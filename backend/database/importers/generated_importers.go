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
	Entry         int    `json:"entry"`
	Name          string `json:"name"`
	Description   string `json:"description"`
	BasePoints1   int    `json:"effectBasePoints1"`
	BasePoints2   int    `json:"effectBasePoints2"`
	BasePoints3   int    `json:"effectBasePoints3"`
	DieSides1     int    `json:"effectDieSides1"`
	DieSides2     int    `json:"effectDieSides2"`
	DieSides3     int    `json:"effectDieSides3"`
	DurationIndex int    `json:"durationIndex"`
	SpellIconId   int    `json:"spellIconId"`
	IconName      string `json:"iconName"`
}

// ImportMissingSpells backfills spell_template from the Spell.dbc export with
// any spell the (frozen) MySQL import lacked. The client DBC carries newer
// octo spells — e.g. set-bonus spells like 52568/52587 — that otherwise show
// as a raw spell id in tooltips and are unsearchable. Existing rows are kept
// (INSERT OR IGNORE), so the richer MySQL data always wins.
func (i *GeneratedImporter) ImportMissingSpells(jsonPath string) error {
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		return nil // optional
	}
	var spells []SpellEnhanced
	if err := json.Unmarshal(data, &spells); err != nil {
		fmt.Printf("  ERROR parsing spells_enhanced.json: %v\n", err)
		return nil
	}

	tx, err := i.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`INSERT OR IGNORE INTO spell_template
		(entry, name, description, effectBasePoints1, effectBasePoints2, effectBasePoints3,
		 effectDieSides1, effectDieSides2, effectDieSides3, durationIndex, spellIconId, iconName)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	count := 0
	for _, s := range spells {
		if s.Entry <= 0 {
			continue
		}
		icon := s.IconName
		if icon == "temp" {
			icon = ""
		}
		res, err := stmt.Exec(s.Entry, s.Name, s.Description, s.BasePoints1, s.BasePoints2, s.BasePoints3,
			s.DieSides1, s.DieSides2, s.DieSides3, s.DurationIndex, s.SpellIconId, icon)
		if err != nil {
			continue
		}
		if n, _ := res.RowsAffected(); n > 0 {
			count++
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	fmt.Printf("  ✓ Backfilled %d missing spells from DBC\n", count)
	return nil
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

	// Also backfill the spell_icons lookup table from the same complete,
	// DBC-derived mapping. It is otherwise sourced from the live
	// aowow.aowow_spellicons table, which is an incomplete import (missing icon
	// ids such as 2164 / Spell_Holy_CrusaderStrike).
	iconStmt, err := tx.Prepare("INSERT OR REPLACE INTO spell_icons (id, icon_name) VALUES (?, ?)")
	if err != nil {
		return err
	}
	defer iconStmt.Close()

	count := 0
	iconCount := 0
	for iconId, iconName := range iconMap {
		res, err := stmt.Exec(iconName, iconId)
		if err == nil {
			if rows, _ := res.RowsAffected(); rows > 0 {
				count++
			}
		}
		if _, err := iconStmt.Exec(iconId, iconName); err == nil {
			iconCount++
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	fmt.Printf("  ✓ Updated %d spells with icons; backfilled %d spell_icons rows\n", count, iconCount)
	return nil
}
