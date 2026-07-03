package main

import (
	"fmt"
	"sort"
)

// The professions browser: skill lines from SkillLine.dbc (spell_skills) whose
// recipes come from SkillLineAbility (spell_skill_spells) joined with
// spell_template. A "recipe" is a craft-like spell — create item (effect 24)
// or enchant item (effect 53) — which keeps the profession-rank/proficiency
// spells out of the list.

const recipeEffectPred = `(sp.effect1 IN (24, 53) OR sp.effect2 IN (24, 53) OR sp.effect3 IN (24, 53))`

// Profession is one entry in the professions sidebar.
type Profession struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Count int    `json:"count"` // recipe count
}

// GetProfessions lists the primary professions (SkillLine category 11) and the
// crafting secondaries (category 9 lines that have at least one recipe, e.g.
// Cooking and First Aid but not Riding), with recipe counts.
func (a *App) GetProfessions() []Profession {
	out := []Profession{}
	rows, err := a.db.DB().Query(`
		SELECT s.id, s.name,
		       (SELECT COUNT(*) FROM spell_skill_spells ss
		        JOIN spell_template sp ON sp.entry = ss.spell_id
		        WHERE ss.skill_id = s.id AND ` + recipeEffectPred + `) AS recipes
		FROM spell_skills s
		WHERE s.category_id IN (9, 11)
		ORDER BY s.name`)
	if err != nil {
		fmt.Printf("[API] GetProfessions: %v\n", err)
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var p Profession
		if rows.Scan(&p.ID, &p.Name, &p.Count) != nil {
			continue
		}
		if p.Count > 0 {
			out = append(out, p)
		}
	}
	return out
}

// RecipeItem is an item attached to a recipe (the crafted product, a reagent,
// or the recipe item that teaches it).
type RecipeItem struct {
	Entry   int    `json:"entry"`
	Name    string `json:"name"`
	Quality int    `json:"quality"`
	Icon    string `json:"icon"`
	Count   int    `json:"count,omitempty"`
}

// ProfessionRecipe is one row of the profession recipe browser.
type ProfessionRecipe struct {
	SpellID int    `json:"spellId"`
	Name    string `json:"name"`
	Icon    string `json:"icon"` // spell icon (fallback when nothing is crafted)

	// Skill-up thresholds (the server's SkillGainChance formula): the recipe is
	// orange below Yellow, yellow to Green, green to Grey, grey after. Learn is
	// the rank it's learnable at (trainer/recipe-item rank, else the SLA value).
	Learn  int `json:"learn"`
	Yellow int `json:"yellow"`
	Green  int `json:"green"`
	Grey   int `json:"grey"`

	Crafts   *RecipeItem  `json:"crafts,omitempty"` // created item (effect 24)
	Reagents []RecipeItem `json:"reagents"`

	Trainer   bool        `json:"trainer"`             // taught by a trainer
	TeachItem *RecipeItem `json:"teachItem,omitempty"` // taught by this recipe item
	Quest     bool        `json:"quest"`               // rewarded by a quest
}

