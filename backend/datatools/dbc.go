package datatools

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// --- DBC reader (classic WDBC: 20-byte header, fixed records, string block) ---

type DBC struct {
	RecordCount int
	FieldCount  int
	RecordSize  int

	data       []byte
	recordsOff int
	stringsOff int
}

func openDBC(path string) (*DBC, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return openDBCBytes(b, path)
}

// openDBCBytes parses a WDBC blob already in memory (e.g. read from an MPQ).
// label is used only in error messages.
func openDBCBytes(b []byte, label string) (*DBC, error) {
	if len(b) < 20 || string(b[0:4]) != "WDBC" {
		return nil, fmt.Errorf("%s: not a WDBC file", label)
	}
	rc := int(binary.LittleEndian.Uint32(b[4:8]))
	fc := int(binary.LittleEndian.Uint32(b[8:12]))
	rs := int(binary.LittleEndian.Uint32(b[12:16]))
	ss := int(binary.LittleEndian.Uint32(b[16:20]))
	recordsOff := 20
	stringsOff := recordsOff + rc*rs
	if stringsOff+ss > len(b) {
		return nil, fmt.Errorf("%s: truncated", label)
	}
	return &DBC{RecordCount: rc, FieldCount: fc, RecordSize: rs, data: b, recordsOff: recordsOff, stringsOff: stringsOff}, nil
}

// openDBCFrom reads a DBC from a ClientFiles source by bare name.
func openDBCFrom(cf ClientFiles, name string) (*DBC, error) {
	b, err := cf.ReadDBC(name)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", name, err)
	}
	return openDBCBytes(b, name)
}

func (d *DBC) off(rec, field int) int { return d.recordsOff + rec*d.RecordSize + field*4 }
func (d *DBC) U32(rec, field int) uint32 {
	o := d.off(rec, field)
	if o+4 > len(d.data) {
		return 0
	}
	return binary.LittleEndian.Uint32(d.data[o : o+4])
}
func (d *DBC) I32(rec, field int) int32     { return int32(d.U32(rec, field)) }
func (d *DBC) F32(rec, field int) float32   { return math.Float32frombits(d.U32(rec, field)) }
// U8 reads a single byte at byteOff within the record. Used for byte-packed
// DBCs like CharBaseInfo.dbc (recordSize 2: raceID, classID) where the 4-byte
// field accessors don't apply.
func (d *DBC) U8(rec, byteOff int) byte {
	o := d.recordsOff + rec*d.RecordSize + byteOff
	if o >= len(d.data) {
		return 0
	}
	return d.data[o]
}
func (d *DBC) Str(rec, field int) string {
	start := d.stringsOff + int(d.U32(rec, field))
	if start < d.stringsOff || start >= len(d.data) {
		return ""
	}
	end := start
	for end < len(d.data) && d.data[end] != 0 {
		end++
	}
	return string(d.data[start:end])
}

// GenerateDBCJSON regenerates every data/*.json file InkLab imports from the
// client DBCs in dbcDir, writing into dataDir.
func GenerateDBCJSON(dbcDir, dataDir string) error {
	return GenerateDBCJSONFrom(NewDirSourceDBC(dbcDir), dataDir)
}

