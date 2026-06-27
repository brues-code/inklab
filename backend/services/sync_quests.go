package services

import (
	"fmt"
	"io"
	"inklab/backend/database/models"
	"inklab/backend/parsers"
	"sync"
	"time"
)

// GetLocalMaxQuestID returns the maximum quest entry in local database
func (s *SyncService) GetLocalMaxQuestID() (int, error) {
	var maxID int
	err := s.db.QueryRow("SELECT MAX(entry) FROM quest_template WHERE entry >= 40000").Scan(&maxID)
	if err != nil {
		return 40000, nil
	}
	return maxID, nil
}

// GetLocalQuestCount returns count of Turtle quests in local database
func (s *SyncService) GetLocalQuestCount() (int, error) {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM quest_template WHERE entry >= 40000").Scan(&count)
	return count, err
}

// QuestExistsLocally checks if a quest exists in local database
func (s *SyncService) QuestExistsLocally(entry int) bool {
	var count int
	s.db.QueryRow("SELECT COUNT(*) FROM quest_template WHERE entry = ?", entry).Scan(&count)
	return count > 0
}

// CheckRemoteQuest checks if a quest exists on turtlecraft.gg and returns its title
func (s *SyncService) CheckRemoteQuest(entry int) (bool, string, error) {
	url := fmt.Sprintf("%s/?quest=%d", s.baseURL, entry)

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

	exists, title := parsers.ParseQuestTitle(content)
	if exists && title == "" {
		// Fallback title
		title = fmt.Sprintf("Quest %d", entry)
	}

	return exists, title, nil
}

// CheckNewQuests checks for new quests beyond local max ID
func (s *SyncService) CheckNewQuests(maxChecks int, delayMs int, progressChan chan<- SyncProgress) ([]RemoteQuest, error) {
	localMax, _ := s.GetLocalMaxQuestID()
	startID := localMax + 1

	var newQuests []RemoteQuest
	consecutiveMisses := 0
	maxConsecutiveMisses := 20 // Stop after 20 consecutive misses

	// If maxChecks <= 0, treat as practically unlimited (max int)
	if maxChecks <= 0 {
		maxChecks = 2147483647 // Max Int32
	}

	checked := 0
	for id := startID; checked < maxChecks && consecutiveMisses < maxConsecutiveMisses; id++ {
		// Check for cancellation
		if s.IsStopped() {
			return newQuests, nil
		}
		checked++

		// Send progress
		if progressChan != nil {
			progressChan <- SyncProgress{
				Type:     "quest",
				Current:  id,
				Total:    startID + maxChecks,
				Found:    len(newQuests),
				NewItems: len(newQuests),
				Status:   "running",
				Message:  fmt.Sprintf("Checking quest %d...", id),
			}
		}

		exists, title, err := s.CheckRemoteQuest(id)
		if err != nil {
			consecutiveMisses++
			continue
		}

		if exists {
			consecutiveMisses = 0
			newQuests = append(newQuests, RemoteQuest{
				Entry: id,
				Title: title,
				URL:   fmt.Sprintf("%s/?quest=%d", s.baseURL, id),
			})
			fmt.Printf("  Found new quest: %d - %s\n", id, title)

			// Import it immediately
			res := s.FetchAndImportQuest(id)
			if res.Success {
				fmt.Printf("  ✓ Imported quest: %s\n", title)
				newQuests = append(newQuests, RemoteQuest{
					Entry: id,
					Title: title,
					URL:   fmt.Sprintf("%s/?quest=%d", s.baseURL, id),
				})
			} else {
				fmt.Printf("  x Failed to import quest: %s\n", res.Error)
			}
		} else {
			consecutiveMisses++
		}

		// Rate limiting
		time.Sleep(time.Duration(delayMs) * time.Millisecond)
	}

	return newQuests, nil
}

