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

	// Quest sorts (class/profession/seasonal categories with no AreaTable entry).
	if err := m.importQuestSorts(dataDir); err != nil {
		fmt.Printf("Warning: Failed to import quest sorts: %v\n", err)
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

	// Category ids match SkillLine.dbc categoryID. 9 is the client's
	// "Secondary Skills" bucket (cooking/first aid/fishing/riding + racials)
	// and 11 is primary Professions — the previous labels had these swapped.
	// 13 is synthetic: racial skill lines are lifted out of 9 in importSkills.
	spellCats := []struct {
		ID   int
		Name string
	}{
		{6, "Weapon Skills"}, {8, "Armor Proficiencies"}, {10, "Languages"},
		{7, "Class Skills"}, {9, "Secondary Skills"}, {11, "Professions"},
		{13, "Racial Traits"},
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

	abilityStmt, _ := tx.Prepare("REPLACE INTO spell_skill_spells (skill_id, spell_id, req_skill_value) VALUES (?, ?, ?)")
	defer abilityStmt.Close()

	// Which skills are class skills (category 7) — only those get a class_id.
	skillCat := make(map[int]int, len(skills))
	for _, s := range skills {
		skillCat[s.ID] = s.CategoryID
	}
	// Tally class bits seen per class-skill from its abilities' classmasks; the
	// dominant bit is the owning class (handles strays, e.g. Balance has a few
	// non-druid abilities).
	classBits := []int{1, 2, 4, 8, 16, 64, 128, 256, 1024}
	tally := make(map[int]map[int]int)

	for _, a := range abilities {
		abilityStmt.Exec(a.SkillID, a.SpellID, a.ReqSkillValue)
		if skillCat[a.SkillID] != 7 || a.ClassMask == 0 {
			continue
		}
		bits := tally[a.SkillID]
		if bits == nil {
			bits = map[int]int{}
			tally[a.SkillID] = bits
		}
		for _, b := range classBits {
			if a.ClassMask&b != 0 {
				bits[b]++
			}
		}
	}

	classUpd, _ := tx.Prepare("UPDATE spell_skills SET class_id = ? WHERE id = ?")
	defer classUpd.Close()
	for skillID, bits := range tally {
		best, bestCount := 0, 0
		for _, b := range classBits {
			if bits[b] > bestCount {
				best, bestCount = b, bits[b]
			}
		}
		if best != 0 {
			classUpd.Exec(best, skillID)
		}
	}

	// Pet ability lines carry no class mask; assign by family so they group
	// under their owning class (warlock demons vs hunter beasts).
	tx.Exec(`UPDATE spell_skills SET class_id = 256 WHERE category_id = 7 AND name IN
		('Pet - Imp','Pet - Voidwalker','Pet - Succubus','Pet - Felhunter','Pet - Doomguard','Pet - Infernal')`)
	tx.Exec("UPDATE spell_skills SET class_id = 4 WHERE category_id = 7 AND class_id = 0 AND name LIKE 'Pet - %'")

	// Racial skill lines ("Orc Racial", "Racial - Human", ...) are imported
	// under Secondary Skills (category 9); lift them into the dedicated
	// Racial Traits category (13) so they browse on their own.
	tx.Exec("UPDATE spell_skills SET category_id = 13 WHERE name LIKE '%Racial%'")

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
		stmt.Exec(z.AreaID, zoneGroup(z), z.Name)
	}
	return tx.Commit()
}

// zoneGroup maps a zone to a quest_category_group id, using Map.dbc instance
// type so raids and battlegrounds land in their own groups instead of all
// non-continent zones falling into Dungeons.
func zoneGroup(z models.ZoneEntry) int {
	switch z.InstanceType {
	case 1:
		return 2 // Dungeons
	case 2:
		return 3 // Raids
	case 3:
		return 6 // Battlegrounds
	}
	// instanceType 0 (continent / world): split by map.
	switch z.MapID {
	case 0:
		return 0 // Eastern Kingdoms
	case 1:
		return 1 // Kalimdor
	default:
		return 7 // Misc
	}
}

// questSortGroup maps a QuestSort.dbc name to a quest_category_group id:
// 4 Classes, 5 Professions, 7 Misc (seasonal/special/holiday).
func questSortGroup(name string) int {
	classes := map[string]bool{
		"Warrior": true, "Paladin": true, "Hunter": true, "Rogue": true,
		"Priest": true, "Shaman": true, "Mage": true, "Warlock": true,
		"Druid": true, "Death Knight": true,
	}
	professions := map[string]bool{
		"Alchemy": true, "Blacksmithing": true, "Cooking": true, "Enchanting": true,
		"Engineering": true, "First Aid": true, "Fishing": true, "Herbalism": true,
		"Leatherworking": true, "Mining": true, "Skinning": true, "Tailoring": true,
		"Inscription": true, "Jewelcrafting": true,
	}
	switch {
	case classes[name]:
		return 4 // Classes
	case professions[name]:
		return 5 // Professions
	default:
		return 7 // Misc
	}
}

// importQuestSorts loads QuestSort.dbc categories (class/profession/seasonal)
// into quest_categories_enhanced under NEGATIVE ids, matching how quests store
// them in ZoneOrSort. Without this, class quests have no browsable category.
func (m *MetadataImporter) importQuestSorts(dataDir string) error {
	file, err := os.Open(fmt.Sprintf("%s/quest_sorts.json", dataDir))
	if err != nil {
		return err
	}
	defer file.Close()

	var sorts []models.QuestSortEntry
	if err := json.NewDecoder(file).Decode(&sorts); err != nil {
		return err
	}

	tx, err := m.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, _ := tx.Prepare("REPLACE INTO quest_categories_enhanced (id, group_id, name) VALUES (?, ?, ?)")
	defer stmt.Close()

	for _, s := range sorts {
		stmt.Exec(-s.SortID, questSortGroup(s.Name), s.Name)
	}
	return tx.Commit()
}
