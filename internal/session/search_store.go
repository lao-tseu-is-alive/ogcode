package session

import (
	"database/sql"
	"fmt"

	"github.com/prasenjeet-symon/ogcode/internal/db"
)

// SearchConfig holds the global toggle for the web search feature.
type SearchConfig struct {
	Enabled        bool  `json:"enabled"`
	UseRealProfile bool  `json:"useRealProfile"`
	UpdatedAt      int64 `json:"updatedAt"`
}

// GetSearchConfig returns the stored config. If no row exists, defaults to disabled.
func GetSearchConfig(database *db.DB) (*SearchConfig, error) {
	var enabled, useRealProfile int
	var updatedAt int64
	err := database.QueryRow(
		`SELECT enabled, use_real_profile, time_updated FROM search_config WHERE id = 1`,
	).Scan(&enabled, &useRealProfile, &updatedAt)
	if err == sql.ErrNoRows {
		return &SearchConfig{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get search config: %w", err)
	}
	return &SearchConfig{
		Enabled:        enabled != 0,
		UseRealProfile: useRealProfile != 0,
		UpdatedAt:      updatedAt,
	}, nil
}

// SetSearchConfig upserts the singleton config row.
func SetSearchConfig(database *db.DB, c *SearchConfig) error {
	enabled, useRealProfile := 0, 0
	if c.Enabled {
		enabled = 1
	}
	if c.UseRealProfile {
		useRealProfile = 1
	}
	_, err := database.Exec(`
		INSERT INTO search_config (id, enabled, use_real_profile, time_updated) VALUES (1, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			enabled          = excluded.enabled,
			use_real_profile = excluded.use_real_profile,
			time_updated     = excluded.time_updated
	`, enabled, useRealProfile, Now())
	if err != nil {
		return fmt.Errorf("set search config: %w", err)
	}
	return nil
}
