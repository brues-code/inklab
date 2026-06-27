package repositories

import (
	"database/sql"
	"fmt"
	"math/bits"
	"sort"
	"strconv"
	"strings"

	"inklab/backend/database/helpers"
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
			       sp.effectBasePoints1, sp.effectBasePoints2, sp.effectBasePoints3,
			       COALESCE(sp.nameSubtext, '')
			FROM spell_template sp
			LEFT JOIN spell_icons si ON sp.spellIconId = si.id
			WHERE sp.entry = ?
		`, id)
	} else {
		// Text search by name
		rows, err = r.db.Query(`
			SELECT sp.entry, sp.name, sp.description, COALESCE(NULLIF(si.icon_name, ''), sp.iconName, ''),
			       sp.effectBasePoints1, sp.effectBasePoints2, sp.effectBasePoints3,
			       COALESCE(sp.nameSubtext, '')
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
		if err := rows.Scan(&s.Entry, &s.Name, &desc, &s.Icon, &bp1, &bp2, &bp3, &s.SubName); err != nil {
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

// spellClassBits lists the class-mask bits (in display order) under the Class
// Skills category, plus a General bucket (0) for class-category skills with no
// single class (Mounts, Companions, Glyphs, ...). Display names come from
// class_info (ChrClasses.dbc), not hardcoded here.
var spellClassBits = []int{1, 2, 4, 8, 16, 64, 128, 256, 1024, 0}

// GetSpellClasses returns the classes under the Class Skills category that have
// at least one non-empty skill line, with the count of those skills. Class names
// come from class_info; spell_skills.class_id is a class-mask bit, so the class
// id is the bit's index + 1.
func (r *SpellRepository) GetSpellClasses() ([]*models.SpellClass, error) {
	nameByID := map[int]string{}
	colorByID := map[int]string{}
	if cr, err := r.db.Query("SELECT id, name, COALESCE(color,'') FROM class_info"); err == nil {
		for cr.Next() {
			var id int
			var name, color string
			if cr.Scan(&id, &name, &color) == nil {
				nameByID[id] = name
				colorByID[id] = color
			}
		}
		cr.Close()
	}
	// bit -> class id (bit's index + 1); 0 is the General bucket.
	classID := func(bit int) int { return bits.TrailingZeros32(uint32(bit)) + 1 }
	className := func(bit int) string {
		if bit == 0 {
			return "General"
		}
		if n := nameByID[classID(bit)]; n != "" {
			return n
		}
		return fmt.Sprintf("Class %d", classID(bit))
	}

	var out []*models.SpellClass
	for _, bit := range spellClassBits {
		var n int
		r.db.QueryRow(`
			SELECT COUNT(*) FROM spell_skills s
			WHERE s.category_id = 7 AND s.class_id = ?
				AND (SELECT COUNT(*) FROM spell_skill_spells ss JOIN spell_template sp ON sp.entry = ss.spell_id WHERE ss.skill_id = s.id) > 0
		`, bit).Scan(&n)
		if n > 0 {
			out = append(out, &models.SpellClass{ID: bit, Name: className(bit), SkillCount: n, Color: colorByID[classID(bit)]})
		}
	}
	// Sort classes alphabetically by name, keeping the General bucket last.
	sort.Slice(out, func(i, j int) bool {
		if (out[i].ID == 0) != (out[j].ID == 0) {
			return out[j].ID == 0
		}
		return out[i].Name < out[j].Name
	})
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
		       sp.effectBasePoints1, sp.effectBasePoints2, sp.effectBasePoints3,
		       COALESCE(sp.nameSubtext, '')
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
		if err := rows.Scan(&s.Entry, &s.Name, &desc, &s.Icon, &bp1, &bp2, &bp3, &s.SubName); err != nil {
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
		SELECT sp.entry, sp.name, sp.description, COALESCE(NULLIF(si.icon_name, ''), sp.iconName, ''),
		       COALESCE(sp.nameSubtext, '')
		FROM spell_template sp
		LEFT JOIN spell_icons si ON sp.spellIconId = si.id
		WHERE sp.entry = ?
	`, entry).Scan(&s.Entry, &s.Name, &desc, &s.Icon, &s.SubName)
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
            sp.effectBaseDice1, sp.effectBaseDice2, sp.effectBaseDice3,
            COALESCE(sp.nameSubtext, ''),
            sp.mechanic, sp.dispel, sp.recoveryTime, sp.categoryRecoveryTime, sp.startRecoveryTime,
            sp.procChance, sp.procCharges, sp.maxAffectedTargets,
            sp.attributes, sp.attributesEx, sp.attributesEx2, sp.attributesEx3, sp.attributesEx4, sp.customFlags,
            sp.effect1, sp.effect2, sp.effect3,
            sp.effectApplyAuraName1, sp.effectApplyAuraName2, sp.effectApplyAuraName3,
            sp.effectAmplitude1, sp.effectAmplitude2, sp.effectAmplitude3,
            sp.effectBonusCoefficient1, sp.effectBonusCoefficient2, sp.effectBonusCoefficient3,
            sp.effectMechanic1, sp.effectMechanic2, sp.effectMechanic3,
            sp.effectRadiusIndex1, sp.effectRadiusIndex2, sp.effectRadiusIndex3,
            sp.effectTriggerSpell1, sp.effectTriggerSpell2, sp.effectTriggerSpell3,
            sp.effectMiscValue1, sp.effectMiscValue2, sp.effectMiscValue3
		FROM spell_template sp
		LEFT JOIN spell_icons si ON sp.spellIconId = si.id
		WHERE sp.entry = ?
	`

	var desc sql.NullString
	var iconName string

	var bp1, bp2, bp3, ds1, ds2, ds3, bd1, bd2, bd3 int
	var mechanic, dispel, recoveryTime, catRecovery, startRecovery, procChance, procCharges, maxTargets int
	var attr [6]int64
	var eff, auraN, amp, effMech, radIdx, trig, misc [3]int
	var coeff [3]float64
	err := r.db.QueryRow(query, entry).Scan(
		&s.Entry, &s.Name, &desc, &s.Durationindex, &s.Rangeindex,
		&s.Manacost, &s.Castingtimeindex, &s.School, &s.Spelllevel, &iconName,
		&bp1, &bp2, &bp3, &ds1, &ds2, &ds3, &bd1, &bd2, &bd3,
		&s.Namesubtext,
		&mechanic, &dispel, &recoveryTime, &catRecovery, &startRecovery,
		&procChance, &procCharges, &maxTargets,
		&attr[0], &attr[1], &attr[2], &attr[3], &attr[4], &attr[5],
		&eff[0], &eff[1], &eff[2],
		&auraN[0], &auraN[1], &auraN[2],
		&amp[0], &amp[1], &amp[2],
		&coeff[0], &coeff[1], &coeff[2],
		&effMech[0], &effMech[1], &effMech[2],
		&radIdx[0], &radIdx[1], &radIdx[2],
		&trig[0], &trig[1], &trig[2],
		&misc[0], &misc[1], &misc[2],
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

	// Localized school name from GlobalStrings.lua (spell_schools); empty when the
	// reference hasn't been imported, so the frontend falls back to English.
	r.db.QueryRow("SELECT name FROM spell_schools WHERE id = ?", s.School).Scan(&detail.SchoolName)

	// Fetch Duration
	var durationStr string = "Instant"
	if s.Durationindex > 0 {
		var durationBase int
		r.db.QueryRow("SELECT duration_base FROM spell_durations WHERE id = ?", s.Durationindex).Scan(&durationBase)
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
		r.db.QueryRow("SELECT range_max FROM spell_range WHERE id = ?", s.Rangeindex).Scan(&rangeMax)
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

	// Cooldown (the greater of the spell's own and its category recovery).
	cd := recoveryTime
	if catRecovery > cd {
		cd = catRecovery
	}
	if cd > 0 {
		detail.Cooldown = fmtDurationMs(cd)
	}
	if startRecovery > 0 {
		detail.GCD = fmt.Sprintf("%.1fs", float64(startRecovery)/1000.0)
	}
	// Raw numbers live on the embedded template (so they serialize once).
	s.Procchance = procChance
	s.Proccharges = procCharges
	s.Maxaffectedtargets = maxTargets
	s.Mechanic = mechanic
	s.Dispel = dispel

	// Real proc rate from the world DB proc tables (more accurate than the DBC
	// procChance, which is usually the 101 "always" sentinel). Prefer an explicit
	// custom %, then a spell PPM, then a weapon-enchant PPM, then a real DBC %.
	var peFlags int
	var pePPM, peCustom, enchPPM float64
	r.db.QueryRow("SELECT procFlags, ppmRate, CustomChance FROM spell_proc_event WHERE entry = ?", entry).Scan(&peFlags, &pePPM, &peCustom)
	r.db.QueryRow("SELECT ppmRate FROM spell_proc_item_enchant WHERE entry = ?", entry).Scan(&enchPPM)
	switch {
	case peCustom > 0:
		detail.Proc = fmt.Sprintf("%s%%", trimFloat(peCustom))
	case pePPM > 0:
		detail.Proc = fmt.Sprintf("%s PPM", trimFloat(pePPM))
	case enchPPM > 0:
		detail.Proc = fmt.Sprintf("%s PPM", trimFloat(enchPPM))
	case procChance > 0 && procChance <= 100:
		detail.Proc = fmt.Sprintf("%d%%", procChance)
	default:
		// On-hit weapon-enchant procs with no explicit rate fall back to the
		// engine's 1-PPM default (e.g. Crusader).
		var isEnchantProc int
		r.db.QueryRow("SELECT 1 FROM enchant_proc_spells WHERE id = ?", entry).Scan(&isEnchantProc)
		if isEnchantProc == 1 {
			detail.Proc = "1 PPM (default)"
		}
	}

	// Mechanic / dispel display names come from the client DBC reference tables.
	if mechanic > 0 {
		r.db.QueryRow("SELECT name FROM spell_mechanics WHERE id = ?", mechanic).Scan(&detail.MechanicName)
	}
	if dispel > 0 {
		r.db.QueryRow("SELECT name FROM spell_dispel_types WHERE id = ?", dispel).Scan(&detail.DispelType)
	}

	// Decoded effects (type/aura names from the server-source enums; values from
	// spell_template; radius from the client SpellRadius reference).
	bps := [3]int{bp1, bp2, bp3}
	dss := [3]int{ds1, ds2, ds3}
	bds := [3]int{bd1, bd2, bd3}
	for i := 0; i < 3; i++ {
		if eff[i] == 0 {
			continue
		}
		e := models.SpellEffectInfo{Index: i + 1, Effect: helpers.SpellEffectNames[eff[i]]}
		if e.Effect == "" {
			e.Effect = fmt.Sprintf("Effect %d", eff[i])
		}
		if auraN[i] != 0 {
			if name := helpers.AuraTypeNames[auraN[i]]; name != "" {
				e.AuraName = name
			} else {
				e.AuraName = fmt.Sprintf("Aura %d", auraN[i])
			}
			// Many auras qualify themselves via the misc value (which stat, school,
			// skill, ...). Append it so "Mod Stat" reads "Mod Stat: Strength".
			if q := r.effectQualifier(auraN[i], misc[i]); q != "" {
				e.AuraName += ": " + q
			}
		}
		e.Value = buildEffectValue(bps[i], dss[i], bds[i], amp[i], coeff[i])
		if radIdx[i] > 0 {
			var rb float64
			r.db.QueryRow("SELECT radius_base FROM spell_radius WHERE id = ?", radIdx[i]).Scan(&rb)
			if rb > 0 {
				e.Radius = fmt.Sprintf("%g yd", rb)
			}
		}
		if effMech[i] > 0 {
			r.db.QueryRow("SELECT name FROM spell_mechanics WHERE id = ?", effMech[i]).Scan(&e.Mechanic)
		}
		e.TriggerSpell = trig[i]
		detail.Effects = append(detail.Effects, e)
	}

	// Attribute flags (server-source labels).
	detail.Flags = helpers.DecodeSpellAttributes([6]uint32{
		uint32(attr[0]), uint32(attr[1]), uint32(attr[2]),
		uint32(attr[3]), uint32(attr[4]), uint32(attr[5]),
	})

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

	// Trainers that teach this spell (reverse of the scraped npc_trainer_spell).
	npcRows, err := r.db.Query(`
		SELECT ts.npc_entry, COALESCE(c.name, ''), COALESCE(c.level_min, 0), COALESCE(c.level_max, 0)
		FROM npc_trainer_spell ts
		LEFT JOIN creature_template c ON c.entry = ts.npc_entry
		WHERE ts.spell_id = ?
		ORDER BY c.name
		LIMIT 100
	`, entry)
	if err == nil {
		defer npcRows.Close()
		for npcRows.Next() {
			n := &models.SpellTrainerNpc{}
			if err := npcRows.Scan(&n.Entry, &n.Name, &n.LevelMin, &n.LevelMax); err == nil {
				detail.TaughtByNpcs = append(detail.TaughtByNpcs, n)
			}
		}
	}

	// Recipe items that teach this spell: an item whose on-use spell is a
	// learn-spell (effect 36) whose triggered spell is this one.
	itemRows, err := r.db.Query(`
		SELECT DISTINCT i.entry, i.name, i.quality, COALESCE(d.icon, '')
		FROM item_template i
		JOIN spell_template ls ON ls.entry IN
			(i.spellid_1, i.spellid_2, i.spellid_3, i.spellid_4, i.spellid_5)
		LEFT JOIN item_display_info d ON i.display_id = d.ID
		WHERE (ls.effect1 = 36 AND ls.effectTriggerSpell1 = ?)
		   OR (ls.effect2 = 36 AND ls.effectTriggerSpell2 = ?)
		   OR (ls.effect3 = 36 AND ls.effectTriggerSpell3 = ?)
		ORDER BY i.quality DESC, i.name
		LIMIT 50
	`, entry, entry, entry)
	if err == nil {
		defer itemRows.Close()
		for itemRows.Next() {
			it := &models.SpellUsedByItem{}
			if err := itemRows.Scan(&it.Entry, &it.Name, &it.Quality, &it.IconPath); err == nil {
				detail.TaughtByItems = append(detail.TaughtByItems, it)
			}
		}
	}

	return detail
}

// fmtDurationMs renders a millisecond duration as a compact "1.5s" / "2m" string.
func fmtDurationMs(ms int) string {
	if ms >= 60000 {
		return fmt.Sprintf("%gm", float64(ms)/60000.0)
	}
	return fmt.Sprintf("%gs", float64(ms)/1000.0)
}

// buildEffectValue renders a spell effect's value the way the detail view shows
// it: base value (basePoints+baseDice, a range when dieSides>1), an "every Ns"
// period for periodic auras, and a spell-power coefficient. Zero-value control
// effects (e.g. Root) render no number.
func buildEffectValue(bp, ds, bd, amp int, coeff float64) string {
	minV := bp + bd
	maxV := bp + bd*ds
	val := ""
	if minV != 0 || maxV != 0 {
		if minV == maxV {
			val = strconv.Itoa(minV)
		} else {
			val = fmt.Sprintf("%d to %d", minV, maxV)
		}
	}
	if amp > 0 {
		secs := trimFloat(float64(amp) / 1000.0)
		if val == "" {
			val = "every " + secs + " sec"
		} else {
			val = fmt.Sprintf("%s every %s sec", val, secs)
		}
	}
	if coeff > 0 {
		val = strings.TrimSpace(val + fmt.Sprintf(" (SP mod: %s)", trimFloat(coeff)))
	}
	return strings.TrimSpace(val)
}

// trimFloat formats a float without trailing zeros (3 -> "3", 0.033 -> "0.033").
func trimFloat(f float64) string {
	s := strconv.FormatFloat(f, 'f', 3, 64)
	s = strings.TrimRight(s, "0")
	return strings.TrimRight(s, ".")
}

// baseStatToStatType maps a unit base-stat index (as stored in MOD_STAT-style
// aura misc values: 0=Strength..4=Spirit) to the item stat_type id, so the name
// resolves from the client-extracted stat_types table.
var baseStatToStatType = [5]int{4, 3, 7, 5, 6}

// effectQualifier resolves the qualifier an aura's misc value carries (which
// stat / school / skill), so "Mod Stat" can read "Mod Stat: Strength". Returns
// "" for auras whose misc value isn't a known qualifier.
func (r *SpellRepository) effectQualifier(auraType, misc int) string {
	switch auraType {
	case 29, 80, 137: // Mod Stat / Mod Percent Stat / Mod Total Stat Percentage
		if misc < 0 {
			return "All Stats"
		}
		if misc < len(baseStatToStatType) {
			var name string
			r.db.QueryRow("SELECT name FROM stat_types WHERE id = ?", baseStatToStatType[misc]).Scan(&name)
			return name
		}
	case 13, 14, 22, 59, 101, 143, 168: // school-mask auras (damage done/taken, resistance)
		return r.schoolMaskNames(misc)
	case 30, 98: // Mod Skill / Mod Skill Talent
		var name string
		r.db.QueryRow("SELECT name FROM spell_skills WHERE id = ?", misc).Scan(&name)
		return name
	}
	return ""
}

// schoolMaskNames turns a spell-school bitmask into "Fire", "Fire/Frost", etc.,
// using the client-extracted spell_schools names (English fallback).
func (r *SpellRepository) schoolMaskNames(mask int) string {
	if mask <= 0 {
		return ""
	}
	var names []string
	for i := 0; i < 7; i++ {
		if mask&(1<<uint(i)) == 0 {
			continue
		}
		var n string
		r.db.QueryRow("SELECT name FROM spell_schools WHERE id = ?", i).Scan(&n)
		if n == "" {
			n = helpers.GetSchoolName(i)
		}
		names = append(names, n)
	}
	return strings.Join(names, "/")
}
