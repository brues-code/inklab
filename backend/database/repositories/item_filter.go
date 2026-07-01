package repositories

import (
	"fmt"
	"strconv"
	"strings"

	"inklab/backend/database/models"
)

// item_filter.go is the item search filter service: it translates a
// models.SearchFilter into a parameterized SQL WHERE clause (+ args) and the
// ORDER BY clause. Keeping every filter here — instead of inline in the
// repository — keeps AdvancedSearch thin and makes adding filters a one-liner.
//
// All filters AND together. List filters (quality/class/...) are IN(...). The
// table is aliased `t` in the queries, so source EXISTS subqueries reference
// t.entry. condBuilder keeps conditions and their args in lockstep so the ?
// placeholders line up with the flat args slice.

type condBuilder struct {
	conds []string
	args  []any
}

// raw appends one condition and its args (kept in call order so they align
// with the joined WHERE).
func (b *condBuilder) raw(cond string, args ...any) {
	b.conds = append(b.conds, cond)
	b.args = append(b.args, args...)
}

// in adds `col IN (?, ...)` for a non-empty int list.
func (b *condBuilder) in(col string, vals []int) {
	if len(vals) == 0 {
		return
	}
	ph := make([]string, len(vals))
	for i, v := range vals {
		ph[i] = "?"
		b.args = append(b.args, v)
	}
	b.conds = append(b.conds, fmt.Sprintf("%s IN (%s)", col, strings.Join(ph, ",")))
}

// gteI/lteI add a `col >=/<= v` bound when v > 0 (0 = unset).
func (b *condBuilder) gteI(col string, v int) {
	if v > 0 {
		b.raw(col+" >= ?", v)
	}
}
func (b *condBuilder) lteI(col string, v int) {
	if v > 0 {
		b.raw(col+" <= ?", v)
	}
}
func (b *condBuilder) gteF(cond string, v float64) {
	if v > 0 {
		b.raw(cond+" >= ?", v)
	}
}
func (b *condBuilder) lteF(cond string, v float64) {
	if v > 0 {
		b.raw(cond+" <= ?", v)
	}
}

func (b *condBuilder) where() string {
	if len(b.conds) == 0 {
		return ""
	}
	return "WHERE " + strings.Join(b.conds, " AND ")
}

// dpsExpr is the per-item weapon DPS, guarded against delay 0 (non-weapons).
const dpsExpr = "(CASE WHEN delay > 0 THEN (dmg_min1 + dmg_max1) / 2.0 / (delay / 1000.0) ELSE 0 END)"

