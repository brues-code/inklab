package datatools

import (
	"database/sql"
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

// CacheResult reports the outcome of patching one WDB cache file.
type CacheResult struct {
	File     string `json:"file"`
	Table    string `json:"table"`
	Records  int    `json:"records"`
	Updated  int    `json:"updated"`
	Inserted int    `json:"inserted"`
	Error    string `json:"error,omitempty"`
}

// wdbFiles maps the standard cache filenames to nothing in particular; we key
// off the file's magic, but this is the set PatchAllCaches looks for.
var wdbFiles = []string{"itemcache.wdb", "questcache.wdb", "creaturecache.wdb", "gameobjectcache.wdb"}

var cacheTable = map[string]string{
	"BDIW": "item_template",       // itemcache
	"TSQW": "quest_template",      // questcache
	"BOMW": "creature_template",   // creaturecache
	"BOGW": "gameobject_template", // gameobjectcache
}

// PatchAllCaches applies every recognized WDB cache found in wdbDir to the
// SQLite database at dbPath. Missing files are skipped silently.
func PatchAllCaches(wdbDir, dbPath string) ([]CacheResult, error) {
	var out []CacheResult
	for _, name := range wdbFiles {
		p := filepath.Join(wdbDir, name)
		if _, err := os.Stat(p); err != nil {
			continue // not present
		}
		out = append(out, PatchCacheFile(p, dbPath))
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no WDB caches found in %s", wdbDir)
	}
	return out, nil
}

// PatchCacheFile decodes a single WDB cache and patches it into dbPath.
func PatchCacheFile(cachePath, dbPath string) CacheResult {
	res := CacheResult{File: filepath.Base(cachePath)}
	b, err := os.ReadFile(cachePath)
	if err != nil {
		res.Error = err.Error()
		return res
	}
	if len(b) < 20 {
		res.Error = "not a WDB file"
		return res
	}
	magic := string(b[0:4])
	table, ok := cacheTable[magic]
	if !ok {
		res.Error = fmt.Sprintf("unsupported cache magic %q", magic)
		return res
	}
	res.Table = table
	recs := records(b)
	res.Records = len(recs)

	var cols []string
	var rows [][]interface{}
	switch magic {
	case "BDIW":
		cols, rows = decodeItems(recs)
	case "TSQW":
		cols, rows = decodeQuests(recs)
	case "BOMW":
		cols, rows = decodeCreatures(recs)
	case "BOGW":
		cols, rows = decodeGameObjects(recs)
	}

	upd, ins, err := patch(dbPath, table, "entry", cols, rows)
	if err != nil {
		res.Error = err.Error()
		return res
	}
	res.Updated, res.Inserted = upd, ins
	return res
}

type record struct {
	entry uint32
	blk   []byte
}

// records walks the 20-byte header + entry/size/block framing.
func records(b []byte) []record {
	var out []record
	off := 20
	for off+8 <= len(b) {
		entry := binary.LittleEndian.Uint32(b[off : off+4])
		size := int(binary.LittleEndian.Uint32(b[off+4 : off+8]))
		off += 8
		if size == 0 || off+size > len(b) {
			break
		}
		out = append(out, record{entry, b[off : off+size]})
		off += size
	}
	return out
}

// --- itemcache: class(0) subclass(4) name1..4 then numeric fields. From the
// displayId field onward: displayId, quality, flags, buyPrice, sellPrice,
// invType, allowableClass, allowableRace, itemLevel, requiredLevel,
// required{Skill,SkillRank,Spell,HonorRank,CityRank,RepFaction,RepRank},
// maxCount, stackable, containerSlots (index 19), 10x stats, 5x damage, armor,
// 6x resistances, delay, ammoType, rangedMod, 5x spell blocks, bonding, then the
// description string and post-string fields. ---
func decodeItems(recs []record) ([]string, [][]interface{}) {
	cols := []string{"name", "class", "subclass", "display_id", "quality", "flags",
		"buy_price", "sell_price", "inventory_type", "allowable_class", "allowable_race",
		"item_level", "required_level"}
	for i := 1; i <= 10; i++ {
		cols = append(cols, fmt.Sprintf("stat_type%d", i), fmt.Sprintf("stat_value%d", i))
	}
	for i := 1; i <= 5; i++ {
		cols = append(cols, fmt.Sprintf("dmg_min%d", i), fmt.Sprintf("dmg_max%d", i), fmt.Sprintf("dmg_type%d", i))
	}
	cols = append(cols, "armor", "holy_res", "fire_res", "nature_res", "frost_res", "shadow_res", "arcane_res",
		"delay", "ammo_type", "range_mod")
	for i := 1; i <= 5; i++ {
		cols = append(cols, fmt.Sprintf("spellid_%d", i), fmt.Sprintf("spelltrigger_%d", i),
			fmt.Sprintf("spellcharges_%d", i), fmt.Sprintf("spellcooldown_%d", i),
			fmt.Sprintf("spellcategory_%d", i), fmt.Sprintf("spellcategorycooldown_%d", i))
	}
	cols = append(cols, "bonding", "description",
		"page_text", "page_language", "page_material", "start_quest", "lock_id",
		"material", "sheath", "random_property", "block", "set_id", "max_durability",
		"area_bound", "map_bound", "bag_family", "container_slots", "entry")

	var rows [][]interface{}
	for _, r := range recs {
		blk := r.blk
		if len(blk) < 12 {
			continue
		}
		class := u32(blk, 0)
		sub := u32(blk, 4)
		o := 8
		var name string
		name, o = cstr(blk, o)
		for i := 0; i < 3; i++ {
			_, o = cstr(blk, o)
		}
		fu := func(f int) interface{} { return u32(blk, o+f*4) }
		fi := func(f int) interface{} { return int32(u32(blk, o+f*4)) }
		ff := func(f int) interface{} { return f32(blk, o+f*4) }

		row := []interface{}{name, class, sub, fu(0), fu(1), fu(2), fu(3), fu(4), fu(5), fi(6), fi(7), fu(8), fu(9)}
		for i := 0; i < 10; i++ {
			row = append(row, fu(20+i*2), fi(21+i*2))
		}
		for i := 0; i < 5; i++ {
			row = append(row, ff(40+i*3), ff(41+i*3), fu(42+i*3))
		}
		row = append(row, fu(55), fu(56), fu(57), fu(58), fu(59), fu(60), fu(61), fu(62), fu(63), ff(64))
		for i := 0; i < 5; i++ {
			b := 65 + i*6
			row = append(row, fu(b), fu(b+1), fi(b+2), fi(b+3), fu(b+4), fi(b+5))
		}
		desc, p := cstr(blk, o+96*4)
		pp := func(k int) interface{} { return u32(blk, p+k*4) }
		ppi := func(k int) interface{} { return int32(u32(blk, p+k*4)) }
		row = append(row, fu(95), desc,
			pp(0), pp(1), pp(2), pp(3), pp(4), ppi(5), pp(6), ppi(7), pp(8), pp(9), pp(10),
			pp(11), pp(12), pp(13), fu(19), r.entry)
		rows = append(rows, row)
	}
	return cols, rows
}

// --- questcache: numeric header [0..38], then Title/Objectives/Details/EndText,
// then 4x (ReqCreatureOrGOId, Count, ReqItemId, Count), then 4x ObjectiveText ---
func decodeQuests(recs []record) ([]string, [][]interface{}) {
	cols := []string{
		"Method", "QuestLevel", "ZoneOrSort", "Type", "SuggestedPlayers",
		"RepObjectiveFaction", "RepObjectiveValue", "NextQuestInChain",
		"RewOrReqMoney", "RewMoneyMaxLevel", "RewSpell", "RewSpellCast", "QuestFlags",
		"RewItemId1", "RewItemCount1", "RewItemId2", "RewItemCount2", "RewItemId3", "RewItemCount3", "RewItemId4", "RewItemCount4",
		"RewChoiceItemId1", "RewChoiceItemCount1", "RewChoiceItemId2", "RewChoiceItemCount2", "RewChoiceItemId3", "RewChoiceItemCount3",
		"RewChoiceItemId4", "RewChoiceItemCount4", "RewChoiceItemId5", "RewChoiceItemCount5", "RewChoiceItemId6", "RewChoiceItemCount6",
		"PointMapId", "PointX", "PointY", "PointOpt",
		"Title", "Objectives", "Details", "EndText",
		"ReqCreatureOrGOId1", "ReqCreatureOrGOCount1", "ReqItemId1", "ReqItemCount1",
		"ReqCreatureOrGOId2", "ReqCreatureOrGOCount2", "ReqItemId2", "ReqItemCount2",
		"ReqCreatureOrGOId3", "ReqCreatureOrGOCount3", "ReqItemId3", "ReqItemCount3",
		"ReqCreatureOrGOId4", "ReqCreatureOrGOCount4", "ReqItemId4", "ReqItemCount4",
		"ObjectiveText1", "ObjectiveText2", "ObjectiveText3", "ObjectiveText4",
		"entry",
	}
	var rows [][]interface{}
	for _, r := range recs {
		blk := r.blk
		if len(blk) < 39*4 {
			continue
		}
		row := []interface{}{
			u32(blk, 1*4), int32(u32(blk, 2*4)), int32(u32(blk, 3*4)), u32(blk, 4*4), u32(blk, 5*4),
			u32(blk, 6*4), u32(blk, 7*4), u32(blk, 9*4),
			int32(u32(blk, 10*4)), u32(blk, 11*4), u32(blk, 12*4), u32(blk, 13*4), u32(blk, 14*4),
		}
		for i := 0; i < 4; i++ {
			row = append(row, u32(blk, (15+i*2)*4), u32(blk, (16+i*2)*4))
		}
		for i := 0; i < 6; i++ {
			row = append(row, u32(blk, (23+i*2)*4), u32(blk, (24+i*2)*4))
		}
		row = append(row, u32(blk, 35*4), f32(blk, 36*4), f32(blk, 37*4), u32(blk, 38*4))

		o := 39 * 4
		var title, obj, det, end string
		title, o = cstr(blk, o)
		obj, o = cstr(blk, o)
		det, o = cstr(blk, o)
		end, o = cstr(blk, o)
		row = append(row, title, obj, det, end)

		objReq := make([]interface{}, 0, 16)
		for i := 0; i < 4; i++ {
			objReq = append(objReq, u32(blk, o), u32(blk, o+4), u32(blk, o+8), u32(blk, o+12))
			o += 16
		}
		row = append(row, objReq...)
		for i := 0; i < 4; i++ {
			var s string
			s, o = cstr(blk, o)
			row = append(row, s)
		}
		row = append(row, r.entry)
		rows = append(rows, row)
	}
	return cols, rows
}

// --- creaturecache: name[0..3], subName, then 7x u32
// (TypeFlags, Type, Family, Rank, unk0, PetSpellDataId, DisplayId) + civilian u16. ---
func decodeCreatures(recs []record) ([]string, [][]interface{}) {
	cols := []string{"name", "subname", "type_flags", "type", "beast_family",
		"rank", "pet_spell_list_id", "display_id1", "civilian", "entry"}
	var rows [][]interface{}
	for _, r := range recs {
		blk := r.blk
		o := 0
		var name, sub string
		name, o = cstr(blk, o)
		for i := 0; i < 3; i++ {
			_, o = cstr(blk, o)
		}
		sub, o = cstr(blk, o)
		if o+30 > len(blk) {
			continue
		}
		rows = append(rows, []interface{}{
			name, sub,
			u32(blk, o), u32(blk, o+4), u32(blk, o+8), u32(blk, o+12),
			u32(blk, o+20), u32(blk, o+24), u16(blk, o+28), r.entry,
		})
	}
	return cols, rows
}

// --- gameobjectcache: type u32, displayId u32, name[0..3], castBarCaption,
// then 24x u32 data[]. ---
func decodeGameObjects(recs []record) ([]string, [][]interface{}) {
	cols := []string{"name", "type", "displayId"}
	for i := 0; i < 24; i++ {
		cols = append(cols, fmt.Sprintf("data%d", i))
	}
	cols = append(cols, "entry")
	var rows [][]interface{}
	for _, r := range recs {
		blk := r.blk
		if len(blk) < 8 {
			continue
		}
		goType := u32(blk, 0)
		display := u32(blk, 4)
		o := 8
		var name string
		name, o = cstr(blk, o)
		for i := 0; i < 3; i++ {
			_, o = cstr(blk, o)
		}
		_, o = cstr(blk, o)
		if o+24*4 > len(blk) {
			continue
		}
		row := []interface{}{name, goType, display}
		for i := 0; i < 24; i++ {
			row = append(row, u32(blk, o+i*4))
		}
		row = append(row, r.entry)
		rows = append(rows, row)
	}
	return cols, rows
}

// patch UPDATEs each row by keyCol, INSERTing when no row matched.
func patch(dbPath, table, keyCol string, cols []string, rows [][]interface{}) (upd, ins int, err error) {
	db, err := sql.Open("sqlite", dbPath+"?_busy_timeout=5000")
	if err != nil {
		return 0, 0, err
	}
	defer db.Close()
	tx, err := db.Begin()
	if err != nil {
		return 0, 0, err
	}
	defer tx.Rollback()

	setList, insCols, insPlace := "", "", ""
	for i, c := range cols {
		insCols += c
		insPlace += "?"
		if i < len(cols)-1 {
			insCols += ","
			insPlace += ","
			setList += c + "=?"
			if i < len(cols)-2 {
				setList += ", "
			}
		}
	}
	updStmt, err := tx.Prepare(fmt.Sprintf("UPDATE %s SET %s WHERE %s=?", table, setList, keyCol))
	if err != nil {
		return 0, 0, err
	}
	defer updStmt.Close()
	insStmt, err := tx.Prepare(fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", table, insCols, insPlace))
	if err != nil {
		return 0, 0, err
	}
	defer insStmt.Close()

	for _, row := range rows {
		res, err := updStmt.Exec(row...)
		if err != nil {
			continue
		}
		if n, _ := res.RowsAffected(); n > 0 {
			upd++
		} else if _, err := insStmt.Exec(row...); err == nil {
			ins++
		}
	}
	return upd, ins, tx.Commit()
}

func u32(b []byte, o int) uint32 {
	if o+4 > len(b) {
		return 0
	}
	return binary.LittleEndian.Uint32(b[o : o+4])
}
func u16(b []byte, o int) uint32 {
	if o+2 > len(b) {
		return 0
	}
	return uint32(binary.LittleEndian.Uint16(b[o : o+2]))
}
func f32(b []byte, o int) float64 { return float64(math.Float32frombits(u32(b, o))) }
func cstr(b []byte, o int) (string, int) {
	if o >= len(b) {
		return "", o
	}
	e := o
	for e < len(b) && b[e] != 0 {
		e++
	}
	return string(b[o:e]), e + 1
}
