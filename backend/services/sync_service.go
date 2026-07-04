package services

import (
	"database/sql"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync/atomic"
	"time"
)

// SyncService handles database synchronization with turtlecraft.gg
type SyncService struct {
	db            *sql.DB
	httpClient    *http.Client
	baseURL       string
	stopRequested atomic.Bool
}

// SyncProgress represents the current sync progress
type SyncProgress struct {
	Type     string `json:"type"`     // "item" or "quest"
	Current  int    `json:"current"`  // Current ID being checked
	Total    int    `json:"total"`    // Total IDs to check
	Found    int    `json:"found"`    // Items found on remote
	Missing  int    `json:"missing"`  // Items in local but not remote
	NewItems int    `json:"newItems"` // Items on remote but not local
	Status   string `json:"status"`   // "running", "done", "error"
	Message  string `json:"message"`  // Status message
}

// SyncResult represents the final sync result
type SyncResult struct {
	ItemsChecked  int           `json:"itemsChecked"`
	NewItems      []RemoteItem  `json:"newItems"`
	QuestsChecked int           `json:"questsChecked"`
	NewQuests     []RemoteQuest `json:"newQuests"`
	Duration      string        `json:"duration"`
}

// FullSyncResult represents the result of a full sync operation
type FullSyncResult struct {
	TotalItems   int      `json:"totalItems"`
	Updated      int      `json:"updated"`
	Failed       int      `json:"failed"`
	IconsFixed   int      `json:"iconsFixed"`
	Errors       []string `json:"errors"`
	Message      string   `json:"message"`
	LastSyncedID int      `json:"lastSyncedId"` // Last successfully synced item ID for resume
	StartFromID  int      `json:"startFromId"`  // ID we started this sync from
}

// RemoteItem represents an item found on turtlecraft.gg
type RemoteItem struct {
	Entry int    `json:"entry"`
	Name  string `json:"name"`
	URL   string `json:"url"`
}

// RemoteQuest represents a quest found on turtlecraft.gg
type RemoteQuest struct {
	Entry int    `json:"entry"`
	Title string `json:"title"`
	URL   string `json:"url"`
}

// ProgressCallback is a function type for reporting sync progress
type ProgressCallback func(current, total int, itemID int, itemName string)

// NewSyncService creates a new sync service
func NewSyncService(db *sql.DB) *SyncService {
	return &SyncService{
		db: db,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		baseURL: DatabaseBaseURL,
	}
}

// errNotFound marks a definitive HTTP 404 — the page doesn't exist, as opposed
// to a transient failure (throttled, network blip) that exhausted its retries.
var errNotFound = errors.New("not found (HTTP 404)")

// getWithRetry fetches url, retrying transient failures — network errors,
// HTTP 429 (throttled: the full syncs run 10 fetches at once, which can trip
// the server's limiter), and 5xx — with the same backoff the page scrapers
// use. A 404 returns errNotFound immediately (retrying can't create the page).
// On success the response is returned with its body open; the caller closes it.
func (s *SyncService) getWithRetry(url string) (*http.Response, error) {
	var lastErr error
	for attempt := 1; ; attempt++ {
		resp, err := s.httpClient.Get(url)
		switch {
		case err != nil:
			lastErr = err
		case resp.StatusCode == 200:
			return resp, nil
		default:
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			if resp.StatusCode == 404 {
				return nil, errNotFound
			}
			if resp.StatusCode != 429 && resp.StatusCode < 500 {
				// Other 4xx: not transient, retrying won't help.
				return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
			}
			lastErr = fmt.Errorf("HTTP %d", resp.StatusCode)
		}
		if attempt >= scrapeAttempts {
			return nil, lastErr
		}
		time.Sleep(time.Duration(attempt) * scrapeRetryDelay)
	}
}

// GetSyncStats returns current sync statistics
func (s *SyncService) GetSyncStats() map[string]interface{} {
	itemCount, _ := s.GetLocalItemCount()
	questCount, _ := s.GetLocalQuestCount()
	maxItem, _ := s.GetLocalMaxItemID()
	maxQuest, _ := s.GetLocalMaxQuestID()
	missingCount, _ := s.GetMissingAtlasLootItemCount()

	var creatureCount, maxCreatureID int
	s.db.QueryRow("SELECT COUNT(*) FROM creature_template").Scan(&creatureCount)
	s.db.QueryRow("SELECT MAX(entry) FROM creature_template").Scan(&maxCreatureID)

	return map[string]interface{}{
		"itemCount":             itemCount,
		"questCount":            questCount,
		"maxItemID":             maxItem,
		"maxQuestID":            maxQuest,
		"missingAtlasLootItems": missingCount,
		"creatureCount":         creatureCount,
		"maxCreatureID":         maxCreatureID,
	}
}

// RequestStop signals the sync process to stop
func (s *SyncService) RequestStop() {
	s.stopRequested.Store(true)
}

// IsStopped returns true if stop was requested
func (s *SyncService) IsStopped() bool {
	return s.stopRequested.Load()
}

// ResetStop resets the stop signal
func (s *SyncService) ResetStop() {
	s.stopRequested.Store(false)
}