// GenerateDBCJSONFrom regenerates the data/*.json files from any ClientFiles
// source (loose folder or in-memory MPQ), writing into dataDir.
func GenerateDBCJSONFrom(cf ClientFiles, dataDir string) error {
	if err := writeFactions(cf, filepath.Join(dataDir, "factions.json")); err != nil {
		return fmt.Errorf("factions: %w", err)
	}
	// FactionTemplate maps creature.faction (a template id) to a Faction.dbc id,
	// used to list faction members. Non-fatal: a missing/odd DBC shouldn't break
	// the whole client import.
	if err := writeFactionTemplates(cf, filepath.Join(dataDir, "faction_templates.json")); err != nil {
		fmt.Printf("[dbc] faction_templates skipped: %v\n", err)
	}
	jobs := []struct{ name, file string }{
		{"itemsets", "item_sets.json"},
		{"skills", "skills.json"},
		{"sla", "skill_line_abilities.json"},
		{"zones", "zones.json"},
		{"questsorts", "quest_sorts.json"},
		{"icons", "item_icons.json"},
		{"spells", "spells_enhanced.json"},
		{"talents", "talents.json"},
		{"taxi", "taxi.json"},
		{"creaturefamilies", "creature_families.json"},
		{"locks", "locks.json"},
		{"classes", "classes.json"},
		{"spellschools", "spell_schools.json"},
		{"itemmods", "stat_names.json"},
		{"races", "races.json"},
		{"mechanics", "spell_mechanics.json"},
		{"dispeltypes", "spell_dispel_types.json"},
		{"enchantprocs", "enchant_proc_spells.json"},
		{"locktypes", "lock_types.json"},
	}
	for _, j := range jobs {
		if err := runGen(j.name, cf, filepath.Join(dataDir, j.file)); err != nil {
			return fmt.Errorf("%s: %w", j.name, err)
		}
	}
	return nil
}

func runGen(name string, cf ClientFiles, out string) error {
	var v interface{}
	var err error
	switch name {
	case "itemsets":
		v, err = genItemSets(cf)
	case "skills":
		v, err = genSkills(cf)
	case "sla":
		v, err = genSLA(cf)
	case "zones":
		v, err = genZones(cf)
	case "questsorts":
		v, err = genQuestSorts(cf)
	case "icons":
		v, err = genIcons(cf)
	case "spells":
		v, err = genSpells(cf)
	case "talents":
		v, err = genTalents(cf)
	case "taxi":
		v, err = genTaxi(cf)
	case "creaturefamilies":
		v, err = genCreatureFamilies(cf)
	case "locks":
		v, err = genLocks(cf)
	case "classes":
		v, err = genClasses(cf)
	case "spellschools":
		v, err = genSpellSchools(cf)
	case "itemmods":
		v, err = genItemMods(cf)
	case "races":
		v, err = genRaces(cf)
	case "mechanics":
		v, err = genMechanics(cf)
	case "dispeltypes":
		v, err = genDispelTypes(cf)
	case "enchantprocs":
		v, err = genEnchantProcSpells(cf)
	case "locktypes":
		v, err = genLockTypes(cf)
	default:
		return fmt.Errorf("unknown gen %q", name)
	}
	if err != nil {
		return err
	}
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return os.WriteFile(out, b, 0644)
}

func iconBase(p string) string {
	if i := strings.LastIndexAny(p, `\/`); i >= 0 {
		p = p[i+1:]
	}
	return p
}

// ItemSet.dbc (45 fields): id(0), name[8](1-8), itemID[17](10-26),
// setSpellID[8](27-34), setThreshold[8](35-42), reqSkill(43), reqSkillRank(44).
func genItemSets(cf ClientFiles) (interface{}, error) {
	d, err := openDBCFrom(cf, "ItemSet.dbc")
	if err != nil {
		return nil, err
	}
	out := make([]map[string]interface{}, 0, d.RecordCount)
	for r := 0; r < d.RecordCount; r++ {
		m := map[string]interface{}{
			"itemsetID": d.U32(r, 0), "name_loc0": d.Str(r, 1),
			"skillID": d.U32(r, 43), "skilllevel": d.U32(r, 44),
		}
		for i := 0; i < 10; i++ {
			m[fmt.Sprintf("item%d", i+1)] = d.U32(r, 10+i)
		}
		for i := 0; i < 8; i++ {
			m[fmt.Sprintf("spell%d", i+1)] = d.U32(r, 27+i)
		}
		for i := 0; i < 8; i++ {
			m[fmt.Sprintf("bonus%d", i+1)] = d.U32(r, 35+i)
		}
		out = append(out, m)
	}
	return out, nil
}

func genSkills(cf ClientFiles) (interface{}, error) {
	d, err := openDBCFrom(cf, "SkillLine.dbc")
	if err != nil {
		return nil, err
	}
	out := make([]map[string]interface{}, 0, d.RecordCount)
	for r := 0; r < d.RecordCount; r++ {
		out = append(out, map[string]interface{}{
			"skillID": d.U32(r, 0), "categoryID": d.I32(r, 1), "name_loc0": d.Str(r, 3),
		})
	}
	return out, nil
}

