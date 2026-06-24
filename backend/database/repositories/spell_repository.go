package repositories

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	"inklab/backend/database/models"
)

// SpellRepository handles spell-related database operations
type SpellRepository struct {
	db *sql.DB
}

// NewSpellRepository creates a new spell repository
func NewSpellRepository(db *sql.DB) *SpellRepository {
	return &SpellRepository{db: db}
}

// parseSpellDescription replaces $s1, $s2, $s3 variables in spell description
func parseSpellDescription(desc string, bp1, bp2, bp3 int) string {
	if desc == "" {
		return ""
	}
	desc = strings.ReplaceAll(desc, "$s1", fmt.Sprintf("%d", bp1+1))
	desc = strings.ReplaceAll(desc, "$S1", fmt.Sprintf("%d", bp1+1))
	desc = strings.ReplaceAll(desc, "$s2", fmt.Sprintf("%d", bp2+1))
	desc = strings.ReplaceAll(desc, "$S2", fmt.Sprintf("%d", bp2+1))
	desc = strings.ReplaceAll(desc, "$s3", fmt.Sprintf("%d", bp3+1))
	desc = strings.ReplaceAll(desc, "$S3", fmt.Sprintf("%d", bp3+1))
	return desc
}

// SearchSpells searches for spells by ID or name
func (r *SpellRepository) SearchSpells(query string) ([]*models.Spell, error) {
	var rows *sql.Rows
	var err error

	// Check if query is a number (ID search)
	if id, parseErr := strconv.Atoi(query); parseErr == nil && id > 0 {
		rows, err = r.db.Query(`
			SELECT sp.entry, sp.name, sp.description, COALESCE(NULLIF(si.icon_name, ''), sp.iconName, ''),
			       sp.effectBasePoints1, sp.effectBasePoints2, sp.effectBasePoints3
			FROM spell_template sp
			LEFT JOIN spell_icons si ON sp.spellIconId = si.id
			WHERE sp.entry = ?
		`, id)
	} else {
		// Text search by name
		rows, err = r.db.Query(`
			SELECT sp.entry, sp.name, sp.description, COALESCE(NULLIF(si.icon_name, ''), sp.iconName, ''),
			       sp.effectBasePoints1, sp.effectBasePoints2, sp.effectBasePoints3
			FROM spell_template sp
			LEFT JOIN spell_icons si ON sp.spellIconId = si.id
			WHERE sp.name LIKE ?
			ORDER BY length(sp.name), sp.name
			LIMIT 100
		`, "%"+query+"%")
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var spells []*models.Spell
	for rows.Next() {
		s := &models.Spell{}
		var desc *string
		var bp1, bp2, bp3 int
		if err := rows.Scan(&s.Entry, &s.Name, &desc, &s.Icon, &bp1, &bp2, &bp3); err != nil {
			fmt.Printf("Scan error: %v\n", err)
			continue
		}
		if desc != nil {
			s.Description = parseSpellDescription(*desc, bp1, bp2, bp3)
		}
		spells = append(spells, s)
	}
	return spells, nil
}

// GetSpellSkillCategories returns all spell skill categories
func (r *SpellRepository) GetSpellSkillCategories() ([]*models.SpellSkillCategory, error) {
	rows, err := r.db.Query(`
		SELECT id, name FROM spell_skill_categories ORDER BY id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []*models.SpellSkillCategory
	for rows.Next() {
		c := &models.SpellSkillCategory{}
		if err := rows.Scan(&c.ID, &c.Name); err != nil {
			continue
		}
		categories = append(categories, c)
	}
	return categories, nil
}

// GetSpellSkillsByCategory returns all skills in a category with spell counts
func (r *SpellRepository) GetSpellSkillsByCategory(categoryID int) ([]*models.SpellSkill, error) {
	rows, err := r.db.Query(`
		SELECT s.id, s.category_id, s.name,
		       (SELECT COUNT(*) FROM spell_skill_spells ss JOIN spell_template sp ON sp.entry = ss.spell_id WHERE ss.skill_id = s.id) as spell_count
		FROM spell_skills s
		WHERE s.category_id = ?
		       AND (SELECT COUNT(*) FROM spell_skill_spells ss JOIN spell_template sp ON sp.entry = ss.spell_id WHERE ss.skill_id = s.id) > 0
		ORDER BY s.name
	`, categoryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var skills []*models.SpellSkill
	for rows.Next() {
		s := &models.SpellSkill{}
		if err := rows.Scan(&s.ID, &s.CategoryID, &s.Name, &s.SpellCount); err != nil {
			continue
		}
		skills = append(skills, s)
	}
	return skills, nil
}

// spellClasses lists the WoW classes (by classmask bit) in display order,
// plus a General bucket (0) for class-category skills with no single class
// (Mounts, Companions, Glyphs, ...).
var spellClasses = []struct {
	ID   int
	Name string
}{
	{1, "Warrior"}, {2, "Paladin"}, {4, "Hunter"}, {8, "Rogue"}, {16, "Priest"},
	{64, "Shaman"}, {128, "Mage"}, {256, "Warlock"}, {1024, "Druid"}, {0, "General"},
}

// GetSpellClasses returns the classes under the Class Skills category that have
// at least one non-empty skill line, with the count of those skills.
func (r *SpellRepository) GetSpellClasses() ([]*models.SpellClass, error) {
	var out []*models.SpellClass
	for _, c := range spellClasses {
		var n int
		r.db.QueryRow(`
			SELECT COUNT(*) FROM spell_skills s
			WHERE s.category_id = 7 AND s.class_id = ?
				AND (SELECT COUNT(*) FROM spell_skill_spells ss JOIN spell_template sp ON sp.entry = ss.spell_id WHERE ss.skill_id = s.id) > 0
		`, c.ID).Scan(&n)
		if n > 0 {
			out = append(out, &models.SpellClass{ID: c.ID, Name: c.Name, SkillCount: n})
		}
	}
	return out, nil
}

// GetSpellSkillsByClass returns the (non-empty) class skill lines for one class.
func (r *SpellRepository) GetSpellSkillsByClass(classID int) ([]*models.SpellSkill, error) {
	rows, err := r.db.Query(`
		SELECT s.id, s.category_id, s.name,
		       (SELECT COUNT(*) FROM spell_skill_spells ss JOIN spell_template sp ON sp.entry = ss.spell_id WHERE ss.skill_id = s.id) as spell_count
		FROM spell_skills s
		WHERE s.category_id = 7 AND s.class_id = ?
		       AND (SELECT COUNT(*) FROM spell_skill_spells ss JOIN spell_template sp ON sp.entry = ss.spell_id WHERE ss.skill_id = s.id) > 0
		ORDER BY s.name
	`, classID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var skills []*models.SpellSkill
	for rows.Next() {
		s := &models.SpellSkill{}
		if err := rows.Scan(&s.ID, &s.CategoryID, &s.Name, &s.SpellCount); err != nil {
			continue
		}
		skills = append(skills, s)
	}
	return skills, nil
}

// GetSpellsBySkill returns all spells for a given skill
func (r *SpellRepository) GetSpellsBySkill(skillID int, nameFilter string) ([]*models.Spell, error) {
	whereClause := "WHERE ss.skill_id = ?"
	args := []interface{}{skillID}

	if nameFilter != "" {
		whereClause += " AND sp.name LIKE ?"
		args = append(args, "%"+nameFilter+"%")
	}

	query := fmt.Sprintf(`
		SELECT sp.entry, sp.name, sp.description, COALESCE(NULLIF(si.icon_name, ''), sp.iconName, ''),
		       sp.effectBasePoints1, sp.effectBasePoints2, sp.effectBasePoints3
		FROM spell_template sp
		INNER JOIN spell_skill_spells ss ON ss.spell_id = sp.entry
		LEFT JOIN spell_icons si ON sp.spellIconId = si.id
		%s
		ORDER BY sp.name
		LIMIT 10000
	`, whereClause)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var spells []*models.Spell
	for rows.Next() {
		s := &models.Spell{}
		var desc *string
		var bp1, bp2, bp3 int
		if err := rows.Scan(&s.Entry, &s.Name, &desc, &s.Icon, &bp1, &bp2, &bp3); err != nil {
			continue
		}
		if desc != nil {
			s.Description = parseSpellDescription(*desc, bp1, bp2, bp3)
		}
		spells = append(spells, s)
	}
	return spells, nil
}

// GetSpellByID retrieves a single spell by ID
func (r *SpellRepository) GetSpellByID(entry int) (*models.Spell, error) {
	s := &models.Spell{}
	var desc *string
	err := r.db.QueryRow(`
		SELECT sp.entry, sp.name, sp.description, COALESCE(NULLIF(si.icon_name, ''), sp.iconName, '')
		FROM spell_template sp
		LEFT JOIN spell_icons si ON sp.spellIconId = si.id
		WHERE sp.entry = ?
	`, entry).Scan(&s.Entry, &s.Name, &desc, &s.Icon)
	if err != nil {
		return nil, err
	}
	if desc != nil {
		s.Description = *desc
	}
	return s, nil
}

// GetSpellDescription retrieves spell description and base points
func (r *SpellRepository) GetSpellDescription(spellID int) (string, []int) {
	var desc string
	var bp1, bp2, bp3 int
	err := r.db.QueryRow(`
		SELECT description, effectBasePoints1, effectBasePoints2, effectBasePoints3
		FROM spell_template WHERE entry = ?
	`, spellID).Scan(&desc, &bp1, &bp2, &bp3)
	if err != nil {
		return "", nil
	}
	return desc, []int{bp1, bp2, bp3}
}

// GetSpellDetail returns detailed information about a spell
func (r *SpellRepository) GetSpellDetail(entry int) *models.SpellDetail {
	// 1. Get base spell template
	s := &models.SpellTemplateFull{}

	// Select relevant fields for the Detail View.
	// Use spell_icons table to get icon name via spellIconId
	query := `
		SELECT 
			sp.entry, sp.name, sp.description, sp.durationIndex, sp.rangeIndex, 
			sp.manaCost, sp.castingTimeIndex, sp.school, sp.spellLevel, COALESCE(NULLIF(si.icon_name, ''), sp.iconName, ''),
            sp.effectBasePoints1, sp.effectBasePoints2, sp.effectBasePoints3,
            sp.effectDieSides1, sp.effectDieSides2, sp.effectDieSides3,
            sp.effectBaseDice1, sp.effectBaseDice2, sp.effectBaseDice3
		FROM spell_template sp
		LEFT JOIN spell_icons si ON sp.spellIconId = si.id
		WHERE sp.entry = ?
	`

	var desc sql.NullString
	var iconName string

	var bp1, bp2, bp3, ds1, ds2, ds3, bd1, bd2, bd3 int
	err := r.db.QueryRow(query, entry).Scan(
		&s.Entry, &s.Name, &desc, &s.Durationindex, &s.Rangeindex,
		&s.Manacost, &s.Castingtimeindex, &s.School, &s.Spelllevel, &iconName,
		&bp1, &bp2, &bp3, &ds1, &ds2, &ds3, &bd1, &bd2, &bd3,
	)

	if err != nil {
		fmt.Printf("GetSpellDetail error: %v\n", err)
		return nil
	}

	if desc.Valid {
		s.Description = desc.String
	}

	detail := &models.SpellDetail{
		SpellTemplateFull: s,
	}

	detail.Icon = iconName

	// Fetch Duration
	var durationStr string = "Instant"
	if s.Durationindex > 0 {
		var durationBase int
		r.db.QueryRow("SELECT DurationBase FROM spell_duration WHERE ID = ?", s.Durationindex).Scan(&durationBase)
		if durationBase > 0 {
			if durationBase >= 60000 {
				durationStr = fmt.Sprintf("%dm", durationBase/60000)
			} else {
				durationStr = fmt.Sprintf("%ds", durationBase/1000)
			}
		}
	}
	detail.Duration = durationStr

	// Fetch Range
	if s.Rangeindex > 0 {
		var rangeMax float64
		r.db.QueryRow("SELECT rangeMax FROM spell_range WHERE ID = ?", s.Rangeindex).Scan(&rangeMax)
		if rangeMax > 0 {
			detail.Range = fmt.Sprintf("%.0f yd", rangeMax)
		} else {
			detail.Range = "Self"
		}
	} else {
		detail.Range = "Self"
	}

	// Fetch Cast Time
	if s.Castingtimeindex > 0 {
		var base int
		r.db.QueryRow("SELECT base FROM spell_cast_times WHERE ID = ?", s.Castingtimeindex).Scan(&base)
		if base > 0 {
			detail.CastTime = fmt.Sprintf("%.1fs", float64(base)/1000.0)
		} else {
			detail.CastTime = "Instant"
		}
	} else {
		detail.CastTime = "Instant"
	}

	detail.ToolTip = s.Description

	// Parse Description Variables
	parser := func(text string) string {
		if text == "" {
			return ""
		}
		// $d - Duration
		text = strings.ReplaceAll(text, "$d", durationStr)
		text = strings.ReplaceAll(text, "$D", durationStr)

		// $s1, $s2, $s3 -> (bp + 1)
		text = strings.ReplaceAll(text, "$s1", fmt.Sprintf("%d", bp1+1))
		text = strings.ReplaceAll(text, "$S1", fmt.Sprintf("%d", bp1+1))

		text = strings.ReplaceAll(text, "$s2", fmt.Sprintf("%d", bp2+1))
		text = strings.ReplaceAll(text, "$S2", fmt.Sprintf("%d", bp2+1))

		text = strings.ReplaceAll(text, "$s3", fmt.Sprintf("%d", bp3+1))
		text = strings.ReplaceAll(text, "$S3", fmt.Sprintf("%d", bp3+1))

		return text
	}

	// Apply parser to both description and tooltip
	if s.Description != "" {
		detail.Description = parser(s.Description)
	}
	// Note: s.Description was assigned to detail.ToolTip above, but we re-parse it.
	// Ideally ToolTip might be different, but in our query we only fetched 'description'.
	// If there is a 'tooltip' column in DB, we should fetch it. currently using description as tooltip.
	detail.ToolTip = detail.Description

	// Query items that use this spell
	usedByQuery := `
		SELECT t.entry, t.name, t.quality, COALESCE(d.icon, ''),
			CASE 
				WHEN t.spellid_1 = ? THEN t.spelltrigger_1
				WHEN t.spellid_2 = ? THEN t.spelltrigger_2
				WHEN t.spellid_3 = ? THEN t.spelltrigger_3
				WHEN t.spellid_4 = ? THEN t.spelltrigger_4
				WHEN t.spellid_5 = ? THEN t.spelltrigger_5
				ELSE 0
			END as trigger_type
		FROM item_template t
		LEFT JOIN item_display_info d ON t.display_id = d.ID
		WHERE t.spellid_1 = ? OR t.spellid_2 = ? OR t.spellid_3 = ? OR t.spellid_4 = ? OR t.spellid_5 = ?
		ORDER BY t.quality DESC, t.name
		LIMIT 50
	`
	usedByRows, err := r.db.Query(usedByQuery, entry, entry, entry, entry, entry, entry, entry, entry, entry, entry)
	if err == nil {
		defer usedByRows.Close()
		for usedByRows.Next() {
			item := &models.SpellUsedByItem{}
			if err := usedByRows.Scan(&item.Entry, &item.Name, &item.Quality, &item.IconPath, &item.TriggerType); err == nil {
				detail.UsedByItems = append(detail.UsedByItems, item)
			}
		}
	}

	return detail
}
