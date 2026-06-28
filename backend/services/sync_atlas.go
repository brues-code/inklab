package services

import (
	"fmt"
	"sync"
	"time"
)

// GetMissingAtlasLootItemCount returns count of items in AtlasLoot that don't exist in item_template
func (s *SyncService) GetMissingAtlasLootItemCount() (int, error) {
	var count int
	err := s.db.QueryRow(`
		SELECT COUNT(DISTINCT ai.item_id) 
		FROM atlasloot_items ai 
		LEFT JOIN item_template i ON ai.item_id = i.entry 
		WHERE i.entry IS NULL
	`).Scan(&count)
	return count, err
}

// MissingItem represents an item in AtlasLoot but not in item_template
type MissingItem struct {
	ItemID    int    `json:"itemId"`
	TableKey  string `json:"tableKey"`
	TableName string `json:"tableName"`
}

// GetMissingAtlasLootItems returns items in AtlasLoot that don't exist in item_template
func (s *SyncService) GetMissingAtlasLootItems(limit int) ([]MissingItem, error) {
	if limit <= 0 {
		limit = 100
	}

	rows, err := s.db.Query(`
		SELECT DISTINCT ai.item_id, t.table_key, t.display_name
		FROM atlasloot_items ai 
		JOIN atlasloot_tables t ON ai.table_id = t.id
		LEFT JOIN item_template i ON ai.item_id = i.entry 
		WHERE i.entry IS NULL
		ORDER BY ai.item_id
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []MissingItem
	for rows.Next() {
		var item MissingItem
		if err := rows.Scan(&item.ItemID, &item.TableKey, &item.TableName); err != nil {
			continue
		}
		items = append(items, item)
	}
	return items, nil
}

// ImportResult represents the result of importing items
type ImportResult struct {
	Checked  int      `json:"checked"`
	Imported int      `json:"imported"`
	Failed   int      `json:"failed"`
	Items    []string `json:"items"`
	Errors   []string `json:"errors"`
}

// ImportMissingItems fetches and imports missing AtlasLoot items
func (s *SyncService) ImportMissingItems(maxItems int, delayMs int) (*ImportResult, error) {
	if maxItems <= 0 {
		maxItems = 2147483647 // Max Int32 (Unlimited)
	}

	missing, err := s.GetMissingAtlasLootItems(maxItems)
	if err != nil {
		return nil, err
	}

	result := &ImportResult{
		Checked: len(missing),
		Items:   []string{},
		Errors:  []string{},
	}

	// Worker Pool Setup
	numWorkers := scrapeWorkers()
	jobs := make(chan MissingItem, len(missing))
	var wg sync.WaitGroup
	var mu sync.Mutex

	fmt.Printf("[ImportMissing] Starting parallel import of %d items with %d workers\n", len(missing), numWorkers)

	worker := func() {
		defer wg.Done()
		for m := range jobs {
			// Check for cancellation
			if s.IsStopped() {
				return
			}

			// Reuse the main sync logic
			syncRes := s.FetchAndImportItem(m.ItemID)

			mu.Lock()
			if syncRes.Success {
				result.Imported++
				result.Items = append(result.Items, fmt.Sprintf("%d: %s", syncRes.ItemID, syncRes.Name))
			} else {
				result.Failed++
				result.Errors = append(result.Errors, fmt.Sprintf("Item %d: %s", m.ItemID, syncRes.Error))
			}
			mu.Unlock()

			if delayMs > 0 {
				time.Sleep(time.Duration(delayMs) * time.Millisecond)
			}
		}
	}

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go worker()
	}

	for _, m := range missing {
		jobs <- m
	}
	close(jobs)
	wg.Wait()

	return result, nil
}
