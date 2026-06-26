package main

import (
	"encoding/json"
	"math/bits"
	"strings"
)

// Talent data is read from the talent_tab / talent tables, which are populated
// from the DBC-derived data/talents.json during a client import (and shipped in
// the embedded inklab.db). Per-rank spell name/icon/description are resolved
// from spell_template, whose descriptions are already $-resolved.

// TalentRank carries the resolved per-rank tooltip text.
type TalentRank struct {
	SpellID     int    `json:"spellId"`
	Description string `json:"description"`
}

// Talent is one node in a tree, enriched with spell name/icon/per-rank text.
type Talent struct {
	ID        int          `json:"id"`
	Row       int          `json:"row"`
	Col       int          `json:"col"`
	Name      string       `json:"name"`
	Icon      string       `json:"icon"` // icon base name; resolved via GetLocalImage("icon", …)
	MaxRank   int          `json:"maxRank"`
	Ranks     []TalentRank `json:"ranks"`
	ReqTalent int          `json:"reqTalent"`
	ReqRank   int          `json:"reqRank"`
}

// TalentTree is one of a class's three trees.
type TalentTree struct {
	ID         int      `json:"id"`
	Name       string   `json:"name"`
	Order      int      `json:"order"`
	Background string   `json:"background"` // lowercased base name for talent_bg image lookup
	Talents    []Talent `json:"talents"`
}

// TalentClassData is the full set of trees for one class.
type TalentClassData struct {
	Class string       `json:"class"`
	Trees []TalentTree `json:"trees"`
}

// TalentClassInfo pairs a class token with its WoW class id. The id is derived
// from the TalentTab class_mask (a single class bit) — classId = bit index + 1
// — so it comes from the client DBC, not a hardcoded table.
type TalentClassInfo struct {
	Class   string `json:"class"`            // token, e.g. "WARRIOR"
	ClassID int    `json:"classId"`          // WoW class id
	Name    string `json:"name,omitempty"`   // display name from ChrClasses.dbc
}

// GetTalentClasses returns the classes that have talent trees with their class
// ids and display names, ordered by the conventional class order (the order
// trees were exported in). The display name comes from ChrClasses.dbc
// (class_info), not derived from the token.
func (a *App) GetTalentClasses() []TalentClassInfo {
	if a.db == nil {
		return nil
	}
	// id -> display name from the client DBC.
	nameByID := map[int]string{}
	if cr, err := a.db.DB().Query("SELECT id, name FROM class_info"); err == nil {
		for cr.Next() {
			var id int
			var name string
			if cr.Scan(&id, &name) == nil {
				nameByID[id] = name
			}
		}
		cr.Close()
	}

	rows, err := a.db.DB().Query("SELECT class, MAX(class_mask) AS mask, MIN(id) AS o FROM talent_tab GROUP BY class ORDER BY o")
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []TalentClassInfo
	for rows.Next() {
		var class string
		var mask, o int
		if rows.Scan(&class, &mask, &o) == nil && class != "" {
			id := 0
			if mask > 0 {
				id = bits.TrailingZeros32(uint32(mask)) + 1
			}
			out = append(out, TalentClassInfo{Class: class, ClassID: id, Name: nameByID[id]})
		}
	}
	return out
}

// GetTalentTrees returns the three talent trees for a class, with each rank's
// spell name/icon/description resolved from the (already $-resolved)
// spell_template rows. The talent's display name and icon come from its rank-1
// spell.
func (a *App) GetTalentTrees(class string) *TalentClassData {
	class = strings.ToUpper(strings.TrimSpace(class))
	out := &TalentClassData{Class: class}
	if a.db == nil {
		return out
	}

	// Load the trees for this class.
	tabRows, err := a.db.DB().Query(
		"SELECT id, name, order_index, background FROM talent_tab WHERE class = ? ORDER BY order_index", class)
	if err != nil {
		return out
	}
	type rawTab struct {
		id, order int
		name, bg  string
	}
	var tabs []rawTab
	for tabRows.Next() {
		var t rawTab
		if tabRows.Scan(&t.id, &t.name, &t.order, &t.bg) == nil {
			tabs = append(tabs, t)
		}
	}
	tabRows.Close()
	if len(tabs) == 0 {
		return out
	}

	// Load talents for those trees and collect every rank spell id.
	type rawTalent struct {
		id, tabID, row, col, reqTalent, reqRank int
		ranks                                   []int
	}
	talentsByTab := map[int][]rawTalent{}
	idSet := map[int]bool{}
	for _, tab := range tabs {
		talRows, err := a.db.DB().Query(
			"SELECT id, row, col, ranks, req_talent, req_rank FROM talent WHERE tab_id = ? ORDER BY row, col", tab.id)
		if err != nil {
			continue
		}
		for talRows.Next() {
			var t rawTalent
			var ranksJSON string
			if talRows.Scan(&t.id, &t.row, &t.col, &ranksJSON, &t.reqTalent, &t.reqRank) != nil {
				continue
			}
			json.Unmarshal([]byte(ranksJSON), &t.ranks)
			t.tabID = tab.id
			for _, s := range t.ranks {
				idSet[s] = true
			}
			talentsByTab[tab.id] = append(talentsByTab[tab.id], t)
		}
		talRows.Close()
	}

	// Resolve all rank spells in one pass.
	type spellInfo struct{ name, desc, icon string }
	info := make(map[int]spellInfo, len(idSet))
	if len(idSet) > 0 {
		ph := make([]string, 0, len(idSet))
		args := make([]interface{}, 0, len(idSet))
		for id := range idSet {
			ph = append(ph, "?")
			args = append(args, id)
		}
		q := "SELECT entry, COALESCE(name,''), COALESCE(description,''), COALESCE(iconName,'') FROM spell_template WHERE entry IN (" + strings.Join(ph, ",") + ")"
		if rows, err := a.db.DB().Query(q, args...); err == nil {
			for rows.Next() {
				var entry int
				var name, desc, icon string
				if rows.Scan(&entry, &name, &desc, &icon) == nil {
					info[entry] = spellInfo{name: name, desc: desc, icon: strings.ToLower(icon)}
				}
			}
			rows.Close()
		}
	}

	for _, tab := range tabs {
		tree := TalentTree{ID: tab.id, Name: tab.name, Order: tab.order, Background: strings.ToLower(tab.bg)}
		for _, t := range talentsByTab[tab.id] {
			tal := Talent{
				ID:        t.id,
				Row:       t.row,
				Col:       t.col,
				MaxRank:   len(t.ranks),
				ReqTalent: t.reqTalent,
				ReqRank:   t.reqRank,
			}
			for _, s := range t.ranks {
				si := info[s]
				if tal.Name == "" && si.name != "" {
					tal.Name = si.name
				}
				if tal.Icon == "" && si.icon != "" {
					tal.Icon = si.icon
				}
				tal.Ranks = append(tal.Ranks, TalentRank{SpellID: s, Description: si.desc})
			}
			tree.Talents = append(tree.Talents, tal)
		}
		out.Trees = append(out.Trees, tree)
	}
	return out
}
