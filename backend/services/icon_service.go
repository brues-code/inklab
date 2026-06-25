package services

// Icon name resolution. Images are never fetched from the network — icons are
// served only from the locally imported client icon set (data/icons), and any
// icon missing there falls back to the questionmark placeholder in the UI. This
// service only discovers and records the correct icon NAME for items/spells whose
// mapping is missing (scraping the database site for the name string is data, not
// an image), so the locally shipped icon can resolve.

import (
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// IconFixService discovers and records missing icon NAMES (item/spell) from the
// database website. It never downloads icon images — the image comes from the
// local client icon set.
type IconFixService struct {
	db      *sql.DB
	iconDir string
	baseURL string
	delayMs int
	client  *http.Client
}

// NewIconFixService creates a new icon fix service.
func NewIconFixService(db *sql.DB, iconDir string) *IconFixService {
	return &IconFixService{
		db:      db,
		iconDir: iconDir,
		baseURL: DatabaseBaseURL + "/?item=",
		delayMs: 500, // Be nice to the server
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// MissingIconItem represents an item with missing icon
type MissingIconItem struct {
	Entry int
	Name  string
}

// GetMissingIcons returns list of items with missing icon (via item_display_info)
func (s *IconFixService) GetMissingIcons() ([]MissingIconItem, error) {
	rows, err := s.db.Query(`
		SELECT t.entry, t.name
		FROM item_template t
		LEFT JOIN item_display_info d ON t.display_id = d.ID
		WHERE d.icon IS NULL OR d.icon = ''
		ORDER BY t.entry
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []MissingIconItem
	for rows.Next() {
		var item MissingIconItem
		if err := rows.Scan(&item.Entry, &item.Name); err != nil {
			continue
		}
		items = append(items, item)
	}

	return items, nil
}

// GetMissingSpellIcons returns list of spells with missing icon
func (s *IconFixService) GetMissingSpellIcons() ([]MissingIconItem, error) {
	rows, err := s.db.Query(`
		SELECT entry, name
		FROM spell_template
		WHERE iconName IS NULL OR iconName = ''
		ORDER BY entry
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var spells []MissingIconItem
	for rows.Next() {
		var spell MissingIconItem
		if err := rows.Scan(&spell.Entry, &spell.Name); err != nil {
			continue
		}
		spells = append(spells, spell)
	}

	return spells, nil
}

// FetchIconFromWebsite fetches the icon NAME (a string like "inv_sword_01") for an
// item from the database website. This is data scraping, not an image download.
func (s *IconFixService) FetchIconFromWebsite(entry int) (string, error) {
	url := fmt.Sprintf("%s%d", s.baseURL, entry)

	resp, err := s.client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// Look for icon in JavaScript data
	// The website uses: Icon.create('iconName', ...) or _[itemId]={icon: 'iconName'}

	// Try pattern 1: Icon.create('iconName', ...)
	re1 := regexp.MustCompile(`Icon\.create\('([^']+)',`)
	matches := re1.FindStringSubmatch(string(body))
	if len(matches) > 1 {
		return matches[1], nil
	}

	// Try pattern 2: _[itemId]={icon: 'iconName'}
	re2 := regexp.MustCompile(fmt.Sprintf(`_\[%d\]=\{icon:\s*'([^']+)'\}`, entry))
	matches = re2.FindStringSubmatch(string(body))
	if len(matches) > 1 {
		return matches[1], nil
	}

	// Try pattern 3: g_items[itemId] = {icon: 'iconName'}
	re3 := regexp.MustCompile(fmt.Sprintf(`g_items\[%d\]\s*=\s*\{[^}]*icon:\s*'([^']+)'`, entry))
	matches = re3.FindStringSubmatch(string(body))
	if len(matches) > 1 {
		return matches[1], nil
	}

	return "", fmt.Errorf("icon not found in HTML")
}

// UpdateIconPath updates icon in item_display_info via display_id
func (s *IconFixService) UpdateIconPath(entry int, iconName string) error {
	// 1. Get display_id
	var displayID int
	err := s.db.QueryRow("SELECT display_id FROM item_template WHERE entry = ?", entry).Scan(&displayID)
	if err != nil {
		return fmt.Errorf("failed to get display_id and update icon for item %d: %w", entry, err)
	}

	// 2. Upsert into item_display_info
	_, err = s.db.Exec(`
		INSERT INTO item_display_info (ID, icon) VALUES (?, ?)
		ON CONFLICT(ID) DO UPDATE SET icon = excluded.icon
	`, displayID, iconName)
	return err
}

// FixSingleItem records the correct icon name for an item (complete workflow).
// It only updates the DB mapping; the icon image must already be in the local
// icon set. Returns: success, iconName, error.
func (s *IconFixService) FixSingleItem(db *sql.DB, itemID int) (bool, string, error) {
	// Check if item exists and join display info
	var currentIcon string
	err := db.QueryRow(`
		SELECT COALESCE(d.icon, '')
		FROM item_template t
		LEFT JOIN item_display_info d ON t.display_id = d.ID
		WHERE t.entry = ?`, itemID).Scan(&currentIcon)
	if err != nil {
		return false, "", fmt.Errorf("item %d not found", itemID)
	}

	// Allow updating if icon is empty or is a generic placeholder
	placeholders := []string{"", "inv_misc_questionmark", "temp", "template"}
	isPlaceholder := false
	currentIconLower := strings.ToLower(currentIcon)
	for _, ph := range placeholders {
		if currentIconLower == ph {
			isPlaceholder = true
			break
		}
	}

	if currentIcon != "" && !isPlaceholder {
		// DB already maps a real icon name; nothing to fix (the image, if any,
		// resolves from the local icon set).
		return false, "", fmt.Errorf("already has icon: %s", currentIcon)
	}

	// Discover the correct icon name from the website (data only) and record it.
	fetchedName, err := s.FetchIconFromWebsite(itemID)
	if err != nil {
		return false, "", err
	}
	iconName := strings.ToLower(fetchedName)

	fetchedIsPlaceholder := false
	for _, ph := range placeholders {
		if iconName == ph {
			fetchedIsPlaceholder = true
			break
		}
	}
	if fetchedIsPlaceholder && iconName == currentIconLower {
		return false, iconName, nil
	}

	if err := s.UpdateIconPath(itemID, iconName); err != nil {
		return false, "", fmt.Errorf("failed to update database: %w", err)
	}
	if fetchedIsPlaceholder {
		return false, iconName, nil
	}
	return true, iconName, nil
}

// UpdateSpellIcon updates iconName in spell_template
func (s *IconFixService) UpdateSpellIcon(spellID int, iconName string) error {
	_, err := s.db.Exec(`
		UPDATE spell_template
		SET iconName = ?
		WHERE entry = ?
	`, iconName, spellID)
	return err
}

// FixSingleSpell records the correct icon name for a spell (complete workflow).
// It only updates the DB mapping; the icon image must already be in the local
// icon set. Returns: success, iconName, error.
func (s *IconFixService) FixSingleSpell(db *sql.DB, spellID int) (bool, string, error) {
	// Check if spell exists
	var currentIcon string
	err := db.QueryRow("SELECT COALESCE(iconName, '') FROM spell_template WHERE entry = ?", spellID).Scan(&currentIcon)
	if err != nil {
		return false, "", fmt.Errorf("spell %d not found", spellID)
	}

	// Allow updating if icon is empty or is a generic placeholder
	placeholders := []string{"", "inv_misc_questionmark", "temp", "template"}
	isPlaceholder := false
	currentIconLower := strings.ToLower(currentIcon)
	for _, ph := range placeholders {
		if currentIconLower == ph {
			isPlaceholder = true
			break
		}
	}

	if currentIcon != "" && !isPlaceholder {
		return false, "", fmt.Errorf("already has icon: %s", currentIcon)
	}

	// Fetch icon name from website (note: spell uses ?spell= parameter).
	url := fmt.Sprintf(DatabaseBaseURL+"/?spell=%d", spellID)
	resp, err := s.client.Get(url)
	if err != nil {
		return false, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, "", fmt.Errorf("HTTP status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, "", err
	}

	// Use same patterns to extract icon
	var iconName string
	re1 := regexp.MustCompile(`Icon\.create\('([^']+)',`)
	matches := re1.FindStringSubmatch(string(body))
	if len(matches) > 1 {
		iconName = matches[1]
	} else {
		re2 := regexp.MustCompile(fmt.Sprintf(`_\[%d\]=\{icon:\s*'([^']+)'\}`, spellID))
		matches = re2.FindStringSubmatch(string(body))
		if len(matches) > 1 {
			iconName = matches[1]
		} else {
			return false, "", fmt.Errorf("icon not found in HTML")
		}
	}

	iconName = strings.ToLower(iconName)

	fetchedIsPlaceholder := false
	for _, ph := range placeholders {
		if iconName == ph {
			fetchedIsPlaceholder = true
			break
		}
	}
	if fetchedIsPlaceholder && iconName == currentIconLower {
		return false, iconName, nil
	}

	if err := s.UpdateSpellIcon(spellID, iconName); err != nil {
		return false, "", fmt.Errorf("failed to update database: %w", err)
	}
	if fetchedIsPlaceholder {
		return false, iconName, nil
	}
	return true, iconName, nil
}
