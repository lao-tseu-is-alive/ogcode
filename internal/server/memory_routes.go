package server

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/prasenjeet-symon/ogcode/internal/memory"
)

// ---------- helpers ----------

func writeError(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}

// ---------- agentic memory routes ----------

func (s *Server) handleMemoryStatus(w http.ResponseWriter, r *http.Request) {
	if s.mem == nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"enabled":    false,
			"collection": 0,
			"document":   0,
			"node":       0,
			"edge":       0,
		})
		return
	}
	c, d, n, e, err := s.mem.Stats(r.Context())
	if err != nil {
		slog.Warn("memory stats", "err", err)
		writeError(w, http.StatusInternalServerError, "memory stats: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"enabled":    true,
		"collection": c,
		"document":   d,
		"node":       n,
		"edge":       e,
	})
}

func (s *Server) handleMemoryCollections(w http.ResponseWriter, r *http.Request) {
	if s.mem == nil {
		writeJSON(w, http.StatusOK, []any{})
		return
	}
	rows, err := s.mem.Store.DB().QueryContext(r.Context(), `SELECT id, name FROM memory_collection ORDER BY name`)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "collections: "+err.Error())
		return
	}
	defer rows.Close()
	var out []map[string]any
	for rows.Next() {
		var id int64
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			continue
		}
		out = append(out, map[string]any{"id": id, "name": name})
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleMemoryCreateCollection(w http.ResponseWriter, r *http.Request) {
	var req struct{ Name string `json:"name"` }
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		writeError(w, http.StatusBadRequest, "collection name required")
		return
	}
	if s.mem == nil {
		writeError(w, http.StatusServiceUnavailable, "agentic memory not enabled")
		return
	}
	id, err := s.mem.CreateCollection(r.Context(), req.Name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"id": id, "name": req.Name})
}

func (s *Server) handleMemoryDeleteCollection(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Path[len("/api/memory/collections/"):]
	if name == "" {
		writeError(w, http.StatusBadRequest, "collection name required")
		return
	}
	if s.mem == nil {
		writeError(w, http.StatusServiceUnavailable, "agentic memory not enabled")
		return
	}
	if err := s.mem.DeleteCollection(r.Context(), name); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleMemoryDocuments(w http.ResponseWriter, r *http.Request) {
	name := memory.DocumentDefaultCollection
	if q := r.URL.Query().Get("collection"); q != "" {
		name = q
	}
	limit := 100
	if q := r.URL.Query().Get("limit"); q != "" {
		if v, err := strconv.Atoi(q); err == nil && v > 0 {
			limit = v
		}
	}
	if s.mem == nil {
		writeJSON(w, http.StatusOK, []any{})
		return
	}
	rows, err := s.mem.Store.DB().QueryContext(r.Context(),
		`SELECT id, content, created_at FROM memory_document WHERE collection = ? ORDER BY created_at DESC LIMIT ?`,
		name, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "documents: "+err.Error())
		return
	}
	defer rows.Close()
	var out []map[string]any
	for rows.Next() {
		var id int64
		var content string
		var created int64
		if err := rows.Scan(&id, &content, &created); err != nil {
			continue
		}
		out = append(out, map[string]any{"id": id, "content": content, "created": created})
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleMemoryAddDocument(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Content    string `json:"content"`
		Collection string `json:"collection,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Content == "" {
		writeError(w, http.StatusBadRequest, "content required")
		return
	}
	if s.mem == nil {
		writeError(w, http.StatusServiceUnavailable, "agentic memory not enabled")
		return
	}
	if _, err := s.mem.UpsertDocument(r.Context(), req.Collection, req.Content); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (s *Server) handleMemoryDeleteDocument(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Path[len("/api/memory/documents/"):]
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid document id")
		return
	}
	if s.mem == nil {
		writeError(w, http.StatusServiceUnavailable, "agentic memory not enabled")
		return
	}
	if _, err := s.mem.Store.DB().ExecContext(r.Context(), `DELETE FROM memory_document WHERE id = ?`, id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleMemoryChat(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Collection string `json:"collection,omitempty"`
		Query      string `json:"query"`
		TopK       int    `json:"topK,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Query == "" {
		writeError(w, http.StatusBadRequest, "query required")
		return
	}
	if s.mem == nil {
		writeError(w, http.StatusServiceUnavailable, "agentic memory not enabled")
		return
	}
	docs, err := s.mem.SemanticSearch(r.Context(), req.Collection, req.Query, req.TopK)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Generate LLM response using memory context
	var results []map[string]any
	for _, d := range docs {
		results = append(results, map[string]any{
			"content": d.Doc.Content,
			"score":   d.Score,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"query":   req.Query,
		"results": results,
		"time":    time.Now().UTC(),
	})
}

func (s *Server) handleMemorySimilar(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	query := q.Get("query")
	coll := q.Get("collection")
	topK := 5
	if v, err := strconv.Atoi(q.Get("topK")); err == nil && v > 0 {
		topK = v
	}
	if query == "" {
		writeError(w, http.StatusBadRequest, "query parameter required")
		return
	}
	if s.mem == nil {
		writeError(w, http.StatusServiceUnavailable, "agentic memory not enabled")
		return
	}
	docs, err := s.mem.SemanticSearch(r.Context(), coll, query, topK)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	var out []map[string]any
	for _, d := range docs {
		out = append(out, map[string]any{
			"id":        d.Doc.ID,
			"content":   d.Doc.Content,
			"score":     d.Score,
			"createdAt": d.Doc.CreatedAt,
		})
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleMemoryReindex(w http.ResponseWriter, r *http.Request) {
	if s.mem == nil {
		writeError(w, http.StatusServiceUnavailable, "agentic memory not enabled")
		return
	}
	if err := s.mem.RefreshAll(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

func (s *Server) handleMemoryReset(w http.ResponseWriter, r *http.Request) {
	if s.mem == nil {
		writeError(w, http.StatusServiceUnavailable, "agentic memory not enabled")
		return
	}
	if _, err := s.mem.Store.DB().ExecContext(r.Context(), `DELETE FROM memory_document`); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if _, err := s.mem.Store.DB().ExecContext(r.Context(), `DELETE FROM memory_node`); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if _, err := s.mem.Store.DB().ExecContext(r.Context(), `DELETE FROM memory_edge`); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if _, err := s.mem.Store.DB().ExecContext(r.Context(), `DELETE FROM memory_collection`); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}
