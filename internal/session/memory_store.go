package session

import (
	"database/sql"
	"fmt"

	"github.com/prasenjeet-symon/ogcode/internal/db"
)

// MemoryConfig holds the user's agentic-memory configuration stored in the DB.
type MemoryConfig struct {
	Enabled         bool   `json:"enabled"`
	EmbedProviderID string `json:"embedProviderId"`
	EmbedModel      string `json:"embedModel"`
	EmbedAPIKey     string `json:"embedApiKey"`
	ChatProviderID  string `json:"chatProviderId"`
	ChatModel       string `json:"chatModel"`
	ChatAPIKey      string `json:"chatApiKey"`
	UpdatedAt       int64  `json:"updatedAt"`
}

// GetMemoryConfig returns the stored agentic-memory config. If the row does not
// exist, it returns a zero-value config (disabled, empty fields).
func GetMemoryConfig(database *db.DB) (*MemoryConfig, error) {
	var c MemoryConfig
	var enabled int
	var updated int64
	err := database.QueryRow(`
		SELECT enabled, embed_provider_id, embed_model, embed_api_key,
		       chat_provider_id, chat_model, chat_api_key, time_updated
		FROM memory_config WHERE id = 1
	`).Scan(
		&enabled, &c.EmbedProviderID, &c.EmbedModel, &c.EmbedAPIKey,
		&c.ChatProviderID, &c.ChatModel, &c.ChatAPIKey, &updated)
	if err == sql.ErrNoRows {
		return &MemoryConfig{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get memory config: %w", err)
	}
	c.Enabled = enabled != 0
	c.UpdatedAt = updated
	return &c, nil
}

// SetMemoryConfig upserts the agentic-memory config row (id is always 1).
func SetMemoryConfig(database *db.DB, c *MemoryConfig) error {
	enabled := 0
	if c.Enabled {
		enabled = 1
	}
	_, err := database.Exec(`
		INSERT INTO memory_config (id, enabled, embed_provider_id, embed_model, embed_api_key,
		                           chat_provider_id, chat_model, chat_api_key, time_updated)
		VALUES (1, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			enabled = excluded.enabled,
			embed_provider_id = excluded.embed_provider_id,
			embed_model = excluded.embed_model,
			embed_api_key = excluded.embed_api_key,
			chat_provider_id = excluded.chat_provider_id,
			chat_model = excluded.chat_model,
			chat_api_key = excluded.chat_api_key,
			time_updated = excluded.time_updated
	`, enabled, c.EmbedProviderID, c.EmbedModel, c.EmbedAPIKey,
		c.ChatProviderID, c.ChatModel, c.ChatAPIKey, Now())
	if err != nil {
		return fmt.Errorf("set memory config: %w", err)
	}
	return nil
}

// MaskedMemoryConfig returns a copy of c with API keys replaced by a sentinel
// so they can be safely sent to the UI.
func MaskedMemoryConfig(c *MemoryConfig) *MemoryConfig {
	mc := *c
	if mc.EmbedAPIKey != "" {
		mc.EmbedAPIKey = "__SET__"
	}
	if mc.ChatAPIKey != "" {
		mc.ChatAPIKey = "__SET__"
	}
	return &mc
}
