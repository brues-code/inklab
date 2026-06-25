package services

import (
	"fmt"
	"io"
	"net/http"
	"inklab/backend/parsers"
	"sync"
	"time"
)

// SyncSpell fetches and imports a spell if it's missing from the database
// Also fixes the spell icon if iconDir is provided
// fallbackDesc is used when the spell page doesn't have a description (common for custom spells)
func (s *SyncService) SyncSpell(spellID int, iconDir string, fallbackDesc string) {
	if spellID == 0 {
		return
	}

	// Always fetch to check for description update if missing
	fmt.Printf("[SyncService] Syncing missing/incomplete spell %d...\n", spellID)

	// Fetch spell details
	url := fmt.Sprintf(DatabaseBaseURL+"/?spell=%d", spellID)
	resp, err := http.Get(url)
	if err != nil {
		fmt.Printf("Error fetching spell %d: %v\n", spellID, err)
		return
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return
	}
	content := string(bodyBytes)

	// Use parser
	name, description := parsers.ParseSpell(content)

	// Use fallback description if spell page doesn't have one
	if description == "" && fallbackDesc != "" {
		description = fallbackDesc
		fmt.Printf("  Using fallback description from item page for spell %d\n", spellID)
	}

	if name != "" {
		// Insert or Update logic to handle filling in missing descriptions
		_, err = s.db.Exec(`
			INSERT INTO spell_template (entry, name, description)
			VALUES (?, ?, ?)
			ON CONFLICT(entry) DO UPDATE SET
				name = excluded.name,
				description = excluded.description
			WHERE description = '' OR description IS NULL OR length(description) < 5
		`, spellID, name, description)

		if err != nil {
			fmt.Printf("Error inserting/updating spell %d: %v\n", spellID, err)
		} else {
			fmt.Printf("✓ Synced Spell %d: %s (Desc len: %d)\n", spellID, name, len(description))

			// Fix spell icon if iconDir is provided
			if iconDir != "" {
				iconFixService := NewIconFixService(s.db, iconDir)
				success, iconName, _ := iconFixService.FixSingleSpell(s.db, spellID)
				if success {
					fmt.Printf("  ✓ Fixed spell icon: %s\n", iconName)
				}
			}
		}
	}
}

// SyncSpellResult represents the result of syncing a single spell
type SyncSpellResult struct {
	Success     bool   `json:"success"`
	SpellID     int    `json:"spellId"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	Error       string `json:"error,omitempty"`
}

// FetchAndImportSpell fetches a single spell from turtlecraft.gg and imports it to local database
func (s *SyncService) FetchAndImportSpell(spellID int, iconDir string) *SyncSpellResult {
	if spellID == 0 {
		return &SyncSpellResult{
			Success: false,
			SpellID: spellID,
			Error:   "Invalid spell ID",
		}
	}

	fmt.Printf("[SyncService] FetchAndImportSpell called for spell %d\n", spellID)

	// Fetch spell details from remote database
	url := fmt.Sprintf(DatabaseBaseURL+"/?spell=%d", spellID)
	resp, err := http.Get(url)
	if err != nil {
		return &SyncSpellResult{
			Success: false,
			SpellID: spellID,
			Error:   fmt.Sprintf("Failed to fetch: %v", err),
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return &SyncSpellResult{
			Success: false,
			SpellID: spellID,
			Error:   fmt.Sprintf("HTTP error: %d", resp.StatusCode),
		}
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return &SyncSpellResult{
			Success: false,
			SpellID: spellID,
			Error:   fmt.Sprintf("Failed to read response: %v", err),
		}
	}
	content := string(bodyBytes)

	// Use parser to extract spell data
	name, description := parsers.ParseSpell(content)

	if name == "" {
		return &SyncSpellResult{
			Success: false,
			SpellID: spellID,
			Error:   "Spell not found or has no name",
		}
	}

	// Insert or Update spell
	_, err = s.db.Exec(`
		INSERT INTO spell_template (entry, name, description)
		VALUES (?, ?, ?)
		ON CONFLICT(entry) DO UPDATE SET
			name = excluded.name,
			description = CASE 
				WHEN excluded.description != '' THEN excluded.description 
				ELSE spell_template.description 
			END
	`, spellID, name, description)

	if err != nil {
		return &SyncSpellResult{
			Success: false,
			SpellID: spellID,
			Error:   fmt.Sprintf("Database error: %v", err),
		}
	}

	fmt.Printf("✓ Synced Spell %d: %s (Desc len: %d)\n", spellID, name, len(description))

	// Fix spell icon if iconDir is provided
	if iconDir != "" {
		iconFixService := NewIconFixService(s.db, iconDir)
		success, iconName, _ := iconFixService.FixSingleSpell(s.db, spellID)
		if success {
			fmt.Printf("  ✓ Fixed spell icon: %s\n", iconName)
		}
	}

	return &SyncSpellResult{
		Success:     true,
		SpellID:     spellID,
		Name:        name,
		Description: description,
	}
}

// FullSyncSpells re-syncs all spells referenced by items
func (s *SyncService) FullSyncSpells(delayMs int, fixIcons bool, iconDir string, startFrom int, progressCb ProgressCallback) *FullSyncResult {
	if delayMs <= 0 {
		delayMs = 200
	}

	// Get all unique spell IDs referenced by items
	rows, err := s.db.Query(`
		SELECT DISTINCT spell_id FROM (
			SELECT spellid_1 AS spell_id FROM item_template WHERE spellid_1 > 0
			UNION
			SELECT spellid_2 AS spell_id FROM item_template WHERE spellid_2 > 0
			UNION
			SELECT spellid_3 AS spell_id FROM item_template WHERE spellid_3 > 0
		) ORDER BY spell_id
	`)
	if err != nil {
		return &FullSyncResult{
			Message: fmt.Sprintf("Error querying spells: %v", err),
			Errors:  []string{err.Error()},
		}
	}
	defer rows.Close()

	var spellIDs []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err == nil {
			if startFrom <= 0 || id >= startFrom {
				spellIDs = append(spellIDs, id)
			}
		}
	}

	result := &FullSyncResult{
		TotalItems:  len(spellIDs),
		Errors:      []string{},
		StartFromID: startFrom,
	}

	// Worker pool to parallelize the network-bound scrape, like the item sync.
	const numWorkers = 10
	total := len(spellIDs)
	fmt.Printf("[FullSync] Starting parallel sync of %d spells with %d workers...\n", total, numWorkers)

	jobs := make(chan int, total)
	var wg sync.WaitGroup
	var mu sync.Mutex // guards result + progress
	processed := 0

	worker := func() {
		defer wg.Done()
		for spellID := range jobs {
			if s.IsStopped() {
				return
			}
			s.SyncSpell(spellID, iconDir, "")
			mu.Lock()
			result.Updated++
			result.LastSyncedID = spellID
			processed++
			if progressCb != nil {
				progressCb(processed, total, spellID, fmt.Sprintf("Spell %d", spellID))
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
	for _, id := range spellIDs {
		jobs <- id
	}
	close(jobs)
	wg.Wait()

	if s.IsStopped() {
		result.Message = "Sync stopped by user"
	} else {
		result.Message = "Full spell sync complete"
	}
	return result
}
