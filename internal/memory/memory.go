package memory

import (
	"context"
	"encoding/binary"
	"fmt"
	"log/slog"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/prasenjeet-symon/ogcode/internal/provider"
)

// Memory provides the agentic memory lifecycle: read, recall, and write.
// It wraps a local SQLite-backed knowledge graph with optional LLM inference.
type Memory struct {
	Store  *Store
	Graph  *Graph
	enabled bool
}

// GraphOpts holds dependencies for initializing agentic memory.
type GraphOpts struct {
	// ChatProvider is the provider used for topic/concept inference and recall.
	// It may be the same as EmbedProvider or different.
	ChatProvider provider.Provider
	// ChatModel is the specific model ID to use for inference. If empty, the
	// provider's first available model is used.
	ChatModel string
	// EmbedProvider is the provider used for text embeddings. Must satisfy provider.Embedder.
	EmbedProvider provider.Provider
}

// New creates a Memory backed by local SQLite graph store.
// If chatProvider is nil but embedProvider is non-nil, AddFact can still store
// facts without LLM topic inference.
func New(store *Store, opts *GraphOpts) *Memory {
	m := &Memory{Store: store}
	if store != nil {
		m.enabled = true
		m.Graph = &Graph{Store: store}
		if opts != nil {
			if opts.ChatProvider != nil {
				m.Graph.Chat = NewChatClient(opts.ChatProvider, opts.ChatModel)
			}
			if opts.EmbedProvider != nil {
				if e, ok := opts.EmbedProvider.(provider.Embedder); ok {
					m.Graph.Embed = NewEmbedClient(e)
				}
			}
		}
	}
	if m.enabled {
		if m.Graph.Embed == nil {
			slog.Warn("agentic memory: no embedder configured — semantic recall unavailable")
		} else {
			slog.Info("agentic memory enabled", "chatProvider", func() string {
				if opts != nil && opts.ChatProvider != nil {
					return opts.ChatProvider.ID()
				}
				return "none"
			}(), "embedProvider", func() string {
				if opts != nil && opts.EmbedProvider != nil {
					return opts.EmbedProvider.ID()
				}
				return "none"
			}())
		}
	}
	return m
}

// Enabled returns whether agentic memory is active.
func (m *Memory) Enabled() bool {
	return m.enabled && m.Graph != nil
}

// PrefetchEmbedder kicks off background initialization of the configured
// embedder when it supports lazy prefetching (e.g. the inbuilt LocalEmbedder,
// which downloads ~86 MB of model weights on first use). This is a no-op for
// embedders that do not implement prefetcher, so it is safe to call
// unconditionally at startup.
func (m *Memory) PrefetchEmbedder(ctx context.Context) {
	if m == nil || m.Graph == nil || m.Graph.Embed == nil {
		return
	}
	if c, ok := m.Graph.Embed.(*embedClient); ok {
		if p, ok := c.e.(interface{ Prefetch(context.Context) }); ok {
			p.Prefetch(ctx)
		}
	}
}

// ReadMemory fetches the full session knowledge graph as text.
func (m *Memory) ReadMemory(ctx context.Context, sessionID string) string {
	if !m.Enabled() {
		return ""
	}
	_ = m.Store.EnsureSession(sessionID)

	tree, facts, err := m.Graph.BuildLightweightTree(ctx, sessionID, NodeFilter{}, nil, 0)
	if err != nil {
		slog.Warn("BuildLightweightTree failed", "err", err)
		return ""
	}
	if len(facts) == 0 {
		slog.Info("memory graph empty", "session", sessionID)
		return ""
	}
	result := skeletonTreeText(tree)
	if strings.TrimSpace(result) == "" {
		return ""
	}
	slog.Info("memory graph returned context", "session", sessionID, "len", len(result))
	return result
}