// buildItemFilter turns a SearchFilter into a WHERE clause + args. Needs the
// repository for the proficiency lookup (forbiddenProficiencySubclasses).
func (r *ItemRepository) buildItemFilter(f models.SearchFilter) (string, []any) {
	b := &condBuilder{}

	// Name or numeric ID.
	if f.Query != "" {
		if id, err := strconv.Atoi(f.Query); err == nil {
			b.raw("entry = ?", id)
		} else {
			b.raw("name LIKE ?", "%"+f.Query+"%")
		}
	}

	// Category + level ranges.
	b.in("quality", f.Quality)
	b.in("class", f.Class)
	b.in("subclass", weaponSubclassesWith2H(f.Class, f.SubClass))
	b.in("inventory_type", f.InventoryType)
	b.gteI("item_level", f.MinLevel)
	b.lteI("item_level", f.MaxLevel)
	b.gteI("required_level", f.MinReqLevel)
	b.lteI("required_level", f.MaxReqLevel)

	// Stat minimums, summed across the 10 stat slots (a stat rarely repeats, but
	// summing is correct if it does).
	for _, sf := range f.Stats {
		if sf.Stat <= 0 {
			continue
		}
		parts := make([]string, 10)
		sargs := make([]any, 0, 11)
		for i := 1; i <= 10; i++ {
			parts[i-1] = fmt.Sprintf("CASE WHEN stat_type%d = ? THEN stat_value%d ELSE 0 END", i, i)
			sargs = append(sargs, sf.Stat)
		}
		sargs = append(sargs, sf.Min)
		b.raw("("+strings.Join(parts, " + ")+") >= ?", sargs...)
	}

	// Usable-by-class: explicit allowable_class bitmask AND weapon/armor
	// proficiency (a mage can't wield a 2H sword even with no class flag).
	if f.UsableByClass > 0 {
		classBit := 1 << (uint(f.UsableByClass) - 1)
		b.raw("(allowable_class = -1 OR (allowable_class & ?) != 0)", classBit)

		// Quest-gated class restriction: items obtainable only via a class-
		// restricted quest (e.g. the class Atiesh variants) aren't class-flagged
		// themselves. If an item is a quest reward, require a rewarding quest that
		// allows this class (RequiredClasses = 0 means any class). Items with no
		// rewarding quest are unaffected.
		const rewardPred = `t.entry IN (q.RewItemId1, q.RewItemId2, q.RewItemId3, q.RewItemId4,
			q.RewChoiceItemId1, q.RewChoiceItemId2, q.RewChoiceItemId3, q.RewChoiceItemId4,
			q.RewChoiceItemId5, q.RewChoiceItemId6)`
		b.raw(`(NOT EXISTS (SELECT 1 FROM quest_template q WHERE `+rewardPred+`)
			OR EXISTS (SELECT 1 FROM quest_template q WHERE `+rewardPred+`
				AND (q.RequiredClasses = 0 OR (q.RequiredClasses & ?) != 0)))`, classBit)

		fw, fa := r.forbiddenProficiencySubclasses(f.UsableByClass)
		for _, x := range []struct {
			cls  int
			subs []int
		}{{2, fw}, {4, fa}} {
			if len(x.subs) == 0 {
				continue
			}
			ph := make([]string, len(x.subs))
			a := make([]any, len(x.subs))
			for i, s := range x.subs {
				ph[i] = "?"
				a[i] = s
			}
			b.raw(fmt.Sprintf("NOT (class = %d AND subclass IN (%s))", x.cls, strings.Join(ph, ",")), a...)
		}
	}

	// Source: obtainable via ANY selected source (OR'd group), AND'd with the rest.
	if cond, sargs := buildSourceCondition(f.Sources); cond != "" {
		b.raw(cond, sargs...)
	}

	// Item properties.
	b.in("bonding", f.Bonding)
	if f.OnlyUnique {
		b.raw("max_count = 1")
	}
	if f.ClassSpecific {
		b.raw("(allowable_class != -1 AND allowable_class > 0)")
	}
	if f.RaceSpecific {
		b.raw("(allowable_race != -1 AND allowable_race > 0)")
	}
	if f.StartsQuest {
		b.raw("start_quest > 0")
	}
	if f.HasEffect {
		b.raw("spellid_1 > 0")
	}

	// Requirements & economy.
	if f.RequiresProf {
		b.raw("required_skill > 0")
	}
	b.gteI("required_skill_rank", f.MinSkillRank)
	b.lteI("required_skill_rank", f.MaxSkillRank)
	if f.RequiresRep {
		b.raw("required_reputation_faction > 0")
	}
	if f.RequiredRepFaction > 0 {
		b.raw("required_reputation_faction = ?", f.RequiredRepFaction)
	}
	b.gteI("buy_price", f.MinBuyPrice)
	b.lteI("buy_price", f.MaxBuyPrice)
	b.gteI("sell_price", f.MinSellPrice)
	b.lteI("sell_price", f.MaxSellPrice)
	b.gteI("max_durability", f.MinDurability)
	b.lteI("max_durability", f.MaxDurability)

	// Weapon & armor stats. DPS/speed are guarded against non-weapons (delay 0).
	b.gteF(dpsExpr, f.MinDps)
	if f.MaxDps > 0 {
		b.raw("delay > 0 AND "+dpsExpr+" <= ?", f.MaxDps)
	}
	if f.MinSpeed > 0 {
		b.raw("delay > 0 AND (delay / 1000.0) >= ?", f.MinSpeed)
	}
	if f.MaxSpeed > 0 {
		b.raw("delay > 0 AND (delay / 1000.0) <= ?", f.MaxSpeed)
	}
	if f.DamageSchool > 0 { // 0 = any (Physical is the common default, not filterable)
		b.raw("dmg_type1 = ?", f.DamageSchool)
	}
	b.gteI("armor", f.MinArmor)
	b.lteI("armor", f.MaxArmor)
	b.gteI("block", f.MinBlock)
	b.lteI("block", f.MaxBlock)
	for _, rf := range f.Resists {
		if col := resistColumn(rf.School); col != "" && rf.Min > 0 {
			b.raw("COALESCE("+col+", 0) >= ?", rf.Min)
		}
	}

	return b.where(), b.args
}

