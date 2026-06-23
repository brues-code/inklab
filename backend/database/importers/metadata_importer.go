package importers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"

	"inklab/backend/database/models"
)

// MetadataImporter handles metadata imports (zones, skills, etc.)
type MetadataImporter struct {
	db *sql.DB
}

// NewMetadataImporter creates a new metadata importer
func NewMetadataImporter(db *sql.DB) *MetadataImporter {
	return &MetadataImporter{db: db}
}

// ImportAll handles all metadata imports
func (m *MetadataImporter) ImportAll(dataDir string) error {
	m.initStaticMetadata()

	// Always check/import skills
	if err := m.importSkills(dataDir); err != nil {
		fmt.Printf("Warning: Failed to import skills: %v\n", err)
	}

	// Always check/import quest zones
	if err := m.importQuestZones(dataDir); err != nil {
		fmt.Printf("Warning: Failed to import quest zones: %v\n", err)
	} else {
		// Log success if needed, but avoid spam if it's identical
	}
	return nil
}

func (m *MetadataImporter) initStaticMetadata() {
	groups := []struct {
		ID   int
		Name string
	}{
		{0, "Eastern Kingdoms"}, {1, "Kalimdor"}, {2, "Dungeons"},
		{3, "Raids"}, {4, "Classes"}, {5, "Professions"},
		{6, "Battlegrounds"}, {7, "Misc"},
	}
	m.db.Exec("DELETE FROM quest_category_groups")
	for _, g := range groups {
		m.db.Exec("INSERT OR IGNORE INTO quest_category_groups (id, name) VALUES (?, ?)", g.ID, g.Name)
	}

	spellCats := []struct {
		ID   int
		Name string
	}{
		{6, "Weapon Skills"}, {8, "Armor Proficiencies"}, {10, "Languages"},
		{7, "Class Skills"}, {9, "Professions"}, {11, "Racial Traits"},
	}
	m.db.Exec("DELETE FROM spell_skill_categories")
	for _, c := range spellCats {
		m.db.Exec("INSERT OR IGNORE INTO spell_skill_categories (id, name) VALUES (?, ?)", c.ID, c.Name)
	}
}

func (m *MetadataImporter) importSkills(dataDir string) error {
	file, err := os.Open(fmt.Sprintf("%s/skills.json", dataDir))
	if err != nil {
		return err
	}
	defer file.Close()

	var skills []models.SkillEntry
	if err := json.NewDecoder(file).Decode(&skills); err != nil {
		return err
	}

	tx, err := m.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	skillStmt, _ := tx.Prepare("REPLACE INTO spell_skills (id, category_id, name) VALUES (?, ?, ?)")
	defer skillStmt.Close()

	for _, s := range skills {
		skillStmt.Exec(s.ID, s.CategoryID, s.Name)
	}

	file2, err := os.Open(fmt.Sprintf("%s/skill_line_abilities.json", dataDir))
	if err != nil {
		return err
	}
	defer file2.Close()

	var abilities []models.SkillLineAbilityEntry
	if err := json.NewDecoder(file2).Decode(&abilities); err != nil {
		return err
	}

	abilityStmt, _ := tx.Prepare("REPLACE INTO spell_skill_spells (skill_id, spell_id) VALUES (?, ?)")
	defer abilityStmt.Close()

	for _, a := range abilities {
		abilityStmt.Exec(a.SkillID, a.SpellID)
	}
	return tx.Commit()
}

func (m *MetadataImporter) importQuestZones(dataDir string) error {
	file, err := os.Open(fmt.Sprintf("%s/zones.json", dataDir))
	if err != nil {
		return err
	}
	defer file.Close()

	var zones []models.ZoneEntry
	if err := json.NewDecoder(file).Decode(&zones); err != nil {
		return err
	}

	tx, err := m.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, _ := tx.Prepare("REPLACE INTO quest_categories_enhanced (id, group_id, name) VALUES (?, ?, ?)")
	defer stmt.Close()

	for _, z := range zones {
		groupID := 7 // Misc default
		if z.MapID == 0 {
			groupID = 0 // Eastern Kingdoms
		} else if z.MapID == 1 {
			groupID = 1 // Kalimdor
		} else {
			groupID = 2 // Dungeons
		}
		stmt.Exec(z.AreaID, groupID, z.Name)
	}
	return tx.Commit()
}
