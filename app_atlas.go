package main

import (
	"fmt"
	"inklab/backend/database"
)

// LegacyBossLoot matches the structure from master branch
type LegacyBossLoot struct {
	BossName string           `json:"bossName"`
	Items    []LegacyLootItem `json:"items"`
}

// LegacyLootItem matches the structure from master branch
type LegacyLootItem struct {
	ItemID     int    `json:"itemId"`
	ItemName   string `json:"itemName"`
	IconName   string `json:"iconName"`
	Quality    int    `json:"quality"`
	DropChance string `json:"dropChance,omitempty"`
	SlotType   string `json:"slotType,omitempty"`
	SpellID    int    `json:"spellId,omitempty"`
}

// GetCategories returns all top-level category names (legacy API)
func (a *App) GetCategories() []string {
	fmt.Println("[API] GetCategories called (AtlasLoot)")
	categories, err := a.atlasLootRepo.GetCategories()
	if err != nil {
		fmt.Printf("[API] Error: %v\n", err)
		return []string{}
	}
	return categories
}

// GetInstances returns modules for a category (legacy API)
func (a *App) GetInstances(categoryName string) []string {
	fmt.Printf("[API] GetInstances called for: %s\n", categoryName)
	modules, err := a.atlasLootRepo.GetModules(categoryName)
	if err != nil {
		fmt.Printf("[API] Error: %v\n", err)
		return []string{}
	}
	return modules
}

// GetTables returns tables/bosses for a module (new API for 3-tier structure)
func (a *App) GetTables(categoryName, moduleName string) []database.AtlasTable {
	fmt.Printf("[API] GetTables called for: %s / %s\n", categoryName, moduleName)
	tables, err := a.atlasLootRepo.GetTables(categoryName, moduleName)
	if err != nil {
		fmt.Printf("[API] Error: %v\n", err)
		return []database.AtlasTable{}
	}
	return tables
}

// GetLoot returns loot for a specific table (legacy API)
func (a *App) GetLoot(categoryName, instanceName, bossKey string) *LegacyBossLoot {
	fmt.Printf("[API] GetLoot called: %s / %s / %s\n", categoryName, instanceName, bossKey)

	// Query atlasloot tables directly
	lootEntries, err := a.atlasLootRepo.GetLootItems(categoryName, instanceName, bossKey)
	if err != nil {
		fmt.Printf("[API] Error getting loot: %v\n", err)
		return &LegacyBossLoot{BossName: bossKey, Items: []LegacyLootItem{}}
	}

	// Figure out boss display name for return (using bossKey as fallback)
	bossName := bossKey

	// Convert to legacy format
	var lootItems []LegacyLootItem
	for _, entry := range lootEntries {
		name := entry.Name

		// Clean up AtlasLoot color/formatting tags
		// =q1=, =ds=, etc.
		if len(name) > 4 && name[0] == '=' {
			// Find second =
			if idx := 4; idx < len(name) { // Simple check, usually =qX= or =ds=
				if name[3] == '=' {
					name = name[4:]
				} else if name[4] == '=' { // =ec1=
					name = name[5:]
				}
			}
		} else if len(name) > 3 && name[0] == '=' && name[3] == '=' {
			name = name[4:]
		}

		lootItems = append(lootItems, LegacyLootItem{
			ItemID:     entry.ItemID,
			ItemName:   name,
			IconName:   entry.IconPath,
			Quality:    entry.Quality,
			DropChance: entry.DropChance,
			SpellID:    entry.SpellID,
		})
	}

	result := &LegacyBossLoot{
		BossName: bossName,
		Items:    lootItems,
	}
	fmt.Printf("[API] GetLoot returning %d items for %s\n", len(lootItems), bossKey)
	return result
}
