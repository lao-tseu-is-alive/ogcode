package provider

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/knights-analytics/hugot"
	"github.com/knights-analytics/hugot/pipelines"
	"github.com/prasenjeet-symon/ogcode/internal/provider/embedmodel"
)

// httpClient is the shared HTTP client used for the one-time model download.
// It has a generous timeout because the ONNX weights are ~86 MB.
var httpClient = &http.Client{Timeout: 10 * time.Minute}

// LocalEmbedder is a provider.Embedder implementation that runs a
// sentence-embedding model entirely in-process — no API key, no network
// call (after the one-time model download), no separate model server.
//
// The small tokenizer/config assets are embedded in the binary (see
// internal/provider/embedmodel) and lazily materialized to a cache directory
// on first use. The large ONNX weight file (~86 MB) is downloaded from
// Hugging Face on first use rather than embedded, so the distributable binary
// stays small — mirroring ogcode's search-bridge download pattern.
//
// It uses Hugot's pure-Go backend (GoMLX simplego), so the binary stays
// CGO-free and self-contained. The default model is
// sentence-transformers/all-MiniLM-L6-v2, producing 384-dim embeddings.
type LocalEmbedder struct {
	id      string
	model   string
	baseDir string // cache directory where the model is materialized

	once     sync.Once
	initErr  error
	session  *hugot.Session
	pipeline *pipelines.FeatureExtractionPipeline
	mu       sync.Mutex // serializes inference calls (the Go backend is not goroutine-safe)
}

// EmbedModelID is the default model identifier returned by EmbedModel().
const EmbedModelID = "all-MiniLM-L6-v2"

const localProviderID = "local"

// NewLocalEmbedder creates a LocalEmbedder. cacheDir overrides the default
// materialization location (~/.ogcode/embed-model) when non-empty.
func NewLocalEmbedder(cacheDir string) *LocalEmbedder {
	if cacheDir == "" {
		if env := os.Getenv("OGCODE_EMBED_MODEL_DIR"); env != "" {
			cacheDir = env
		} else if home, err := os.UserHomeDir(); err == nil && home != "" {
			cacheDir = filepath.Join(home, ".ogcode", "embed-model")
		} else {
			cacheDir = filepath.Join(os.TempDir(), "ogcode-embed-model")
		}
	}
	return &LocalEmbedder{
		id:      localProviderID,
		model:   embedmodel.ModelName,
		baseDir: cacheDir,
	}
}

// init materializes the embedded tokenizer assets to baseDir, ensures the ONNX
// model is present (downloading it on first use), and builds the Hugot
// pipeline. It is idempotent (guarded by once) and safe for concurrent use.
func (e *LocalEmbedder) init(ctx context.Context) error {
	e.once.Do(func() {
		e.initErr = e.initLocked(ctx)
	})
	return e.initErr
}

// Prefetch kicks off the lazy initialization in a background goroutine so the
// (potentially slow) one-time model download happens at server startup rather
// than blocking the first memory-related agent turn. It is fire-and-forget:
// the goroutine runs init under the sync.Once guard, so any later Embed call
// either joins the in-flight init (the Once serializes them) or returns
// immediately once it has completed. Errors are logged but never surfaced to
// the caller — the next Embed call will return them if init failed.
//
// The supplied context governs the download; callers should pass a long-lived
// context (e.g. context.Background()) since the ~86 MB download can take a
// while on slow connections.
func (e *LocalEmbedder) Prefetch(ctx context.Context) {
	go func() {
		t0 := time.Now()
		if err := e.init(ctx); err != nil {
			slog.Warn("local embedder: background prefetch failed; will retry on next Embed", "err", err, "duration", time.Since(t0))
			return
		}
		slog.Info("local embedder: background prefetch complete", "duration", time.Since(t0))
	}()
}

func (e *LocalEmbedder) initLocked(ctx context.Context) error {
	t0 := time.Now()
	if err := os.MkdirAll(e.baseDir, 0o755); err != nil {
		e.initErr = fmt.Errorf("local embedder: create cache dir: %w", err)
		return e.initErr
	}

	// Materialize the small embedded assets (tokenizer/config). These are tiny
	// (~700 KB) so we always write them; they double as a freshness marker.
	slog.Info("local embedder: materializing embedded tokenizer assets", "dir", e.baseDir)
	for _, name := range embedmodel.AssetNames {
		dest := filepath.Join(e.baseDir, name)
		if err := os.WriteFile(dest, embedmodel.ReadAsset(name), 0o644); err != nil {
			return fmt.Errorf("local embedder: write %s: %w", name, err)
		}
	}

	// Ensure the large ONNX model is present. Download it on first use; verify
	// integrity by SHA-256. A sidecar marker records the verified hash so
	// subsequent runs skip both the download and the (relatively expensive)
	// full-file hash check.
	modelPath := filepath.Join(e.baseDir, embedmodel.ModelFileName)
	markerPath := filepath.Join(e.baseDir, ".ogcode-model.sha256")
	if existing, err := os.ReadFile(markerPath); err == nil &&
		string(existing) == embedmodel.ModelSHA256 {
		if _, statErr := os.Stat(modelPath); statErr == nil {
			slog.Info("local embedder: using cached model", "dir", e.baseDir)
			return e.buildPipeline(ctx)
		}
	}

	if err := e.ensureModel(ctx, modelPath); err != nil {
		return err
	}
	if err := os.WriteFile(markerPath, []byte(embedmodel.ModelSHA256), 0o644); err != nil {
		slog.Warn("local embedder: failed to write model marker", "err", err)
	}
	slog.Info("local embedder: model ready", "duration", time.Since(t0))

	return e.buildPipeline(ctx)
}

