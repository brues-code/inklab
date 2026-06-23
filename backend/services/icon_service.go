package services

import (
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"shelllab/backend/database"
)

// IconService handles downloading icons
type IconService struct {
	db        *database.SQLiteDB
	outputDir string
	client    *http.Client
}

// NewIconService creates a new IconService
func NewIconService(db *database.SQLiteDB) *IconService {
	// Default output dir: "data/icons" (relative to executable)
	// This allows icons to be persistent and external to the embedded binary
	outputDir := filepath.Join("data", "icons")
	return &IconService{
		db:        db,
		outputDir: outputDir,
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// StartDownload initiates the download process in background
func (s *IconService) StartDownload() {
	go func() {
		fmt.Println("[IconService] Starting background icon download...")
		if err := s.downloadProcess(); err != nil {
			fmt.Printf("[IconService] Error: %v\n", err)
		} else {
			fmt.Println("[IconService] Download complete.")
		}
	}()
}

func (s *IconService) downloadProcess() error {
	// Ensure directory exists
	if err := os.MkdirAll(s.outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output dir: %w", err)
	}

	// 1. Collect all unique icon names from database
	iconNames := make(map[string]bool)

	// Items
	rows, err := s.db.DB().Query(`
		SELECT DISTINCT icon_path 
		FROM item_template 
		WHERE icon_path IS NOT NULL AND icon_path != ''
	`)
	if err != nil {
		return fmt.Errorf("query items failed: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err == nil && name != "" {
			iconNames[strings.ToLower(name)] = true
		}
	}

	// Note: spell_template doesn't have icon_name column (uses spellIconId instead)
	// Spell icons would need a separate lookup table if needed

	fmt.Printf("[IconService] Found %d unique icons to check.\n", len(iconNames))

	// 2. Filter out existing icons
	var toDownload []string
	for name := range iconNames {
		pathJPG := filepath.Join(s.outputDir, name+".jpg")
		pathPNG := filepath.Join(s.outputDir, name+".png")
		_, errJPG := os.Stat(pathJPG)
		_, errPNG := os.Stat(pathPNG)

		// Only download if NEITHER exists
		if os.IsNotExist(errJPG) && os.IsNotExist(errPNG) {
			toDownload = append(toDownload, name)
		}
	}

	if len(toDownload) == 0 {
		fmt.Println("[IconService] All icons exist. Skipping download.")
		return nil
	}

	fmt.Printf("[IconService] Downloading %d missing icons...\n", len(toDownload))

	// 3. Download worker pool
	var wg sync.WaitGroup
	sem := make(chan struct{}, 10) // Concurrency limit

	// Sources to try in order - try PNG first for Turtle WoW custom icons, then fallback to JPG
	sources := []string{
		DatabaseBaseURL + "/images/icons/large/%s.png",                        // WoW Database (PNG)
		DatabaseBaseURL + "/images/icons/large/%s.jpg",                        // WoW Database (JPG fallback)
		"https://wow.zamimg.com/images/wow/icons/large/%s.jpg",                // Wowhead CDN (supports Classic)
		"https://aowow.trinitycore.info/static/images/wow/icons/large/%s.jpg", // Trinity Aowow
	}

	var successCount, failCount int
	var mu sync.Mutex

	for _, name := range toDownload {
		wg.Add(1)
		go func(iconName string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			success := false
			for _, srcFmt := range sources {
				url := fmt.Sprintf(srcFmt, iconName)
				if err := s.downloadFile(url, iconName); err == nil {
					success = true
					break
				}
			}

			mu.Lock()
			if success {
				successCount++
			} else {
				failCount++
				// create a placeholder or just log?
				// fmt.Printf("Failed to download: %s\n", iconName)
			}
			mu.Unlock()

			// Slight delay to be nice
			time.Sleep(50 * time.Millisecond)
		}(name)
	}

	wg.Wait()
	fmt.Printf("[IconService] Downloaded: %d, Failed: %d\n", successCount, failCount)
	return nil
}

func (s *IconService) downloadFile(url, name string) error {
	resp, err := s.client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("status %d", resp.StatusCode)
	}

	// Check content type and determine correct extension
	ct := resp.Header.Get("Content-Type")
	if !strings.Contains(ct, "image") {
		return fmt.Errorf("invalid content type: %s", ct)
	}

	// Determine extension from Content-Type
	ext := ".jpg"
	if strings.Contains(ct, "png") {
		ext = ".png"
	}

	filename := filepath.Join(s.outputDir, name+ext)
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	return err
}

// DownloadSingleIcon downloads a single icon from URL to destination path
func (s *IconService) DownloadSingleIcon(url, destPath string) error {
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	resp, err := s.client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("HTTP status %d", resp.StatusCode)
	}

	file, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		os.Remove(destPath) // Clean up on error
		return err
	}

	return nil
}

// ============================================================================
// Icon Fix Methods
// ============================================================================

// IconFixService handles fetching and fixing missing item icons
type IconFixService struct {
	db      *sql.DB
	iconDir string
	baseURL string
	delayMs int
	client  *http.Client
}

// NewIconFixService creates a new icon fix service
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

// FetchIconFromWebsite fetches icon name from Turtle WoW database website
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

// UpdateIconPath updates icon_path in database
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

