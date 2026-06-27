package services

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"

	"inklab/backend/parsers"
)

// HttpClient defines the interface for HTTP requests
type HttpClient interface {
	Do(req *http.Request) (*http.Response, error)
	Get(url string) (*http.Response, error)
}

// ScraperService handles scraping data from external websites
type ScraperService struct {
	Client HttpClient // Dep: Injectable client
}

// NewScraperService creates a new scraper service
func NewScraperService() *ScraperService {
	return &ScraperService{
		Client: &http.Client{},
	}
}

// ScrapedNpcData holds data scraped from Wowhead
type ScrapedNpcData = parsers.ScrapedNpcData

// ScrapeNpcData scrapes NPC data - tries both TurtleCraft and Wowhead and merges them
func (s *ScraperService) ScrapeNpcData(npcID int) (*ScrapedNpcData, error) {
	// Channels to receive results
	tcChan := make(chan *ScrapedNpcData)
	whChan := make(chan *ScrapedNpcData)

	// Fetch concurrently
	go func() {
		data, err := s.scrapeFromTurtlecraft(npcID)
		if err != nil {
			tcChan <- nil
		} else {
			tcChan <- data
		}
	}()

	go func() {
		data, err := s.scrapeFromWowhead(npcID)
		if err != nil {
			whChan <- nil
		} else {
			whChan <- data
		}
	}()

	// Wait for results
	tcData := <-tcChan
	whData := <-whChan

	// If both failed, return empty
	if tcData == nil && whData == nil {
		return nil, fmt.Errorf("failed to scrape from both sources")
	}

	// Start with TurtleCraft data (primary) or fallback to Wowhead
	finalData := tcData
	if finalData == nil {
		finalData = whData
	}

	// Merge logic if we have both
	if tcData != nil && whData != nil {
		// 1. Model Image: Wowhead > TurtleCraft
		if whData.ModelImageURL != "" {
			finalData.ModelImageURL = whData.ModelImageURL
		}

		// 2. Map Image: TurtleCraft > Wowhead
		// (Already set to TurtleCraft by default assignment above)
		if finalData.MapURL == "" && whData.MapURL != "" {
			finalData.MapURL = whData.MapURL
		}

		// 3. InfoBox: Merge (keep TurtleCraft unique values, add missing from Wowhead)
		if finalData.Infobox == nil {
			finalData.Infobox = make(map[string]string)
		}
		for k, v := range whData.Infobox {
			if _, exists := finalData.Infobox[k]; !exists {
				finalData.Infobox[k] = v
			}
		}

		// 4. Coordinates: TurtleCraft > Wowhead
		// (TurtleCraft parser doesn't get X/Y usually, but if it did, we keep it)
		if (finalData.X == 0 && finalData.Y == 0) && (whData.X != 0 || whData.Y != 0) {
			finalData.X = whData.X
			finalData.Y = whData.Y
			// Also take ZoneName if missing
			if finalData.ZoneName == "" {
				finalData.ZoneName = whData.ZoneName
			}
		}
	}

	return finalData, nil
}

// scrapeFromTurtlecraft scrapes NPC data from database.turtlecraft.gg
func (s *ScraperService) scrapeFromTurtlecraft(npcID int) (*ScrapedNpcData, error) {
	url := fmt.Sprintf(DatabaseBaseURL+"/?npc=%d", npcID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := s.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("turtlecraft returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	data, err := parsers.ParseNpcDataTurtlecraft(bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	// Trainer spells live in a script Listview, not the DOM table — parse from the
	// raw HTML.
	data.TrainerSpells = parsers.ParseTrainerSpells(string(body))
	// octowow serves model images as a relative path (images/models/<id>.png);
	// resolve it against the database base URL.
	if data.ModelImageURL != "" && !strings.HasPrefix(data.ModelImageURL, "http") {
		data.ModelImageURL = DatabaseBaseURL + "/" + strings.TrimPrefix(data.ModelImageURL, "/")
	}
	return data, nil
}

// scrapeFromWowhead scrapes NPC data from Wowhead Classic
func (s *ScraperService) scrapeFromWowhead(npcID int) (*ScrapedNpcData, error) {
	url := fmt.Sprintf("https://www.wowhead.com/classic/npc=%d", npcID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")

	resp, err := s.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("failed to fetch page: %d", resp.StatusCode)
	}

	// Use parser to parse content
	return parsers.ParseNpcData(resp.Body)
}

// ScrapedObject bundles everything we pull from a single octowow object page:
// spawn points (per-zone myMapper.update calls with 0-100 map percentages) and,
// for chests, the contains list. One fetch drives both so they stay in sync.
type ScrapedObject struct {
	Spawns []parsers.SpawnPoint
	Loot   []parsers.ObjectLootEntry
}

// ScrapeObject fetches a game object's octowow.st page once and parses both its
// spawns and its loot (chests with custom contents not in our world-DB snapshot,
// e.g. Cache of the Firelord).
func (s *ScraperService) ScrapeObject(objectID int) (*ScrapedObject, error) {
	url := fmt.Sprintf(DatabaseBaseURL+"/?object=%d", objectID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := s.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("octowow returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	spawns, _ := parsers.ParseSpawnPoints(bytes.NewReader(body))
	loot, _ := parsers.ParseObjectLoot(bytes.NewReader(body))
	return &ScrapedObject{Spawns: spawns, Loot: loot}, nil
}

// ScrapeQuestData scrapes Quest data from TurtleCraft
func (s *ScraperService) ScrapeQuestData(entry int) (*parsers.ScrapedQuestData, error) {
	url := fmt.Sprintf(DatabaseBaseURL+"/?quest=%d", entry)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := s.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("turtlecraft returned status %d", resp.StatusCode)
	}

	return parsers.ParseQuestDataTurtlecraft(resp.Body, entry)
}
