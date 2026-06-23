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
