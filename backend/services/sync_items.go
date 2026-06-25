package services

import (
	"fmt"
	"io"
	"path/filepath"
	"inklab/backend/database/models"
	"inklab/backend/parsers"
	"strings"
	"sync"
	"time"
)

// GetLocalMaxItemID returns the maximum item entry in local database
func (s *SyncService) GetLocalMaxItemID() (int, error) {
	var maxID int
	err := s.db.QueryRow("SELECT MAX(entry) FROM item_template").Scan(&maxID)
	if err != nil {
		return 0, nil
	}
	return maxID, nil
}

// GetLocalItemCount returns count of items in local database
func (s *SyncService) GetLocalItemCount() (int, error) {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM item_template").Scan(&count)
	return count, err
}

// ItemExistsLocally checks if an item exists in local database
func (s *SyncService) ItemExistsLocally(entry int) bool {
	var count int
	s.db.QueryRow("SELECT COUNT(*) FROM item_template WHERE entry = ?", entry).Scan(&count)
	return count > 0
}

// CheckRemoteItem checks if an item exists on turtlecraft.gg and returns its name
func (s *SyncService) CheckRemoteItem(entry int) (bool, string, error) {
	url := fmt.Sprintf("%s/?item=%d", s.baseURL, entry)

	resp, err := s.httpClient.Get(url)
	if err != nil {
		return false, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return false, "", nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, "", err
	}

	content := string(body)

	exists, name := parsers.ParseItemTitle(content)
	if exists && name == "" {
		name = fmt.Sprintf("Item %d", entry)
	}

	return exists, name, nil
}

