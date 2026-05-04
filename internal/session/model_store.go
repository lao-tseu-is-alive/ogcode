package session

import "github.com/prasenjeet-symon/ogcode/internal/db"

func modelStore() *ModelPreferenceStore {
	return &ModelPreferenceStore{}
}

type ModelPreferenceStore struct{}

// GetModelPreferences returns all model preference overrides from the database.
func GetModelPreferences(database *db.DB) ([]*ModelPreference, error) {
	rows, err := database.Query(
		`SELECT id, enabled, provider_id, display_name, is_custom, time_created, time_updated
		 FROM model_preference ORDER BY time_created ASC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var prefs []*ModelPreference
	for rows.Next() {
		var p ModelPreference
		var enabled int
		var isCustom int
		if err := rows.Scan(&p.ID, &enabled, &p.ProviderID, &p.DisplayName, &isCustom, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		p.Enabled = enabled == 1
		p.IsCustom = isCustom == 1
		prefs = append(prefs, &p)
	}
	return prefs, nil
}

// SetModelPreference upserts a model preference into the database.
func SetModelPreference(database *db.DB, p *ModelPreference) error {
	enabled := 0
	if p.Enabled {
		enabled = 1
	}
	isCustom := 0
	if p.IsCustom {
		isCustom = 1
	}
	_, err := database.Exec(
		`INSERT OR REPLACE INTO model_preference (id, enabled, provider_id, display_name, is_custom, time_created, time_updated)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		p.ID, enabled, p.ProviderID, p.DisplayName, isCustom, p.CreatedAt, p.UpdatedAt,
	)
	return err
}

// DeleteModelPreference removes a model preference from the database.
func DeleteModelPreference(database *db.DB, id string) error {
	_, err := database.Exec(`DELETE FROM model_preference WHERE id = ?`, id)
	return err
}