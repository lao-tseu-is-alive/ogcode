package server

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/prasenjeet-symon/ogcode/internal/id"
	"github.com/prasenjeet-symon/ogcode/internal/note"
	"github.com/prasenjeet-symon/ogcode/internal/session"
)

func (s *Server) handleListNotes(w http.ResponseWriter, r *http.Request) {
	directory := r.URL.Query().Get("directory")
	if directory == "" {
		directory = s.dir
	}
	notes, err := s.noteStore.List(directory)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if notes == nil {
		notes = []*note.Note{}
	}
	writeJSON(w, http.StatusOK, notes)
}

func (s *Server) handleCreateNote(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Query     string `json:"query"`
		Directory string `json:"directory"`
		Model     string `json:"model,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if input.Query == "" {
		http.Error(w, "query is required", http.StatusBadRequest)
		return
	}
	dir := input.Directory
	if dir == "" {
		dir = s.dir
	}

	// Create a session to run the note agent
	sess := &session.Session{
		ID:          session.NewSessionID(),
		ProjectID:   dir,
		Directory:   dir,
		Title:       "Note: " + truncate(input.Query, 60),
		Model:       input.Model,
		SessionType: "note",
		CreatedAt:   session.Now(),
		UpdatedAt:   session.Now(),
	}
	if err := s.store.Create(sess); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Create the note record linked to the session
	n := &note.Note{
		ID:        id.NewNoteID(),
		Directory: dir,
		Title:     truncate(input.Query, 60),
		Query:     input.Query,
		Content:   "",
		SessionID: string(sess.ID),
		Status:    note.StatusGenerating,
		CreatedAt: note.Now(),
		UpdatedAt: note.Now(),
	}
	if err := s.noteStore.Create(n); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.bus.Publish("note.created", n)

	// Fire agent loop with the query as the first user message
	go func() {
		userMsg := &session.MessageInfo{
			ID:        session.NewMessageID(),
			SessionID: sess.ID,
			Role:      session.RoleUser,
			Agent:     "note",
			CreatedAt: session.Now(),
		}
		if err := s.store.CreateMessage(userMsg); err != nil {
			slog.Error("create note user message", "err", err)
			return
		}
		textData, _ := json.Marshal(session.TextPartData{Text: input.Query})
		userPart := &session.Part{
			ID:        session.NewPartID(),
			MessageID: userMsg.ID,
			SessionID: sess.ID,
			Type:      session.PartText,
			Data:      textData,
			CreatedAt: session.Now(),
			UpdatedAt: session.Now(),
		}
		if err := s.store.CreatePart(userPart); err != nil {
			slog.Error("create note user part", "err", err)
			return
		}
		s.bus.Publish("message.updated", userMsg)

		ctx, cancel := context.WithCancel(context.Background())
		s.mu.Lock()
		s.nextToken++
		token := s.nextToken
		s.running[sess.ID] = cancel
		s.runningToken[sess.ID] = token
		s.mu.Unlock()

		defer func() {
			s.mu.Lock()
			if s.runningToken[sess.ID] == token {
				delete(s.running, sess.ID)
				delete(s.runningToken, sess.ID)
			}
			s.mu.Unlock()
		}()

		if err := s.loopRunner.RunLoop(ctx, sess.ID, "note"); err != nil {
			slog.Error("note agent loop error", "session", sess.ID, "err", err)
		}
	}()

	writeJSON(w, http.StatusCreated, n)
}

func (s *Server) handleGetNote(w http.ResponseWriter, r *http.Request) {
	noteID := chi.URLParam(r, "noteID")
	n, err := s.noteStore.Get(noteID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if n == nil {
		http.Error(w, "note not found", http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, n)
}

func (s *Server) handleListNoteVersions(w http.ResponseWriter, r *http.Request) {
	noteID := chi.URLParam(r, "noteID")
	versions, err := s.noteStore.ListVersions(noteID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if versions == nil {
		versions = []*note.NoteVersion{}
	}
	writeJSON(w, http.StatusOK, versions)
}

func (s *Server) handleDeleteNote(w http.ResponseWriter, r *http.Request) {
	noteID := chi.URLParam(r, "noteID")
	if err := s.noteStore.Delete(noteID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.bus.Publish("note.deleted", map[string]string{"id": noteID})
	w.WriteHeader(http.StatusNoContent)
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}