// CheckNewItems checks for new items beyond local max ID
func (s *SyncService) CheckNewItems(maxChecks int, delayMs int, progressChan chan<- SyncProgress) ([]RemoteItem, error) {
	localMax, _ := s.GetLocalMaxItemID()
	startID := localMax + 1

	var newItems []RemoteItem
	consecutiveMisses := 0
	maxConsecutiveMisses := 10000 // Stop after 10000 consecutive misses (handles large ID gaps)

	// If maxChecks <= 0, treat as practically unlimited (max int)
	if maxChecks <= 0 {
		maxChecks = 2147483647 // Max Int32
	}

	checked := 0
	for id := startID; checked < maxChecks && consecutiveMisses < maxConsecutiveMisses; id++ {
		// Check for cancellation
		if s.IsStopped() {
			return newItems, nil
		}
		checked++

		// Send progress
		if progressChan != nil {
			progressChan <- SyncProgress{
				Type:     "item",
				Current:  id,
				Total:    startID + maxChecks,
				Found:    len(newItems),
				NewItems: len(newItems),
				Status:   "running",
				Message:  fmt.Sprintf("Checking item %d...", id),
			}
		}

		// Use FetchItemDetails to get full info for import
		item, itemSet, err := s.FetchItemDetails(id)
		if err != nil {
			// If error is "not found", count as miss
			if strings.Contains(err.Error(), "not found") {
				consecutiveMisses++
			} else {
				// Network error or other, treat as miss to keep flow going
				consecutiveMisses++
			}
			continue
		}

		// If name is empty, it's a "shell" item (exists on web but no data).
		// Treat as non-existent/miss to avoid polluting DB.
		if item.Name == "" {
			consecutiveMisses++
			continue
		}

		// Found valid item! Import it.
		consecutiveMisses = 0

		// Insert into item_template
		_, dbErr := s.db.Exec(`
			INSERT OR IGNORE INTO item_template 
			(entry, name, quality, item_level, required_level, class, subclass, inventory_type, display_id, armor, bonding, max_durability, description, sell_price, stat_type1, stat_value1, stat_type2, stat_value2, stat_type3, stat_value3, stat_type4, stat_value4, stat_type5, stat_value5, stat_type6, stat_value6, stat_type7, stat_value7, stat_type8, stat_value8, stat_type9, stat_value9, stat_type10, stat_value10, holy_res, fire_res, nature_res, frost_res, shadow_res, arcane_res, allowable_class, allowable_race, spellid_1, spelltrigger_1, spellid_2, spelltrigger_2, spellid_3, spelltrigger_3, spellid_4, spelltrigger_4, spellid_5, spelltrigger_5, delay, dmg_min1, dmg_max1, dmg_type1, set_id, bag_family, food_type, container_slots)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, item.Entry, item.Name, item.Quality, item.ItemLevel, item.RequiredLevel, item.Class, item.Subclass, item.InventoryType, item.DisplayId, item.Armor, item.Bonding, item.MaxDurability, item.Description, item.SellPrice, item.StatType1, item.StatValue1, item.StatType2, item.StatValue2, item.StatType3, item.StatValue3, item.StatType4, item.StatValue4, item.StatType5, item.StatValue5, item.StatType6, item.StatValue6, item.StatType7, item.StatValue7, item.StatType8, item.StatValue8, item.StatType9, item.StatValue9, item.StatType10, item.StatValue10, item.HolyRes, item.FireRes, item.NatureRes, item.FrostRes, item.ShadowRes, item.ArcaneRes, item.AllowableClass, item.AllowableRace, item.Spellid1, item.Spelltrigger1, item.Spellid2, item.Spelltrigger2, item.Spellid3, item.Spelltrigger3, item.Spellid4, item.Spelltrigger4, item.Spellid5, item.Spelltrigger5, item.Delay, item.DmgMin1, item.DmgMax1, item.DmgType1, item.SetId, item.BagFamily, item.FoodType, item.ContainerSlots)

		if dbErr != nil {
			fmt.Printf("  Error importing item %d: %v\n", id, dbErr)
		} else {
			fmt.Printf("  ✓ Auto-imported: %d - %s\n", id, item.Name)

			// Update Dropped By relations (Batch Sync)
			if len(item.DroppedByNpcs) > 0 {
				for _, npcID := range item.DroppedByNpcs {
					_, _ = s.db.Exec(`INSERT OR IGNORE INTO creature_loot_template (entry, item, ChanceOrQuestChance, groupid, mincountOrRef, maxcount) VALUES (?, ?, 0, 0, 1, 1)`, npcID, item.Entry)
				}
			}

			// Sync item set info
			if itemSet != nil {
				if err := s.UpsertItemSet(itemSet); err != nil {
					fmt.Printf("  Error importing item set %d: %v\n", itemSet.ID, err)
				}
				// Spell descriptions are resolved locally (ResolveSpellByID /
				// FullSyncSpells), not fetched here.
			}
		}

		newItems = append(newItems, RemoteItem{
			Entry: id,
			Name:  item.Name,
			URL:   fmt.Sprintf("%s/?item=%d", s.baseURL, id),
		})

		// Rate limiting
		time.Sleep(time.Duration(delayMs) * time.Millisecond)
	}

	return newItems, nil
}

// FetchItemDetails fetches detailed item info from turtlecraft.gg
func (s *SyncService) FetchItemDetails(itemID int) (*models.ItemTemplateFull, *models.ItemSetEntry, error) {
	url := fmt.Sprintf("%s/?item=%d", s.baseURL, itemID)

	resp, err := s.httpClient.Get(url)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, nil, fmt.Errorf("item not found: %d", itemID)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, err
	}

	content := string(body)

	// Use the new parser
	return parsers.ParseItem(content, itemID)
}

// SyncItemResult represents the result of syncing a single item
type SyncItemResult struct {
	Success bool   `json:"success"`
	ItemID  int    `json:"itemId"`
	Name    string `json:"name,omitempty"`
	Error   string `json:"error,omitempty"`
}

// FetchAndImportItem fetches a single item from turtlecraft.gg and imports it to local database
func (s *SyncService) FetchAndImportItem(itemID int) *SyncItemResult {
	fmt.Printf("[SyncService] FetchAndImportItem called for item %d\n", itemID)

	// Fetch item details from remote
	item, itemSet, err := s.FetchItemDetails(itemID)
	if err != nil {
		return &SyncItemResult{
			Success: false,
			ItemID:  itemID,
			Error:   err.Error(),
		}
	}

	// Check if item has meaningful data
	if item.Name == "" {
		return &SyncItemResult{
			Success: false,
			ItemID:  itemID,
			Error:   "Item exists but has no name (shell item)",
		}
	}

	// Note: icon_path is no longer in item_template. Icons are handling via display_id join.

	iconDir := filepath.Join("data", "icons")

	// Sync item set info. Spell descriptions (item + set bonuses) are resolved
	// locally from DBC data (ResolveSpellByID / FullSyncSpells), not fetched here.
	if itemSet != nil {
		if err := s.UpsertItemSet(itemSet); err != nil {
			fmt.Printf("  Error importing item set %d: %v\n", itemSet.ID, err)
		}
	}

	// Check if item already exists and fetch old data for diff
	var existingCount int
	s.db.QueryRow("SELECT COUNT(*) FROM item_template WHERE entry = ?", itemID).Scan(&existingCount)

	if existingCount > 0 {
		// Fetch existing item for diff logging
		oldItem := s.fetchExistingItemForDiff(itemID)

		// UPDATE existing item - preserves columns not included here (buy_price, buy_count, flags, etc.)
		_, err = s.db.Exec(`
			UPDATE item_template SET
				name = ?, quality = ?, item_level = ?, required_level = ?, class = ?, subclass = ?,
				inventory_type = ?, display_id = ?, armor = ?, bonding = ?, max_durability = ?,
				description = ?, sell_price = CASE WHEN ? > 0 THEN ? ELSE sell_price END, stat_type1 = ?, stat_value1 = ?, stat_type2 = ?,
				stat_value2 = ?, stat_type3 = ?, stat_value3 = ?, stat_type4 = ?, stat_value4 = ?,
				stat_type5 = ?, stat_value5 = ?, stat_type6 = ?, stat_value6 = ?, stat_type7 = ?,
				stat_value7 = ?, stat_type8 = ?, stat_value8 = ?, stat_type9 = ?, stat_value9 = ?,
				stat_type10 = ?, stat_value10 = ?, holy_res = ?, fire_res = ?, nature_res = ?,
				frost_res = ?, shadow_res = ?, arcane_res = ?, allowable_class = ?, allowable_race = ?,
				spellid_1 = ?, spelltrigger_1 = ?, spellid_2 = ?, spelltrigger_2 = ?, spellid_3 = ?,
				spelltrigger_3 = ?, spellid_4 = ?, spelltrigger_4 = ?, spellid_5 = ?, spelltrigger_5 = ?,
				delay = ?, dmg_min1 = ?, dmg_max1 = ?, dmg_type1 = ?, set_id = ?,
				bag_family = ?, food_type = ?
			WHERE entry = ?
		`, item.Name, item.Quality, item.ItemLevel, item.RequiredLevel, item.Class, item.Subclass,
			item.InventoryType, item.DisplayId, item.Armor, item.Bonding, item.MaxDurability,
			item.Description, item.SellPrice, item.SellPrice, item.StatType1, item.StatValue1, item.StatType2,
			item.StatValue2, item.StatType3, item.StatValue3, item.StatType4, item.StatValue4,
			item.StatType5, item.StatValue5, item.StatType6, item.StatValue6, item.StatType7,
			item.StatValue7, item.StatType8, item.StatValue8, item.StatType9, item.StatValue9,
			item.StatType10, item.StatValue10, item.HolyRes, item.FireRes, item.NatureRes,
			item.FrostRes, item.ShadowRes, item.ArcaneRes, item.AllowableClass, item.AllowableRace,
			item.Spellid1, item.Spelltrigger1, item.Spellid2, item.Spelltrigger2, item.Spellid3,
			item.Spelltrigger3, item.Spellid4, item.Spelltrigger4, item.Spellid5, item.Spelltrigger5,
			item.Delay, item.DmgMin1, item.DmgMax1, item.DmgType1, item.SetId,
			item.BagFamily, item.FoodType, itemID)

		// Log diff after update
		if oldItem != nil {
			s.logItemDiff(itemID, oldItem, item)
		}
	} else {
		// INSERT new item
		fmt.Printf("[Sync] + NEW ITEM %d: %s\n", itemID, item.Name)
		_, err = s.db.Exec(`
			INSERT INTO item_template 
			(entry, name, quality, item_level, required_level, class, subclass, inventory_type, display_id, armor, bonding, max_durability, description, sell_price, stat_type1, stat_value1, stat_type2, stat_value2, stat_type3, stat_value3, stat_type4, stat_value4, stat_type5, stat_value5, stat_type6, stat_value6, stat_type7, stat_value7, stat_type8, stat_value8, stat_type9, stat_value9, stat_type10, stat_value10, holy_res, fire_res, nature_res, frost_res, shadow_res, arcane_res, allowable_class, allowable_race, spellid_1, spelltrigger_1, spellid_2, spelltrigger_2, spellid_3, spelltrigger_3, spellid_4, spelltrigger_4, spellid_5, spelltrigger_5, delay, dmg_min1, dmg_max1, dmg_type1, set_id, bag_family, food_type, container_slots)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, item.Entry, item.Name, item.Quality, item.ItemLevel, item.RequiredLevel, item.Class, item.Subclass, item.InventoryType, item.DisplayId, item.Armor, item.Bonding, item.MaxDurability, item.Description, item.SellPrice, item.StatType1, item.StatValue1, item.StatType2, item.StatValue2, item.StatType3, item.StatValue3, item.StatType4, item.StatValue4, item.StatType5, item.StatValue5, item.StatType6, item.StatValue6, item.StatType7, item.StatValue7, item.StatType8, item.StatValue8, item.StatType9, item.StatValue9, item.StatType10, item.StatValue10, item.HolyRes, item.FireRes, item.NatureRes, item.FrostRes, item.ShadowRes, item.ArcaneRes, item.AllowableClass, item.AllowableRace, item.Spellid1, item.Spelltrigger1, item.Spellid2, item.Spelltrigger2, item.Spellid3, item.Spelltrigger3, item.Spellid4, item.Spelltrigger4, item.Spellid5, item.Spelltrigger5, item.Delay, item.DmgMin1, item.DmgMax1, item.DmgType1, item.SetId, item.BagFamily, item.FoodType, item.ContainerSlots)
	}

	if err != nil {
		return &SyncItemResult{
			Success: false,
			ItemID:  itemID,
			Error:   fmt.Sprintf("Database error: %v", err),
		}
	}

	// Update Dropped By relations (Single Sync)
	if len(item.DroppedByNpcs) > 0 {
		fmt.Printf("[SyncService] Updating %d Dropped By records for item %d\n", len(item.DroppedByNpcs), itemID)
		for _, npcID := range item.DroppedByNpcs {
			_, _ = s.db.Exec(`INSERT OR IGNORE INTO creature_loot_template (entry, item, ChanceOrQuestChance, groupid, mincountOrRef, maxcount) VALUES (?, ?, 0, 0, 1, 1)`, npcID, itemID)
		}
	}

	// Store the "sold by" vendor list scraped from the item page.
	s.storeItemVendors(itemID, item.Vendors)

	// Auto-fix icon if needed
	iconFixService := NewIconFixService(s.db, iconDir)
	_, _, _ = iconFixService.FixSingleItem(s.db, itemID)

	return &SyncItemResult{
		Success: true,
		ItemID:  itemID,
		Name:    item.Name,
	}
}