const (
	TurtleIconPNG = DatabaseBaseURL + "/images/icons/large/%s.png"
	TurtleIconJPG = DatabaseBaseURL + "/images/icons/large/%s.jpg"
	WowheadIcon   = "https://wow.zamimg.com/images/wow/icons/large/%s.jpg"
	TrinityIcon   = "https://aowow.trinitycore.info/static/images/wow/icons/large/%s.jpg"
)

// downloadIconFromSources attempts to download an icon from multiple sources
func (s *IconFixService) downloadIconFromSources(iconName string) error {
	sources := []string{
		fmt.Sprintf(TurtleIconPNG, iconName),
		fmt.Sprintf(TurtleIconJPG, iconName),
		fmt.Sprintf(WowheadIcon, iconName),
		fmt.Sprintf(TrinityIcon, iconName),
	}

	for _, url := range sources {
		if err := s.downloadFile(url, iconName); err == nil {
			return nil
		}
	}
	return fmt.Errorf("failed to download icon %s from all sources", iconName)
}

// downloadFile downloads a single file and saves it with correct extension
func (s *IconFixService) downloadFile(url, name string) error {
	resp, err := s.client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("status %d", resp.StatusCode)
	}

	// Determine extension from Content-Type
	ct := resp.Header.Get("Content-Type")
	ext := ".jpg"
	if strings.Contains(ct, "png") {
		ext = ".png"
	}

	filename := filepath.Join(s.iconDir, name+ext)
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	return err
}

// FixSingleItem fixes icon for a single item (complete workflow)
// Returns: success, iconName, error
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

	var iconName string
	needFetch := true

	if currentIcon != "" && !isPlaceholder {
		// If DB has a valid icon, check if the file actually exists (JPG or PNG)
		pathJPG := filepath.Join(s.iconDir, currentIconLower+".jpg")
		pathPNG := filepath.Join(s.iconDir, currentIconLower+".png")
		_, errJPG := os.Stat(pathJPG)
		_, errPNG := os.Stat(pathPNG)

		if errJPG == nil || errPNG == nil {
			return false, "", fmt.Errorf("[v2] already has valid icon: %s", currentIcon)
		}
		// File missing, skip fetch and use current value to redownload
		iconName = currentIconLower
		needFetch = false
	}

	if needFetch {
		// Fetch icon name from website
		fetchedName, err := s.FetchIconFromWebsite(itemID)
		if err != nil {
			return false, "", err
		}

		// Normalize to lowercase
		iconName = strings.ToLower(fetchedName)

		// Check if the fetched icon is also a placeholder
		fetchedIsPlaceholder := false
		for _, ph := range placeholders {
			if iconName == ph {
				fetchedIsPlaceholder = true
				break
			}
		}

		// If fetched icon is still a placeholder, and matches what we have, return success but don't count as "Fixed"
		if fetchedIsPlaceholder && iconName == currentIconLower {
			return false, iconName, nil
		}

		// Update database
		if err := s.UpdateIconPath(itemID, iconName); err != nil {
			return false, "", fmt.Errorf("failed to update database: %w", err)
		}

		// If result is a placeholder, consider it not a "success" fix for UI stats
		if fetchedIsPlaceholder {
			return false, iconName, nil
		}
	}

	// Check if file already exists with either extension before downloading
	iconPathJPG := filepath.Join(s.iconDir, iconName+".jpg")
	iconPathPNG := filepath.Join(s.iconDir, iconName+".png")
	_, errJPG := os.Stat(iconPathJPG)
	_, errPNG := os.Stat(iconPathPNG)

	if os.IsNotExist(errJPG) && os.IsNotExist(errPNG) {
		// Use shared download logic
		if err := s.downloadIconFromSources(iconName); err != nil {
			// Don't fail the whole operation if download fails, as we updated the DB
			fmt.Printf("Warning: Failed to download icon %s: %v\n", iconName, err)
		}
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

// FixSingleSpell fixes icon for a single spell (complete workflow)
// Returns: success, iconName, error
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

	var iconName string
	needFetch := true

	if currentIcon != "" && !isPlaceholder {
		pathJPG := filepath.Join(s.iconDir, currentIconLower+".jpg")
		pathPNG := filepath.Join(s.iconDir, currentIconLower+".png")
		_, errJPG := os.Stat(pathJPG)
		_, errPNG := os.Stat(pathPNG)

		if errJPG == nil || errPNG == nil {
			return false, "", fmt.Errorf("already has valid icon: %s", currentIcon)
		}
		iconName = currentIconLower
		needFetch = false
	}

	if needFetch {
		// Fetch icon name from website (note: spell uses ?spell= parameter)
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

		// Normalize to lowercase
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
	}

	// Check if file already exists with either extension before downloading
	iconPathJPG := filepath.Join(s.iconDir, iconName+".jpg")
	iconPathPNG := filepath.Join(s.iconDir, iconName+".png")
	_, errJPG := os.Stat(iconPathJPG)
	_, errPNG := os.Stat(iconPathPNG)

	if os.IsNotExist(errJPG) && os.IsNotExist(errPNG) {
		if err := s.downloadIconFromSources(iconName); err != nil {
			fmt.Printf("Warning: Failed to download icon %s: %v\n", iconName, err)
		}
	}

	return true, iconName, nil
}
