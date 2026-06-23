package importers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"

	"inklab/backend/database/models"
)

// ItemSetImporter handles item set data imports
type ItemSetImporter struct {
	db *sql.DB
}

// NewItemSetImporter creates a new item set importer
func NewItemSetImporter(db *sql.DB) *ItemSetImporter {
	return &ItemSetImporter{db: db}
}

// ImportFromJSON imports item sets from JSON
func (i *ItemSetImporter) ImportFromJSON(jsonPath string) error {
	file, err := os.Open(jsonPath)
	if err != nil {
		return fmt.Errorf("failed to open item sets JSON: %w", err)
	}
	defer file.Close()

	var sets []models.ItemSetEntry
	if err := json.NewDecoder(file).Decode(&sets); err != nil {
		return fmt.Errorf("failed to decode item sets JSON: %w", err)
	}

	tx, err := i.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		REPLACE INTO itemsets (
			itemset_id, name,
			item1, item2, item3, item4, item5, item6, item7, item8, item9, item10,
			skill_id, skill_level,
			bonus1, bonus2, bonus3, bonus4, bonus5, bonus6, bonus7, bonus8,
			spell1, spell2, spell3, spell4, spell5, spell6, spell7, spell8
		) VALUES (
			?, ?,
			?, ?, ?, ?, ?, ?, ?, ?, ?, ?,
			?, ?,
			?, ?, ?, ?, ?, ?, ?, ?,
			?, ?, ?, ?, ?, ?, ?, ?
		)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, s := range sets {
		stmt.Exec(
			s.ID, s.Name,
			s.Item1, s.Item2, s.Item3, s.Item4, s.Item5, s.Item6, s.Item7, s.Item8, s.Item9, s.Item10,
			s.SkillID, s.SkillLevel,
			s.Bonus1, s.Bonus2, s.Bonus3, s.Bonus4, s.Bonus5, s.Bonus6, s.Bonus7, s.Bonus8,
			s.Spell1, s.Spell2, s.Spell3, s.Spell4, s.Spell5, s.Spell6, s.Spell7, s.Spell8,
		)
	}
	return tx.Commit()
}

// CheckAndImport checks if itemsets table is empty and imports if JSON exists
func (i *ItemSetImporter) CheckAndImport(dataDir string) error {
	var count int
	if err := i.db.QueryRow("SELECT COUNT(*) FROM itemsets").Scan(&count); err != nil {
		return nil
	}
	if count == 0 {
		path := fmt.Sprintf("%s/item_sets.json", dataDir)
		if _, err := os.Stat(path); err == nil {
			fmt.Println("Importing Item Sets...")
			return i.ImportFromJSON(path)
		}
	}
	return nil
}