// storeItemVendors replaces an item's "sold by" rows with the freshly scraped
// list (authoritative refresh — clears stale vendors).
func (s *SyncService) storeItemVendors(itemID int, vendors []models.ItemVendor) {
	s.db.Exec("DELETE FROM item_vendor WHERE item_entry = ?", itemID)
	for _, v := range vendors {
		s.db.Exec(
			`INSERT OR REPLACE INTO item_vendor (item_entry, npc_entry, npc_name, level_min, level_max, cost, stock)
			 VALUES (?, ?, ?, ?, ?, ?, ?)`,
			itemID, v.Entry, v.Name, v.LevelMin, v.LevelMax, v.Cost, v.Stock)
	}
}

// fetchExistingItemForDiff fetches an existing item from DB for diff comparison
func (s *SyncService) fetchExistingItemForDiff(itemID int) *models.ItemTemplateFull {
	row := s.db.QueryRow(`
		SELECT name, quality, item_level, required_level, class, subclass, inventory_type, display_id,
		       armor, bonding, max_durability, description, sell_price,
		       stat_type1, stat_value1, stat_type2, stat_value2, stat_type3, stat_value3,
		       stat_type4, stat_value4, stat_type5, stat_value5, stat_type6, stat_value6,
		       stat_type7, stat_value7, stat_type8, stat_value8, stat_type9, stat_value9,
		       stat_type10, stat_value10,
		       holy_res, fire_res, nature_res, frost_res, shadow_res, arcane_res,
		       allowable_class, allowable_race,
		       spellid_1, spelltrigger_1, spellid_2, spelltrigger_2, spellid_3, spelltrigger_3,
		       spellid_4, spelltrigger_4, spellid_5, spelltrigger_5,
		       delay, dmg_min1, dmg_max1, dmg_type1, set_id, bag_family, food_type
		FROM item_template WHERE entry = ?
	`, itemID)

	item := &models.ItemTemplateFull{Entry: itemID}
	var desc, name string
	err := row.Scan(
		&name, &item.Quality, &item.ItemLevel, &item.RequiredLevel, &item.Class, &item.Subclass,
		&item.InventoryType, &item.DisplayId, &item.Armor, &item.Bonding, &item.MaxDurability,
		&desc, &item.SellPrice,
		&item.StatType1, &item.StatValue1, &item.StatType2, &item.StatValue2,
		&item.StatType3, &item.StatValue3, &item.StatType4, &item.StatValue4,
		&item.StatType5, &item.StatValue5, &item.StatType6, &item.StatValue6,
		&item.StatType7, &item.StatValue7, &item.StatType8, &item.StatValue8,
		&item.StatType9, &item.StatValue9, &item.StatType10, &item.StatValue10,
		&item.HolyRes, &item.FireRes, &item.NatureRes, &item.FrostRes, &item.ShadowRes, &item.ArcaneRes,
		&item.AllowableClass, &item.AllowableRace,
		&item.Spellid1, &item.Spelltrigger1, &item.Spellid2, &item.Spelltrigger2,
		&item.Spellid3, &item.Spelltrigger3, &item.Spellid4, &item.Spelltrigger4,
		&item.Spellid5, &item.Spelltrigger5,
		&item.Delay, &item.DmgMin1, &item.DmgMax1, &item.DmgType1, &item.SetId, &item.BagFamily, &item.FoodType,
	)
	if err != nil {
		return nil
	}
	item.Name = name
	item.Description = desc
	return item
}

