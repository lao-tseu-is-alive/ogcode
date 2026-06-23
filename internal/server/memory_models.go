package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/prasenjeet-symon/ogcode/internal/provider"
	"github.com/prasenjeet-symon/ogcode/internal/session"
)

// handleMemoryModels returns available model IDs for a given provider and type (embed|chat).
// Query params: provider, type, apiKey (optional — backend falls back to stored DB key),
// baseUrl (optional — backend falls back to stored DB config or env var).
func (s *Server) handleMemoryModels(w http.ResponseWriter, r *http.Request) {
	providerID := r.URL.Query().Get("provider")
	modelType := r.URL.Query().Get("type")
	apiKey := r.URL.Query().Get("apiKey")
	baseURL := r.URL.Query().Get("baseUrl")

	if providerID == "" {
		writeError(w, http.StatusBadRequest, "provider is required")
		return
	}
	if modelType != "embed" && modelType != "chat" {
		writeError(w, http.StatusBadRequest, "type must be 'embed' or 'chat'")
		return
	}

	// Use stored key as fallback when none is sent (means "keep existing").
	if cfg, err := session.GetMemoryConfig(s.db); err == nil {
		if modelType == "embed" {
			if apiKey == "" {
				apiKey = cfg.EmbedAPIKey
			}
			if baseURL == "" {
				baseURL = cfg.EmbedBaseURL
			}
		} else {
			if apiKey == "" {
				apiKey = cfg.ChatAPIKey
			}
			if baseURL == "" {
				baseURL = cfg.ChatBaseURL
			}
		}
	}

	models, err := fetchProviderModels(r.Context(), providerID, modelType, apiKey, baseURL)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, models)
}

// fetchProviderModels dispatches to the right fetcher for the given provider.
// baseURL overrides the env-var default when non-empty.
func fetchProviderModels(ctx context.Context, providerID, modelType, apiKey, baseURL string) ([]string, error) {
	switch providerID {
	case "local":
		// The built-in embedder ships a fixed model; there is nothing to fetch.
		// Returning the model name keeps the model picker non-empty for users
		// who switch providers and switch back.
		return []string{provider.EmbedModelID}, nil

	case "anthropic":
		if modelType == "embed" {
			return nil, fmt.Errorf("Anthropic does not support text embeddings")
		}
		return []string{
			"claude-sonnet-4-6",
			"claude-opus-4-7",
			"claude-haiku-4-5-20251001",
			"claude-sonnet-4-20250514",
			"claude-opus-4-20250514",
		}, nil

	case "openai":
		if baseURL == "" {
			baseURL = os.Getenv("OPENAI_BASE_URL")
		}
		if baseURL == "" {
			baseURL = "https://api.openai.com/v1"
		}
		return fetchOpenAIStyleModels(ctx, baseURL, apiKey, modelType)

	case "openrouter":
		return fetchOpenAIStyleModels(ctx, "https://openrouter.ai/api/v1", apiKey, modelType)

	case "ollama":
		if baseURL == "" {
			baseURL = os.Getenv("OLLAMA_BASE_URL")
		}
		if baseURL == "" {
			baseURL = "http://localhost:11434/v1"
		}
		return fetchOpenAIStyleModels(ctx, baseURL, apiKey, modelType)

	default:
		return nil, fmt.Errorf("unknown provider %q", providerID)
	}
}

// fetchOpenAIStyleModels hits the /v1/models endpoint (works for OpenAI, OpenRouter, Ollama).
func fetchOpenAIStyleModels(ctx context.Context, baseURL, apiKey, modelType string) ([]string, error) {
	url := strings.TrimRight(baseURL, "/") + "/models"

	reqCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch models: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("provider returned HTTP %d", resp.StatusCode)
	}

	var result struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	var models []string
	for _, m := range result.Data {
		if modelType == "embed" && isEmbedModelID(m.ID) {
			models = append(models, m.ID)
		} else if modelType == "chat" && isChatModelID(m.ID) {
			models = append(models, m.ID)
		}
	}

	// If nothing matched the filter, return the full list so the user still has options.
	if len(models) == 0 {
		for _, m := range result.Data {
			models = append(models, m.ID)
		}
	}

	sort.Strings(models)
	return models, nil
}

func isEmbedModelID(id string) bool {
	return strings.Contains(strings.ToLower(id), "embed")
}

func isChatModelID(id string) bool {
	lower := strings.ToLower(id)
	// Exclude well-known non-chat model types.
	excluded := []string{"embed", "whisper", "tts", "dall-e", "babbage", "davinci-00", "ada-", "curie", "transcribe", "realtime", "moderation", "search"}
	for _, e := range excluded {
		if strings.Contains(lower, e) {
			return false
		}
	}
	return true
}
