package server

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

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

	go func() {
		defer func() {
			s.docindexMu.Lock()
			s.docindexRunning = false
			s.docindexMu.Unlock()
			s.bus.Publish("docindex.built", map[string]string{"directory": dir})
		}()

		idx := indexer.New(dir, s.docindexStore, s.loopRunner)
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
