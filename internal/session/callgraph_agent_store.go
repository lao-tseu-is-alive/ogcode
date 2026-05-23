package session

import (
	"database/sql"
	"fmt"

	"github.com/prasenjeet-symon/ogcode/internal/db"
)

// CallGraphAgentConfig holds the global toggle for callgraph agent instructions.
type CallGraphAgentConfig struct {
	Enabled bool `json:"enabled"`
}

// GetCallGraphAgentConfig returns the stored config. If no row exists, returns
// the default (enabled = true).
func GetCallGraphAgentConfig(database *db.DB) (*CallGraphAgentConfig, error) {
	var enabled int
	err := database.QueryRow(`SELECT enabled FROM callgraph_agent_config WHERE id = 1`).Scan(&enabled)
	if err == sql.ErrNoRows {
		return &CallGraphAgentConfig{Enabled: true}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get callgraph agent config: %w", err)
	}
	return &CallGraphAgentConfig{Enabled: enabled != 0}, nil
}

// SetCallGraphAgentConfig upserts the singleton config row.
func SetCallGraphAgentConfig(database *db.DB, c *CallGraphAgentConfig) error {
	enabled := 0
	if c.Enabled {
		enabled = 1
	}
	_, err := database.Exec(`
		INSERT INTO callgraph_agent_config (id, enabled) VALUES (1, ?)
		ON CONFLICT(id) DO UPDATE SET enabled = excluded.enabled
	`, enabled)
	if err != nil {
		return fmt.Errorf("set callgraph agent config: %w", err)
	}
	return nil
}
