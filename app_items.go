package main

import (
	"fmt"
	"path/filepath"
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
	}

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