// GetProfessionRecipes returns every recipe of a profession skill line with
// skill-up thresholds, reagents, the crafted item, and how it's learned.
// Assembled set-based (a handful of queries per call, not per recipe).
func (a *App) GetProfessionRecipes(skillID int) []ProfessionRecipe {
	db := a.db.DB()
	out := []ProfessionRecipe{}

	type rawRecipe struct {
		rec       *ProfessionRecipe
		craftItem int
		craftCnt  int
		reagents  [8]int
		reagentN  [8]int
	}
	var raws []*rawRecipe
	craftSet := map[int]*rawRecipe{} // spellID -> row

	rows, err := db.Query(`
		SELECT sp.entry, sp.name, COALESCE(si.icon_name, ''),
		       ss.req_skill_value, ss.min_value, ss.max_value,
		       CASE WHEN sp.effect1 = 24 THEN sp.effectItemType1
		            WHEN sp.effect2 = 24 THEN sp.effectItemType2
		            WHEN sp.effect3 = 24 THEN sp.effectItemType3 ELSE 0 END,
		       CASE WHEN sp.effect1 = 24 THEN sp.effectBasePoints1 + 1
		            WHEN sp.effect2 = 24 THEN sp.effectBasePoints2 + 1
		            WHEN sp.effect3 = 24 THEN sp.effectBasePoints3 + 1 ELSE 1 END,
		       sp.reagent1, sp.reagent2, sp.reagent3, sp.reagent4,
		       sp.reagent5, sp.reagent6, sp.reagent7, sp.reagent8,
		       sp.reagentCount1, sp.reagentCount2, sp.reagentCount3, sp.reagentCount4,
		       sp.reagentCount5, sp.reagentCount6, sp.reagentCount7, sp.reagentCount8
		FROM spell_skill_spells ss
		JOIN spell_template sp ON sp.entry = ss.spell_id
		LEFT JOIN spell_icons si ON si.id = sp.spellIconId
		WHERE ss.skill_id = ? AND `+recipeEffectPred, skillID)
	if err != nil {
		fmt.Printf("[API] GetProfessionRecipes(%d): %v\n", skillID, err)
		return out
	}
	for rows.Next() {
		r := &rawRecipe{rec: &ProfessionRecipe{Reagents: []RecipeItem{}}}
		var reqSkill, minV, maxV int
		if rows.Scan(&r.rec.SpellID, &r.rec.Name, &r.rec.Icon,
			&reqSkill, &minV, &maxV,
			&r.craftItem, &r.craftCnt,
			&r.reagents[0], &r.reagents[1], &r.reagents[2], &r.reagents[3],
			&r.reagents[4], &r.reagents[5], &r.reagents[6], &r.reagents[7],
			&r.reagentN[0], &r.reagentN[1], &r.reagentN[2], &r.reagentN[3],
			&r.reagentN[4], &r.reagentN[5], &r.reagentN[6], &r.reagentN[7]) != nil {
			continue
		}
		r.rec.Learn = reqSkill
		r.rec.Yellow = minV
		r.rec.Green = (minV + maxV) / 2
		r.rec.Grey = maxV
		raws = append(raws, r)
		craftSet[r.rec.SpellID] = r
	}
	rows.Close()
	if len(raws) == 0 {
		return out
	}

	// Resolve all referenced items (crafted + reagents) in one pass.
	itemIDs := map[int]bool{}
	for _, r := range raws {
		if r.craftItem > 0 {
			itemIDs[r.craftItem] = true
		}
		for _, id := range r.reagents {
			if id > 0 {
				itemIDs[id] = true
			}
		}
	}
	items := a.recipeItemsByID(itemIDs)
	for _, r := range raws {
		if it, ok := items[r.craftItem]; ok {
			c := it
			c.Count = r.craftCnt
			r.rec.Crafts = &c
		}
		for i, id := range r.reagents {
			if it, ok := items[id]; ok && r.reagentN[i] > 0 {
				c := it
				c.Count = r.reagentN[i]
				r.rec.Reagents = append(r.rec.Reagents, c)
			}
		}
	}

	// Learn spells (effect 36): learnSpellID -> the craft spells it teaches.
	// Bridges trainers / recipe items / quests to the craft spell.
	teaches := map[int][]int{}
	if lRows, err := db.Query(`
		SELECT entry, effectTriggerSpell1, effectTriggerSpell2, effectTriggerSpell3
		FROM spell_template
		WHERE effect1 = 36 OR effect2 = 36 OR effect3 = 36`); err == nil {
		for lRows.Next() {
			var id, t1, t2, t3 int
			if lRows.Scan(&id, &t1, &t2, &t3) != nil {
				continue
			}
			for _, t := range []int{t1, t2, t3} {
				if _, ok := craftSet[t]; ok && t > 0 {
					teaches[id] = append(teaches[id], t)
				}
			}
		}
		lRows.Close()
	}

	// Trainer source + authoritative taught-at rank. npc_trainer(.template)
	// reference the learn spell; reqskillvalue is the rank a trainer teaches it
	// at, which the craft spell's own SkillLineAbility row doesn't carry.
	if tRows, err := db.Query(`
		SELECT spell, MIN(reqskillvalue) FROM (
			SELECT spell, reqskillvalue FROM npc_trainer
			UNION ALL
			SELECT spell, reqskillvalue FROM npc_trainer_template
		) GROUP BY spell`); err == nil {
		for tRows.Next() {
			var spell, req int
			if tRows.Scan(&spell, &req) != nil {
				continue
			}
			// npc_trainer.spell may be the learn spell or the craft spell itself.
			targets := teaches[spell]
			if r, ok := craftSet[spell]; ok {
				targets = append(targets, r.rec.SpellID)
			}
			for _, t := range targets {
				r := craftSet[t]
				r.rec.Trainer = true
				if req > 0 && (r.rec.Learn == 0 || req > r.rec.Learn) {
					r.rec.Learn = req
				}
			}
		}
		tRows.Close()
	}

	// Recipe items that teach a craft spell (item on-use = learn spell).
	if iRows, err := db.Query(`
		SELECT i.entry, i.name, i.quality, COALESCE(d.icon, ''), i.required_skill_rank,
		       i.spellid_1, i.spellid_2, i.spellid_3, i.spellid_4, i.spellid_5
		FROM item_template i
		LEFT JOIN item_display_info d ON i.display_id = d.ID
		WHERE i.spellid_1 > 0 OR i.spellid_2 > 0`); err == nil {
		for iRows.Next() {
			var it RecipeItem
			var reqRank int
			var sp [5]int
			if iRows.Scan(&it.Entry, &it.Name, &it.Quality, &it.Icon, &reqRank,
				&sp[0], &sp[1], &sp[2], &sp[3], &sp[4]) != nil {
				continue
			}
			for _, s := range sp {
				for _, t := range teaches[s] {
					r := craftSet[t]
					if r.rec.TeachItem == nil {
						tc := it
						r.rec.TeachItem = &tc
						if reqRank > 0 && reqRank > r.rec.Learn {
							r.rec.Learn = reqRank
						}
					}
				}
			}
		}
		iRows.Close()
	}

	// Quest rewards that teach a craft spell.
	if qRows, err := db.Query(`
		SELECT RewSpell, RewSpellCast FROM quest_template
		WHERE RewSpell > 0 OR RewSpellCast > 0`); err == nil {
		for qRows.Next() {
			var rew, cast int
			if qRows.Scan(&rew, &cast) != nil {
				continue
			}
			if r, ok := craftSet[rew]; ok {
				r.rec.Quest = true
			}
			for _, s := range []int{rew, cast} {
				for _, t := range teaches[s] {
					craftSet[t].rec.Quest = true
				}
			}
		}
		qRows.Close()
	}

	for _, r := range raws {
		// A recipe with no learn rank anywhere: fall back to the yellow
		// threshold so the [learn/yellow/green/grey] chips make sense.
		if r.rec.Learn == 0 {
			r.rec.Learn = r.rec.Yellow
		}
		out = append(out, *r.rec)
	}
	// Order by effective difficulty: some recipes carry a placeholder learn
	// rank of 1 with high skill-up thresholds (e.g. custom smelts), so the
	// yellow threshold caps the sort key — the list reads as a leveling path.
	key := func(r ProfessionRecipe) int {
		if r.Yellow > r.Learn {
			return r.Yellow
		}
		return r.Learn
	}
	sort.SliceStable(out, func(i, j int) bool {
		if key(out[i]) != key(out[j]) {
			return key(out[i]) < key(out[j])
		}
		return out[i].Name < out[j].Name
	})
	return out
}

// recipeItemsByID resolves item entries to name/quality/icon in one query.
func (a *App) recipeItemsByID(ids map[int]bool) map[int]RecipeItem {
	items := map[int]RecipeItem{}
	if len(ids) == 0 {
		return items
	}
	ph := ""
	args := make([]any, 0, len(ids))
	for id := range ids {
		if ph != "" {
			ph += ","
		}
		ph += "?"
		args = append(args, id)
	}
	rows, err := a.db.DB().Query(`
		SELECT t.entry, t.name, t.quality, COALESCE(d.icon, '')
		FROM item_template t
		LEFT JOIN item_display_info d ON t.display_id = d.ID
		WHERE t.entry IN (`+ph+`)`, args...)
	if err != nil {
		return items
	}
	defer rows.Close()
	for rows.Next() {
		var it RecipeItem
		if rows.Scan(&it.Entry, &it.Name, &it.Quality, &it.Icon) == nil {
			items[it.Entry] = it
		}
	}
	return items
}
