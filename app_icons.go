package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// IconEntry is one unique icon file in data/icons plus how many items/spells
// reference it, so the Icons tab can list, filter, and sort by usage.
type IconEntry struct {
	Name       string `json:"name"`
	ItemCount  int    `json:"itemCount"`
	SpellCount int    `json:"spellCount"`
}

// iconUsageCounts tallies references per icon name (lowercased) from a GROUP BY
// query, tolerating a missing table so a fresh DB just shows zero counts.
func (a *App) iconUsageCounts(query string) map[string]int {
	counts := map[string]int{}
	rows, err := a.db.DB().Query(query)
	if err != nil {
		return counts
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		var n int
		if err := rows.Scan(&name, &n); err == nil {
			counts[name] = n
		}
	}
	return counts
}

// ListLocalIcons enumerates the unique icons in data/icons (the client-imported
// set — the only place icon images come from). Names are the lowercase file
// names without extension, the same keys useIcon/GetLocalImage resolve.
func (a *App) ListLocalIcons() []IconEntry {
	iconDir := filepath.Join(a.DataDir, "icons")
	dirents, err := os.ReadDir(iconDir)
	if err != nil {
		fmt.Printf("[API] ListLocalIcons: cannot read %s: %v\n", iconDir, err)
		return []IconEntry{}
	}

	itemCounts := a.iconUsageCounts(`
		SELECT LOWER(d.icon), COUNT(*)
		FROM item_template t
		JOIN item_display_info d ON t.display_id = d.ID
		WHERE d.icon != ''
		GROUP BY LOWER(d.icon)`)
	spellCounts := a.iconUsageCounts(`
		SELECT LOWER(si.icon_name), COUNT(*)
		FROM spell_template sp
		JOIN spell_icons si ON sp.spellIconId = si.id
		WHERE si.icon_name != ''
		GROUP BY LOWER(si.icon_name)`)

	seen := map[string]bool{}
	out := []IconEntry{}
	for _, de := range dirents {
		if de.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(de.Name()))
		if ext != ".png" && ext != ".jpg" && ext != ".jpeg" {
			continue
		}
		name := strings.ToLower(strings.TrimSuffix(de.Name(), filepath.Ext(de.Name())))
		if seen[name] {
			continue
		}
		seen[name] = true
		out = append(out, IconEntry{
			Name:       name,
			ItemCount:  itemCounts[name],
			SpellCount: spellCounts[name],
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	fmt.Printf("[API] ListLocalIcons: %d unique icons\n", len(out))
	return out
}

// IconUsageItem is an item that renders a given icon.
type IconUsageItem struct {
	Entry   int    `json:"entry"`
	Name    string `json:"name"`
	Quality int    `json:"quality"`
}

// IconUsageSpell is a spell that renders a given icon.
type IconUsageSpell struct {
	Entry int    `json:"entry"`
	Name  string `json:"name"`
	Rank  string `json:"rank"`
}

// IconUsage lists every entity that uses one icon — the icon detail page.
type IconUsage struct {
	Name   string           `json:"name"`
	Items  []IconUsageItem  `json:"items"`
	Spells []IconUsageSpell `json:"spells"`
}

// GetIconUsage returns all items and spells whose icon resolves to `name`
// (case-insensitive, matching how icon files are looked up on disk).
func (a *App) GetIconUsage(name string) *IconUsage {
	usage := &IconUsage{Name: strings.ToLower(name), Items: []IconUsageItem{}, Spells: []IconUsageSpell{}}

	rows, err := a.db.DB().Query(`
		SELECT t.entry, t.name, t.quality
		FROM item_template t
		JOIN item_display_info d ON t.display_id = d.ID
		WHERE LOWER(d.icon) = LOWER(?)
		ORDER BY t.quality DESC, t.name`, name)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var it IconUsageItem
			if err := rows.Scan(&it.Entry, &it.Name, &it.Quality); err == nil {
				usage.Items = append(usage.Items, it)
			}
		}
	}

	sRows, err := a.db.DB().Query(`
		SELECT sp.entry, sp.name, COALESCE(sp.nameSubtext, '')
		FROM spell_template sp
		JOIN spell_icons si ON sp.spellIconId = si.id
		WHERE LOWER(si.icon_name) = LOWER(?)
		ORDER BY sp.name, sp.entry`, name)
	if err == nil {
		defer sRows.Close()
		for sRows.Next() {
			var sp IconUsageSpell
			if err := sRows.Scan(&sp.Entry, &sp.Name, &sp.Rank); err == nil {
				usage.Spells = append(usage.Spells, sp)
			}
		}
	}

	return usage
}
