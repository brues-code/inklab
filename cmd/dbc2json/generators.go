package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// runGen dispatches to the per-file generator and writes the JSON output.
func runGen(name, dir, out string) error {
	var v interface{}
	var err error
	switch name {
	case "itemsets":
		v, err = genItemSets(dir)
	case "skills":
		v, err = genSkills(dir)
	case "sla":
		v, err = genSLA(dir)
	case "zones":
		v, err = genZones(dir)
	case "questsorts":
		v, err = genQuestSorts(dir)
	case "icons":
		v, err = genIcons(dir)
	case "spells":
		v, err = genSpells(dir)
	default:
		return fmt.Errorf("unknown gen name %q", name)
	}
	if err != nil {
		return err
	}
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	if err := os.WriteFile(out, b, 0644); err != nil {
		return err
	}
	fmt.Printf("wrote %s\n", out)
	return nil
}

// iconBase strips the "Interface\\Icons\\" path, leaving just the icon name.
func iconBase(p string) string {
	if i := strings.LastIndexAny(p, `\/`); i >= 0 {
		p = p[i+1:]
	}
	return p
}

// ItemSet.dbc (1.12, 45 fields): id(0), name[8](1-8), nameFlags(9),
// itemID[17](10-26), setSpellID[8](27-34), setThreshold[8](35-42),
// requiredSkill(43), requiredSkillRank(44).
func genItemSets(dir string) (interface{}, error) {
	d, err := Open(filepath.Join(dir, "ItemSet.dbc"))
	if err != nil {
		return nil, err
	}
	out := make([]map[string]interface{}, 0, d.RecordCount)
	for r := 0; r < d.RecordCount; r++ {
		m := map[string]interface{}{
			"itemsetID":  d.Uint32(r, 0),
			"name_loc0":  d.Str(r, 1),
			"skillID":    d.Uint32(r, 43),
			"skilllevel": d.Uint32(r, 44),
		}
		for i := 0; i < 10; i++ {
			m[fmt.Sprintf("item%d", i+1)] = d.Uint32(r, 10+i)
		}
		for i := 0; i < 8; i++ {
			m[fmt.Sprintf("spell%d", i+1)] = d.Uint32(r, 27+i)
		}
		for i := 0; i < 8; i++ {
			m[fmt.Sprintf("bonus%d", i+1)] = d.Uint32(r, 35+i)
		}
		out = append(out, m)
	}
	return out, nil
}

// SkillLine.dbc (1.12, 22 fields): id(0), categoryID(1), skillCostsID(2),
// displayName[8](3-10), nameFlags(11), ...
func genSkills(dir string) (interface{}, error) {
	d, err := Open(filepath.Join(dir, "SkillLine.dbc"))
	if err != nil {
		return nil, err
	}
	out := make([]map[string]interface{}, 0, d.RecordCount)
	for r := 0; r < d.RecordCount; r++ {
		out = append(out, map[string]interface{}{
			"skillID":    d.Uint32(r, 0),
			"categoryID": d.Int32(r, 1),
			"name_loc0":  d.Str(r, 3),
		})
	}
	return out, nil
}

// SkillLineAbility.dbc (1.12, 15 fields): id(0), skillLine(1), spell(2),
// raceMask(3), classMask(4), excludeRace(5), excludeClass(6),
// minSkillLineRank(7), supercededBySpell(8), acquireMethod(9),
// trivialSkillLineRankHigh(10), trivialSkillLineRankLow(11), ...
func genSLA(dir string) (interface{}, error) {
	d, err := Open(filepath.Join(dir, "SkillLineAbility.dbc"))
	if err != nil {
		return nil, err
	}
	out := make([]map[string]interface{}, 0, d.RecordCount)
	for r := 0; r < d.RecordCount; r++ {
		out = append(out, map[string]interface{}{
			"skillID":         d.Uint32(r, 1),
			"spellID":         d.Uint32(r, 2),
			"racemask":        d.Uint32(r, 3),
			"classmask":       d.Uint32(r, 4),
			"req_skill_value": d.Uint32(r, 7),
			"max_value":       d.Uint32(r, 10),
			"min_value":       d.Uint32(r, 11),
		})
	}
	return out, nil
}