// CreatureFamily.dbc (18 fields): id(0), name_loc[8](8-15), enUS at field 8.
// Maps creature_template.beast_family -> a family name (Wolf, Cat, …). Octo's
// ids extend the vanilla set (e.g. 35 Serpent, 36 Fox), so they're read from
// the client rather than assumed.
func genCreatureFamilies(cf ClientFiles) (interface{}, error) {
	d, err := openDBCFrom(cf, "CreatureFamily.dbc")
	if err != nil {
		return nil, err
	}
	out := make([]map[string]interface{}, 0, d.RecordCount)
	for r := 0; r < d.RecordCount; r++ {
		name := d.Str(r, 8)
		if name == "" {
			continue
		}
		out = append(out, map[string]interface{}{
			"id": d.U32(r, 0), "name_loc0": name,
		})
	}
	return out, nil
}

// Lock.dbc (33 fields): id(0), Type[8](1-8), Property/Index[8](9-16),
// RequiredSkill[8](17-24), Action[8](25-32). We keep the first 5 slots, enough
// for the gathering/lockpicking categories: a skill lock has Type=2 and
// Property = LockType id (Herbalism=2, Mining=3, Lockpicking=1).
func genLocks(cf ClientFiles) (interface{}, error) {
	d, err := openDBCFrom(cf, "Lock.dbc")
	if err != nil {
		return nil, err
	}
	out := make([]map[string]interface{}, 0, d.RecordCount)
	for r := 0; r < d.RecordCount; r++ {
		m := map[string]interface{}{"id": d.U32(r, 0)}
		for i := 1; i <= 5; i++ {
			m[fmt.Sprintf("type%d", i)] = d.U32(r, i)
			m[fmt.Sprintf("prop%d", i)] = d.U32(r, 8+i)
			m[fmt.Sprintf("req%d", i)] = d.U32(r, 16+i)
		}
		out = append(out, m)
	}
	return out, nil
}

// classColorRe matches a RAID_CLASS_COLORS entry in the client FrameXML, e.g.
// ["WARRIOR"] = { r = 0.78, g = 0.61, b = 0.43, hex = "|cffc79c6e" }.
var classColorRe = regexp.MustCompile(`\["(\w+)"\][^}]*?hex\s*=\s*"\|c[fF][fF]([0-9a-fA-F]{6})"`)

// readClassColors pulls the RAID_CLASS_COLORS table (token -> "#rrggbb") from the
// client's FrameXML/Fonts.xml. Class colors aren't in a DBC in 1.12 — they live
// in the UI code — so this is best-effort: returns nil if the file isn't found.
func readClassColors(cf ClientFiles) map[string]string {
	for _, p := range []string{`BlizzardInterfaceCode\FrameXML\Fonts.xml`, `Interface\FrameXML\Fonts.xml`} {
		b, err := cf.ReadFile(p)
		if err != nil {
			continue
		}
		out := map[string]string{}
		for _, m := range classColorRe.FindAllStringSubmatch(string(b), -1) {
			out[strings.ToUpper(m[1])] = "#" + strings.ToLower(m[2])
		}
		if len(out) > 0 {
			return out
		}
	}
	return nil
}

// ChrClasses.dbc (17 fields): id(0), name_loc[8](5-12, enUS at 5), token(14).
// Provides player class display names ("Warrior", …); class colors come from the
// FrameXML RAID_CLASS_COLORS table (also client data, just not a DBC).
func genClasses(cf ClientFiles) (interface{}, error) {
	d, err := openDBCFrom(cf, "ChrClasses.dbc")
	if err != nil {
		return nil, err
	}
	colors := readClassColors(cf)
	out := make([]map[string]interface{}, 0, d.RecordCount)
	for r := 0; r < d.RecordCount; r++ {
		token := d.Str(r, 14)
		out = append(out, map[string]interface{}{
			"id": d.U32(r, 0), "name_loc0": d.Str(r, 5), "token": token,
			"color": colors[strings.ToUpper(token)],
		})
	}
	return out, nil
}

