package server

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/prasenjeet-symon/ogcode/internal/session"
)

var knownProviders = []string{"anthropic", "openai", "openrouter", "ollama"}

// handleGetProviderConfigs returns masked credentials for all known providers.
func (s *Server) handleGetProviderConfigs(w http.ResponseWriter, r *http.Request) {
	var out []*session.ProviderConfig
	for _, id := range knownProviders {
		cfg, err := session.GetProviderConfig(s.db, id)
		if err != nil {
			http.Error(w, "failed to read provider config", http.StatusInternalServerError)
			return
		}
		out = append(out, session.MaskedProviderConfig(cfg))
	}
	writeJSON(w, http.StatusOK, out)
}

// handleSetProviderConfig upserts credentials for a single provider.
func (s *Server) handleSetProviderConfig(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var incoming session.ProviderConfig
	if err := json.NewDecoder(r.Body).Decode(&incoming); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	incoming.ProviderID = id

	// Preserve existing API key when the sentinel is sent.
	existing, err := session.GetProviderConfig(s.db, id)
	if err != nil {
		http.Error(w, "failed to read provider config", http.StatusInternalServerError)
		return
	}
	if incoming.APIKey == "__SET__" {
		incoming.APIKey = existing.APIKey
	}

	if err := session.SetProviderConfig(s.db, &incoming); err != nil {
		http.Error(w, "failed to save provider config", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, session.MaskedProviderConfig(&incoming))
}