// RecallMemory performs semantic recall for a specific question.
func (m *Memory) RecallMemory(ctx context.Context, sessionID, question string) string {
	if !m.Enabled() {
		return ""
	}
	if m.Graph.Embed == nil {
		slog.Warn("RecallMemory: no embedder configured")
		return m.ReadMemory(ctx, sessionID)
	}
	_ = m.Store.EnsureSession(sessionID)

	result, err := m.Graph.Recall(ctx, RecallOptions{
		SessionID: sessionID,
		Question:  question,
		Limit:     50,
		MaxRounds: 3,
		Threshold: 0.7,
	})
	if err != nil {
		slog.Warn("memory recall failed", "err", err)
		return ""
	}

	var display string
	if result.Confidence > 0 {
		display = fmt.Sprintf("[confidence: %.0f%% | rounds: %d | facts used: %d]\n\n%s",
			result.Confidence*100, result.Rounds, result.FactsUsed, result.Answer)
	} else {
		display = result.Answer
	}
	slog.Info("memory recall returned context", "session", sessionID, "len", len(display))
	return display
}

// WriteMemory persists a conversation turn.
func (m *Memory) WriteMemory(ctx context.Context, sessionID, question, response string) {
	if !m.Enabled() {
		return
	}
	go func() {
		bgCtx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		defer cancel()

		_, err := m.Graph.AddFact(bgCtx, GraphOptions{
			SessionID: sessionID,
			Question:  question,
			Response:  response,
		})
		if err != nil {
			slog.Warn("memory_add call failed", "err", err)
		} else {
			slog.Info("memory_add succeeded", "session", sessionID)
		}
	}()
}

// DefaultDBPath returns the default path for the memory database.
func DefaultDBPath() string {
	home, _ := os.UserHomeDir()
	if home == "" {
		home = os.TempDir()
	}
	if p := os.Getenv("OGCODE_MEMORY_DB_PATH"); p != "" {
		return p
	}
	return filepath.Join(home, ".ogcode", "memory.db")
}

// DocumentDefaultCollection is the fallback collection name.
const DocumentDefaultCollection = "default"

// Document is an unstructured text fragment tied to a collection.
type Document struct {
	ID         int64  `json:"id"`
	Collection string `json:"collection"`
	Content    string `json:"content"`
	CreatedAt  int64  `json:"createdAt"`
}

// SearchResult pairs a Document with a relevance score.
type SearchResult struct {
	Doc   Document `json:"doc"`
	Score float32  `json:"score"`
}

// CollectionStats is returned by Stats.
type CollectionStats struct {
	Collections int `json:"collections"`
	Documents   int `json:"documents"`
	Nodes       int `json:"nodes"`
	Edges       int `json:"edges"`
}

// Stats returns total counts across all collections and graph tables.
func (m *Memory) Stats(ctx context.Context) (col, doc, nodes, edges int, err error) {
	if m.Store == nil {
		return 0, 0, 0, 0, nil
	}
	if err = m.Store.DB().QueryRowContext(ctx, `SELECT COUNT(*) FROM memory_collection`).Scan(&col); err != nil {
		return
	}
	if err = m.Store.DB().QueryRowContext(ctx, `SELECT COUNT(*) FROM memory_document`).Scan(&doc); err != nil {
		return
	}
	if err = m.Store.DB().QueryRowContext(ctx, `SELECT COUNT(*) FROM nodes`).Scan(&nodes); err != nil {
		return
	}
	if err = m.Store.DB().QueryRowContext(ctx, `SELECT COUNT(*) FROM edges`).Scan(&edges); err != nil {
		return
	}
	return
}