// spellSchoolRe matches a localized spell-school name in the client FrameXML
// GlobalStrings.lua, e.g. SPELL_SCHOOL3_CAP = "Nature".
var spellSchoolRe = regexp.MustCompile(`SPELL_SCHOOL(\d+)_CAP\s*=\s*"([^"]*)"`)

// genSpellSchools reads the spell-school display names from the client's
// FrameXML/GlobalStrings.lua (index -> localized name). Schools aren't in a DBC;
// the names live in the UI strings, so they follow the client's locale.
// Best-effort: returns an empty set if the file isn't present (callers fall back
// to built-in English names).
func genSpellSchools(cf ClientFiles) (interface{}, error) {
	var b []byte
	for _, p := range []string{`BlizzardInterfaceCode\FrameXML\GlobalStrings.lua`, `Interface\FrameXML\GlobalStrings.lua`} {
		if data, err := cf.ReadFile(p); err == nil {
			b = data
			break
		}
	}
	out := make([]map[string]interface{}, 0, 7)
	if b == nil {
		fmt.Printf("[dbc] spell schools skipped: GlobalStrings.lua not found\n")
		return out, nil
	}
	for _, m := range spellSchoolRe.FindAllStringSubmatch(string(b), -1) {
		id, err := strconv.Atoi(m[1])
		if err != nil {
			continue
		}
		out = append(out, map[string]interface{}{"id": id, "name": m[2]})
	}
	return out, nil
}

// spellStatNameRe matches the character-sheet stat names in GlobalStrings.lua,
// e.g. SPELL_STAT0_NAME = "Strength". These are the only clean (un-templated)
// stat names the 1.12 client exposes.
var spellStatNameRe = regexp.MustCompile(`SPELL_STAT(\d+)_NAME\s*=\s*"([^"]*)"`)

// spellStatToItemMod maps the SPELL_STATn_NAME index to its item stat_type id
// (the ITEM_MOD enum): 0 Strength, 1 Agility, 2 Stamina, 3 Intellect, 4 Spirit.
// The ordering differs from the stat_type ids, so the mapping is explicit.
var spellStatToItemMod = []int{4, 3, 7, 5, 6}

// genItemMods reads the base item-stat display names from the client's
// FrameXML/GlobalStrings.lua, keyed by item stat_type id. Only the five primary
// stats have client strings in 1.12 (via SPELL_STATn_NAME); the secondary/rating
// stats aren't in the client and keep their built-in names (helpers.StatNames).
// Best-effort: returns an empty set if the file isn't present.
func genItemMods(cf ClientFiles) (interface{}, error) {
	var b []byte
	for _, p := range []string{`BlizzardInterfaceCode\FrameXML\GlobalStrings.lua`, `Interface\FrameXML\GlobalStrings.lua`} {
		if data, err := cf.ReadFile(p); err == nil {
			b = data
			break
		}
	}
	out := make([]map[string]interface{}, 0, len(spellStatToItemMod))
	if b == nil {
		fmt.Printf("[dbc] item mods skipped: GlobalStrings.lua not found\n")
		return out, nil
	}
	for _, m := range spellStatNameRe.FindAllStringSubmatch(string(b), -1) {
		idx, err := strconv.Atoi(m[1])
		if err != nil || idx < 0 || idx >= len(spellStatToItemMod) {
			continue
		}
		out = append(out, map[string]interface{}{"id": spellStatToItemMod[idx], "name": m[2]})
	}
	return out, nil
}

// Race flavor/ability text comes from the glue (character-create) strings, not
// the in-game GlobalStrings. RACE_INFO_<FILESTRING> is the select-screen
// paragraph; ABILITY_INFO_<FILESTRING><n> are the racial trait blurbs.
var raceInfoRe = regexp.MustCompile(`RACE_INFO_([A-Z]+)\s*=\s*"([^"]*)"`)
var raceAbilityRe = regexp.MustCompile(`ABILITY_INFO_([A-Z]+)(\d+)\s*=\s*"([^"]*)"`)

