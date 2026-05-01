package server

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/ogcode/ogcode/internal/session"
)

// handlePermissionReply handles user responses to permission requests
func (s *Server) handlePermissionReply(w http.ResponseWriter, r *http.Request) {
	sessionID := session.SessionID(chi.URLParam(r, "sessionID"))
	permID := chi.URLParam(r, "permissionID")

	var input struct {
		Response string `json:"response"` // "once", "always", "reject"
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	s.bus.Publish("permission.replied", map[string]string{
		"sessionId":    string(sessionID),
		"permissionId": permID,
		"response":     input.Response,
	})

	w.WriteHeader(http.StatusNoContent)
}