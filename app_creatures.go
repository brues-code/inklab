package main

import (
	"fmt"
	"inklab/backend/database"
	"inklab/backend/services"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// GetCreatureTypes returns all creature types with counts
func (a *App) GetCreatureTypes() []*database.CreatureType {
	fmt.Println("[API] GetCreatureTypes called")
	types, err := a.creatureRepo.GetCreatureTypes()
	if err != nil {
		fmt.Printf("[API] Error getting creature types: %v\n", err)
		return []*database.CreatureType{}
	}
	fmt.Printf("[API] GetCreatureTypes returning %d types\n", len(types))
	return types
}

// BrowseCreaturesByType returns creatures filtered by type
func (a *App) BrowseCreaturesByType(creatureType int, nameFilter string) []*database.Creature {
	fmt.Printf("[API] BrowseCreaturesByType called type=%d filter='%s'\n", creatureType, nameFilter)
	creatures, _, err := a.creatureRepo.GetCreaturesByType(creatureType, nameFilter, 9999, 0)
	if err != nil {
		fmt.Printf("[API] Error browsing creatures: %v\n", err)
		return []*database.Creature{}
	}
	fmt.Printf("[API] BrowseCreaturesByType returning %d creatures\n", len(creatures))
	return creatures
}

// CreaturePageResult is the result of paginated creature query
type CreaturePageResult struct {
	Creatures []*database.Creature `json:"creatures"`
	Total     int                  `json:"total"`
	HasMore   bool                 `json:"hasMore"`
}

// BrowseCreaturesByTypePaged returns creatures with pagination support
func (a *App) BrowseCreaturesByTypePaged(creatureType int, nameFilter string, limit, offset int) *CreaturePageResult {
	fmt.Printf("[API] BrowseCreaturesByTypePaged: type=%d filter='%s' limit=%d offset=%d\n", creatureType, nameFilter, limit, offset)

	creatures, total, err := a.creatureRepo.GetCreaturesByType(creatureType, nameFilter, limit, offset)
	if err != nil {
		fmt.Printf("[API] Error browsing creatures: %v\n", err)
		return &CreaturePageResult{
			Creatures: []*database.Creature{},
			Total:     0,
			HasMore:   false,
		}
	}

	return &CreaturePageResult{
		Creatures: creatures,
		Total:     total,
		HasMore:   (offset + len(creatures)) < total,
	}
}

// SearchCreatures searches for creatures by name
func (a *App) SearchCreatures(query string) []*database.Creature {
	creatures, err := a.creatureRepo.SearchCreatures(query, 50)
	if err != nil {
		fmt.Printf("Error searching creatures: %v\n", err)
		return []*database.Creature{}
	}
	return creatures
}

// GetCreatureDetail returns full details for a creature
func (a *App) GetCreatureDetail(entry int) (*database.CreatureDetail, error) {
	c, err := a.creatureRepo.GetCreatureDetail(entry)
	if err != nil {
		fmt.Printf("Error getting creature detail [%d]: %v\n", entry, err)
		return nil, err
	}
	return c, nil
}

// GetCreatureLoot returns the loot for a creature
func (a *App) GetCreatureLoot(entry int) []*database.LootItem {
	loot, err := a.lootRepo.GetCreatureLoot(entry)
	if err != nil {
		fmt.Printf("Error getting creature loot: %v\n", err)
		return []*database.LootItem{}
	}
	return loot
}

// GetNpcDetails returns full details for an NPC (Scraped + DB)
func (a *App) GetNpcDetails(entry int) *services.NpcFullDetails {
	fmt.Printf("[API] GetNpcDetails called for %d\n", entry)
	details, err := a.npcService.GetNpcDetails(entry)
	if err != nil {
		fmt.Printf("Error getting NPC details: %v\n", err)
		return nil
	}
	return details
}

// SyncNpcData forces a re-sync of NPC data
func (a *App) SyncNpcData(entry int) *services.NpcFullDetails {
	fmt.Printf("[API] SyncNpcData called for %d\n", entry)
	err := a.npcService.SyncNpcData(entry)
	if err != nil {
		fmt.Printf("Error syncing NPC data: %v\n", err)
	}
	// Return fresh details regardless of sync error (might be partial)
	details, _ := a.npcService.GetNpcDetails(entry)
	return details
}

// RefreshNpcImages re-fetches only the model/map images (and visual metadata)
// for an NPC, without re-syncing creature_template stats from MySQL.
func (a *App) RefreshNpcImages(entry int) *services.NpcFullDetails {
	fmt.Printf("[API] RefreshNpcImages called for %d\n", entry)
	if err := a.npcService.RefreshNpcImages(entry); err != nil {
		fmt.Printf("Error refreshing NPC images: %v\n", err)
	}
	details, _ := a.npcService.GetNpcDetails(entry)
	return details
}

// FullSyncNpcs re-syncs all NPC data (Web + MySQL) starting from a specific ID
func (a *App) FullSyncNpcs(startFrom int, delayMs int) string {
	fmt.Printf("[API] FullSyncNpcs called with startFrom=%d, delayMs=%d\n", startFrom, delayMs)

	if delayMs <= 0 {
		delayMs = 200 // Default delay
	}

	a.npcService.ResetStop()

	go func() {
		progressCb := func(current, total int, id int) {
			runtime.EventsEmit(a.ctx, "sync:npc_full:progress", map[string]interface{}{
				"current": current,
				"total":   total,
				"id":      id,
			})
		}

		err := a.npcService.FullSyncNpcs(startFrom, delayMs, progressCb)
		if err != nil {
			fmt.Printf("Error syncing all NPCs: %v\n", err)
			runtime.EventsEmit(a.ctx, "sync:npc_full:error", err.Error())
		} else {
			runtime.EventsEmit(a.ctx, "sync:npc_full:complete", "Full NPC sync complete")
		}
	}()

	return "Started"
}