// logItemDiff logs a git-style diff of changes between old and new item
func (s *SyncService) logItemDiff(itemID int, old, new *models.ItemTemplateFull) {
	var diffs []string

	// Helper to add diff line
	addDiff := func(field string, oldVal, newVal interface{}) {
		oldStr := fmt.Sprintf("%v", oldVal)
		newStr := fmt.Sprintf("%v", newVal)
		if oldStr != newStr {
			diffs = append(diffs, fmt.Sprintf("  - %s: %v", field, oldVal))
			diffs = append(diffs, fmt.Sprintf("  + %s: %v", field, newVal))
		}
	}

	// Compare fields
	addDiff("name", old.Name, new.Name)
	addDiff("quality", old.Quality, new.Quality)
	addDiff("item_level", old.ItemLevel, new.ItemLevel)
	addDiff("required_level", old.RequiredLevel, new.RequiredLevel)
	addDiff("class", old.Class, new.Class)
	addDiff("subclass", old.Subclass, new.Subclass)
	addDiff("inventory_type", old.InventoryType, new.InventoryType)
	addDiff("display_id", old.DisplayId, new.DisplayId)
	addDiff("armor", old.Armor, new.Armor)
	addDiff("bonding", old.Bonding, new.Bonding)
	addDiff("max_durability", old.MaxDurability, new.MaxDurability)
	addDiff("sell_price", old.SellPrice, new.SellPrice)
	addDiff("stat_type1", old.StatType1, new.StatType1)
	addDiff("stat_value1", old.StatValue1, new.StatValue1)
	addDiff("stat_type2", old.StatType2, new.StatType2)
	addDiff("stat_value2", old.StatValue2, new.StatValue2)
	addDiff("stat_type3", old.StatType3, new.StatType3)
	addDiff("stat_value3", old.StatValue3, new.StatValue3)
	addDiff("stat_type4", old.StatType4, new.StatType4)
	addDiff("stat_value4", old.StatValue4, new.StatValue4)
	addDiff("stat_type5", old.StatType5, new.StatType5)
	addDiff("stat_value5", old.StatValue5, new.StatValue5)
	addDiff("stat_type6", old.StatType6, new.StatType6)
	addDiff("stat_value6", old.StatValue6, new.StatValue6)
	addDiff("stat_type7", old.StatType7, new.StatType7)
	addDiff("stat_value7", old.StatValue7, new.StatValue7)
	addDiff("stat_type8", old.StatType8, new.StatType8)
	addDiff("stat_value8", old.StatValue8, new.StatValue8)
	addDiff("stat_type9", old.StatType9, new.StatType9)
	addDiff("stat_value9", old.StatValue9, new.StatValue9)
	addDiff("stat_type10", old.StatType10, new.StatType10)
	addDiff("stat_value10", old.StatValue10, new.StatValue10)
	addDiff("holy_res", old.HolyRes, new.HolyRes)
	addDiff("fire_res", old.FireRes, new.FireRes)
	addDiff("nature_res", old.NatureRes, new.NatureRes)
	addDiff("frost_res", old.FrostRes, new.FrostRes)
	addDiff("shadow_res", old.ShadowRes, new.ShadowRes)
	addDiff("arcane_res", old.ArcaneRes, new.ArcaneRes)
	addDiff("allowable_class", old.AllowableClass, new.AllowableClass)
	addDiff("allowable_race", old.AllowableRace, new.AllowableRace)
	addDiff("spellid_1", old.Spellid1, new.Spellid1)
	addDiff("spelltrigger_1", old.Spelltrigger1, new.Spelltrigger1)
	addDiff("spellid_2", old.Spellid2, new.Spellid2)
	addDiff("spelltrigger_2", old.Spelltrigger2, new.Spelltrigger2)
	addDiff("spellid_3", old.Spellid3, new.Spellid3)
	addDiff("spelltrigger_3", old.Spelltrigger3, new.Spelltrigger3)
	addDiff("spellid_4", old.Spellid4, new.Spellid4)
	addDiff("spelltrigger_4", old.Spelltrigger4, new.Spelltrigger4)
	addDiff("spellid_5", old.Spellid5, new.Spellid5)
	addDiff("spelltrigger_5", old.Spelltrigger5, new.Spelltrigger5)
	addDiff("delay", old.Delay, new.Delay)
	addDiff("dmg_min1", old.DmgMin1, new.DmgMin1)
	addDiff("dmg_max1", old.DmgMax1, new.DmgMax1)
	addDiff("dmg_type1", old.DmgType1, new.DmgType1)
	addDiff("set_id", old.SetId, new.SetId)
	addDiff("bag_family", old.BagFamily, new.BagFamily)
	addDiff("food_type", old.FoodType, new.FoodType)
	if old.Description != new.Description {
		oldDesc := old.Description
		newDesc := new.Description
		if len(oldDesc) > 50 {
			oldDesc = oldDesc[:50] + "..."
		}
		if len(newDesc) > 50 {
			newDesc = newDesc[:50] + "..."
		}
		diffs = append(diffs, fmt.Sprintf("  - description: %s", oldDesc))
		diffs = append(diffs, fmt.Sprintf("  + description: %s", newDesc))
	}

	// Print diff
	if len(diffs) > 0 {
		fmt.Printf("[Sync] diff item_template (entry=%d) \"%s\":\n", itemID, new.Name)
		for _, line := range diffs {
			fmt.Println(line)
		}
	} else {
		fmt.Printf("[Sync] ○ item %d \"%s\" - no changes\n", itemID, new.Name)
	}
}

