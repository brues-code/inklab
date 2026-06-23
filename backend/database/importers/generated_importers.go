package importers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
)

type GeneratedImporter struct {
	db *sql.DB
}

func NewGeneratedImporter(db *sql.DB) *GeneratedImporter {
	return &GeneratedImporter{db: db}
}

// ImportItemIcons loads display-id -> icon mappings from item_icons.json into
// the item_display_info table (the table item queries join against for icons).
// This is the client ItemDisplayInfo.dbc data and is more complete than the
// server-side copy, so it upserts over whatever the MySQL import provided.
func (i *GeneratedImporter) ImportItemIcons(jsonPath string) error {
	fmt.Printf("  -> Reading item icons from %s...\n", jsonPath)
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		return nil // Icons are optional
	}

	var iconMap map[string]string
	if err := json.Unmarshal(data, &iconMap); err != nil {
		fmt.Printf("  ERROR parsing item_icons.json: %v\n", err)
		return nil
	}

	fmt.Printf("  -> Upserting %d icon mappings into item_display_info...\n", len(iconMap))
	tx, err := i.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare("INSERT OR REPLACE INTO item_display_info (ID, icon) VALUES (?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	count := 0
	for displayIDStr, iconName := range iconMap {
		var displayID int
		fmt.Sscanf(displayIDStr, "%d", &displayID)
		if displayID > 0 {
			if _, err := stmt.Exec(displayID, iconName); err != nil {
				continue
			}
			count++
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	fmt.Printf("  ✓ Successfully wrote %d item icon mappings\n", count)
	return nil
}

// SpellEnhanced represents a spell record from spells_enhanced.json
type SpellEnhanced struct {
	SpellIconId int    `json:"spellIconId"`
	IconName    string `json:"iconName"`
}

// ImportSpellIcons loads spell icons from spells_enhanced.json and updates spell_template
func (i *GeneratedImporter) ImportSpellIcons(jsonPath string) error {
	fmt.Printf("  -> Reading spell icons from %s...\n", jsonPath)
	file, err := os.Open(jsonPath)
	if err != nil {
		return nil // Optional
	}
	defer file.Close()

	var spells []SpellEnhanced
	if err := json.NewDecoder(file).Decode(&spells); err != nil {
		fmt.Printf("  ERROR parsing spells_enhanced.json: %v\n", err)
		return nil
	}

	// Build unique icon map
	iconMap := make(map[int]string)
	for _, s := range spells {
		if s.SpellIconId > 0 && s.IconName != "" && s.IconName != "temp" {
			iconMap[s.SpellIconId] = s.IconName
		}
	}

	fmt.Printf("  -> Updating database with %d spell icon mappings...\n", len(iconMap))
	tx, err := i.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare("UPDATE spell_template SET iconName = ? WHERE spellIconId = ?")
	if err != nil {
		return err
	}
	defer stmt.Close()

	count := 0
	for iconId, iconName := range iconMap {
		res, err := stmt.Exec(iconName, iconId)
		if err != nil {
			continue
		}
		if rows, _ := res.RowsAffected(); rows > 0 {
			count++
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	fmt.Printf("  ✓ Successfully updated %d spells with icons\n", count)
	return nil
}
