package services

import (
	"database/sql"
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