// raceGen is one race assembled entirely from client data for races.json.
type raceGen struct {
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

// readGlueStrings returns the client's GlueXML/GlueStrings.lua (character-create
// UI strings), or "" if not present.
func readGlueStrings(cf ClientFiles) string {
	for _, p := range []string{`BlizzardInterfaceCode\GlueXML\GlueStrings.lua`, `Interface\GlueXML\GlueStrings.lua`} {
		if b, err := cf.ReadFile(p); err == nil {
			return string(b)
		}
	}
	return ""
}

// readCharBaseInfo maps each race id to its available class ids from the
// byte-packed CharBaseInfo.dbc (each record is raceID, classID).
func readCharBaseInfo(cf ClientFiles) map[int][]int {
	out := map[int][]int{}
	d, err := openDBCFrom(cf, "CharBaseInfo.dbc")
	if err != nil {
		fmt.Printf("[dbc] CharBaseInfo skipped: %v\n", err)
		return out
	}
	for r := 0; r < d.RecordCount; r++ {
		race := int(d.U8(r, 0))
		cls := int(d.U8(r, 1))
		if race == 0 || cls == 0 {
			continue
		}
		out[race] = append(out[race], cls)
	}
	return out
}

// readRacialSpells maps each race id to its racial spell ids by scanning the
// racial skill lines (SkillLine names containing "Racial") and assigning each
// SkillLineAbility spell to the races in its race bitmask.
func readRacialSpells(cf ClientFiles) map[int][]int {
	out := map[int][]int{}
	skills, err := openDBCFrom(cf, "SkillLine.dbc")
	if err != nil {
		fmt.Printf("[dbc] SkillLine skipped for racials: %v\n", err)
		return out
	}
	racial := map[uint32]bool{}
	for r := 0; r < skills.RecordCount; r++ {
		if strings.Contains(skills.Str(r, 3), "Racial") {
			racial[skills.U32(r, 0)] = true
		}
	}
	sla, err := openDBCFrom(cf, "SkillLineAbility.dbc")
	if err != nil {
		fmt.Printf("[dbc] SkillLineAbility skipped for racials: %v\n", err)
		return out
	}
	seen := map[[2]int]bool{}
	for r := 0; r < sla.RecordCount; r++ {
		if !racial[sla.U32(r, 1)] {
			continue
		}
		spell := int(sla.U32(r, 2))
		racemask := sla.U32(r, 3)
		if racemask == 0 {
			continue // would apply to every race; skip rather than over-assign
		}
		for race := 1; race <= 32; race++ {
			if racemask&(1<<uint(race-1)) == 0 {
				continue
			}
			key := [2]int{race, spell}
			if seen[key] {
				continue
			}
			seen[key] = true
			out[race] = append(out[race], spell)
		}
	}
	return out
}

// genRaces assembles the playable races entirely from client data: ChrRaces.dbc
// (id, name, client prefix, base language → faction), CharBaseInfo.dbc (available
// classes), the glue strings (select-screen flavor + racial blurbs), and the
// racial skill lines (clickable racial spell ids).
func genRaces(cf ClientFiles) (interface{}, error) {
	races, err := openDBCFrom(cf, "ChrRaces.dbc")
	if err != nil {
		return nil, err
	}
	glue := readGlueStrings(cf)
	info := map[string]string{}
	for _, m := range raceInfoRe.FindAllStringSubmatch(glue, -1) {
		info[m[1]] = strings.TrimSpace(m[2])
	}
	abilities := map[string]map[int]string{}
	for _, m := range raceAbilityRe.FindAllStringSubmatch(glue, -1) {
		idx, _ := strconv.Atoi(m[2])
		if abilities[m[1]] == nil {
			abilities[m[1]] = map[int]string{}
		}
		abilities[m[1]][idx] = strings.TrimSpace(m[3])
	}
	classByRace := readCharBaseInfo(cf)
	spellByRace := readRacialSpells(cf)

	out := make([]raceGen, 0, races.RecordCount)
	for r := 0; r < races.RecordCount; r++ {
		id := int(races.U32(r, 0))
		fileStr := races.Str(r, 15)
		up := strings.ToUpper(fileStr)
		faction := ""
		switch races.U32(r, 8) { // base language: 7 Common, 1 Orcish
		case 7:
			faction = "Alliance"
		case 1:
			faction = "Horde"
		}
		// Racial ability blurbs in index order.
		var abil []string
		for i := 1; i <= 12; i++ {
			if t, ok := abilities[up][i]; ok {
				abil = append(abil, t)
			}
		}
		out = append(out, raceGen{
			ID:             id,
			Name:           races.Str(r, 17),
			FileString:     fileStr,
			Prefix:         races.Str(r, 6),
			Faction:        faction,
			Info:           info[up],
			Abilities:      abil,
			ClassIDs:       classByRace[id],
			RacialSpellIDs: spellByRace[id],
		})
	}
	return out, nil
}

// genMechanics reads SpellMechanic.dbc (id -> name, e.g. "rooted") — the spell
// mechanic display names, which (unlike effect/aura type names) do live in the
// client.
func genMechanics(cf ClientFiles) (interface{}, error) {
	return genIDName(cf, "SpellMechanic.dbc")
}

// genDispelTypes reads SpellDispelType.dbc (id -> name, e.g. "Magic").
func genDispelTypes(cf ClientFiles) (interface{}, error) {
	return genIDName(cf, "SpellDispelType.dbc")
}

// genLockTypes reads LockType.dbc (id -> name, e.g. "Herbalism", "Mining",
// "Survival"). Drives the derived gathering/lock object categories.
func genLockTypes(cf ClientFiles) (interface{}, error) {
	return genIDName(cf, "LockType.dbc")
}

// genEnchantProcSpells reads SpellItemEnchantment.dbc and returns the set of
// spell ids that are on-hit weapon-enchant procs (enchant slot type 1 =
// COMBAT_SPELL). These resolve to a 1 PPM default when they have no explicit
// rate in spell_proc_item_enchant (matching the server's enchant proc fallback).
// Layout: type[3] at fields 1-3, spellid[3] at fields 10-12.
func genEnchantProcSpells(cf ClientFiles) (interface{}, error) {
	d, err := openDBCFrom(cf, "SpellItemEnchantment.dbc")
	if err != nil {
		return nil, err
	}
	seen := map[uint32]bool{}
	out := make([]map[string]interface{}, 0)
	for r := 0; r < d.RecordCount; r++ {
		for slot := 0; slot < 3; slot++ {
			if d.U32(r, 1+slot) != 1 { // ITEM_ENCHANTMENT_TYPE_COMBAT_SPELL
				continue
			}
			spellID := d.U32(r, 10+slot)
			if spellID == 0 || seen[spellID] {
				continue
			}
			seen[spellID] = true
			out = append(out, map[string]interface{}{"id": spellID})
		}
	}
	return out, nil
}

// genIDName reads a simple DBC whose first field is the id and field 1 is the
// enUS name, into [{id, name}].
func genIDName(cf ClientFiles, dbc string) (interface{}, error) {
	d, err := openDBCFrom(cf, dbc)
	if err != nil {
		return nil, err
	}
	out := make([]map[string]interface{}, 0, d.RecordCount)
	for r := 0; r < d.RecordCount; r++ {
		name := d.Str(r, 1)
		if name == "" {
			continue
		}
		out = append(out, map[string]interface{}{"id": d.U32(r, 0), "name": name})
	}
	return out, nil
}

func genSLA(cf ClientFiles) (interface{}, error) {
	d, err := openDBCFrom(cf, "SkillLineAbility.dbc")
	if err != nil {
		return nil, err
	}
	out := make([]map[string]interface{}, 0, d.RecordCount)
	for r := 0; r < d.RecordCount; r++ {
		out = append(out, map[string]interface{}{
			"skillID": d.U32(r, 1), "spellID": d.U32(r, 2),
			"racemask": d.U32(r, 3), "classmask": d.U32(r, 4),
			"req_skill_value": d.U32(r, 7), "max_value": d.U32(r, 10), "min_value": d.U32(r, 11),
		})
	}
	return out, nil
}

// WorldMapArea.dbc: mapID(1), areaID(2), areaName(3), loc bounds(4-7).
func genZones(cf ClientFiles) (interface{}, error) {
	maps, err := openDBCFrom(cf, "Map.dbc")
	if err != nil {
		return nil, err
	}
	instByMap := make(map[uint32]uint32, maps.RecordCount)
	for r := 0; r < maps.RecordCount; r++ {
		instByMap[maps.U32(r, 0)] = maps.U32(r, 2)
	}
	d, err := openDBCFrom(cf, "WorldMapArea.dbc")
	if err != nil {
		return nil, err
	}
	out := make([]map[string]interface{}, 0, d.RecordCount)
	for r := 0; r < d.RecordCount; r++ {
		mapID := d.U32(r, 1)
		out = append(out, map[string]interface{}{
			"mapID": mapID, "instanceType": instByMap[mapID],
			"areatableID": d.U32(r, 2), "name_loc0": d.Str(r, 3),
			"x_min": d.F32(r, 7), "x_max": d.F32(r, 6),
			"y_min": d.F32(r, 5), "y_max": d.F32(r, 4),
		})
	}
	return out, nil
}

// QuestSort.dbc: id(0), name[8](1-8).
func genQuestSorts(cf ClientFiles) (interface{}, error) {
	d, err := openDBCFrom(cf, "QuestSort.dbc")
	if err != nil {
		return nil, err
	}
	out := make([]map[string]interface{}, 0, d.RecordCount)
	for r := 0; r < d.RecordCount; r++ {
		out = append(out, map[string]interface{}{"sortID": d.U32(r, 0), "name_loc0": d.Str(r, 1)})
	}
	return out, nil
}

// ItemDisplayInfo.dbc: id(0), inventoryIcon1(5). -> map displayID -> icon name.
func genIcons(cf ClientFiles) (interface{}, error) {
	d, err := openDBCFrom(cf, "ItemDisplayInfo.dbc")
	if err != nil {
		return nil, err
	}
	out := make(map[string]string, d.RecordCount)
	for r := 0; r < d.RecordCount; r++ {
		icon := d.Str(r, 5)
		if icon == "" {
			continue
		}
		out[fmt.Sprintf("%d", d.U32(r, 0))] = icon
	}
	return out, nil
}

// Spell.dbc (1.12, 173 fields): id(0), procChance(25), procCharges(26),
// durationIndex(30), rangeIndex(36), effectDieSides[3](64-66),
// effectBasePoints[3](76-78), effectRadiusIndex[3](88-90),
// effectAmplitude[3](94-96), effectChainTarget[3](100-102), spellIconID(117),
// name[8](120-127), description[8](138-145), maxTargetLevel(159),
// maxAffectedTargets(163). iconName via SpellIcon.dbc: id(0)->texturePath(1).
//
// These are exactly the fields the local $-placeholder resolver reads, so the
// client DBC can be the authoritative source for spell text + values (it is the
// patched, current data for a custom server) — see ImportSpellsFromDBC.
func genSpells(cf ClientFiles) (interface{}, error) {
	icons, err := openDBCFrom(cf, "SpellIcon.dbc")
	if err != nil {
		return nil, err
	}
	iconByID := make(map[uint32]string, icons.RecordCount)
	for r := 0; r < icons.RecordCount; r++ {
		iconByID[icons.U32(r, 0)] = iconBase(icons.Str(r, 1))
	}
	d, err := openDBCFrom(cf, "Spell.dbc")
	if err != nil {
		return nil, err
	}
	out := make([]map[string]interface{}, 0, d.RecordCount)
	for r := 0; r < d.RecordCount; r++ {
		iconID := d.U32(r, 117)
		name := iconByID[iconID]
		if name == "" {
			name = "temp"
		}
		out = append(out, map[string]interface{}{
			"entry": d.U32(r, 0), "name": d.Str(r, 120), "description": d.Str(r, 138),
			"effectBasePoints1": d.I32(r, 76), "effectBasePoints2": d.I32(r, 77), "effectBasePoints3": d.I32(r, 78),
			"effectDieSides1": d.I32(r, 64), "effectDieSides2": d.I32(r, 65), "effectDieSides3": d.I32(r, 66),
			"effectAmplitude1": d.I32(r, 94), "effectAmplitude2": d.I32(r, 95), "effectAmplitude3": d.I32(r, 96),
			"effectChainTarget1": d.I32(r, 100), "effectChainTarget2": d.I32(r, 101), "effectChainTarget3": d.I32(r, 102),
			"effectRadiusIndex1": d.I32(r, 88), "effectRadiusIndex2": d.I32(r, 89), "effectRadiusIndex3": d.I32(r, 90),
			"durationIndex": d.U32(r, 30), "rangeIndex": d.U32(r, 36),
			"procChance": d.I32(r, 25), "procCharges": d.I32(r, 26),
			"maxTargetLevel": d.I32(r, 159), "maxAffectedTargets": d.I32(r, 163),
			"spellIconId": iconID, "iconName": name,
		})
	}
	return out, nil
}

// --- Faction.dbc (37 fields, 148-byte records) ---
const (
	facID     = 0
	facParent = 18
	facName   = 19 // enUS
	facDesc   = 28 // enUS
)

type factionOut struct {
	FactionID uint32 `json:"factionID"`
	NameLoc0  string `json:"name_loc0"`
	DescLoc0  string `json:"description1_loc0"`
	Side      int    `json:"side"`
	Team      uint32 `json:"team"`
}

var (
	allianceParents = map[uint32]bool{469: true, 891: true}
	hordeParents    = map[uint32]bool{67: true, 892: true}
	alliancePlayers = map[uint32]bool{1: true, 3: true, 4: true, 8: true}
	hordePlayers    = map[uint32]bool{2: true, 5: true, 6: true, 9: true}
	allianceSpecial = map[uint32]bool{469: true, 189: true, 61: true, 71: true, 49: true, 269: true, 589: true}
	hordeSpecial    = map[uint32]bool{67: true, 66: true}
)

func factionSide(id, parent uint32) int {
	switch {
	case alliancePlayers[id] || allianceParents[parent] || allianceSpecial[id]:
		return 1
	case hordePlayers[id] || hordeParents[parent] || hordeSpecial[id]:
		return 2
	default:
		return 0
	}
}

// factionTemplateOut maps a FactionTemplate id (creature.faction) to its
// Faction.dbc id.
type factionTemplateOut struct {
	TemplateID int `json:"id"`
	FactionID  int `json:"faction"`
}

// writeFactionTemplates emits FactionTemplate.dbc as {id, faction} pairs. In
// vanilla the layout is field 0 = template id, field 1 = Faction.dbc id.
func writeFactionTemplates(cf ClientFiles, outPath string) error {
	d, err := openDBCFrom(cf, "FactionTemplate.dbc")
	if err != nil {
		return err
	}
	out := make([]factionTemplateOut, 0, d.RecordCount)
	for rec := 0; rec < d.RecordCount; rec++ {
		out = append(out, factionTemplateOut{
			TemplateID: int(d.U32(rec, 0)),
			FactionID:  int(d.U32(rec, 1)),
		})
	}
	b, err := json.Marshal(out)
	if err != nil {
		return err
	}
	return os.WriteFile(outPath, b, 0644)
}

func writeFactions(cf ClientFiles, outPath string) error {
	d, err := openDBCFrom(cf, "Faction.dbc")
	if err != nil {
		return err
	}
	out := make([]factionOut, 0, d.RecordCount)
	for rec := 0; rec < d.RecordCount; rec++ {
		id := d.U32(rec, facID)
		parent := d.U32(rec, facParent)
		out = append(out, factionOut{
			FactionID: id, NameLoc0: d.Str(rec, facName), DescLoc0: d.Str(rec, facDesc),
			Side: factionSide(id, parent), Team: parent,
		})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Side != out[j].Side {
			return out[i].Side < out[j].Side
		}
		return out[i].NameLoc0 < out[j].NameLoc0
	})
	b, err := json.Marshal(out)
	if err != nil {
		return err
	}
	return os.WriteFile(outPath, b, 0644)
}