// CreateCollection inserts a new collection.
func (m *Memory) CreateCollection(ctx context.Context, name string) (int64, error) {
	now := time.Now().UnixMilli()
	res, err := m.Store.DB().ExecContext(ctx,
		`INSERT INTO memory_collection (name, created_at) VALUES (?, ?)`,
		name, now)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// DeleteCollection removes a collection including its documents.
func (m *Memory) DeleteCollection(ctx context.Context, name string) error {
	_, err := m.Store.DB().ExecContext(ctx, `DELETE FROM memory_document WHERE collection = ?`, name)
	if err != nil {
		return err
	}
	_, err = m.Store.DB().ExecContext(ctx, `DELETE FROM memory_collection WHERE name = ?`, name)
	return err
}

// UpsertDocument stores or updates a document and computes its embedding using the currently configured embedder.
func (m *Memory) UpsertDocument(ctx context.Context, collection, content string) (int64, error) {
	if collection == "" {
		collection = DocumentDefaultCollection
	}
	var emb []float32
	var embBlob []byte
	if m.Graph != nil && m.Graph.Embed != nil {
		vecs, err := m.Graph.Embed.Embed(ctx, []string{content})
		if err != nil {
			return 0, fmt.Errorf("embedding failed: %w", err)
		}
		if len(vecs) > 0 {
			emb = vecs[0]
			embBlob = floatsToBytes(emb)
		}
	}
	now := time.Now().UnixMilli()
	res, err := m.Store.DB().ExecContext(ctx,
		`INSERT INTO memory_document (collection, content, embedding, created_at)
		 VALUES (?, ?, ?, ?)
		 ON CONFLICT(collection, content) DO UPDATE SET
			 content = excluded.content,
			 embedding = excluded.embedding,
			 created_at = excluded.created_at`,
		collection, content, embBlob, now)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// floatsToBytes converts a []float32 to a []byte (little-endian).
func floatsToBytes(f []float32) []byte {
	buf := make([]byte, len(f)*4)
	for i, v := range f {
		binary.LittleEndian.PutUint32(buf[i*4:], math.Float32bits(v))
	}
	return buf
}

// bytesToFloats converts a []byte to a []float32 (little-endian).
func bytesToFloats(b []byte) []float32 {
	if len(b)%4 != 0 {
		return nil
	}
	out := make([]float32, len(b)/4)
	for i := range out {
		out[i] = math.Float32frombits(binary.LittleEndian.Uint32(b[i*4:]))
	}
	return out
}

// SemanticSearch runs a vector search across a collection.
func (m *Memory) SemanticSearch(ctx context.Context, collection, query string, topK int) ([]SearchResult, error) {
	if m.Graph == nil || m.Graph.Embed == nil {
		return nil, fmt.Errorf("semantic search unavailable: no embedder")
	}
	if collection == "" {
		collection = DocumentDefaultCollection
	}
	vecs, err := m.Graph.Embed.Embed(ctx, []string{query})
	if err != nil {
		return nil, fmt.Errorf("embedding query failed: %w", err)
	}
	qvec := vecs[0]

	// SQLite does not have a native vector index, so we do brute-force cosine similarity.
	rows, err := m.Store.DB().QueryContext(ctx,
		`SELECT id, content, embedding, created_at FROM memory_document WHERE collection = ?`, collection)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var candidates []SearchResult
	cosineStart := time.Now()
	docsCompared := 0
	for rows.Next() {
		var id int64
		var content string
		var embBlob []byte
		var created int64
		if err := rows.Scan(&id, &content, &embBlob, &created); err != nil {
			continue
		}
		dvec := bytesToFloats(embBlob)
		if dvec == nil || len(dvec) != len(qvec) {
			continue
		}
		docsCompared++
		score := cosine(qvec, dvec)
		if score < 0 {
			score = 0
		}
		candidates = append(candidates, SearchResult{
			Doc: Document{ID: id, Collection: collection, Content: content, CreatedAt: created},
			Score: score,
		})
	}
	slog.Info("cosine similarity timing (SemanticSearch)",
		"docs_compared", docsCompared,
		"duration", time.Since(cosineStart),
	)
	_ = rows.Err()

	sort.Slice(candidates, func(i, j int) bool { return candidates[i].Score > candidates[j].Score })
	if len(candidates) > topK {
		candidates = candidates[:topK]
	}
	return candidates, nil
}

// RefreshAll recomputes all document embeddings; useful after provider changes.
func (m *Memory) RefreshAll(ctx context.Context) error {
	if m.Graph == nil || m.Graph.Embed == nil {
		return fmt.Errorf("refresh unavailable: no embedder")
	}
	rows, err := m.Store.DB().QueryContext(ctx, `SELECT id, content FROM memory_document`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		var content string
		if err := rows.Scan(&id, &content); err != nil {
			continue
		}
		vecs, err := m.Graph.Embed.Embed(ctx, []string{content})
		if err != nil {
			continue
		}
		if len(vecs) == 0 {
			continue
		}
		embBlob := floatsToBytes(vecs[0])
		_, _ = m.Store.DB().ExecContext(ctx,
			`UPDATE memory_document SET embedding = ? WHERE id = ?`,
			embBlob, id)
	}
	return rows.Err()
}
