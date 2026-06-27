package main

import (
	"fmt"
	"path/filepath"
	"inklab/backend/services"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// ============================================================================
// Icon Fix APIs
// ============================================================================

// FixMissingIconsResult holds the result of icon fixing operation
type FixMissingIconsResult struct {
	TotalMissing int    `json:"totalMissing"`
	Fixed        int    `json:"fixed"`
	Failed       int    `json:"failed"`
	Message      string `json:"message"`
}

// FixSingleItemIcon fixes icon for a single item
func (a *App) FixSingleItemIcon(itemID int) *FixMissingIconsResult {
	fmt.Printf("[API] FixSingleItemIcon called for item %d\n", itemID)

	iconFixService := services.NewIconFixService(a.db.DB(), filepath.Join(a.DataDir, "icons"))

	success, iconName, err := iconFixService.FixSingleItem(a.db.DB(), itemID)
	if err != nil {
		return &FixMissingIconsResult{
			TotalMissing: 1,
			Fixed:        0,
			Failed:       1,
			Message:      err.Error(),
		}
	}

	if success {
		fmt.Printf("[API] Successfully fixed icon for item %d: %s\n", itemID, iconName)
		return &FixMissingIconsResult{
			TotalMissing: 1,
			Fixed:        1,
			Failed:       0,
			Message:      fmt.Sprintf("Successfully updated icon to: %s", iconName),
		}
	}

	return &FixMissingIconsResult{
		TotalMissing: 1,
		Fixed:        0,
		Failed:       1,
		Message:      "Unknown error",
	}
}

// FixMissingIcons manually triggers the icon fix process
func (a *App) FixMissingIcons(iconType string, maxItems int) *FixMissingIconsResult {
	fmt.Printf("[API] FixMissingIcons called with type=%s, maxItems=%d\n", iconType, maxItems)

	iconFixService := services.NewIconFixService(a.db.DB(), filepath.Join(a.DataDir, "icons"))

	var allMissing []services.MissingIconItem
	var err error

	if iconType == "spell" {
		allMissing, err = iconFixService.GetMissingSpellIcons()
	} else {
		allMissing, err = iconFixService.GetMissingIcons()
	}

	if err != nil {
		return &FixMissingIconsResult{
			Message: fmt.Sprintf("Error: %v", err),
		}
	}

	totalMissing := len(allMissing)
	if totalMissing == 0 {
		return &FixMissingIconsResult{
			TotalMissing: 0,
			Fixed:        0,
			Failed:       0,
			Message:      fmt.Sprintf("All %s icons are already fixed!", iconType),
		}
	}

	itemsToFix := allMissing
	if maxItems > 0 && maxItems < totalMissing {
		itemsToFix = allMissing[:maxItems]
	}

	successCount := 0
	failedCount := 0

	fmt.Printf("Fixing %d %s icons...\n", len(itemsToFix), iconType)

	for _, item := range itemsToFix {
		var success bool
		var iconName string

		if iconType == "spell" {
			success, iconName, err = iconFixService.FixSingleSpell(a.db.DB(), item.Entry)
		} else {
			success, iconName, err = iconFixService.FixSingleItem(a.db.DB(), item.Entry)
		}

		if err != nil || !success {
			failedCount++
			fmt.Printf("  Failed %s %d: %v\n", iconType, item.Entry, err)
		} else {
			successCount++
			fmt.Printf("  Fixed %s %d: %s\n", iconType, item.Entry, iconName)
		}
	}

	result := &FixMissingIconsResult{
		TotalMissing: totalMissing,
		Fixed:        successCount,
		Failed:       failedCount,
		Message:      fmt.Sprintf("Fixed %d %s icons, %d failed, %d remaining", successCount, iconType, failedCount, totalMissing-successCount),
	}

	fmt.Printf("[API] FixMissingIcons result: %+v\n", result)
	return result
}

// ============================================================================
// Database Sync APIs (turtlecraft.gg)
// ============================================================================

// GetSyncStats returns statistics about local Turtle WoW data
func (a *App) GetSyncStats() map[string]interface{} {
	fmt.Println("[API] GetSyncStats called")
	return a.syncService.GetSyncStats()
}

// CheckNewItems checks for new items on turtlecraft.gg beyond local max ID
func (a *App) CheckNewItems(maxChecks int, delayMs int) []services.RemoteItem {
	fmt.Printf("[API] CheckNewItems called with maxChecks=%d, delayMs=%d\n", maxChecks, delayMs)

	// Allow 0 for unlimited
	if maxChecks < 0 {
		maxChecks = 100
	}
	if delayMs <= 0 {
		delayMs = 200
	}

	a.syncService.ResetStop()
	items, err := a.syncService.CheckNewItems(maxChecks, delayMs, nil)
	if err != nil {
		fmt.Printf("[API] Error checking new items: %v\n", err)
		return []services.RemoteItem{}
	}

	fmt.Printf("[API] Found %d new items\n", len(items))
	return items
}

// CheckNewQuests checks for new quests on turtlecraft.gg beyond local max ID
func (a *App) CheckNewQuests(maxChecks int, delayMs int) []services.RemoteQuest {
	fmt.Printf("[API] CheckNewQuests called with maxChecks=%d, delayMs=%d\n", maxChecks, delayMs)

	// Allow 0 for unlimited
	if maxChecks < 0 {
		maxChecks = 100
	}
	if delayMs <= 0 {
		delayMs = 200
	}

	a.syncService.ResetStop()
	quests, err := a.syncService.CheckNewQuests(maxChecks, delayMs, nil)
	if err != nil {
		fmt.Printf("[API] Error checking new quests: %v\n", err)
		return []services.RemoteQuest{}
	}

	fmt.Printf("[API] Found %d new quests\n", len(quests))
	return quests
}

// GetMissingAtlasLootItems returns items in AtlasLoot that don't exist in item_template
func (a *App) GetMissingAtlasLootItems(limit int) []services.MissingItem {
	fmt.Printf("[API] GetMissingAtlasLootItems called with limit=%d\n", limit)

	if limit <= 0 {
		limit = 100
	}

	items, err := a.syncService.GetMissingAtlasLootItems(limit)
	if err != nil {
		fmt.Printf("[API] Error getting missing items: %v\n", err)
		return []services.MissingItem{}
	}

	fmt.Printf("[API] Found %d missing items\n", len(items))
	return items
}

// SyncMissingAtlasLoot syncs all missing AtlasLoot items
func (a *App) SyncMissingAtlasLoot(maxItems int, delayMs int) *services.ImportResult {
	fmt.Printf("[API] SyncMissingAtlasLoot called with maxItems=%d\n", maxItems)

	a.syncService.ResetStop()
	result, err := a.syncService.ImportMissingItems(maxItems, delayMs)
	if err != nil {
		return &services.ImportResult{
			Errors: []string{err.Error()},
		}
	}
	return result
}

// SyncSingleItem fetches and imports a single item from turtlecraft.gg
func (a *App) SyncSingleItem(itemID int) *services.SyncItemResult {
	fmt.Printf("[API] SyncSingleItem called for item %d\n", itemID)

	return a.syncService.FetchAndImportItem(itemID)
}

// FullSyncItems re-syncs all Turtle items from turtlecraft.gg
// startFrom: if > 0, resume sync from this ID (for progress recovery)
func (a *App) FullSyncItems(delayMs int, fixIcons bool, startFrom int) string {
	fmt.Printf("[API] FullSyncItems called with delayMs=%d, fixIcons=%v, startFrom=%d\n", delayMs, fixIcons, startFrom)

	if delayMs <= 0 {
		delayMs = 200
	}

	a.syncService.ResetStop()
	iconDir := filepath.Join(a.DataDir, "icons")

	go func() {
		// Create progress callback that emits events to frontend
		progressCb := func(current, total int, itemID int, itemName string) {
			runtime.EventsEmit(a.ctx, "sync:progress", map[string]interface{}{
				"current":  current,
				"total":    total,
				"itemId":   itemID,
				"itemName": itemName,
			})
		}

		result := a.syncService.FullSyncItems(delayMs, fixIcons, iconDir, startFrom, progressCb)
		if len(result.Errors) > 0 && result.Updated == 0 {
			runtime.EventsEmit(a.ctx, "sync:item_full:error", result.Message)
		} else {
			runtime.EventsEmit(a.ctx, "sync:item_full:complete", result.Message)
		}
	}()

	return "Started"
}

// FullSyncSpells resolves all spell descriptions locally from DBC data (the
// delayMs/fixIcons args are kept for API/binding compatibility but unused).
func (a *App) FullSyncSpells(delayMs int, fixIcons bool, startFrom int) string {
	fmt.Printf("[API] FullSyncSpells called with delayMs=%d, fixIcons=%v, startFrom=%d\n", delayMs, fixIcons, startFrom)

	if delayMs <= 0 {
		delayMs = 200
	}

	a.syncService.ResetStop()
	iconDir := filepath.Join(a.DataDir, "icons")

	go func() {
		// Create progress callback that emits events to frontend
		progressCb := func(current, total int, itemID int, itemName string) {
			runtime.EventsEmit(a.ctx, "sync:spells:progress", map[string]interface{}{
				"current":  current,
				"total":    total,
				"itemId":   itemID,
				"itemName": itemName,
			})
		}

		result := a.syncService.FullSyncSpells(delayMs, fixIcons, iconDir, startFrom, progressCb)
		runtime.EventsEmit(a.ctx, "sync:spells_full:complete", result.Message)
	}()

	return "Started"
}

// FullSyncQuests re-syncs all quests
func (a *App) FullSyncQuests(delayMs int, startFrom int) string {
	fmt.Printf("[API] FullSyncQuests called with delayMs=%d, startFrom=%d\n", delayMs, startFrom)

	if delayMs <= 0 {
		delayMs = 200
	}

	a.syncService.ResetStop()

	go func() {
		// Create progress callback that emits events to frontend
		progressCb := func(current, total int, itemID int, itemName string) {
			runtime.EventsEmit(a.ctx, "sync:quests:progress", map[string]interface{}{
				"current":  current,
				"total":    total,
				"itemId":   itemID,
				"itemName": itemName,
			})
		}

		result := a.syncService.FullSyncQuests(delayMs, startFrom, progressCb)
		runtime.EventsEmit(a.ctx, "sync:quests_full:complete", result.Message)
	}()

	return "Started"
}

// NPC model renders are produced on demand when a creature is viewed (see
// RenderNpcModel) and the data/npc_images cache is age-swept, so there's no
// bulk "render everything" pass — it just churned a multi-GB cache.

// FullSyncObjects re-syncs every known game object from the web (octowow.st),
// refreshing spawn points and (for chests) loot — the bulk counterpart to
// SyncObject. Runs in the background, emits progress, and is cancellable via
// StopSync.
func (a *App) FullSyncObjects(delayMs int, startFrom int) string {
	fmt.Printf("[API] FullSyncObjects called with delayMs=%d, startFrom=%d\n", delayMs, startFrom)
	if delayMs <= 0 {
		delayMs = 200
	}
	a.npcService.ResetStop()

	go func() {
		progressCb := func(current, total int, id int) {
			runtime.EventsEmit(a.ctx, "sync:objects:progress", map[string]interface{}{
				"current":  current,
				"total":    total,
				"itemId":   id,
				"itemName": fmt.Sprintf("Object %d", id),
			})
		}
		if err := a.npcService.FullSyncObjects(startFrom, delayMs, progressCb); err != nil {
			runtime.EventsEmit(a.ctx, "sync:objects_full:error", err.Error())
		} else {
			runtime.EventsEmit(a.ctx, "sync:objects_full:complete", "GameObject sync complete")
		}
	}()

	return "Started"
}

func (a *App) SyncSingleSpell(spellID int) *services.SyncSpellResult {
	fmt.Printf("[API] SyncSingleSpell called for spell %d\n", spellID)
	// Resolve the spell's description locally from DBC data (no web scrape).
	return a.syncService.ResolveSpellByID(spellID, "")
}

// StopSync requests all ongoing sync processes to stop
func (a *App) StopSync() string {
	fmt.Println("[API] StopSync called")
	if a.syncService != nil {
		a.syncService.RequestStop()
	}
	if a.npcService != nil {
		a.npcService.RequestStop()
	}
	return "Stop requested"
}
