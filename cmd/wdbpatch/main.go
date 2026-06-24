// Command wdbpatch reads WoW 1.12 WDB caches (the client's cache of server
// query responses) and patches the freshest values into inklab.db. The cache
// only holds entries the client has actually seen, so this is an incremental
// overlay: re-run it over time as the cache grows to patch in newer data and
// server-new entries the (frozen) tw_world dump never had.
//
// Supports itemcache.wdb (-> item_template), questcache.wdb (->
// quest_template), creaturecache.wdb (-> creature_template) and
// gameobjectcache.wdb (-> gameobject_template). Existing rows are UPDATEd;
// missing ones are INSERTed.
//
// Usage:
//
//	go run ./cmd/wdbpatch <cache.wdb>             # dry run: parse + sample
//	go run ./cmd/wdbpatch <cache.wdb> <db.sqlite> # apply
package main

import (
	"database/sql"
	"encoding/binary"
	"fmt"
	"math"
	"os"

	_ "modernc.org/sqlite"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("usage: wdbpatch <cache.wdb> [db.sqlite]")
		os.Exit(2)
	}
	path := os.Args[1]
	b, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	if len(b) < 20 {
		fmt.Fprintln(os.Stderr, "not a WDB file")
		os.Exit(1)
	}
	magic := string(b[0:4])
	recs := records(b)
	fmt.Printf("magic=%s records=%d\n", magic, len(recs))

	var cols []string
	var rows [][]interface{}
	switch magic {
	case "BDIW": // itemcache -> item_template
		cols, rows = decodeItems(recs)
	case "TSQW": // questcache -> quest_template
		cols, rows = decodeQuests(recs)
	case "BOMW": // creaturecache -> creature_template
		cols, rows = decodeCreatures(recs)
	case "BOGW": // gameobjectcache -> gameobject_template
		cols, rows = decodeGameObjects(recs)
	default:
		fmt.Fprintf(os.Stderr, "unsupported cache magic %q (have BDIW item / TSQW quest / BOMW creature / BOGW gameobject)\n", magic)
		os.Exit(1)
	}

	// sample / verification: print named fields for a few known entries
	idx := map[string]int{}
	for i, c := range cols {
		idx[c] = i
	}
	keyPos := len(cols) - 1
	want := map[uint32]bool{1015: true, 40959: true, 10041: true, 19019: true}
	shown := 0
	for _, row := range rows {
		e, _ := row[keyPos].(uint32)
		if !want[e] && shown >= 3 {
			continue
		}
		get := func(c string) interface{} {
			if p, ok := idx[c]; ok {
				return row[p]
			}
			return "-"
		}
		switch magic {
		case "TSQW":
			fmt.Printf("  quest %d: level=%v zone=%v title=%q reqItem1=%v objText1=%q\n",
				e, get("QuestLevel"), get("ZoneOrSort"), get("Title"), get("ReqItemId1"), get("ObjectiveText1"))
		case "BOMW":
			fmt.Printf("  creature %d: %q sub=%q type=%v family=%v rank=%v display=%v\n",
				e, get("name"), get("subname"), get("type"), get("beast_family"), get("rank"), get("display_id1"))
		case "BOGW":
			fmt.Printf("  gameobject %d: %q type=%v display=%v data0=%v\n",
				e, get("name"), get("type"), get("displayId"), get("data0"))
		default:
			fmt.Printf("  item %d: %q class=%v sub=%v q=%v\n", e, get("name"), get("class"), get("subclass"), get("quality"))
		}
		shown++
		if shown >= 8 {
			break
		}
	}
	if len(os.Args) < 3 {
		fmt.Println("(dry run — pass a db path to apply)")
		return
	}
	table := map[string]string{
		"BDIW": "item_template",
		"TSQW": "quest_template",
		"BOMW": "creature_template",
		"BOGW": "gameobject_template",
	}[magic]
	upd, ins, err := patch(os.Args[2], table, "entry", cols, rows)
	if err != nil {
		fmt.Fprintln(os.Stderr, "patch error:", err)
		os.Exit(1)
	}
	fmt.Printf("updated %d, inserted %d in %s\n", upd, ins, table)
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
// displayId field onward they are: displayId, quality, flags, buyPrice,
// sellPrice, invType, allowableClass, allowableRace, itemLevel, requiredLevel,
// requiredSkill, requiredSkillRank, requiredSpell, requiredHonorRank,
// requiredCityRank, requiredRepFaction, requiredRepRank, maxCount, stackable,
// containerSlots (index 19, o+76). ---
func decodeItems(recs []record) ([]string, [][]interface{}) {
	// Numeric field layout (index from the displayId field, after the 4 names):
	//  0 displayId 1 quality 2 flags 3 buyPrice 4 sellPrice 5 invType
	//  6 allowableClass 7 allowableRace 8 itemLevel 9 requiredLevel
	//  10..16 required{Skill,SkillRank,Spell,HonorRank,CityRank,RepFaction,RepRank}
	//  17 maxCount 18 stackable 19 containerSlots
	//  20..39 stats (10x type,value)  40..54 damage (5x min,max,type as float,float,u32)
	//  55 armor 56..61 resistances 62 delay 63 ammoType 64 rangedModRange(float)
	//  65..94 spells (5x id,trigger,charges,cooldown,category,categoryCooldown)
	//  95 bonding, then the description string and post-string fields we skip.
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
	cols = append(cols, "bonding", "container_slots", "entry")

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
		// field helpers (index relative to the first numeric field at o)
		fu := func(f int) interface{} { return u32(blk, o+f*4) }
		fi := func(f int) interface{} { return int32(u32(blk, o+f*4)) }
		ff := func(f int) interface{} { return f32(blk, o+f*4) }

		row := []interface{}{name, class, sub, fu(0), fu(1), fu(2), fu(3), fu(4), fu(5), fi(6), fi(7), fu(8), fu(9)}
		for i := 0; i < 10; i++ { // stats
			row = append(row, fu(20+i*2), fi(21+i*2))
		}
		for i := 0; i < 5; i++ { // damage
			row = append(row, ff(40+i*3), ff(41+i*3), fu(42+i*3))
		}
		row = append(row, fu(55), fu(56), fu(57), fu(58), fu(59), fu(60), fu(61), fu(62), fu(63), ff(64))
		for i := 0; i < 5; i++ { // spells (ppmRate is not in the query, left untouched)
			b := 65 + i*6
			row = append(row, fu(b), fu(b+1), fi(b+2), fi(b+3), fu(b+4), fi(b+5))
		}
		row = append(row, fu(95), fu(19), r.entry)
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
		for i := 0; i < 4; i++ { // reward items id/count
			row = append(row, u32(blk, (15+i*2)*4), u32(blk, (16+i*2)*4))
		}
		for i := 0; i < 6; i++ { // reward choice id/count
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

		// 4 objective groups: ReqCreatureOrGOId, Count, ReqItemId, Count
		objReq := make([]interface{}, 0, 16)
		for i := 0; i < 4; i++ {
			objReq = append(objReq, u32(blk, o), u32(blk, o+4), u32(blk, o+8), u32(blk, o+12))
			o += 16
		}
		row = append(row, objReq...)
		// 4 objective texts
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
// (TypeFlags, Type, Family, Rank, unk0, PetSpellDataId, DisplayId) + civilian u16.
// Only one display id is delivered (display_id1). ---
func decodeCreatures(recs []record) ([]string, [][]interface{}) {
	cols := []string{"name", "subname", "type_flags", "type", "beast_family",
		"rank", "pet_spell_list_id", "display_id1", "civilian", "entry"}
	var rows [][]interface{}
	for _, r := range recs {
		blk := r.blk
		o := 0
		var name, sub string
		name, o = cstr(blk, o)
		for i := 0; i < 3; i++ { // name[1..3] (server fills all four the same)
			_, o = cstr(blk, o)
		}
		sub, o = cstr(blk, o)
		if o+30 > len(blk) {
			continue
		}
		rows = append(rows, []interface{}{
			name, sub,
			u32(blk, o),       // TypeFlags
			u32(blk, o+4),     // Type
			u32(blk, o+8),     // Family
			u32(blk, o+12),    // Rank
			u32(blk, o+20),    // PetSpellDataId (o+16 is unk0)
			u32(blk, o+24),    // DisplayId
			u16(blk, o+28),    // civilian
			r.entry,
		})
	}
	return cols, rows
}

// --- gameobjectcache: type u32, displayId u32, name[0..3], castBarCaption,
// then 24x u32 data[]. This build carries no trailing size float. ---
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
		for i := 0; i < 3; i++ { // name[1..3]
			_, o = cstr(blk, o)
		}
		_, o = cstr(blk, o) // castBarCaption
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

	// cols ends with keyCol; build UPDATE set list from all but the last.
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
		res, err := updStmt.Exec(row...) // row already ends with key value
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
