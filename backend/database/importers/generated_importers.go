package importers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"inklab/backend/database/helpers"
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

// SpellEnhanced represents a spell record from spells_enhanced.json. The fields
// mirror exactly what the $-placeholder resolver reads, so the client DBC can be
// the authoritative source for spell text + values.
type SpellEnhanced struct {
	Entry              int    `json:"entry"`
	Name               string `json:"name"`
	Description        string `json:"description"`
	BasePoints1        int    `json:"effectBasePoints1"`
	BasePoints2        int    `json:"effectBasePoints2"`
	BasePoints3        int    `json:"effectBasePoints3"`
	DieSides1          int    `json:"effectDieSides1"`
	DieSides2          int    `json:"effectDieSides2"`
	DieSides3          int    `json:"effectDieSides3"`
	Amplitude1         int    `json:"effectAmplitude1"`
	Amplitude2         int    `json:"effectAmplitude2"`
	Amplitude3         int    `json:"effectAmplitude3"`
	ChainTarget1       int    `json:"effectChainTarget1"`
	ChainTarget2       int    `json:"effectChainTarget2"`
	ChainTarget3       int    `json:"effectChainTarget3"`
	RadiusIndex1       int    `json:"effectRadiusIndex1"`
	RadiusIndex2       int    `json:"effectRadiusIndex2"`
	RadiusIndex3       int    `json:"effectRadiusIndex3"`
	DurationIndex      int    `json:"durationIndex"`
	RangeIndex         int    `json:"rangeIndex"`
	ProcChance         int    `json:"procChance"`
	ProcCharges        int    `json:"procCharges"`
	MaxLevel           int    `json:"maxLevel"`
	BaseLevel          int    `json:"baseLevel"`
	SpellLevel         int    `json:"spellLevel"`
	MaxTargetLevel     int    `json:"maxTargetLevel"`
	MaxAffectedTargets int    `json:"maxAffectedTargets"`
	SpellIconId        int    `json:"spellIconId"`
	IconName           string `json:"iconName"`
}

