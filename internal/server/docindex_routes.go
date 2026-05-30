package server

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/prasenjeet-symon/ogcode/internal/docindex"
	"github.com/prasenjeet-symon/ogcode/internal/indexer"
)

func (s *Server) handleDocIndexBuildStatus(w http.ResponseWriter, r *http.Request) {
	s.docindexMu.Lock()
	running := s.docindexRunning
	s.docindexMu.Unlock()
	writeJSON(w, http.StatusOK, map[string]any{"running": running})
}

func (s *Server) handleBuildDocIndex(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Directory string `json:"directory"`
		Rebuild   bool   `json:"rebuild"`
		Model     string `json:"model,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	dir := input.Directory
	if dir == "" {
		dir = s.dir
	}

	s.docindexMu.Lock()
	if s.docindexRunning {
		s.docindexMu.Unlock()
		writeJSON(w, http.StatusConflict, map[string]any{"running": true})
		return
	}
	s.docindexRunning = true
	s.docindexMu.Unlock()

	if input.Rebuild {
		if err := s.docindexStore.DeleteAllByPrefix(dir); err != nil {
			s.docindexMu.Lock()
			s.docindexRunning = false
			s.docindexMu.Unlock()
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	excludes, err := s.docindexStore.ListExcludes(dir)
	if err != nil {
		slog.Warn("fetch excludes failed, indexing without them", "err", err)
	}
	var excludePatterns []string
	for _, e := range excludes {
		excludePatterns = append(excludePatterns, e.Pattern)
	}

	go func() {
		defer func() {
			s.docindexMu.Lock()
			s.docindexRunning = false
			s.docindexMu.Unlock()
			s.bus.Publish("docindex.built", map[string]string{"directory": dir})
		}()

		idx := indexer.New(dir, s.docindexStore, s.loopRunner).WithExcludes(excludePatterns)
		if input.Model != "" {
			idx = idx.WithModel(input.Model)
		}
		if err := idx.Run(context.Background()); err != nil {
			slog.Error("docindex build failed", "dir", dir, "err", err)
		}
	}()

	writeJSON(w, http.StatusAccepted, map[string]any{"running": true})
}

func (s *Server) handleListIndexedDocs(w http.ResponseWriter, r *http.Request) {
	dir := r.URL.Query().Get("directory")
	if dir == "" {
		dir = s.dir
	}
	docs, err := s.docindexStore.ListDocsSummary(dir)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if docs == nil {
		docs = []*docindex.DocSummary{}
	}
	writeJSON(w, http.StatusOK, docs)
}

func (s *Server) handleListExcludes(w http.ResponseWriter, r *http.Request) {
	dir := r.URL.Query().Get("directory")
	if dir == "" {
		dir = s.dir
	}
	if err := s.docindexStore.SeedDefaultExcludes(dir); err != nil {
		slog.Warn("seed excludes failed", "err", err)
	}
	entries, err := s.docindexStore.ListExcludes(dir)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if entries == nil {
		entries = []*docindex.ExcludeEntry{}
	}
	writeJSON(w, http.StatusOK, entries)
}

func (s *Server) handleAddExclude(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Directory string `json:"directory"`
		Pattern   string `json:"pattern"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	if input.Pattern == "" {
		http.Error(w, "pattern is required", http.StatusBadRequest)
		return
	}
	if input.Directory == "" {
		input.Directory = s.dir
	}
	e, err := s.docindexStore.AddExclude(input.Directory, input.Pattern)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, e)
}

func (s *Server) handleDeleteExclude(w http.ResponseWriter, r *http.Request) {
	if err := s.docindexStore.DeleteExclude(chi.URLParam(r, "id")); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
