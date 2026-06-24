package services

import (
	"context"
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"inklab/backend/database"
	"strings"
	"sync/atomic"
	"time"
)

type NpcService struct {
	sqlite        *sql.DB
	mysql         *database.MySQLConnection
	scraper       *ScraperService
	itemRepo      *database.ItemRepository
	creatureRepo  *database.CreatureRepository
	dataDir       string // Path to data directory for storing images
	stopRequested atomic.Bool
}

func NewNpcService(sqlite *sql.DB, mysql *database.MySQLConnection, scraper *ScraperService, itemRepo *database.ItemRepository, creatureRepo *database.CreatureRepository, dataDir string) *NpcService {
	return &NpcService{
		sqlite:       sqlite,
		mysql:        mysql,
		scraper:      scraper,
		itemRepo:     itemRepo,
		creatureRepo: creatureRepo,
		dataDir:      dataDir,
	}
}

type NpcLoot struct {
	ItemID   int     `json:"itemId"`
	Name     string  `json:"name"`
	Chance   float64 `json:"chance"`
	MinCount int     `json:"minCount"`
	MaxCount int     `json:"maxCount"`
	Quality  int     `json:"quality"`
	IconPath string  `json:"iconPath"`
}

type NpcQuest struct {
	QuestID int    `json:"questId"`
	Title   string `json:"title"`
	Type    string `json:"type"` // "starts" or "ends"
	Level   int    `json:"level"`
}