// ImportSpellsFromDBC makes the client Spell.dbc the authoritative source for
// spell text and the values the $-placeholder resolver reads. On a custom server
// the client DBC is patched to match what the server actually does, so it is the
// latest/correct data — newer than the frozen MySQL world-DB import. For every
// DBC spell it overwrites name/description/effect values/proc/range/icon on the
// existing row (DBC wins), and inserts spells the MySQL import lacked. The raw
// $-templates it writes are turned into readable text by the spell resolver pass
// that runs immediately after (FullSyncSpells). Name/description/icon are only
// overwritten when the DBC actually provides them, so a DBC gap can't blank good
// data.
func (i *GeneratedImporter) ImportSpellsFromDBC(jsonPath string) error {
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		return nil // optional — only present after a client import
	}
	// Parse into raw maps keyed by spell_template column name (genSpells emits the
	// column names directly), so the whole row stays data-driven — adding a DBC
	// field is a one-line change in genSpells + spellNumCols here, with no risk of
	// the UPDATE/INSERT positional args drifting.
	var spells []map[string]interface{}
	if err := json.Unmarshal(data, &spells); err != nil {
		fmt.Printf("  ERROR parsing spells_enhanced.json: %v\n", err)
		return nil
	}

	// Plain (overwrite) numeric columns the client DBC is authoritative for.
	numCols := []string{
		"spellIconId", "school", "category", "dispel", "mechanic",
		"castingTimeIndex", "recoveryTime", "categoryRecoveryTime", "startRecoveryTime",
		"powerType", "manaCost", "durationIndex", "rangeIndex", "procChance", "procCharges",
		"maxLevel", "baseLevel", "spellLevel", "maxTargetLevel", "maxAffectedTargets",
	}
	for _, p := range []string{
		"effect", "effectBasePoints", "effectDieSides", "effectBaseDice", "effectMechanic",
		"effectImplicitTargetA", "effectImplicitTargetB", "effectRadiusIndex",
		"effectApplyAuraName", "effectAmplitude", "effectMultipleValue", "effectChainTarget",
		"effectItemType", "effectMiscValue", "effectTriggerSpell",
	} {
		for n := 1; n <= 3; n++ {
			numCols = append(numCols, fmt.Sprintf("%s%d", p, n))
		}
	}
	for n := 1; n <= 8; n++ {
		numCols = append(numCols, fmt.Sprintf("reagent%d", n), fmt.Sprintf("reagentCount%d", n))
	}
	// Text columns: keep the prior value when the DBC has none (COALESCE/NULLIF).
	textCols := []string{"name", "description", "nameSubtext", "iconName"}

	tx, err := i.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Build the UPDATE / INSERT once from the column lists so they stay in lockstep.
	setParts := make([]string, 0, len(textCols)+len(numCols))
	for _, c := range textCols {
		setParts = append(setParts, fmt.Sprintf("%s=COALESCE(NULLIF(?,''), %s)", c, c))
	}
	for _, c := range numCols {
		setParts = append(setParts, c+"=?")
	}
	upd, err := tx.Prepare("UPDATE spell_template SET " + strings.Join(setParts, ", ") + " WHERE entry=?")
	if err != nil {
		return err
	}
	defer upd.Close()

	insCols := append([]string{"entry"}, append(append([]string{}, textCols...), numCols...)...)
	ph := make([]string, len(insCols))
	for i := range ph {
		ph[i] = "?"
	}
	ins, err := tx.Prepare("INSERT OR IGNORE INTO spell_template (" + strings.Join(insCols, ",") + ") VALUES (" + strings.Join(ph, ",") + ")")
	if err != nil {
		return err
	}
	defer ins.Close()

	updated, inserted := 0, 0
	for _, m := range spells {
		entry := jsonInt(m["entry"])
		if entry <= 0 {
			continue
		}
		// Shared text + numeric args, in textCols then numCols order.
		text := make([]interface{}, len(textCols))
		for j, c := range textCols {
			s := jsonStr(m[c])
			if c == "iconName" && s == "temp" {
				s = ""
			}
			text[j] = s
		}
		nums := make([]interface{}, len(numCols))
		for j, c := range numCols {
			nums[j] = jsonInt(m[c])
		}

		updArgs := append(append(append([]interface{}{}, text...), nums...), entry)
		res, err := upd.Exec(updArgs...)
		if err != nil {
			continue
		}
		if n, _ := res.RowsAffected(); n > 0 {
			updated++
			continue
		}
		insArgs := append(append(append([]interface{}{entry}, text...), nums...))
		if _, err := ins.Exec(insArgs...); err == nil {
			inserted++
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	fmt.Printf("  ✓ Spell DBC import: %d updated, %d inserted (client DBC wins)\n", updated, inserted)
	return nil
}

// jsonInt coerces a decoded JSON value (float64 from encoding/json) to an int for
// SQL binding; missing/non-numeric yields 0.
func jsonInt(v interface{}) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	}
	return 0
}

