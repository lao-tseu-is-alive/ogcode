package server

import (
	"encoding/json"
	"net/http"
	"os"
	"os/exec"

	"github.com/prasenjeet-symon/ogcode/internal/session"
)

func (s *Server) handlePath(w http.ResponseWriter, r *http.Request) {
	home, _ := os.UserHomeDir()
	writeJSON(w, http.StatusOK, map[string]string{
		"home":      home,
		"directory": s.dir,
		"state":     s.dir + "/.ogcode",
	})
}

func (s *Server) handleAgents(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, []map[string]string{
		{"id": "build", "name": "Build", "description": "Full-access coding agent"},
		{"id": "plan", "name": "Plan", "description": "Planning agent — reads and understands code, plans changes but never writes"},
	})
}

func (s *Server) handleMode(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"mode": string(s.mode),
	})
}

func (s *Server) handleModels(w http.ResponseWriter, r *http.Request) {
	builtIn := s.registry.ListModels()

	prefs, _ := session.GetModelPreferences(s.db)
	prefMap := make(map[string]*session.ModelPreference)
	for _, p := range prefs {
		prefMap[p.ID] = p
	}

	availableProviders := make(map[string]bool)
	for _, id := range s.registry.List() {
		availableProviders[id] = true
	}

	type ModelEntry struct {
		ID         string `json:"id"`
		Name       string `json:"name"`
		ProviderID string `json:"providerId"`
		Default    bool   `json:"default"`
		Enabled    bool   `json:"enabled"`
		IsCustom   bool   `json:"isCustom"`
	}

	var result []ModelEntry
	for _, m := range builtIn {
		entry := ModelEntry{
			ID:         m.ID,
			Name:       m.Name,
			ProviderID: m.ProviderID,
			Default:    m.Default,
			Enabled:    true,
			IsCustom:   false,
		}
		if pref, ok := prefMap[m.ID]; ok {
			entry.Enabled = pref.Enabled
		}
		result = append(result, entry)
	}

	for _, p := range prefs {
		if !p.IsCustom {
			continue
		}
		if !availableProviders[p.ProviderID] && !p.Enabled {
			continue
		}
		result = append(result, ModelEntry{
			ID:         p.ID,
			Name:       p.DisplayName,
			ProviderID: p.ProviderID,
			Default:    false,
			Enabled:    p.Enabled,
			IsCustom:   true,
		})
	}

	if result == nil {
		result = []ModelEntry{}
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) configPayload() map[string]any {
	memoryEnabled := s.mem != nil && s.mem.Enabled()
	memoryProvider := ""
	if s.mem != nil && s.mem.Graph != nil && s.mem.Graph.Embed != nil {
		memoryProvider = "ogcode-embedded"
	}
	mcpEnabled := s.mcpClient != nil
	mcpProvider := ""
	if mcpEnabled {
		mcpProvider = s.mcpCfg.Command
	}
	return map[string]any{
		"directory":      s.dir,
		"port":           s.port,
		"memoryEnabled":  memoryEnabled,
		"memoryProvider": memoryProvider,
		"mcpEnabled":     mcpEnabled,
		"mcpProvider":    mcpProvider,
	}
}

func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.configPayload())
}

func (s *Server) handleGetMemoryConfig(w http.ResponseWriter, r *http.Request) {
	cfg, err := session.GetMemoryConfig(s.db)
	if err != nil {
		http.Error(w, "failed to read memory config", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, session.MaskedMemoryConfig(cfg))
}

func (s *Server) handleSetMemoryConfig(w http.ResponseWriter, r *http.Request) {
	var incoming session.MemoryConfig
	if err := json.NewDecoder(r.Body).Decode(&incoming); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Preserve existing API keys when the sentinel "__SET__" is sent.
	existing, err := session.GetMemoryConfig(s.db)
	if err != nil {
		http.Error(w, "failed to read memory config", http.StatusInternalServerError)
		return
	}
	if incoming.EmbedAPIKey == "__SET__" {
		incoming.EmbedAPIKey = existing.EmbedAPIKey
	}
	if incoming.ChatAPIKey == "__SET__" {
		incoming.ChatAPIKey = existing.ChatAPIKey
	}

	if err := session.SetMemoryConfig(s.db, &incoming); err != nil {
		http.Error(w, "failed to save memory config", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, session.MaskedMemoryConfig(&incoming))
}

func (s *Server) handleModelsRefresh(w http.ResponseWriter, r *http.Request) {
	s.registry.RefreshModels()
	// Return the updated model list
	s.handleModels(w, r)
}

func (s *Server) handleVCS(w http.ResponseWriter, r *http.Request) {
	branch := getCurrentBranch(s.dir)
	writeJSON(w, http.StatusOK, map[string]string{
		"branch": branch,
	})
}

func getCurrentBranch(dir string) string {
	out, err := execInDir(dir, "git", "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return ""
	}
	return out
}

func execInDir(dir string, name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...) //nolint:gosec
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	result := string(out)
	if len(result) > 0 && result[len(result)-1] == '\n' {
		result = result[:len(result)-1]
	}
	return result, nil
}