package main

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"

	"inklab/backend/database"
)

// GetRootCategories returns top-level categories (e.g., "Mage Sets", "Molten Core")
func (a *App) GetRootCategories() []*database.Category {
	fmt.Println("[API] GetRootCategories called")
	cats, err := a.categoryRepo.GetRootCategories()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return []*database.Category{}
	}
	return cats
}

// GetChildCategories returns sub-categories (e.g., Bosses in an Instance)
func (a *App) GetChildCategories(parentID int) []*database.Category {
	cats, err := a.categoryRepo.GetChildCategories(parentID)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return []*database.Category{}
	}
	return cats
}

// GetCategoryItems returns items for a specific category (e.g., drops from Ragnaros)
func (a *App) GetCategoryItems(categoryID int) []*database.Item {
	items, err := a.categoryRepo.GetCategoryItems(categoryID)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return []*database.Item{}
	}
	return a.enrichItemsWithIcons(items)
}

// SearchItems searches for items by name (Simple)
func (a *App) SearchItems(query string) []*database.Item {
	items, err := a.itemRepo.SearchItems(query, 50)
	if err != nil {
		fmt.Printf("Error searching items: %v\n", err)
		return []*database.Item{}
	}
	return a.enrichItemsWithIcons(items)
}

// GetItemClasses returns hierarchical item classes
func (a *App) GetItemClasses() []*database.ItemClass {
	fmt.Println("[API] GetItemClasses called")
	classes, err := a.itemRepo.GetItemClasses()
	if err != nil {
		fmt.Printf("[API] Error getting classes: %v\n", err)
		return []*database.ItemClass{}
	}
	fmt.Printf("[API] GetItemClasses returning %d classes\n", len(classes))
	return classes
}

// GetItemStatTypes returns the stat types present in the item data, with display
// names, for the filter's stat dropdown.
func (a *App) GetItemStatTypes() []*database.StatType {
	stats, err := a.itemRepo.GetStatTypes()
	if err != nil {
		fmt.Printf("[API] Error getting stat types: %v\n", err)
		return []*database.StatType{}
	}
	return stats
}

// BrowseItemsByClass returns items for a specific class/subclass
func (a *App) BrowseItemsByClass(class, subClass int, nameFilter string) []*database.Item {
	fmt.Printf("[API] BrowseItemsByClass called: class=%d, subClass=%d, filter='%s'\n", class, subClass, nameFilter)
	// No limit - return all matching items
	items, _, err := a.itemRepo.GetItemsByClass(class, subClass, nameFilter, 999999, 0)
	if err != nil {
		fmt.Printf("[API] Error browsing items: %v\n", err)
		return []*database.Item{}
	}
	fmt.Printf("[API] BrowseItemsByClass returning %d items\n", len(items))
	return a.enrichItemsWithIcons(items)
}

// BrowseItemsByClassAndSlot returns items for a specific class/subclass/inventoryType
func (a *App) BrowseItemsByClassAndSlot(class, subClass, inventoryType int, nameFilter string) []*database.Item {
	// No limit - return all matching items
	items, _, err := a.itemRepo.GetItemsByClassAndSlot(class, subClass, inventoryType, nameFilter, 999999, 0)
	if err != nil {
		fmt.Printf("Error browsing items by slot: %v\n", err)
		return []*database.Item{}
	}
	return a.enrichItemsWithIcons(items)
}