// FetchQuestDetails fetches detailed quest info from turtlecraft.gg
func (s *SyncService) FetchQuestDetails(questID int) (*models.QuestDetail, error) {
	url := fmt.Sprintf("%s/?quest=%d", s.baseURL, questID)

	resp, err := s.httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("quest not found: %d", questID)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	content := string(body)

	return parsers.ParseQuest(content, questID)
}

// SyncQuestResult represents the result of syncing a single quest
type SyncQuestResult struct {
	Success bool   `json:"success"`
	QuestID int    `json:"questId"`
	Title   string `json:"title,omitempty"`
	Error   string `json:"error,omitempty"`
}

// FetchAndImportQuest scrapes a quest and fills the fields the WDB quest cache
// can't carry — start/end NPCs, Race/Class masks, reputation rewards, and the
// prev/next chain — WITHOUT overwriting the WDB/world-dump data. Text and core
// numeric fields are only backfilled when they're empty locally, so the
// authoritative cached values always win.
func (s *SyncService) FetchAndImportQuest(questID int) *SyncQuestResult {
	fmt.Printf("[SyncService] FetchAndImportQuest called for quest %d\n", questID)

	url := fmt.Sprintf("%s/?quest=%d", s.baseURL, questID)
	resp, err := s.httpClient.Get(url)
	if err != nil {
		return &SyncQuestResult{Success: false, QuestID: questID, Error: err.Error()}
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return &SyncQuestResult{Success: false, QuestID: questID, Error: fmt.Sprintf("quest not found: %d", questID)}
	}
	data, err := parsers.ParseQuestDataTurtlecraft(resp.Body, questID)
	if err != nil {
		return &SyncQuestResult{Success: false, QuestID: questID, Error: err.Error()}
	}
	if data.Title == "" {
		return &SyncQuestResult{Success: false, QuestID: questID, Error: "no quest data"}
	}

	tx, err := s.db.Begin()
	if err != nil {
		return &SyncQuestResult{Success: false, QuestID: questID, Error: err.Error()}
	}
	defer tx.Rollback()

	// Ensure the row exists (custom quests not yet imported from WDB/world dump).
	tx.Exec("INSERT OR IGNORE INTO quest_template (entry, Title) VALUES (?, ?)", questID, data.Title)

	// fill* writes a scraped value only into an empty/zero column — used for the
	// fields the WDB quest cache carries, so we never clobber authoritative cached
	// (or world-dump) data.
	fillText := func(col, val string) {
		if val == "" {
			return
		}
		tx.Exec(fmt.Sprintf("UPDATE quest_template SET %s = ? WHERE entry = ? AND (%s IS NULL OR %s = '')", col, col, col), val, questID)
	}
	fillInt := func(col string, val int) {
		if val == 0 {
			return
		}
		tx.Exec(fmt.Sprintf("UPDATE quest_template SET %s = ? WHERE entry = ? AND (%s IS NULL OR %s = 0)", col, col, col), val, questID)
	}
	// set* overwrites unconditionally — used for the fields the WDB can't carry,
	// where the scrape is the authoritative source. Still guarded against wiping a
	// good value with a failed-parse 0/"".
	setText := func(col, val string) {
		if val == "" {
			return
		}
		tx.Exec(fmt.Sprintf("UPDATE quest_template SET %s = ? WHERE entry = ?", col), val, questID)
	}
	setInt := func(col string, val int) {
		if val == 0 {
			return
		}
		tx.Exec(fmt.Sprintf("UPDATE quest_template SET %s = ? WHERE entry = ?", col), val, questID)
	}

	// WDB-provided fields: backfill only when empty (cache/world-dump wins).
	fillText("Title", data.Title)
	fillText("Details", data.Details)
	fillText("EndText", data.EndText)
	fillInt("QuestLevel", data.QuestLevel)
	fillInt("ZoneOrSort", data.ZoneOrSort)

	// Fields the WDB cache doesn't carry: the scrape is authoritative.
	setText("OfferRewardText", data.OfferRewardText)
	setInt("MinLevel", data.MinLevel)
	setInt("RewXP", data.RewXP)
	setInt("RewSpellCast", data.RewSpellCast) // WDB cache lacks this; scrape is the source
	setInt("RequiredRaces", data.RaceMask)
	setInt("RequiredClasses", data.ClassMask)
	setInt("PrevQuestId", data.PrevQuestID)
	setInt("NextQuestId", data.NextQuestID)

	// Start/End NPC relations (additive; PK(id,quest) dedupes).
	for _, npc := range data.Starters {
		tx.Exec("INSERT OR IGNORE INTO creature_questrelation (id, quest) VALUES (?, ?)", npc, questID)
	}
	for _, npc := range data.Enders {
		tx.Exec("INSERT OR IGNORE INTO creature_involvedrelation (id, quest) VALUES (?, ?)", npc, questID)
	}

	// Reputation rewards aren't in the WDB, so the scrape is authoritative: when it
	// finds any, replace the slots. (When it finds none we leave existing data
	// alone — absence may just be a parse miss.) Faction id comes from the link.
	if len(data.RepRewards) > 0 {
		tx.Exec(`UPDATE quest_template SET
			RewRepFaction1=0, RewRepValue1=0, RewRepFaction2=0, RewRepValue2=0,
			RewRepFaction3=0, RewRepValue3=0, RewRepFaction4=0, RewRepValue4=0,
			RewRepFaction5=0, RewRepValue5=0 WHERE entry = ?`, questID)
		slot := 1
		for _, rr := range data.RepRewards {
			if slot > 5 || rr.FactionID == 0 {
				continue
			}
			tx.Exec(fmt.Sprintf("UPDATE quest_template SET RewRepFaction%d = ?, RewRepValue%d = ? WHERE entry = ?", slot, slot), rr.FactionID, rr.Value, questID)
			slot++
		}
	}

	if err := tx.Commit(); err != nil {
		return &SyncQuestResult{Success: false, QuestID: questID, Error: err.Error()}
	}
	return &SyncQuestResult{Success: true, QuestID: questID, Title: data.Title}
}