// weaponSubclassesWith2H expands a weapon-family subclass filter to also match
// its Two-Handed variant. GetItemClasses merges 2H weapon subclasses into their
// base (Axe 1→0, Mace 5→4, Sword 8→7) for the filter sidebar, so a "Sword"
// selection sends base subclass 7; without this, `subclass IN (7)` would miss
// 2H swords (subclass 8). Only expands when weapons (class 2) are selected, so
// it can't bleed into the armor subclasses that reuse ids 1/5/8.
func weaponSubclassesWith2H(classes, subs []int) []int {
	if len(subs) == 0 {
		return subs
	}
	weapon := false
	for _, c := range classes {
		if c == 2 {
			weapon = true
		}
	}
	if !weapon {
		return subs
	}
	base2h := map[int]int{0: 1, 4: 5, 7: 8}
	out := append([]int{}, subs...)
	for _, s := range subs {
		if h, ok := base2h[s]; ok {
			out = append(out, h)
		}
	}
	return out
}

// resistColumn maps a resistance school id (matching spell_schools) to its
// item_template column.
func resistColumn(school int) string {
	switch school {
	case 1:
		return "holy_res"
	case 2:
		return "fire_res"
	case 3:
		return "nature_res"
	case 4:
		return "frost_res"
	case 5:
		return "shadow_res"
	case 6:
		return "arcane_res"
	}
	return ""
}

// forbiddenProficiencySubclasses returns the weapon (item class 2) and armor
// (item class 4) subclasses the given player class CANNOT use, derived purely
// from the client DBC: proficiency-granting spells (SPELL_EFFECT_PROFICIENCY =
// 60) carry the item class + a subclass bitmask, and SkillLineAbility.classmask
// (imported into spell_skill_spells) says which classes may learn each. This
// mirrors the server's IsSpellFitByClassAndRace gate that drives a weapon
// trainer's per-class list.
//
// A subclass is forbidden iff some proficiency grants it AND none of the
// proficiencies granting it are learnable by this class. A subclass no
// proficiency mentions (e.g. Misc armor: rings/cloaks/trinkets) is never
// forbidden. classmask 0 (or no SkillLineAbility row) means "all classes".
func (r *ItemRepository) forbiddenProficiencySubclasses(class int) (weapon, armor []int) {
	if class <= 0 {
		return nil, nil
	}
	classBit := 1 << (uint(class) - 1)

	rows, err := r.db.Query(`
		SELECT st.equippedItemClass, st.equippedItemSubClassMask, COALESCE(ss.classmask, 0)
		FROM spell_template st
		LEFT JOIN spell_skill_spells ss ON ss.spell_id = st.entry
		WHERE (st.effect1 = 60 OR st.effect2 = 60 OR st.effect3 = 60)
		  AND st.equippedItemClass IN (2, 4)
	`)
	if err != nil {
		return nil, nil
	}
	defer rows.Close()

	restricted := map[int]map[int]bool{2: {}, 4: {}}
	allowed := map[int]map[int]bool{2: {}, 4: {}}
	for rows.Next() {
		var itemClass, subMask, classMask int
		if rows.Scan(&itemClass, &subMask, &classMask) != nil {
			continue
		}
		learnable := classMask == 0 || classMask&classBit != 0
		for s := 0; s < 32; s++ {
			if subMask&(1<<uint(s)) == 0 {
				continue
			}
			restricted[itemClass][s] = true
			if learnable {
				allowed[itemClass][s] = true
			}
		}
	}

	collect := func(itemClass int) []int {
		var out []int
		for s := range restricted[itemClass] {
			if !allowed[itemClass][s] {
				out = append(out, s)
			}
		}
		return out
	}
	return collect(2), collect(4)
}