// ensureModel downloads the ONNX weights to modelPath, streaming to a temp
// file and verifying the SHA-256 before atomically renaming into place. It
// respects context cancellation.
func (e *LocalEmbedder) ensureModel(ctx context.Context, modelPath string) error {
	slog.Info("local embedder: downloading model weights", "url", embedmodel.ModelURL, "dir", e.baseDir)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, embedmodel.ModelURL, nil)
	if err != nil {
		return fmt.Errorf("local embedder: build download request: %w", err)
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("local embedder: download model: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("local embedder: download model: HTTP %d", resp.StatusCode)
	}

	tmpPath := modelPath + ".part"
	f, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("local embedder: create temp model file: %w", err)
	}

	hasher := sha256.New()
	start := time.Now()
	n, err := io.Copy(io.MultiWriter(f, hasher), resp.Body)
	if err != nil {
		f.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("local embedder: write model: %w", err)
	}
	if err := f.Close(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("local embedder: close model file: %w", err)
	}

	got := hex.EncodeToString(hasher.Sum(nil))
	if got != embedmodel.ModelSHA256 {
		os.Remove(tmpPath)
		return fmt.Errorf("local embedder: model checksum mismatch: got %s, want %s", got, embedmodel.ModelSHA256)
	}
	if err := os.Rename(tmpPath, modelPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("local embedder: move model into place: %w", err)
	}
	slog.Info("local embedder: model downloaded",
		"bytes", n, "duration", time.Since(start), "path", modelPath)
	return nil
}

func (e *LocalEmbedder) buildPipeline(ctx context.Context) error {
	session, err := hugot.NewGoSession(ctx)
	if err != nil {
		return fmt.Errorf("local embedder: create session: %w", err)
	}
	e.session = session

	pipeline, err := hugot.NewPipeline(session, hugot.FeatureExtractionConfig{
		ModelPath: e.baseDir,
		Name:      "ogcode-embeddings",
	})
	if err != nil {
		_ = session.Destroy()
		e.session = nil
		return fmt.Errorf("local embedder: build pipeline: %w", err)
	}
	e.pipeline = pipeline
	slog.Info("local embedder ready", "model", e.model, "dim", embedmodel.EmbeddingDim, "dir", e.baseDir)
	return nil
}

// Embed returns embedding vectors for the given inputs. It lazily initializes
// the model on first call. Inference is serialized because the pure-Go GoMLX
// backend is not safe for concurrent use from multiple goroutines.
func (e *LocalEmbedder) Embed(ctx context.Context, inputs []string) ([][]float32, error) {
	if len(inputs) == 0 {
		return nil, nil
	}
	if err := e.init(ctx); err != nil {
		return nil, err
	}
	e.mu.Lock()
	defer e.mu.Unlock()

	result, err := e.pipeline.RunPipeline(ctx, inputs)
	if err != nil {
		return nil, fmt.Errorf("local embedder: inference: %w", err)
	}
	vecs := make([][]float32, len(inputs))
	for i, emb := range result.Embeddings {
		// Guarantee the declared dimension even if the model returns a padded slice.
		if len(emb) >= embedmodel.EmbeddingDim {
			vecs[i] = emb[:embedmodel.EmbeddingDim]
		} else {
			vecs[i] = emb
		}
	}
	return vecs, nil
}

// EmbedModel returns the identifier of the model used for embeddings.
func (e *LocalEmbedder) EmbedModel() string {
	return e.model
}

// --- provider.Provider (minimal, so it can live in the registry) ---

// ID returns the provider identifier.
func (e *LocalEmbedder) ID() string { return e.id }

// Models returns an empty model list: the local embedder is embedding-only
// and not selectable as a chat provider.
func (e *LocalEmbedder) Models() []ModelInfo { return nil }

// StreamChat is not supported — the local embedder is embedding-only.
func (e *LocalEmbedder) StreamChat(ctx context.Context, req StreamRequest) (<-chan StreamEvent, error) {
	return nil, fmt.Errorf("local embedder does not support chat")
}

// Close releases the Hugot session and model resources. Safe to call
// multiple times.
func (e *LocalEmbedder) Close() error {
	if e.session != nil {
		return e.session.Destroy()
	}
	return nil
}