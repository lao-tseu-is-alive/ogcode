package indexer

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/prasenjeet-symon/ogcode/internal/agent"
	"github.com/prasenjeet-symon/ogcode/internal/docindex"
	"github.com/prasenjeet-symon/ogcode/internal/session"
)

// skipDirs are directories never worth scanning for documents.
var skipDirs = map[string]struct{}{
	"node_modules": {}, "vendor": {}, ".git": {}, "dist": {}, "build": {},
	"out": {}, "target": {}, "__pycache__": {}, ".venv": {}, "venv": {},
	"env": {}, "coverage": {}, ".next": {}, ".nuxt": {}, ".cache": {}, ".ogcode": {},
}

// Indexer scans a workspace directory for PDF files and runs the IndexAgent
// on each one to produce semantic labels per page.
type Indexer struct {
	dir        string
	model      string // optional model override for the IndexAgent
	excludes   []string
	docStore   *docindex.Store
	loopRunner *agent.LoopRunner
}

// New creates a new Indexer. Pass an empty model to use the runner's default.
func New(dir string, docStore *docindex.Store, lr *agent.LoopRunner) *Indexer {
	return &Indexer{
		dir:        dir,
		docStore:   docStore,
		loopRunner: lr,
	}
}

// WithModel sets the model override for sessions created by the indexer.
func (idx *Indexer) WithModel(model string) *Indexer {
	idx.model = model
	return idx
}

// WithExcludes sets additional patterns to skip during the directory walk.
// Patterns are matched against directory names and file basenames using filepath.Match.
func (idx *Indexer) WithExcludes(patterns []string) *Indexer {
	idx.excludes = patterns
	return idx
}

// isExcluded reports whether a file or directory name matches any user-configured exclude pattern.
func (idx *Indexer) isExcluded(name string) bool {
	for _, pattern := range idx.excludes {
		if pattern == name {
			return true
		}
		if matched, _ := filepath.Match(pattern, name); matched {
			return true
		}
	}
	return false
}

// Run scans dir recursively for PDF and text/code files and indexes each one.
func (idx *Indexer) Run(ctx context.Context) error {
	// filepath.Glob does not support "**" recursion, so walk the tree manually.
	var allFiles []string
	walkErr := filepath.WalkDir(idx.dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // skip unreadable entries, keep walking
		}
		if d.IsDir() {
			if path != idx.dir {
				if _, skip := skipDirs[d.Name()]; skip {
					return filepath.SkipDir
				}
				if idx.isExcluded(d.Name()) {
					return filepath.SkipDir
				}
			}
			return nil
		}
		if idx.isExcluded(filepath.Base(path)) {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext == ".pdf" || IsTextFile(ext) {
			allFiles = append(allFiles, path)
		}
		return nil
	})
	if walkErr != nil {
		return fmt.Errorf("walk files: %w", walkErr)
	}

	if len(allFiles) == 0 {
		slog.Info("no indexable files found", "dir", idx.dir)
		return nil
	}

	slog.Info("found indexable files", "count", len(allFiles))
	for _, filePath := range allFiles {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		indexed, err := idx.docStore.IsDocIndexed(filePath)
		if err != nil {
			slog.Warn("could not check index status, skipping", "path", filePath, "err", err)
			continue
		}
		if indexed {
			slog.Info("skipping already-indexed document", "path", filePath)
			continue
		}
		if err := idx.IndexDocument(ctx, filePath); err != nil {
			slog.Warn("failed to index document", "path", filePath, "err", err)
		}
	}
	return nil
}


// batchSize is the number of pages sent to the IndexAgent per session.
// Compact keyword format keeps each batch well under 50KB.
const batchSize = 100

// IndexDocument extracts text from a PDF or text/code file, builds keyword corpora,
// then runs the IndexAgent in batches to produce labels.
// Labels are upserted directly by the agent tool — no pre-registration needed.
func (idx *Indexer) IndexDocument(ctx context.Context, filePath string) error {
	slog.Info("indexing document", "path", filePath)

	var pages []PageText
	var err error
	if strings.ToLower(filepath.Ext(filePath)) == ".pdf" {
		pages, err = ExtractPages(filePath)
	} else {
		pages, err = ExtractTextFile(filePath)
	}
	if err != nil {
		return fmt.Errorf("extract pages: %w", err)
	}
	if len(pages) == 0 {
		slog.Info("no pages found", "path", filePath)
		return nil
	}

	corpora := BuildCorpora(pages)

	for start := 0; start < len(corpora); start += batchSize {
		end := start + batchSize
		if end > len(corpora) {
			end = len(corpora)
		}
		batch := corpora[start:end]
		slog.Info("indexing batch", "doc", filepath.Base(filePath),
			"pages", fmt.Sprintf("%d-%d", batch[0].PageNum, batch[len(batch)-1].PageNum))
		if err := idx.indexBatch(ctx, filePath, batch); err != nil {
			slog.Warn("batch failed", "doc", filePath,
				"start", batch[0].PageNum, "err", err)
		}
	}

	slog.Info("document indexed", "path", filePath)
	return nil
}

// indexBatch runs a single IndexAgent session for a slice of page corpora.
func (idx *Indexer) indexBatch(ctx context.Context, filePath string, batch []PageCorpus) error {
	// Compact format: one line per page — "page_num: kw1,kw2,kw3"
	// Much smaller than JSON arrays; keeps 100-page batches well under 50KB.
	var sb strings.Builder
	for _, c := range batch {
		kw := c.Keywords
		if len(kw) == 0 {
			kw = []string{"(empty)"}
		}
		fmt.Fprintf(&sb, "%d: %s\n", c.PageNum, strings.Join(kw, ","))
	}
	userText := fmt.Sprintf("Index this document: %s\n\nPages (page_num: keywords):\n%s",
		filePath, sb.String())

	sess := &session.Session{
		ID:          session.NewSessionID(),
		ProjectID:   idx.dir,
		Directory:   idx.dir,
		Title:       fmt.Sprintf("Index: %s (p%d-%d)", filepath.Base(filePath), batch[0].PageNum, batch[len(batch)-1].PageNum),
		Model:       idx.model,
		SessionType: "index",
		CreatedAt:   session.Now(),
		UpdatedAt:   session.Now(),
	}
	if err := idx.loopRunner.Store.Create(sess); err != nil {
		return fmt.Errorf("create session: %w", err)
	}

	userMsg := &session.MessageInfo{
		ID:        session.NewMessageID(),
		SessionID: sess.ID,
		Role:      session.RoleUser,
		Agent:     "index",
		CreatedAt: session.Now(),
	}
	if err := idx.loopRunner.Store.CreateMessage(userMsg); err != nil {
		return fmt.Errorf("create user message: %w", err)
	}
	textData, _ := json.Marshal(session.TextPartData{Text: userText})
	userPart := &session.Part{
		ID:        session.NewPartID(),
		MessageID: userMsg.ID,
		SessionID: sess.ID,
		Type:      session.PartText,
		Data:      textData,
		CreatedAt: session.Now(),
		UpdatedAt: session.Now(),
	}
	if err := idx.loopRunner.Store.CreatePart(userPart); err != nil {
		return fmt.Errorf("create user part: %w", err)
	}

	if err := idx.loopRunner.RunLoop(ctx, sess.ID, "index", 0, 0); err != nil {
		return fmt.Errorf("run index agent: %w", err)
	}
	return nil
}