// buildSourceCondition turns a set of source keys into a single OR'd EXISTS
// clause (matching the item-detail relation logic) plus its args. Unknown keys
// are ignored; an empty/all-unknown set returns "".
func buildSourceCondition(sources []string) (string, []any) {
	var ors []string
	for _, s := range sources {
		switch s {
		case "drop": // dropped by a creature (directly or via a referenced loot table)
			ors = append(ors, `(EXISTS (SELECT 1 FROM creature_loot_template WHERE item = t.entry)
				OR EXISTS (SELECT 1 FROM reference_loot_template WHERE item = t.entry))`)
		case "object": // looted from a game object (chest, node, ...)
			ors = append(ors, "EXISTS (SELECT 1 FROM gameobject_loot_template WHERE item = t.entry)")
		case "container": // contained in another item (opening a container item)
			ors = append(ors, "EXISTS (SELECT 1 FROM item_loot_template WHERE item = t.entry)")
		case "disenchant": // obtained by disenchanting an item
			ors = append(ors, "EXISTS (SELECT 1 FROM disenchant_loot_template WHERE item = t.entry)")
		case "vendor": // sold by an NPC vendor
			ors = append(ors, "EXISTS (SELECT 1 FROM item_vendor WHERE item_entry = t.entry)")
		case "quest": // rewarded by a quest (fixed or choice reward)
			ors = append(ors, `EXISTS (SELECT 1 FROM quest_template q WHERE
				q.RewItemId1 = t.entry OR q.RewItemId2 = t.entry OR q.RewItemId3 = t.entry OR q.RewItemId4 = t.entry
				OR q.RewChoiceItemId1 = t.entry OR q.RewChoiceItemId2 = t.entry OR q.RewChoiceItemId3 = t.entry
				OR q.RewChoiceItemId4 = t.entry OR q.RewChoiceItemId5 = t.entry OR q.RewChoiceItemId6 = t.entry)`)
		case "crafted": // created by a spell (SPELL_EFFECT_CREATE_ITEM = 24)
			ors = append(ors, `EXISTS (SELECT 1 FROM spell_template st WHERE
				(st.effect1 = 24 AND st.effectItemType1 = t.entry)
				OR (st.effect2 = 24 AND st.effectItemType2 = t.entry)
				OR (st.effect3 = 24 AND st.effectItemType3 = t.entry))`)
		}
	}
	if len(ors) == 0 {
		return "", nil
	}
	return "(" + strings.Join(ors, " OR ") + ")", nil
}

// orderByClause maps a sort field + direction to a safe ORDER BY (whitelisted —
// never interpolates user input). Falls back to the quality/ilvl default.
func orderByClause(sort, dir string) string {
	col, ok := map[string]string{
		"name":           "name",
		"itemLevel":      "item_level",
		"requiredLevel":  "required_level",
		"quality":        "quality",
		"containerSlots": "container_slots",
	}[sort]
	if !ok {
		return "quality DESC, item_level DESC"
	}
	d := "ASC"
	if strings.EqualFold(dir, "desc") {
		d = "DESC"
	}
	// Stable tiebreaker on entry so paging is deterministic.
	return fmt.Sprintf("%s %s, entry ASC", col, d)
}