type NpcAbility struct {
	SpellID     int    `json:"spellId"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
}

type NpcSpawn struct {
	MapId    int     `json:"mapId"`
	ZoneName string  `json:"zoneName"`
	X        float64 `json:"x"`
	Y        float64 `json:"y"`
}

type NpcFullDetails struct {
	*database.Creature
	Infobox       map[string]string `json:"infobox"`
	MapURL        string            `json:"mapUrl"`
	ModelImageURL string            `json:"modelImageUrl"`
	ZoneName      string            `json:"zoneName"` // New
	X             float64           `json:"x"`        // New
	Y             float64           `json:"y"`        // New
	Loot          []NpcLoot         `json:"loot"`
	Quests        []NpcQuest        `json:"quests"`
	Abilities     []NpcAbility      `json:"abilities"`
	Spawns        []NpcSpawn        `json:"spawns"`
}

func (s *NpcService) GetNpcDetails(entry int) (*NpcFullDetails, error) {
	// 1. Try to load from SQLite (Primary Source)
	details, err := s.loadFromSQLite(entry)
	if err == nil && details != nil {
		// Found in SQLite - return immediately!
		// Metadata (infobox, map) can be fetched on-demand via separate API
		return details, nil
	}

	// 2. Not found in SQLite at all - try to sync from MySQL first
	fmt.Printf("NPC %d not found in SQLite, attempting to sync from MySQL...\n", entry)
	if s.mysql != nil {
		// Sync basic creature data from MySQL (fast)
		if err := s.syncCreatureFromMySQL(entry); err != nil {
			fmt.Printf("Warning: Failed to sync creature from MySQL: %v\n", err)
		}
	}

	// 3. Reload from SQLite
	details, err = s.loadFromSQLite(entry)
	if err != nil || details == nil {
		return nil, fmt.Errorf("NPC %d not found", entry)
	}

	return details, nil
}

func (s *NpcService) loadFromSQLite(entry int) (*NpcFullDetails, error) {
	// Use Repository to get base creature data (includes new Quick Facts fields)
	creature, err := s.creatureRepo.GetCreatureByID(entry)
	if err != nil {
		return nil, err
	}

	details := &NpcFullDetails{
		Creature:  creature,
		Infobox:   make(map[string]string),
		Loot:      []NpcLoot{},
		Quests:    []NpcQuest{},
		Abilities: []NpcAbility{},
	}

	// Load Metadata
	var mapUrl, infoboxJson, modelImageUrl, zoneName string
	var modelImageLocal, mapImageLocal string
	var x, y float64

	// Read fields, handling potential NULLs or missing columns gracefully via Scan logic if needed,
	// but here we just select COALESCE defaults.
	// Note: We need to ensure columns exist in DB schema.
	err = s.sqlite.QueryRow(`
		SELECT map_url, infobox_json, COALESCE(model_image_url, ''), 
		       COALESCE(model_image_local, ''), COALESCE(map_image_local, ''),
		       COALESCE(zone_name, ''), COALESCE(x, 0), COALESCE(y, 0)
		FROM creature_metadata WHERE entry = ?
	`, entry).Scan(&mapUrl, &infoboxJson, &modelImageUrl, &modelImageLocal, &mapImageLocal, &zoneName, &x, &y)

	if err == nil {
		// Use remote URLs directly
		// Local storage feature can be added later with proper asset serving
		details.ModelImageURL = modelImageUrl
		details.MapURL = mapUrl

		details.ZoneName = zoneName
		details.X = x
		details.Y = y
		if infoboxJson != "" {
			_ = json.Unmarshal([]byte(infoboxJson), &details.Infobox)
		}
	} else {
		// Ignore error if metadata missing
	}

	// Load spawns from creature_spawn table (synced from MySQL)
	spawnRows, err := s.sqlite.Query(`
		SELECT map_id, zone_id, zone_name, position_x, position_y, position_z
		FROM creature_spawn
		WHERE creature_entry = ?
		ORDER BY id
		LIMIT 20
	`, entry)
	if err == nil {
		defer spawnRows.Close()
		for spawnRows.Next() {
			var spawn NpcSpawn
			var zoneId int
			var z float64
			if err := spawnRows.Scan(&spawn.MapId, &zoneId, &spawn.ZoneName, &spawn.X, &spawn.Y, &z); err == nil {
				details.Spawns = append(details.Spawns, spawn)
			}
		}
	}

	// If no spawns from creature_spawn table, fallback to metadata spawns
	if len(details.Spawns) == 0 && (zoneName != "" || x != 0 || y != 0) {
		details.Spawns = []NpcSpawn{{
			MapId:    0,
			ZoneName: zoneName,
			X:        x,
			Y:        y,
		}}
	}

	// Update details.ZoneName and X/Y from first spawn if available
	// Prefer spawn data over metadata since spawn comes from MySQL coordinates conversion
	if len(details.Spawns) > 0 && details.Spawns[0].ZoneName != "" {
		details.ZoneName = details.Spawns[0].ZoneName
		details.X = details.Spawns[0].X
		details.Y = details.Spawns[0].Y
	}

	// Load Loot
	// First resolve loot_id
	var lootID int
	s.sqlite.QueryRow("SELECT loot_id FROM creature_template WHERE entry = ?", entry).Scan(&lootID)
	if lootID == 0 {
		lootID = entry
	}

	// Fetch loot (Direct + Reference)
	rows, err := s.sqlite.Query(`
		SELECT l.item, i.name, l.ChanceOrQuestChance, 
		       l.mincountOrRef, l.maxcount, i.quality, COALESCE(idi.icon, '')
		FROM creature_loot_template l
		LEFT JOIN item_template i ON l.item = i.entry
		LEFT JOIN item_display_info idi ON i.display_id = idi.ID
		WHERE l.entry = ? AND l.mincountOrRef >= 0

		UNION ALL

		SELECT r.item, i.name, 
		       l.ChanceOrQuestChance, -- Simplification: showing group chance or ref chance requires more logic
		       r.mincountOrRef, r.maxcount, i.quality, COALESCE(idi.icon, '')
		FROM creature_loot_template l
		JOIN reference_loot_template r ON l.mincountOrRef = -r.entry
		LEFT JOIN item_template i ON r.item = i.entry
		LEFT JOIN item_display_info idi ON i.display_id = idi.ID
		WHERE l.entry = ?
	`, lootID, lootID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var l NpcLoot
			var name, icon sql.NullString
			var quality sql.NullInt32
			// Use Null types for safety on left join
			if err := rows.Scan(&l.ItemID, &name, &l.Chance, &l.MinCount, &l.MaxCount, &quality, &icon); err == nil {
				l.Name = name.String
				l.Quality = int(quality.Int32)
				l.IconPath = icon.String
				details.Loot = append(details.Loot, l)
			}
		}
	}

	// Load Quests
	// Starts
	qRows, err := s.sqlite.Query(`
		SELECT qs.quest, q.Title, q.MinLevel
		FROM creature_questrelation qs
		JOIN quest_template q ON qs.quest = q.entry
		WHERE qs.id = ?
	`, entry)
	if err == nil {
		defer qRows.Close()
		for qRows.Next() {
			var q NpcQuest
			q.Type = "starts"
			if err := qRows.Scan(&q.QuestID, &q.Title, &q.Level); err == nil {
				details.Quests = append(details.Quests, q)
			}
		}
	}
	// Ends
	qRowsEnd, err := s.sqlite.Query(`
		SELECT qe.quest, q.Title, q.MinLevel
		FROM creature_involvedrelation qe
		JOIN quest_template q ON qe.quest = q.entry
		WHERE qe.id = ?
	`, entry)
	if err == nil {
		defer qRowsEnd.Close()
		for qRowsEnd.Next() {
			var q NpcQuest
			q.Type = "ends"
			if err := qRowsEnd.Scan(&q.QuestID, &q.Title, &q.Level); err == nil {
				details.Quests = append(details.Quests, q)
			}
		}
	}

	// Load Abilities
	// Note: We need a table for NPC abilities or query from creature_template columns if mapped
	// Assuming syncNpcData puts abilities into a helper table `npc_abilities` or we just read from creature_template
	// Since generated schema has spell_id1..4, we can read directly.
	var s1, s2, s3, s4 int
	err = s.sqlite.QueryRow("SELECT spell_id1, spell_id2, spell_id3, spell_id4 FROM creature_template WHERE entry = ?", entry).Scan(&s1, &s2, &s3, &s4)
	if err == nil {
		spellIDs := []int{s1, s2, s3, s4}
		for _, id := range spellIDs {
			if id > 0 {
				var name, desc string
				var icon sql.NullString
				// Check spell_template and join spell_icons
				err := s.sqlite.QueryRow(`
					SELECT st.name, st.description, si.icon_name 
					FROM spell_template st 
					LEFT JOIN spell_icons si ON st.spellIconId = si.id
					WHERE st.entry = ?
				`, id).Scan(&name, &desc, &icon)
				if err != nil {
					name = fmt.Sprintf("Spell %d", id)
				}
				details.Abilities = append(details.Abilities, NpcAbility{
					SpellID:     id,
					Name:        name,
					Description: desc,
					Icon:        icon.String,
				})
			}
		}
	}

	return details, nil
}

// syncCreatureFromMySQL syncs basic creature data from MySQL to SQLite (fast, no web scraping)
func (s *NpcService) syncCreatureFromMySQL(entry int) error {
	if s.mysql == nil {
		return fmt.Errorf("MySQL connection not available")
	}

	// Check if creature exists in MySQL
	var name, subname string
	var levelMin, levelMax, healthMax, manaMax, faction, rank, typeId, displayId int
	var goldMin, goldMax int
	var s1, s2, s3, s4 int

	err := s.mysql.DB().QueryRow(`
		SELECT name, COALESCE(subname, ''), level_min, level_max, health_max, mana_max, 
			   faction, `+"`rank`"+`, type, display_id1, gold_min, gold_max,
			   spell_id1, spell_id2, spell_id3, spell_id4
		FROM creature_template WHERE entry = ?
	`, entry).Scan(&name, &subname, &levelMin, &levelMax, &healthMax, &manaMax,
		&faction, &rank, &typeId, &displayId, &goldMin, &goldMax,
		&s1, &s2, &s3, &s4)

	if err != nil {
		return fmt.Errorf("creature not found in MySQL: %w", err)
	}

	// Insert into SQLite (UPSERT)
	_, err = s.sqlite.Exec(`
		INSERT INTO creature_template (entry, name, subname, level_min, level_max, health_max, mana_max,
			faction, rank, type, display_id1, gold_min, gold_max,
			spell_id1, spell_id2, spell_id3, spell_id4)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(entry) DO UPDATE SET
			name = excluded.name, subname = excluded.subname,
			level_min = excluded.level_min, level_max = excluded.level_max,
			health_max = excluded.health_max, mana_max = excluded.mana_max,
			faction = excluded.faction, rank = excluded.rank, type = excluded.type,
			display_id1 = excluded.display_id1, gold_min = excluded.gold_min, gold_max = excluded.gold_max,
			spell_id1 = excluded.spell_id1, spell_id2 = excluded.spell_id2,
			spell_id3 = excluded.spell_id3, spell_id4 = excluded.spell_id4
	`, entry, name, subname, levelMin, levelMax, healthMax, manaMax,
		faction, rank, typeId, displayId, goldMin, goldMax,
		s1, s2, s3, s4)

	if err != nil {
		return fmt.Errorf("failed to insert creature into SQLite: %w", err)
	}

	// Sync spawn coordinates from MySQL creature table
	s.syncCreatureSpawnsFromMySQL(entry)

	// Also sync the referenced spells if they don't exist in local spell_template
	spells := []int{s1, s2, s3, s4}
	for _, spellID := range spells {
		if spellID > 0 {
			s.syncSpellFromMySQL(spellID)
		}
	}

	fmt.Printf("✓ Synced creature %d (%s) from MySQL\n", entry, name)
	return nil
}

// syncCreatureSpawnsFromMySQL syncs creature spawn coordinates from MySQL creature table
func (s *NpcService) syncCreatureSpawnsFromMySQL(entry int) {
	if s.mysql == nil {
		return
	}

	// Ensure creature_spawn table exists
	_, _ = s.sqlite.Exec(`
		CREATE TABLE IF NOT EXISTS creature_spawn (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			creature_entry INTEGER NOT NULL,
			map_id INTEGER DEFAULT 0,
			zone_id INTEGER DEFAULT 0,
			zone_name TEXT DEFAULT '',
			position_x REAL DEFAULT 0,
			position_y REAL DEFAULT 0,
			position_z REAL DEFAULT 0,
			UNIQUE(creature_entry, map_id, position_x, position_y)
		)
	`)

	// Delete existing spawns for this creature (we replce them)
	s.sqlite.Exec("DELETE FROM creature_spawn WHERE creature_entry = ?", entry)

	spawnCount := 0

	// Query spawn points from MySQL if available
	if s.mysql != nil {
		// Using aggregation functions to satisfy only_full_group_by sql_mode
		rows, err := s.mysql.DB().Query(`
			SELECT map, AVG(position_x) as avg_x, AVG(position_y) as avg_y, AVG(position_z) as avg_z
			FROM creature 
			WHERE id = ? 
			GROUP BY map, ROUND(position_x, -1), ROUND(position_y, -1)
			LIMIT 20
		`, entry)

		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var mapId int
				var worldX, worldY, z float64
				if err := rows.Scan(&mapId, &worldX, &worldY, &z); err != nil {
					continue
				}

				// Convert world coordinates to map percentage (0-100)
				zoneName, mapX, mapY := s.convertWorldToMapCoords(mapId, 0, worldX, worldY)

				_, err = s.sqlite.Exec(`
					INSERT INTO creature_spawn (creature_entry, map_id, zone_id, zone_name, position_x, position_y, position_z)
					VALUES (?, ?, ?, ?, ?, ?, ?)
				`, entry, mapId, 0, zoneName, mapX, mapY, z)
				if err == nil {
					spawnCount++
				}
			}
		} else {
			fmt.Printf("Warning: Could not query creature spawns from MySQL: %v\n", err)
		}
	}

	if spawnCount > 0 {
		fmt.Printf("  ✓ Synced %d spawn points for creature %d\n", spawnCount, entry)
	} else {
		// Fallback: If no MySQL spawns, try to use scraped metadata from SQLite
		// We just inserted/updated it in SyncNpcData, so it should be fresh.
		var metaX, metaY float64
		var metaZone string
		err := s.sqlite.QueryRow("SELECT x, y, zone_name FROM creature_metadata WHERE entry = ?", entry).Scan(&metaX, &metaY, &metaZone)
		if err == nil && metaZone != "" && (metaX > 0 || metaY > 0) {
			fmt.Printf("  ⚠ No MySQL spawns for %d, falling back to scraped data: %s (%.1f, %.1f)\n", entry, metaZone, metaX, metaY)

			// Insert pseudo-spawn
			// We don't have MapID or Z, but we have ZoneName and Map Coords
			// Use a dummy MapID (maybe 0 or derived if possible, but 0 is safe-ish for display only)
			// Or try to map ZoneName to MapID if we really want to be fancy.

			_, err = s.sqlite.Exec(`
				INSERT INTO creature_spawn (creature_entry, map_id, zone_id, zone_name, position_x, position_y, position_z)
				VALUES (?, ?, ?, ?, ?, ?, ?)
				ON CONFLICT(creature_entry, map_id, position_x, position_y) DO UPDATE SET
					zone_name = excluded.zone_name
			`, entry, 0, 0, metaZone, metaX, metaY, 0)

			if err == nil {
				fmt.Printf("  ✓ Created pseudo-spawn from web data for creature %d\n", entry)
			} else {
				fmt.Printf("  ✕ Failed to create pseudo-spawn: %v\n", err)
			}
		} else {
			fmt.Printf("  ⚠ No spawn points found in MySQL and no valid scraped metadata for entry %d\n", entry)
		}
	}
}

// convertWorldToMapCoords converts world coordinates to map percentage coordinates (0-100)
// Using the aowow_zones table boundaries similar to the PHP coord_db2wow function
func (s *NpcService) convertWorldToMapCoords(mapId, zoneId int, worldX, worldY float64) (zoneName string, mapX, mapY float64) {
	if s.mysql == nil {
		return s.getZoneNameFromID(zoneId, mapId), 0, 0
	}

	// Query zone boundaries from aowow_zones
	// Note: In WoW, X and Y are swapped compared to typical conventions
	// The formula is: mapX = 100 - (worldY - y_min) / ((y_max - y_min) / 100)
	//                 mapY = 100 - (worldX - x_min) / ((x_max - x_min) / 100)
	var xMin, xMax, yMin, yMax float64
	var name string

	// Find the most specific zone by selecting the smallest area that contains the coordinates
	// This ensures we get "Tanaris" instead of "Kalimdor" when both match
	err := s.mysql.DB().QueryRow(`
		SELECT name_loc0, x_min, x_max, y_min, y_max 
		FROM aowow.aowow_zones 
		WHERE mapID = ? 
		  AND x_min < ? AND x_max > ? 
		  AND y_min < ? AND y_max > ?
		  AND x_min != 0 AND x_max != 0
		ORDER BY (x_max - x_min) * (y_max - y_min) ASC
		LIMIT 1
	`, mapId, worldX, worldX, worldY, worldY).Scan(&name, &xMin, &xMax, &yMin, &yMax)

	if err == nil && name != "" && (xMax-xMin) > 0 && (yMax-yMin) > 0 {
		// Convert coordinates
		// WoW World (MySQL) -> Map Percentage (0-100)
		// Standard Formula:
		// MapX = (y_max - worldY) / (y_max - y_min) * 100
		// MapY = (x_max - worldX) / (x_max - x_min) * 100

		mapX = (yMax - worldY) / (yMax - yMin) * 100
		mapY = (xMax - worldX) / (xMax - xMin) * 100

		// Clamp to valid range
		if mapX < 0 {
			mapX = 0
		} else if mapX > 100 {
			mapX = 100
		}
		if mapY < 0 {
			mapY = 0
		} else if mapY > 100 {
			mapY = 100
		}

		return name, mapX, mapY
	}

	// Fallback: Try to get zone info for instances (zones with 0,0,0,0 boundaries)
	err = s.mysql.DB().QueryRow(`
		SELECT name_loc0 FROM aowow.aowow_zones 
		WHERE mapID = ? AND x_min = 0 AND x_max = 0 AND y_min = 0 AND y_max = 0
		LIMIT 1
	`, mapId).Scan(&name)

	if err == nil && name != "" {
		// For instances, we can't calculate map coordinates, return 50,50 as center
		return name, 50, 50
	}

	// Final fallback
	return s.getZoneNameFromID(zoneId, mapId), 0, 0
}

// getZoneNameFromID attempts to get zone name from zone ID
func (s *NpcService) getZoneNameFromID(zoneId, mapId int) string {
	// Try to get zone name from aowow_zones table in MySQL
	if s.mysql != nil {
		var zoneName string
		err := s.mysql.DB().QueryRow(`
			SELECT name_loc0 FROM aowow.aowow_zones WHERE areatableID = ?
		`, zoneId).Scan(&zoneName)
		if err == nil && zoneName != "" {
			return zoneName
		}

		// Fallback: Try map_template for instance maps
		err = s.mysql.DB().QueryRow(`
			SELECT map_name FROM map_template WHERE entry = ?
		`, mapId).Scan(&zoneName)
		if err == nil && zoneName != "" {
			return zoneName
		}
	}

	// Hardcoded fallback for common zones
	zoneNames := map[int]string{
		1:    "Dun Morogh",
		12:   "Elwynn Forest",
		14:   "Durotar",
		17:   "The Barrens",
		33:   "Stranglethorn Vale",
		40:   "Westfall",
		85:   "Tirisfal Glades",
		130:  "Silverpine Forest",
		148:  "Darkshore",
		215:  "Mulgore",
		331:  "Ashenvale",
		357:  "Feralas",
		361:  "Felwood",
		400:  "Thousand Needles",
		405:  "Desolace",
		406:  "Stonetalon Mountains",
		440:  "Tanaris",
		490:  "Un'Goro Crater",
		493:  "Moonglade",
		618:  "Winterspring",
		1377: "Silithus",
		1422: "Western Plaguelands",
		1423: "Eastern Plaguelands",
		2677: "Blackwing Lair",
		2717: "Molten Core",
	}
	if name, ok := zoneNames[zoneId]; ok {
		return name
	}
	return ""
}

// syncSpellFromMySQL syncs a single spell from MySQL to SQLite
func (s *NpcService) syncSpellFromMySQL(spellID int) {
	// Check if already exists with description (simple check)
	var count int
	s.sqlite.QueryRow("SELECT COUNT(*) FROM spell_template WHERE entry = ? AND description != ''", spellID).Scan(&count)
	if count > 0 {
		// Even if exists, check if icon is linked?
		// For now assume if description exists, it's fine.
		// But let's be safe and check spell_icons linkage if we have time.
		// For performance, return.
		return
	}

	// Fetch from MySQL
	var name, desc string
	var iconID int
	err := s.mysql.DB().QueryRow("SELECT name, description, spellIconId FROM spell_template WHERE entry = ?", spellID).Scan(&name, &desc, &iconID)
	if err != nil {
		fmt.Printf("Warning: Could not fetch spell %d from MySQL: %v\n", spellID, err)
		return
	}

	// Insert into SQLite
	_, err = s.sqlite.Exec(`
		INSERT INTO spell_template (entry, name, description, spellIconId) VALUES (?, ?, ?, ?)
		ON CONFLICT(entry) DO UPDATE SET name=excluded.name, description=excluded.description, spellIconId=excluded.spellIconId
	`, spellID, name, desc, iconID)

	if err != nil {
		fmt.Printf("Warning: Failed to save spell %d to SQLite: %v\n", spellID, err)
	}

	// Sync Icon if needed
	if iconID > 0 {
		var iconCount int
		s.sqlite.QueryRow("SELECT COUNT(*) FROM spell_icons WHERE id = ?", iconID).Scan(&iconCount)
		if iconCount == 0 {
			var iconName string
			// Fetch from Aowow DB
			err = s.mysql.DB().QueryRow("SELECT iconname FROM aowow.aowow_spellicons WHERE id = ?", iconID).Scan(&iconName)
			if err == nil {
				_, _ = s.sqlite.Exec("INSERT INTO spell_icons (id, icon_name) VALUES (?, ?)", iconID, iconName)
			} else {
				fmt.Printf("Warning: Could not fetch icon %d from Aowow: %v\n", iconID, err)
			}
		}
	}
}

// SyncAllCreatureSpawns syncs spawn points for all creatures
func (s *NpcService) SyncAllCreatureSpawns(progressCb func(current, total int, id int)) error {
	if s.mysql == nil {
		return fmt.Errorf("no mysql connection")
	}

	// Get all entries
	rows, err := s.sqlite.Query("SELECT entry FROM creature_template ORDER BY entry")
	if err != nil {
		return err
	}
	defer rows.Close()

	var entries []int
	for rows.Next() {
		var e int
		if err := rows.Scan(&e); err == nil {
			entries = append(entries, e)
		}
	}

	total := len(entries)
	for i, entry := range entries {
		s.syncCreatureSpawnsFromMySQL(entry)
		if progressCb != nil && i%10 == 0 { // Update every 10 items
			progressCb(i+1, total, entry)
		}
	}

	if progressCb != nil {
		progressCb(total, total, 0)
	}

	return nil
}

// FullSyncNpcs performs a full sync (scrape + DB) for all NPCs starting from a specific ID
func (s *NpcService) FullSyncNpcs(startFrom int, delayMs int, progressCb func(current, total int, id int)) error {
	// Get all entries starting from startFrom
	rows, err := s.sqlite.Query("SELECT entry FROM creature_template WHERE entry >= ? ORDER BY entry", startFrom)
	if err != nil {
		return err
	}
	defer rows.Close()

	var entries []int
	for rows.Next() {
		var e int
		if err := rows.Scan(&e); err == nil {
			entries = append(entries, e)
		}
	}

	total := len(entries)
	for i, entry := range entries {
		// Check for stop request
		if s.IsStopped() {
			return nil
		}

		// Perform full sync (scrape + DB)
		s.SyncNpcData(entry)

		if progressCb != nil {
			progressCb(i+1, total, entry)
		}

		if delayMs > 0 {
			time.Sleep(time.Duration(delayMs) * time.Millisecond)
		}
	}

	return nil
}

// RefreshNpcImages scrapes only the visual metadata (model + map images, zone,
// coords) and stores it, WITHOUT touching creature_template. Use this to pull a
// missing model/map without re-syncing (and potentially overwriting) the
// creature's stat data from the frozen MySQL dump.
func (s *NpcService) RefreshNpcImages(entry int) error {
	return s.syncNpcImages(entry)
}

// syncNpcImages performs the scrape + image download + creature_metadata upsert.
func (s *NpcService) syncNpcImages(entry int) error {
	// A. Scrape Wowhead for Metadata
	scrapedData, err := s.scraper.ScrapeNpcData(entry)
	if err != nil {
		fmt.Printf("Scrape failed: %v\n", err)
		scrapedData = &ScrapedNpcData{Infobox: make(map[string]string)}
	}

	// B. Download images to local storage
	npcImagesDir := filepath.Join(s.dataDir, "npc_images")
	if err := os.MkdirAll(npcImagesDir, 0755); err != nil {
		fmt.Printf("Warning: Failed to create npc_images directory: %v\n", err)
	}

	// Download model image using Hash-based filename for deduplication
	localModelPath := ""
	if scrapedData.ModelImageURL != "" {
		localModelPath = s.downloadImage(scrapedData.ModelImageURL, npcImagesDir, "")
		if localModelPath != "" {
			fmt.Printf("[DEBUG] Model image synced: %s\n", localModelPath)
		}
	}

	// Download map image using Hash-based filename for deduplication
	localMapPath := ""
	if scrapedData.MapURL != "" {
		localMapPath = s.downloadImage(scrapedData.MapURL, npcImagesDir, "")
		if localMapPath != "" {
			fmt.Printf("[DEBUG] Map image synced: %s\n", localMapPath)
		}
	}

	// Store Metadata to SQLite
	// Ensure columns exist (quick dirty adjustment)
	_, _ = s.sqlite.Exec("ALTER TABLE creature_metadata ADD COLUMN model_image_url TEXT")
	_, _ = s.sqlite.Exec("ALTER TABLE creature_metadata ADD COLUMN model_image_local TEXT")
	_, _ = s.sqlite.Exec("ALTER TABLE creature_metadata ADD COLUMN map_image_local TEXT")
	_, _ = s.sqlite.Exec("ALTER TABLE creature_metadata ADD COLUMN zone_name TEXT")
	_, _ = s.sqlite.Exec("ALTER TABLE creature_metadata ADD COLUMN x REAL")
	_, _ = s.sqlite.Exec("ALTER TABLE creature_metadata ADD COLUMN y REAL")

	infoboxBytes, _ := json.Marshal(scrapedData.Infobox)
	_, err = s.sqlite.Exec(`
		INSERT INTO creature_metadata (entry, map_url, infobox_json, model_image_url, model_image_local, map_image_local, zone_name, x, y)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(entry) DO UPDATE SET
			map_url = excluded.map_url,
			infobox_json = excluded.infobox_json,
			model_image_url = excluded.model_image_url,
			model_image_local = excluded.model_image_local,
			map_image_local = excluded.map_image_local,
			zone_name = excluded.zone_name,
			x = excluded.x,
			y = excluded.y
	`, entry, scrapedData.MapURL, string(infoboxBytes), scrapedData.ModelImageURL, localModelPath, localMapPath, scrapedData.ZoneName, scrapedData.X, scrapedData.Y)
	if err != nil {
		return fmt.Errorf("failed to save metadata: %w", err)
	}
	return nil
}

func (s *NpcService) SyncNpcData(entry int) error {
	// A. Scrape + images + metadata (no creature_template changes).
	if err := s.syncNpcImages(entry); err != nil {
		return err
	}
	var err error

	// B. Sync from MySQL (if available)
	if s.mysql != nil {
		// 1. creature_template
		// Read 20+ columns needed or just use `SELECT *` map?
		// For simplicity, let's fetch key columns including spells
		var name, subname string
		var lootID, s1, s2, s3, s4, minLvl, maxLvl, hpMax, manaMax, rank, faction int
		var typeId, armor, holy, fire, nature, frost, shadow, arcane, displayId, goldMin, goldMax int
		var dmgMin, dmgMax float64

		// Note: Column names in MySQL might differ slightly (e.g. Health vs health_max)
		// InkLab uses `creature_template` structure.
		// Let's assume standard names.
		query := `
			SELECT 
				name, subname, loot_id, 
				spell1, spell2, spell3, spell4, 
				minlevel, maxlevel, maxhealth, maxmana, 
				rank, faction_A, type,
				mindmg, maxdmg, armor,
				resistance1, resistance2, resistance3, resistance4, resistance5, resistance6,
				modelid1, mingold, maxgold
			FROM creature_template WHERE entry = ?`

		// Adjust query based on actual MySQL schema if needed.
		// Trying a best-effort simpler query matching what we usually have.
		err = s.mysql.DB().QueryRow(query, entry).Scan(
			&name, &subname, &lootID,
			&s1, &s2, &s3, &s4,
			&minLvl, &maxLvl, &hpMax, &manaMax,
			&rank, &faction, &typeId,
			&dmgMin, &dmgMax, &armor,
			&holy, &fire, &nature, &frost, &shadow, &arcane,
			&displayId, &goldMin, &goldMax,
		)

		if err == nil {
			// Update SQLite creature_template
			// We use INSERT OR REPLACE to update all these stats
			_, _ = s.sqlite.Exec(`
				UPDATE creature_template SET 
					name=?, subname=?, loot_id=?,
					spell_id1=?, spell_id2=?, spell_id3=?, spell_id4=?,
					level_min=?, level_max=?, health_max=?, mana_max=?,
					rank=?, faction=?, type=?,
					dmg_min=?, dmg_max=?, armor=?,
					holy_res=?, fire_res=?, nature_res=?, frost_res=?, shadow_res=?, arcane_res=?,
					display_id1=?, gold_min=?, gold_max=?
				WHERE entry=?
			`, name, subname, lootID, s1, s2, s3, s4, minLvl, maxLvl, hpMax, manaMax, rank, faction, typeId,
				dmgMin, dmgMax, armor, holy, fire, nature, frost, shadow, arcane, displayId, goldMin, goldMax, entry)

			// If it didn't exist (updated 0 rows), insert it
			// This might fail if row doesn't exist.
			// Ideally we rely on the large import, but for dev sync:
			_, _ = s.sqlite.Exec(`
				INSERT INTO creature_template 
				(entry, name, subname, loot_id, spell_id1, spell_id2, spell_id3, spell_id4, level_min, level_max, health_max, mana_max, rank, faction, type,
				 dmg_min, dmg_max, armor, holy_res, fire_res, nature_res, frost_res, shadow_res, arcane_res, display_id1, gold_min, gold_max)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
				ON CONFLICT(entry) DO UPDATE SET
					name=excluded.name, subname=excluded.subname, loot_id=excluded.loot_id,
					spell_id1=excluded.spell_id1, spell_id2=excluded.spell_id2, spell_id3=excluded.spell_id3, spell_id4=excluded.spell_id4,
					dmg_min=excluded.dmg_min, dmg_max=excluded.dmg_max, display_id1=excluded.display_id1,
					gold_min=excluded.gold_min, gold_max=excluded.gold_max
			`, entry, name, subname, lootID, s1, s2, s3, s4, minLvl, maxLvl, hpMax, manaMax, rank, faction, typeId,
				dmgMin, dmgMax, armor, holy, fire, nature, frost, shadow, arcane, displayId, goldMin, goldMax)
		}

		// 2. Loot
		if lootID > 0 {
			// Fetch from MySQL loot tables and insert into SQLite creature_loot_template
			// Note: ensure column names match MySQL `creature_loot_template`
			lRows, lErr := s.mysql.DB().Query("SELECT Item, Chance, MinCount, MaxCount, GroupId FROM creature_loot_template WHERE Entry = ?", lootID)
			if lErr == nil {
				defer lRows.Close()
				s.sqlite.Exec("DELETE FROM creature_loot_template WHERE entry = ?", entry)

				for lRows.Next() {
					var item, min, max, group int
					var chance float64
					if err := lRows.Scan(&item, &chance, &min, &max, &group); err == nil {
						s.sqlite.Exec(`
							INSERT INTO creature_loot_template (entry, item, ChanceOrQuestChance, mincountOrRef, maxcount, groupid)
							VALUES (?, ?, ?, ?, ?, ?)
						`, entry, item, chance, min, max, group)
					}
				}
			}
		}

		// 3. Quests (Starts/Ends)
		// Starts
		qsRows, qsErr := s.mysql.DB().Query("SELECT quest FROM creature_questrelation WHERE id = ?", entry)
		if qsErr == nil {
			defer qsRows.Close()
			s.sqlite.Exec("DELETE FROM creature_questrelation WHERE id = ?", entry)
			for qsRows.Next() {
				var q int
				if err := qsRows.Scan(&q); err == nil {
					s.sqlite.Exec("INSERT INTO creature_questrelation (id, quest) VALUES (?,?)", entry, q)
				}
			}
		}
		// Ends
		qeRows, qeErr := s.mysql.DB().Query("SELECT quest FROM creature_involvedrelation WHERE id = ?", entry)
		if qeErr == nil {
			defer qeRows.Close()
			s.sqlite.Exec("DELETE FROM creature_involvedrelation WHERE id = ?", entry)
			for qeRows.Next() {
				var q int
				if err := qeRows.Scan(&q); err == nil {
					s.sqlite.Exec("INSERT INTO creature_involvedrelation (id, quest) VALUES (?,?)", entry, q)
				}
			}
		}

		// 4. Sync spawn coordinates from creature table
		fmt.Printf("[SyncNpcData] Syncing spawn coordinates for creature %d...\n", entry)
		s.syncCreatureSpawnsFromMySQL(entry)
	} else {
		// Even if MySQL is missing, try to generate spawn from scraped metadata
		fmt.Printf("[SyncNpcData] No MySQL connection. Attempting to use scraped spawn data for %d...\n", entry)
		s.syncCreatureSpawnsFromMySQL(entry)
	}

	return nil
}

// GetNpcDetailsContext adds a context-aware version if needed for Wails
func (s *NpcService) GetNpcDetailsContext(ctx context.Context, entry int) (*NpcFullDetails, error) {
	return s.GetNpcDetails(entry)
}

// downloadImage downloads an image from URL and saves it locally using MD5 hash of URL as filename for deduplication
func (s *NpcService) downloadImage(url string, dir string, _ string) string {
	if url == "" {
		return ""
	}

	// Compute MD5 hash of URL for deduplication
	hash := md5.Sum([]byte(url))
	hashName := hex.EncodeToString(hash[:])

	// Determine file extension from URL
	ext := ".jpg"
	if strings.Contains(strings.ToLower(url), ".png") {
		ext = ".png"
	} else if strings.Contains(strings.ToLower(url), ".gif") {
		ext = ".gif"
	} else if strings.Contains(strings.ToLower(url), ".webp") {
		ext = ".webp"
	}

	localPath := filepath.Join(dir, hashName+ext)

	// Skip if file already exists (DEDUPLICATION)
	if _, err := os.Stat(localPath); err == nil {
		return localPath
	}

	// Download the image
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Printf("Failed to create request for %s: %v\n", url, err)
		return ""
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Failed to download image from %s: %v\n", url, err)
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		fmt.Printf("Failed to download image from %s: HTTP %d\n", url, resp.StatusCode)
		return ""
	}

	// Create file
	file, err := os.Create(localPath)
	if err != nil {
		fmt.Printf("Failed to create file %s: %v\n", localPath, err)
		return ""
	}
	defer file.Close()

	// Copy data
	_, err = io.Copy(file, resp.Body)
	if err != nil {
		fmt.Printf("Failed to save image to %s: %v\n", localPath, err)
		os.Remove(localPath) // Clean up partial file
		return ""
	}

	return localPath
}

// RequestStop signals the sync process to stop
func (s *NpcService) RequestStop() {
	s.stopRequested.Store(true)
}

// IsStopped returns true if stop was requested
func (s *NpcService) IsStopped() bool {
	return s.stopRequested.Load()
}

// ResetStop resets the stop signal
func (s *NpcService) ResetStop() {
	s.stopRequested.Store(false)
}