// FullSyncQuests re-syncs all quests in the database
func (s *SyncService) FullSyncQuests(delayMs int, startFrom int, progressCb ProgressCallback) *FullSyncResult {
	if delayMs <= 0 {
		delayMs = 200
	}

	// Get all quest IDs ordered by entry
	rows, err := s.db.Query("SELECT entry FROM quest_template ORDER BY entry ASC")
	if err != nil {
		return &FullSyncResult{
			Message: fmt.Sprintf("Error querying quests: %v", err),
			Errors:  []string{err.Error()},
		}
	}
	defer rows.Close()

	var questIDs []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err == nil {
			if startFrom <= 0 || id >= startFrom {
				questIDs = append(questIDs, id)
			}
		}
	}

	result := &FullSyncResult{
		TotalItems:  len(questIDs),
		Errors:      []string{},
		StartFromID: startFrom,
	}

	// Worker pool to parallelize the network-bound scrape, like the item sync.
	const numWorkers = 10
	total := len(questIDs)
	fmt.Printf("[FullSync] Starting parallel sync of %d quests with %d workers...\n", total, numWorkers)

	jobs := make(chan int, total)
	var wg sync.WaitGroup
	var mu sync.Mutex // guards result + progress
	processed := 0

	worker := func() {
		defer wg.Done()
		for questID := range jobs {
			if s.IsStopped() {
				return
			}
			res := s.FetchAndImportQuest(questID)
			mu.Lock()
			if res.Success {
				result.Updated++
			} else {
				result.Failed++
				if len(result.Errors) < 10 {
					result.Errors = append(result.Errors, fmt.Sprintf("Quest %d: %s", questID, res.Error))
				}
			}
			result.LastSyncedID = questID
			processed++
			if progressCb != nil {
				progressCb(processed, total, questID, res.Title)
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
	for _, id := range questIDs {
		jobs <- id
	}
	close(jobs)
	wg.Wait()

	if s.IsStopped() {
		result.Message = "Sync stopped by user"
	} else {
		result.Message = "Full quest sync complete"
	}
	return result
}