// WorldMapArea.dbc (1.12, 8 fields): id(0), mapID(1), areaID(2),
// areaName(3), locLeft(4), locRight(5), locTop(6), locBottom(7).
// NOTE: x/y bound field order is verified against the committed JSON.
// instanceType is resolved from Map.dbc field 2 (0 continent, 1 dungeon,
// 2 raid, 3 battleground) so quests can be grouped beyond just continents.
func genZones(dir string) (interface{}, error) {
	maps, err := Open(filepath.Join(dir, "Map.dbc"))
	if err != nil {
		return nil, err
	}
	instByMap := make(map[uint32]uint32, maps.RecordCount)
	for r := 0; r < maps.RecordCount; r++ {
		instByMap[maps.Uint32(r, 0)] = maps.Uint32(r, 2)
	}

	d, err := Open(filepath.Join(dir, "WorldMapArea.dbc"))
	if err != nil {
		return nil, err
	}
	out := make([]map[string]interface{}, 0, d.RecordCount)
	for r := 0; r < d.RecordCount; r++ {
		mapID := d.Uint32(r, 1)
		out = append(out, map[string]interface{}{
			"mapID":        mapID,
			"instanceType": instByMap[mapID],
			"areatableID":  d.Uint32(r, 2),
			"name_loc0":    d.Str(r, 3),
			// WoW's coordinate system: x is the top/bottom axis, y is the
			// right/left axis; min/max line up with the reversed loc fields.
			"x_min": d.Float32(r, 7), // locBottom
			"x_max": d.Float32(r, 6), // locTop
			"y_min": d.Float32(r, 5), // locRight
			"y_max": d.Float32(r, 4), // locLeft
		})
	}
	return out, nil
}

// QuestSort.dbc (1.12, 10 fields): id(0), name[8](1-8), nameFlags(9).
// Quests store the NEGATIVE of these ids in ZoneOrSort — these cover the
// class quest sorts (Warlock, Mage, ...), professions and seasonal/special
// categories that have no AreaTable entry.
func genQuestSorts(dir string) (interface{}, error) {
	d, err := Open(filepath.Join(dir, "QuestSort.dbc"))
	if err != nil {
		return nil, err
	}
	out := make([]map[string]interface{}, 0, d.RecordCount)
	for r := 0; r < d.RecordCount; r++ {
		out = append(out, map[string]interface{}{
			"sortID":    d.Uint32(r, 0),
			"name_loc0": d.Str(r, 1),
		})
	}
	return out, nil
}

// ItemDisplayInfo.dbc (1.12, 23 fields): id(0), ..., inventoryIcon1(5).
// item_icons.json is a map of displayID -> icon name (non-empty only).
func genIcons(dir string) (interface{}, error) {
	d, err := Open(filepath.Join(dir, "ItemDisplayInfo.dbc"))
	if err != nil {
		return nil, err
	}
	out := make(map[string]string, d.RecordCount)
	for r := 0; r < d.RecordCount; r++ {
		icon := d.Str(r, 5)
		if icon == "" {
			continue
		}
		out[fmt.Sprintf("%d", d.Uint32(r, 0))] = icon
	}
	return out, nil
}

// Spell.dbc (1.12, 173 fields). Relevant offsets: id(0), durationIndex(30),
// effectDieSides[3](64-66), effectBasePoints[3](76-78), spellIconID(117),
// name[8](120-127), description[8](138-145). iconName resolves via
// SpellIcon.dbc: id(0) -> texturePath(1).
func genSpells(dir string) (interface{}, error) {
	icons, err := Open(filepath.Join(dir, "SpellIcon.dbc"))
	if err != nil {
		return nil, err
	}
	iconByID := make(map[uint32]string, icons.RecordCount)
	for r := 0; r < icons.RecordCount; r++ {
		iconByID[icons.Uint32(r, 0)] = iconBase(icons.Str(r, 1))
	}

	d, err := Open(filepath.Join(dir, "Spell.dbc"))
	if err != nil {
		return nil, err
	}
	out := make([]map[string]interface{}, 0, d.RecordCount)
	for r := 0; r < d.RecordCount; r++ {
		iconID := d.Uint32(r, 117)
		name := iconByID[iconID]
		if name == "" {
			name = "temp"
		}
		out = append(out, map[string]interface{}{
			"entry":             d.Uint32(r, 0),
			"name":              d.Str(r, 120),
			"description":       d.Str(r, 138),
			"effectBasePoints1": d.Int32(r, 76),
			"effectBasePoints2": d.Int32(r, 77),
			"effectBasePoints3": d.Int32(r, 78),
			"effectDieSides1":   d.Int32(r, 64),
			"effectDieSides2":   d.Int32(r, 65),
			"effectDieSides3":   d.Int32(r, 66),
			"durationIndex":     d.Uint32(r, 30),
			"spellIconId":       iconID,
			"iconName":          name,
		})
	}
	return out, nil
}