// jsonStr coerces a decoded JSON value to a string; missing/non-string yields "".
func jsonStr(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

// talentTabJSON mirrors a record in data/talents.json (datatools.TalentTabOut).
type talentTabJSON struct {
	ID         int    `json:"id"`
	Name       string `json:"name"`
	Class      string `json:"class"`
	ClassMask  int    `json:"classMask"`
	Order      int    `json:"order"`
	Background string `json:"background"`
	Talents    []struct {
		ID        int   `json:"id"`
		Row       int   `json:"row"`
		Col       int   `json:"col"`
		Ranks     []int `json:"ranks"`
		ReqTalent int   `json:"reqTalent"`
		ReqRank   int   `json:"reqRank"`
	} `json:"talents"`
}

// ImportTalents loads the DBC-derived talent trees from talents.json into the
// talent_tab / talent tables. The per-rank spell ids are stored as a JSON array;
// name/description/icon are resolved at query time from spell_template. Existing
// rows are replaced so a re-import refreshes the data.
func (i *GeneratedImporter) ImportTalents(jsonPath string) error {
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		return nil // optional — talents only present after a client import
	}
	var tabs []talentTabJSON
	if err := json.Unmarshal(data, &tabs); err != nil {
		fmt.Printf("  ERROR parsing talents.json: %v\n", err)
		return nil
	}

	tx, err := i.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Clear and repopulate (small, fully derived data set).
	tx.Exec("DELETE FROM talent")
	tx.Exec("DELETE FROM talent_tab")

	tabStmt, err := tx.Prepare(`INSERT OR REPLACE INTO talent_tab
		(id, name, class, class_mask, order_index, background) VALUES (?,?,?,?,?,?)`)
	if err != nil {
		return err
	}
	defer tabStmt.Close()
	talStmt, err := tx.Prepare(`INSERT OR REPLACE INTO talent
		(id, tab_id, row, col, ranks, req_talent, req_rank) VALUES (?,?,?,?,?,?,?)`)
	if err != nil {
		return err
	}
	defer talStmt.Close()

	tabCount, talCount := 0, 0
	for _, t := range tabs {
		if _, err := tabStmt.Exec(t.ID, t.Name, t.Class, t.ClassMask, t.Order, t.Background); err != nil {
			continue
		}
		tabCount++
		for _, tal := range t.Talents {
			ranks, _ := json.Marshal(tal.Ranks)
			if _, err := talStmt.Exec(tal.ID, t.ID, tal.Row, tal.Col, string(ranks), tal.ReqTalent, tal.ReqRank); err != nil {
				continue
			}
			talCount++
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	fmt.Printf("  ✓ Imported %d talent trees, %d talents\n", tabCount, talCount)
	return nil
}

// creatureFamilyJSON mirrors an entry in data/creature_families.json.
type creatureFamilyJSON struct {
	ID   int    `json:"id"`
	Name string `json:"name_loc0"`
}

// ImportCreatureFamilies loads CreatureFamily.dbc names (id -> "Wolf", "Cat", …)
// into creature_family, used to label the NPC Beast family sub-filter. Existing
// rows are replaced so a re-import refreshes the data.
func (i *GeneratedImporter) ImportCreatureFamilies(jsonPath string) error {
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		return nil // optional — only present after a client import
	}
	var fams []creatureFamilyJSON
	if err := json.Unmarshal(data, &fams); err != nil {
		fmt.Printf("  ERROR parsing creature_families.json: %v\n", err)
		return nil
	}
	tx, err := i.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	tx.Exec("DELETE FROM creature_family")
	stmt, err := tx.Prepare("INSERT OR REPLACE INTO creature_family (id, name) VALUES (?,?)")
	if err != nil {
		return err
	}
	defer stmt.Close()
	n := 0
	for _, f := range fams {
		if _, err := stmt.Exec(f.ID, f.Name); err == nil {
			n++
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	fmt.Printf("  ✓ Imported %d creature families\n", n)
	return nil
}

// classJSON mirrors an entry in data/classes.json (ChrClasses.dbc).
type classJSON struct {
	ID    int    `json:"id"`
	Name  string `json:"name_loc0"`
	Token string `json:"token"`
	Color string `json:"color"`
}

// ImportClasses loads ChrClasses.dbc (id -> display name + token) into
// class_info, so class names come from game data rather than being derived from
// tokens. Existing rows are replaced so a re-import refreshes the data.
func (i *GeneratedImporter) ImportClasses(jsonPath string) error {
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		return nil // optional — only present after a client import
	}
	var classes []classJSON
	if err := json.Unmarshal(data, &classes); err != nil {
		fmt.Printf("  ERROR parsing classes.json: %v\n", err)
		return nil
	}
	tx, err := i.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	tx.Exec("DELETE FROM class_info")
	stmt, err := tx.Prepare("INSERT OR REPLACE INTO class_info (id, name, token, color) VALUES (?,?,?,?)")
	if err != nil {
		return err
	}
	defer stmt.Close()
	n := 0
	for _, c := range classes {
		if c.Name == "" {
			continue
		}
		if _, err := stmt.Exec(c.ID, c.Name, c.Token, c.Color); err == nil {
			n++
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	fmt.Printf("  ✓ Imported %d classes\n", n)
	return nil
}

// spellSchoolJSON mirrors an entry in data/spell_schools.json (GlobalStrings.lua).
type spellSchoolJSON struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// ImportSpellSchools loads localized spell-school names (index -> name) from the
// client's GlobalStrings.lua into spell_schools, so school names follow the
// client's locale rather than built-in English. Rows are replaced on re-import.
func (i *GeneratedImporter) ImportSpellSchools(jsonPath string) error {
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		return nil // optional — only present after a client import
	}
	var schools []spellSchoolJSON
	if err := json.Unmarshal(data, &schools); err != nil {
		fmt.Printf("  ERROR parsing spell_schools.json: %v\n", err)
		return nil
	}
	tx, err := i.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	tx.Exec("DELETE FROM spell_schools")
	stmt, err := tx.Prepare("INSERT OR REPLACE INTO spell_schools (id, name) VALUES (?,?)")
	if err != nil {
		return err
	}
	defer stmt.Close()
	n := 0
	for _, s := range schools {
		if s.Name == "" {
			continue
		}
		if _, err := stmt.Exec(s.ID, s.Name); err == nil {
			n++
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	fmt.Printf("  ✓ Imported %d spell schools\n", n)
	return nil
}

// statTypeJSON mirrors an entry in data/stat_names.json (GlobalStrings.lua).
type statTypeJSON struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// ImportStatNames populates stat_types (item stat_type id -> display name). It
// always seeds the full canonical set (helpers.StatNames) so every stat resolves
// even without a client import, then overlays the base stats with the client's
// localized names from stat_names.json when present.
func (i *GeneratedImporter) ImportStatNames(jsonPath string) error {
	tx, err := i.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	tx.Exec("DELETE FROM stat_types")
	stmt, err := tx.Prepare("INSERT OR REPLACE INTO stat_types (id, name) VALUES (?,?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	// 1. Built-in canonical names (covers the rating stats with no client string).
	for id, name := range helpers.StatNames {
		stmt.Exec(id, name)
	}
	// 2. Overlay the client's localized base-stat names (optional file).
	overlaid := 0
	if data, err := os.ReadFile(jsonPath); err == nil {
		var stats []statTypeJSON
		if err := json.Unmarshal(data, &stats); err != nil {
			fmt.Printf("  ERROR parsing stat_names.json: %v\n", err)
		} else {
			for _, s := range stats {
				if s.Name == "" {
					continue
				}
				if _, err := stmt.Exec(s.ID, s.Name); err == nil {
					overlaid++
				}
			}
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	fmt.Printf("  ✓ Imported stat types (%d client-localized)\n", overlaid)
	return nil
}

// importIDName loads a simple [{id,name}] JSON into an (id, name) table. Used
// for the small client-derived spell reference tables. A missing file is a
// no-op (only present after a client import).
func (i *GeneratedImporter) importIDName(jsonPath, table string) error {
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		return nil
	}
	var rows []struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}
	if err := json.Unmarshal(data, &rows); err != nil {
		fmt.Printf("  ERROR parsing %s: %v\n", filepath.Base(jsonPath), err)
		return nil
	}
	tx, err := i.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	tx.Exec("DELETE FROM " + table)
	stmt, err := tx.Prepare("INSERT OR REPLACE INTO " + table + " (id, name) VALUES (?,?)")
	if err != nil {
		return err
	}
	defer stmt.Close()
	n := 0
	for _, r := range rows {
		if r.Name == "" {
			continue
		}
		if _, err := stmt.Exec(r.ID, r.Name); err == nil {
			n++
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	fmt.Printf("  ✓ Imported %d rows into %s\n", n, table)
	return nil
}

// ImportSpellMechanics loads spell_mechanics.json (SpellMechanic.dbc) into spell_mechanics.
func (i *GeneratedImporter) ImportSpellMechanics(jsonPath string) error {
	return i.importIDName(jsonPath, "spell_mechanics")
}

// ImportDispelTypes loads spell_dispel_types.json (SpellDispelType.dbc) into spell_dispel_types.
func (i *GeneratedImporter) ImportDispelTypes(jsonPath string) error {
	return i.importIDName(jsonPath, "spell_dispel_types")
}

// ImportLockTypes loads lock_types.json (LockType.dbc) into lock_types.
func (i *GeneratedImporter) ImportLockTypes(jsonPath string) error {
	return i.importIDName(jsonPath, "lock_types")
}

// ImportEnchantProcSpells loads enchant_proc_spells.json (the on-hit enchant proc
// spell ids from SpellItemEnchantment.dbc) into enchant_proc_spells. Optional.
func (i *GeneratedImporter) ImportEnchantProcSpells(jsonPath string) error {
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		return nil
	}
	var rows []struct {
		ID int `json:"id"`
	}
	if err := json.Unmarshal(data, &rows); err != nil {
		fmt.Printf("  ERROR parsing enchant_proc_spells.json: %v\n", err)
		return nil
	}
	tx, err := i.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	tx.Exec("DELETE FROM enchant_proc_spells")
	stmt, err := tx.Prepare("INSERT OR REPLACE INTO enchant_proc_spells (id) VALUES (?)")
	if err != nil {
		return err
	}
	defer stmt.Close()
	for _, r := range rows {
		stmt.Exec(r.ID)
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	fmt.Printf("  ✓ Imported %d enchant proc spells\n", len(rows))
	return nil
}

// raceJSON mirrors an entry in data/races.json (ChrRaces + glue strings +
// CharBaseInfo + racial skill lines).
type raceJSON struct {
	ID             int      `json:"id"`
	Name           string   `json:"name"`
	FileString     string   `json:"fileString"`
	Prefix         string   `json:"prefix"`
	Faction        string   `json:"faction"`
	Info           string   `json:"info"`
	Abilities      []string `json:"abilities"`
	ClassIDs       []int    `json:"classIds"`
	RacialSpellIDs []int    `json:"racialSpellIds"`
}

// ImportRaces loads the client-derived race data (races + racial blurbs +
// available classes + racial spell ids) into the race tables. Optional: a
// missing file (no client import yet) is a no-op.
func (i *GeneratedImporter) ImportRaces(jsonPath string) error {
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		return nil
	}
	var races []raceJSON
	if err := json.Unmarshal(data, &races); err != nil {
		fmt.Printf("  ERROR parsing races.json: %v\n", err)
		return nil
	}
	tx, err := i.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, t := range []string{"races", "race_abilities", "race_classes", "race_spells"} {
		tx.Exec("DELETE FROM " + t)
	}
	raceStmt, _ := tx.Prepare("INSERT OR REPLACE INTO races (id, name, file_string, prefix, faction, info) VALUES (?,?,?,?,?,?)")
	abilStmt, _ := tx.Prepare("INSERT OR REPLACE INTO race_abilities (race_id, idx, text) VALUES (?,?,?)")
	classStmt, _ := tx.Prepare("INSERT OR REPLACE INTO race_classes (race_id, class_id) VALUES (?,?)")
	spellStmt, _ := tx.Prepare("INSERT OR REPLACE INTO race_spells (race_id, spell_id) VALUES (?,?)")
	defer raceStmt.Close()
	defer abilStmt.Close()
	defer classStmt.Close()
	defer spellStmt.Close()

	for _, r := range races {
		if r.Name == "" {
			continue
		}
		raceStmt.Exec(r.ID, r.Name, r.FileString, r.Prefix, r.Faction, r.Info)
		for idx, text := range r.Abilities {
			abilStmt.Exec(r.ID, idx, text)
		}
		for _, c := range r.ClassIDs {
			classStmt.Exec(r.ID, c)
		}
		for _, s := range r.RacialSpellIDs {
			spellStmt.Exec(r.ID, s)
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	fmt.Printf("  ✓ Imported %d races\n", len(races))
	return nil
}

// lockJSON mirrors an entry in data/locks.json (Lock.dbc, first 5 slots).
type lockJSON struct {
	ID    int `json:"id"`
	Type1 int `json:"type1"`
	Type2 int `json:"type2"`
	Type3 int `json:"type3"`
	Type4 int `json:"type4"`
	Type5 int `json:"type5"`
	Prop1 int `json:"prop1"`
	Prop2 int `json:"prop2"`
	Prop3 int `json:"prop3"`
	Prop4 int `json:"prop4"`
	Prop5 int `json:"prop5"`
	Req1  int `json:"req1"`
	Req2  int `json:"req2"`
	Req3  int `json:"req3"`
	Req4  int `json:"req4"`
	Req5  int `json:"req5"`
}

// ImportLocks loads Lock.dbc (lock id -> required skill/key per slot) into the
// locks table. Gameobject_template.data0 references a lock id; the Objects page
// derives Herbalism/Mining/Lockpicking categories by joining type-3 chests to
// these rows. Existing rows are replaced so a re-import refreshes the data.
func (i *GeneratedImporter) ImportLocks(jsonPath string) error {
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		return nil // optional — only present after a client import
	}
	var locks []lockJSON
	if err := json.Unmarshal(data, &locks); err != nil {
		fmt.Printf("  ERROR parsing locks.json: %v\n", err)
		return nil
	}
	tx, err := i.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	tx.Exec("DELETE FROM locks")
	stmt, err := tx.Prepare(`INSERT OR REPLACE INTO locks
		(id, type1, type2, type3, type4, type5, prop1, prop2, prop3, prop4, prop5, req1, req2, req3, req4, req5)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	n := 0
	for _, l := range locks {
		if _, err := stmt.Exec(l.ID, l.Type1, l.Type2, l.Type3, l.Type4, l.Type5,
			l.Prop1, l.Prop2, l.Prop3, l.Prop4, l.Prop5,
			l.Req1, l.Req2, l.Req3, l.Req4, l.Req5); err == nil {
			n++
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	fmt.Printf("  ✓ Imported %d locks\n", n)
	return nil
}

// taxiJSON mirrors data/taxi.json (datatools.TaxiData).
type taxiJSON struct {
	Continents []struct {
		MapID int    `json:"mapId"`
		Name  string `json:"name"`
	} `json:"continents"`
	Nodes []struct {
		ID       int     `json:"id"`
		MapID    int     `json:"mapId"`
		Name     string  `json:"name"`
		Alliance bool    `json:"alliance"`
		Horde    bool    `json:"horde"`
		PX       float64 `json:"px"`
		PY       float64 `json:"py"`
	} `json:"nodes"`
	Paths []struct {
		ID   int `json:"id"`
		From int `json:"from"`
		To   int `json:"to"`
	} `json:"paths"`
	Waypoints []struct {
		PathID int     `json:"pathId"`
		Idx    int     `json:"idx"`
		PX     float64 `json:"px"`
		PY     float64 `json:"py"`
	} `json:"waypoints"`
	Transports []struct {
		ID    int    `json:"id"`
		Type  string `json:"type"`
		NameA string `json:"nameA"`
		MapA  int    `json:"mapA"`
		NameB string `json:"nameB"`
		MapB  int    `json:"mapB"`
	} `json:"transports"`
	TransportWps []struct {
		RouteID int     `json:"routeId"`
		Idx     int     `json:"idx"`
		MapID   int     `json:"mapId"`
		PX      float64 `json:"px"`
		PY      float64 `json:"py"`
	} `json:"transportWps"`
}

// ImportTaxi loads the DBC-derived flight network from taxi.json into the
// taxi_node / taxi_path / taxi_path_node tables. Clears and repopulates (small,
// fully derived). Continent name is resolved from the continents list by map id.
func (i *GeneratedImporter) ImportTaxi(jsonPath string) error {
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		return nil // optional — only present after a client import
	}
	var t taxiJSON
	if err := json.Unmarshal(data, &t); err != nil {
		fmt.Printf("  ERROR parsing taxi.json: %v\n", err)
		return nil
	}
	contName := map[int]string{}
	for _, c := range t.Continents {
		contName[c.MapID] = c.Name
	}

	tx, err := i.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	tx.Exec("DELETE FROM taxi_node")
	tx.Exec("DELETE FROM taxi_path")
	tx.Exec("DELETE FROM taxi_path_node")
	tx.Exec("DELETE FROM transport_route")
	tx.Exec("DELETE FROM transport_waypoint")

	nodeStmt, _ := tx.Prepare(`INSERT OR REPLACE INTO taxi_node
		(id, map_id, continent, name, alliance, horde, px, py) VALUES (?,?,?,?,?,?,?,?)`)
	defer nodeStmt.Close()
	pathStmt, _ := tx.Prepare(`INSERT OR REPLACE INTO taxi_path (id, from_node, to_node) VALUES (?,?,?)`)
	defer pathStmt.Close()
	wpStmt, _ := tx.Prepare(`INSERT OR REPLACE INTO taxi_path_node (path_id, idx, px, py) VALUES (?,?,?,?)`)
	defer wpStmt.Close()

	b2i := func(b bool) int {
		if b {
			return 1
		}
		return 0
	}
	for _, n := range t.Nodes {
		nodeStmt.Exec(n.ID, n.MapID, contName[n.MapID], n.Name, b2i(n.Alliance), b2i(n.Horde), n.PX, n.PY)
	}
	for _, p := range t.Paths {
		pathStmt.Exec(p.ID, p.From, p.To)
	}
	for _, w := range t.Waypoints {
		wpStmt.Exec(w.PathID, w.Idx, w.PX, w.PY)
	}

	trStmt, _ := tx.Prepare(`INSERT OR REPLACE INTO transport_route
		(id, type, name_a, map_a, name_b, map_b) VALUES (?,?,?,?,?,?)`)
	defer trStmt.Close()
	twStmt, _ := tx.Prepare(`INSERT OR REPLACE INTO transport_waypoint
		(route_id, idx, map_id, px, py) VALUES (?,?,?,?,?)`)
	defer twStmt.Close()
	for _, tr := range t.Transports {
		trStmt.Exec(tr.ID, tr.Type, tr.NameA, tr.MapA, tr.NameB, tr.MapB)
	}
	for _, w := range t.TransportWps {
		twStmt.Exec(w.RouteID, w.Idx, w.MapID, w.PX, w.PY)
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	fmt.Printf("  ✓ Imported %d flight nodes, %d paths, %d waypoints; %d transports\n",
		len(t.Nodes), len(t.Paths), len(t.Waypoints), len(t.Transports))
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