// AdvancedSearch performs a detailed search
func (a *App) AdvancedSearch(filter database.SearchFilter) *database.SearchResult {
	result, err := a.itemRepo.AdvancedSearch(filter)
	if err != nil {
		fmt.Printf("Error in advanced search: %v\n", err)
		return &database.SearchResult{Items: []*database.Item{}, TotalCount: 0}
	}
	result.Items = a.enrichItemsWithIcons(result.Items)

	// Search spells (supports both ID and name)
	if filter.Query != "" {
		spells, err := a.spellRepo.SearchSpells(filter.Query)
		if err == nil && len(spells) > 0 {
			result.Spells = spells
		}

		// Search Creatures (Repository handles Name OR ID)
		creatures, err := a.creatureRepo.SearchCreatures(filter.Query, 50)
		if err == nil && len(creatures) > 0 {
			result.Creatures = creatures
		}

		// Search Quests
		// 1. By ID (if numeric)
		if id, err := strconv.Atoi(filter.Query); err == nil && id > 0 {
			quest, _ := a.questRepo.GetQuestByID(id)
			if quest != nil && quest.Entry > 0 {
				result.Quests = append(result.Quests, quest)
			}
		}
		// 2. By Title
		quests, err := a.questRepo.SearchQuests(filter.Query)
		if err == nil && len(quests) > 0 {
			// Deduplicate in case ID match is the same
			for _, q := range quests {
				found := false
				for _, existing := range result.Quests {
					if existing.Entry == q.Entry {
						found = true
						break
					}
				}
				if !found {
					result.Quests = append(result.Quests, q)
				}
			}
		}

		// Search Game Objects (by ID if numeric, then by name)
		if id, err := strconv.Atoi(filter.Query); err == nil && id > 0 {
			if d, _ := a.objectRepo.GetObjectDetail(id); d != nil && d.Entry > 0 {
				result.Objects = append(result.Objects, &database.GameObject{
					Entry: d.Entry, Name: d.Name, Type: d.Type, TypeName: d.TypeName,
					DisplayID: d.DisplayID, Size: d.Size,
				})
			}
		}
		objects, err := a.objectRepo.SearchObjects(filter.Query)
		if err == nil {
			for _, o := range objects {
				dup := false
				for _, ex := range result.Objects {
					if ex.Entry == o.Entry {
						dup = true
						break
					}
				}
				if !dup {
					result.Objects = append(result.Objects, o)
				}
			}
		}
	}

	return result
}

// BrowseItems is the items-only paginated/filtered search powering the Items
// page. Unlike AdvancedSearch (which also fans out to spells/creatures/quests
// for the global search bar), this returns just the matching items plus the
// total count for pagination — driven entirely by the SearchFilter (name,
// quality, class/subclass/slot, level ranges, stats, usable-by-class, source,
// sort).
func (a *App) BrowseItems(filter database.SearchFilter) *database.SearchResult {
	result, err := a.itemRepo.AdvancedSearch(filter)
	if err != nil {
		fmt.Printf("[API] Error in BrowseItems: %v\n", err)
		return &database.SearchResult{Items: []*database.Item{}, TotalCount: 0}
	}
	result.Items = a.enrichItemsWithIcons(result.Items)
	return result
}

// GetTooltipData returns detailed item information (no Wails binding generation)
func (a *App) GetTooltipData(itemID int) *database.TooltipData {
	data, err := a.itemRepo.GetTooltipData(itemID)
	if err != nil {
		return nil
	}
	return data
}

// GetItemSets returns all item sets for browsing
func (a *App) GetItemSets() []*database.ItemSetBrowse {
	fmt.Println("[API] GetItemSets called")
	sets, err := a.itemRepo.GetItemSets()
	if err != nil {
		fmt.Printf("[API] Error getting item sets: %v\n", err)
		return []*database.ItemSetBrowse{}
	}
	fmt.Printf("[API] GetItemSets returning %d sets\n", len(sets))
	return sets
}

// GetItemSetDetail returns detailed information about a specific item set
func (a *App) GetItemSetDetail(itemSetID int) *database.ItemSetDetail {
	detail, err := a.itemRepo.GetItemSetDetail(itemSetID)
	if err != nil {
		fmt.Printf("Error getting item set detail: %v\n", err)
		return nil
	}
	// Enrich items with icons
	detail.Items = a.enrichItemsWithIcons(detail.Items)
	return detail
}

// GetItemDetail returns full details for an item
func (a *App) GetItemDetail(entry int) (*database.ItemDetail, error) {
	i, err := a.itemRepo.GetItemDetail(entry)
	if err != nil {
		fmt.Printf("Error getting item detail [%d]: %v\n", entry, err)
		return nil, err
	}
	if i != nil && i.Item != nil {
		a.enrichItemIcon(i.Item)
	}
	return i, nil
}

// ItemRandomSuffix is one random enchantment ("of the Monkey") an item can
// roll: the suffix text, its stat lines, and its roll chance in percent.
// LinkID is the ItemRandomProperties id of the group's best stat roll — the
// randomPropertyId field of a classic item link
// (|Hitem:itemId:enchantId:randomPropertyId:uniqueId|h).
type ItemRandomSuffix struct {
	Suffix  string   `json:"suffix"`
	Effects []string `json:"effects"`
	Chance  float64  `json:"chance"`
	LinkID  int      `json:"linkId"`
}

