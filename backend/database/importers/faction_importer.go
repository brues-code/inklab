package importers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"

	"inklab/backend/database/models"
)

// FactionImporter handles faction data imports
type FactionImporter struct {
	db *sql.DB
}

// NewFactionImporter creates a new faction importer
func NewFactionImporter(db *sql.DB) *FactionImporter {
	return &FactionImporter{db: db}
}

// ImportFromJSON imports factions from JSON into SQLite
func (f *FactionImporter) ImportFromJSON(jsonPath string) error {
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		return fmt.Errorf("failed to read JSON file: %w", err)
	}

	var factions []models.FactionEntry
	if err := json.Unmarshal(data, &factions); err != nil {
		return fmt.Errorf("failed to parse JSON: %w", err)
	}

	tx, err := f.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	tx.Exec("DELETE FROM factions")

	stmt, err := tx.Prepare("INSERT INTO factions (id, name, description, side, category_id) VALUES (?, ?, ?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, fc := range factions {
		stmt.Exec(fc.FactionID, fc.Name, fc.Description, fc.Side, fc.Team)
	}
	return tx.Commit()
}

type factionTemplateJSON struct {
	TemplateID int `json:"id"`
	FactionID  int `json:"faction"`
}

// ImportTemplatesFromJSON loads faction_templates.json (FactionTemplate.dbc:
// template id -> Faction.dbc id) into faction_template, used to list an NPC's
// faction membership.
func (f *FactionImporter) ImportTemplatesFromJSON(jsonPath string) error {
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		return fmt.Errorf("failed to read JSON file: %w", err)
	}
	var rows []factionTemplateJSON
	if err := json.Unmarshal(data, &rows); err != nil {
		return fmt.Errorf("failed to parse JSON: %w", err)
	}
	tx, err := f.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	tx.Exec("DELETE FROM faction_template")
	stmt, err := tx.Prepare("INSERT OR REPLACE INTO faction_template (template_id, faction_id) VALUES (?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()
	for _, t := range rows {
		stmt.Exec(t.TemplateID, t.FactionID)
	}
	return tx.Commit()
}

// CheckAndImportTemplates imports faction_template from JSON when the table is
// empty and the JSON exists (it only exists after a client DBC export).
func (f *FactionImporter) CheckAndImportTemplates(dataDir string) error {
	var count int
	if err := f.db.QueryRow("SELECT COUNT(*) FROM faction_template").Scan(&count); err != nil {
		return nil
	}
	if count == 0 {
		path := fmt.Sprintf("%s/faction_templates.json", dataDir)
		if _, err := os.Stat(path); err == nil {
			fmt.Println("Importing Faction Templates...")
			return f.ImportTemplatesFromJSON(path)
		}
	}
	return nil
}

// CheckAndImport checks if factions table is empty and imports if JSON exists
func (f *FactionImporter) CheckAndImport(dataDir string) error {
	var count int
	if err := f.db.QueryRow("SELECT COUNT(*) FROM factions").Scan(&count); err != nil {
		return nil
	}
	if count == 0 {
		path := fmt.Sprintf("%s/factions.json", dataDir)
		if _, err := os.Stat(path); err == nil {
			fmt.Println("Importing Factions...")
			return f.ImportFromJSON(path)
		}
	}
	return nil
}