// FullSyncItems re-syncs all items from turtlecraft.gg
// startFrom: if > 0, skip items with ID < startFrom (for resume)
// progressCb: callback for progress updates (can be nil)
func (s *SyncService) FullSyncItems(delayMs int, fixIcons bool, iconDir string, startFrom int, progressCb ProgressCallback) *FullSyncResult {
	// User requested faster sync ("unnecessary delay").
	// We use a worker pool to parallelize requests.

	// Get all item IDs ordered by entry for consistent resume
	rows, err := s.db.Query("SELECT entry FROM item_template ORDER BY entry ASC")
	if err != nil {
		return &FullSyncResult{
			Message: fmt.Sprintf("Error querying items: %v", err),
			Errors:  []string{err.Error()},
		}
	}
	defer rows.Close()

	var itemIDs []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err == nil {
			itemIDs = append(itemIDs, id)
		}
	}

	// Filter items based on startFrom for resume
	var filteredIDs []int
	for _, id := range itemIDs {
		if startFrom <= 0 || id >= startFrom {
			filteredIDs = append(filteredIDs, id)
		}
	}

	result := &FullSyncResult{
		TotalItems:  len(itemIDs),
		Errors:      []string{},
		StartFromID: startFrom,
	}

	skipped := len(itemIDs) - len(filteredIDs)
	if skipped > 0 {
		fmt.Printf("[FullSync] Resuming from ID %d, skipped %d items already synced\n", startFrom, skipped)
	}

	// Worker Pool Setup
	numWorkers := 10 // Concurrent workers
	fmt.Printf("[FullSync] Starting parallel sync of %d items with %d workers...\n", len(filteredIDs), numWorkers)

	jobs := make(chan int, len(filteredIDs))
	var wg sync.WaitGroup
	var mu sync.Mutex // Protects result, progress, console output

	processedCount := 0
	totalFiltered := len(filteredIDs)

	iconFixService := NewIconFixService(s.db, iconDir)

	// Worker function
	worker := func() {
		defer wg.Done()
		for itemID := range jobs {
			// Check if stop requested
			if s.IsStopped() {
				// Drain remaining jobs if necessary or just return
				// For pool workers, we should check it frequently
				return
			}

			// Fetch item
			item, itemSet, err := s.FetchItemDetails(itemID)

			var itemName string
			if item != nil {
				itemName = item.Name
			}

			var success bool

			if err != nil {
				mu.Lock()
				result.Failed++
				if len(result.Errors) < 10 {
					result.Errors = append(result.Errors, fmt.Sprintf("Item %d: %v", itemID, err))
				}
				mu.Unlock()
			} else if item.Name == "" {
				mu.Lock()
				result.Failed++
				mu.Unlock()
			} else {
				// Update item (thread-safe via sql.DB)
				_, err = s.db.Exec(`
					UPDATE item_template SET
						name = COALESCE(NULLIF(?, ''), name),
						quality = COALESCE(NULLIF(?, 0), quality),
						item_level = COALESCE(NULLIF(?, 0), item_level),
						required_level = CASE WHEN ? > 0 THEN ? ELSE required_level END,
						class = COALESCE(NULLIF(?, 0), class),
						subclass = CASE WHEN ? > 0 OR class = 0 THEN ? ELSE subclass END,
						inventory_type = COALESCE(NULLIF(?, 0), inventory_type),
						display_id = COALESCE(NULLIF(?, 0), display_id),
						armor = CASE WHEN ? > 0 THEN ? ELSE armor END,
						bonding = CASE WHEN ? > 0 THEN ? ELSE bonding END,
						max_durability = CASE WHEN ? > 0 THEN ? ELSE max_durability END,
						description = COALESCE(NULLIF(?, ''), description),
						sell_price = CASE WHEN ? > 0 THEN ? ELSE sell_price END,
						stat_type1 = CASE WHEN ? > 0 THEN ? ELSE stat_type1 END,
						stat_value1 = CASE WHEN ? != 0 THEN ? ELSE stat_value1 END,
						stat_type2 = CASE WHEN ? > 0 THEN ? ELSE stat_type2 END,
						stat_value2 = CASE WHEN ? != 0 THEN ? ELSE stat_value2 END,
						stat_type3 = CASE WHEN ? > 0 THEN ? ELSE stat_type3 END,
						stat_value3 = CASE WHEN ? != 0 THEN ? ELSE stat_value3 END,
						stat_type4 = CASE WHEN ? > 0 THEN ? ELSE stat_type4 END,
						stat_value4 = CASE WHEN ? != 0 THEN ? ELSE stat_value4 END,
						stat_type5 = CASE WHEN ? > 0 THEN ? ELSE stat_type5 END,
						stat_value5 = CASE WHEN ? != 0 THEN ? ELSE stat_value5 END,
						stat_type6 = CASE WHEN ? > 0 THEN ? ELSE stat_type6 END,
						stat_value6 = CASE WHEN ? != 0 THEN ? ELSE stat_value6 END,
						stat_type7 = CASE WHEN ? > 0 THEN ? ELSE stat_type7 END,
						stat_value7 = CASE WHEN ? != 0 THEN ? ELSE stat_value7 END,
						stat_type8 = CASE WHEN ? > 0 THEN ? ELSE stat_type8 END,
						stat_value8 = CASE WHEN ? != 0 THEN ? ELSE stat_value8 END,
						stat_type9 = CASE WHEN ? > 0 THEN ? ELSE stat_type9 END,
						stat_value9 = CASE WHEN ? != 0 THEN ? ELSE stat_value9 END,
						stat_type10 = CASE WHEN ? > 0 THEN ? ELSE stat_type10 END,
						stat_value10 = CASE WHEN ? != 0 THEN ? ELSE stat_value10 END,
						holy_res = CASE WHEN ? > 0 THEN ? ELSE holy_res END,
						fire_res = CASE WHEN ? > 0 THEN ? ELSE fire_res END,
						nature_res = CASE WHEN ? > 0 THEN ? ELSE nature_res END,
						frost_res = CASE WHEN ? > 0 THEN ? ELSE frost_res END,
						shadow_res = CASE WHEN ? > 0 THEN ? ELSE shadow_res END,
						arcane_res = CASE WHEN ? > 0 THEN ? ELSE arcane_res END,
						allowable_class = CASE WHEN ? != 0 THEN ? ELSE allowable_class END,
						allowable_race = CASE WHEN ? != 0 THEN ? ELSE allowable_race END,
						spellid_1 = CASE WHEN ? > 0 THEN ? ELSE spellid_1 END,
						spelltrigger_1 = CASE WHEN ? > 0 THEN ? ELSE spelltrigger_1 END,
						spellid_2 = CASE WHEN ? > 0 THEN ? ELSE spellid_2 END,
						spelltrigger_2 = CASE WHEN ? > 0 THEN ? ELSE spelltrigger_2 END,
						spellid_3 = CASE WHEN ? > 0 THEN ? ELSE spellid_3 END,
						spelltrigger_3 = CASE WHEN ? > 0 THEN ? ELSE spelltrigger_3 END,
						spellid_4 = CASE WHEN ? > 0 THEN ? ELSE spellid_4 END,
						spelltrigger_4 = CASE WHEN ? > 0 THEN ? ELSE spelltrigger_4 END,
						spellid_5 = CASE WHEN ? > 0 THEN ? ELSE spellid_5 END,
						spelltrigger_5 = CASE WHEN ? > 0 THEN ? ELSE spelltrigger_5 END,
						delay = CASE WHEN ? > 0 THEN ? ELSE delay END,
						dmg_min1 = CASE WHEN ? > 0 THEN ? ELSE dmg_min1 END,
						dmg_max1 = CASE WHEN ? > 0 THEN ? ELSE dmg_max1 END,
						dmg_type1 = CASE WHEN ? > 0 THEN ? ELSE dmg_type1 END,
						set_id = CASE WHEN ? > 0 THEN ? ELSE set_id END,
						bag_family = CASE WHEN ? > 0 THEN ? ELSE bag_family END,
						food_type = CASE WHEN ? > 0 THEN ? ELSE food_type END,
						container_slots = CASE WHEN ? > 0 THEN ? ELSE container_slots END
					WHERE entry = ?
				`,
					item.Name,
					item.Quality,
					item.ItemLevel,
					item.RequiredLevel, item.RequiredLevel,
					item.Class,
					item.Subclass, item.Subclass,
					item.InventoryType,
					item.DisplayId,
					item.Armor, item.Armor,
					item.Bonding, item.Bonding,
					item.MaxDurability, item.MaxDurability,
					item.Description,
					item.SellPrice, item.SellPrice,
					item.StatType1, item.StatType1, item.StatValue1, item.StatValue1,
					item.StatType2, item.StatType2, item.StatValue2, item.StatValue2,
					item.StatType3, item.StatType3, item.StatValue3, item.StatValue3,
					item.StatType4, item.StatType4, item.StatValue4, item.StatValue4,
					item.StatType5, item.StatType5, item.StatValue5, item.StatValue5,
					item.StatType6, item.StatType6, item.StatValue6, item.StatValue6,
					item.StatType7, item.StatType7, item.StatValue7, item.StatValue7,
					item.StatType8, item.StatType8, item.StatValue8, item.StatValue8,
					item.StatType9, item.StatType9, item.StatValue9, item.StatValue9,
					item.StatType10, item.StatType10, item.StatValue10, item.StatValue10,
					item.HolyRes, item.HolyRes,
					item.FireRes, item.FireRes,
					item.NatureRes, item.NatureRes,
					item.FrostRes, item.FrostRes,
					item.ShadowRes, item.ShadowRes,
					item.ArcaneRes, item.ArcaneRes,
					item.AllowableClass, item.AllowableClass,
					item.AllowableRace, item.AllowableRace,
					item.Spellid1, item.Spellid1, item.Spelltrigger1, item.Spelltrigger1,
					item.Spellid2, item.Spellid2, item.Spelltrigger2, item.Spelltrigger2,
					item.Spellid3, item.Spellid3, item.Spelltrigger3, item.Spelltrigger3,
					item.Spellid4, item.Spellid4, item.Spelltrigger4, item.Spelltrigger4,
					item.Spellid5, item.Spellid5, item.Spelltrigger5, item.Spelltrigger5,
					item.Delay, item.Delay,
					item.DmgMin1, item.DmgMin1,
					item.DmgMax1, item.DmgMax1,
					item.DmgType1, item.DmgType1,
					item.SetId, item.SetId,
					item.BagFamily, item.BagFamily,
					item.FoodType, item.FoodType,
					item.ContainerSlots, item.ContainerSlots,
					itemID)

				if err != nil {
					mu.Lock()
					result.Failed++
					if len(result.Errors) < 10 {
						result.Errors = append(result.Errors, fmt.Sprintf("Item %d update: %v", itemID, err))
					}
					mu.Unlock()
				} else {
					success = true
				}
			}

			if success {
				mu.Lock()
				result.Updated++
				result.LastSyncedID = itemID // Roughly tracks progress
				if fixIcons {
					// Count fixed icons if we really want to, but it requires another call
				}
				mu.Unlock()

				// Sync Set + Set Bonuses. Spell descriptions are resolved locally
				// from DBC data (FullSyncSpells), not fetched per item here.
				if itemSet != nil {
					_ = s.UpsertItemSet(itemSet)
				}

				// Store the "sold by" vendor list scraped from the item page.
				s.storeItemVendors(itemID, item.Vendors)

				// Fix icon for item
				if fixIcons {
					if success, _, _ := iconFixService.FixSingleItem(s.db, itemID); success {
						mu.Lock()
						result.IconsFixed++
						mu.Unlock()
					}
				}
			}

			// Reporting and Progress
			mu.Lock()
			processedCount++
			if progressCb != nil {
				progressCb(processedCount, totalFiltered, itemID, itemName)
			}
			if processedCount%50 == 0 {
				fmt.Printf("[FullSync] Progress: %d/%d items\n", processedCount, totalFiltered)
			}
			mu.Unlock()

			// Delay (if requested)
			if delayMs > 0 {
				time.Sleep(time.Duration(delayMs) * time.Millisecond)
			}
		}
	}

	// Launch workers
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go worker()
	}

	// Feed jobs
	for _, id := range filteredIDs {
		jobs <- id
	}
	close(jobs)

	// Wait for completion
	wg.Wait()

	result.Message = fmt.Sprintf("Full sync complete: %d updated, %d failed, %d icons fixed",
		result.Updated, result.Failed, result.IconsFixed)
	fmt.Printf("[FullSync] %s\n", result.Message)

	return result
}