// suffixEffectRe splits an enchantment line like "+3 Agility" into amount and
// label so same-suffix variants can merge into a "+1-3 Agility" range.
var suffixEffectRe = regexp.MustCompile(`^\+(\d+) (.+)$`)

// GetItemRandomSuffixes returns the possible random suffixes for an item with
// a random_property, resolved through item_enchantment_template (the roll
// pool) and item_random_suffix (the DBC-derived suffix text + stat lines).
// The pool holds one row per stat roll (e.g. "of the Monkey" seven times with
// different +Agi/+Sta combos), so rows aggregate by suffix: amounts collapse
// into min-max ranges and weights sum, normalized to percent. Empty for items
// without random properties.
func (a *App) GetItemRandomSuffixes(entry int) []ItemRandomSuffix {
	out := []ItemRandomSuffix{}
	rows, err := a.db.DB().Query(`
		SELECT s.id, s.suffix, s.effects, t.chance
		FROM item_template i
		JOIN item_enchantment_template t ON t.entry = i.random_property
		JOIN item_random_suffix s ON s.id = t.ench
		WHERE i.entry = ?`, entry)
	if err != nil {
		return out
	}
	defer rows.Close()

	type rng struct{ min, max int }
	type agg struct {
		chance  float64
		labels  []string // effect labels in first-seen order
		ranges  map[string]*rng
		raw     []string // unparseable lines, kept verbatim
		bestID  int      // suffix id of the highest-total roll (for item links)
		bestSum int
	}
	byName := map[string]*agg{}
	order := []string{}
	var total float64

	for rows.Next() {
		var id int
		var suffix, effectsJSON string
		var chance float64
		if err := rows.Scan(&id, &suffix, &effectsJSON, &chance); err != nil {
			continue
		}
		var effects []string
		_ = json.Unmarshal([]byte(effectsJSON), &effects)

		a := byName[suffix]
		if a == nil {
			a = &agg{ranges: map[string]*rng{}}
			byName[suffix] = a
			order = append(order, suffix)
		}
		a.chance += chance
		total += chance
		sum := 0
		for _, e := range effects {
			m := suffixEffectRe.FindStringSubmatch(e)
			if m == nil {
				if !slices.Contains(a.raw, e) {
					a.raw = append(a.raw, e)
				}
				continue
			}
			n, _ := strconv.Atoi(m[1])
			sum += n
			label := m[2]
			if r, ok := a.ranges[label]; ok {
				r.min = min(r.min, n)
				r.max = max(r.max, n)
			} else {
				a.ranges[label] = &rng{n, n}
				a.labels = append(a.labels, label)
			}
		}
		if a.bestID == 0 || sum > a.bestSum {
			a.bestID, a.bestSum = id, sum
		}
	}

	for _, suffix := range order {
		a := byName[suffix]
		s := ItemRandomSuffix{Suffix: suffix, Effects: []string{}, LinkID: a.bestID}
		for _, label := range a.labels {
			r := a.ranges[label]
			if r.min == r.max {
				s.Effects = append(s.Effects, fmt.Sprintf("+%d %s", r.min, label))
			} else {
				s.Effects = append(s.Effects, fmt.Sprintf("+%d-%d %s", r.min, r.max, label))
			}
		}
		s.Effects = append(s.Effects, a.raw...)
		if total > 0 {
			// Weights in the world DB are relative, not percentages.
			s.Chance = a.chance / total * 100
		}
		out = append(out, s)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Chance > out[j].Chance })
	return out
}

// Helper to add full icon URLs
func (a *App) enrichItemsWithIcons(items []*database.Item) []*database.Item {
	for _, item := range items {
		a.enrichItemIcon(item)
	}
	return items
}

func (a *App) enrichItemIcon(item *database.Item) *database.Item {
	if item == nil {
		return nil
	}
	if item.IconPath != "" && !filepath.IsAbs(item.IconPath) && len(item.IconPath) < 100 {
		item.IconPath = strings.ToLower(item.IconPath)
	}
	return item
}
