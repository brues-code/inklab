package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"inklab/backend/database"
	"inklab/backend/datatools"
	"runtime"
	"sync"
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

	zonesOnce  sync.Once
	zoneBounds []zoneBound
	zoneByArea map[int]*zoneBound // areatableID -> bounds, for area-grid resolution

	areaOnce sync.Once
	areaGrid *datatools.AreaGrid // client-derived, nil when data/area_grid.bin absent
}

// zoneBound mirrors an entry in data/zones.json (client WorldMapArea-derived).
// Bounds are in world coordinates; name_loc0 matches the data/maps file name.
type zoneBound struct {
	MapID       int     `json:"mapID"`
	AreatableID int     `json:"areatableID"`
	Name        string  `json:"name_loc0"`
	XMax        float64 `json:"x_max"`
	XMin        float64 `json:"x_min"`
	YMax        float64 `json:"y_max"`
	YMin        float64 `json:"y_min"`
}

func NewNpcService(sqlite *sql.DB, mysql *database.MySQLConnection, scraper *ScraperService, itemRepo *database.ItemRepository, creatureRepo *database.CreatureRepository, dataDir string) *NpcService {
	s := &NpcService{
		sqlite:       sqlite,
		mysql:        mysql,
		scraper:      scraper,
		itemRepo:     itemRepo,
		creatureRepo: creatureRepo,
		dataDir:      dataDir,
	}
	s.ensureSchema()
	return s
}

// ensureSchema creates the spawn tables and adds the creature_metadata columns
// once, at startup. These used to run on every per-NPC sync call, which is fine
// serially but causes write-lock contention (and SQLITE_BUSY failures that abort
// the sync) when the full sync runs them concurrently across the worker pool.
func (s *NpcService) ensureSchema() {
	s.sqlite.Exec(`
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
		)`)
	for _, col := range []string{
		"ALTER TABLE creature_metadata ADD COLUMN model_image_url TEXT",
		"ALTER TABLE creature_metadata ADD COLUMN model_image_local TEXT",
		"ALTER TABLE creature_metadata ADD COLUMN map_image_local TEXT",
		"ALTER TABLE creature_metadata ADD COLUMN zone_name TEXT",
		"ALTER TABLE creature_metadata ADD COLUMN x REAL",
		"ALTER TABLE creature_metadata ADD COLUMN y REAL",
	} {
		s.sqlite.Exec(col) // ignore "duplicate column" errors
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
	FactionName   string            `json:"factionName"` // resolved from the faction template
	FactionID     int               `json:"factionId"`   // resolved Faction.dbc id
	ZoneName      string            `json:"zoneName"`    // New
	X             float64           `json:"x"`        // New
	Y             float64           `json:"y"`        // New
	Loot          []NpcLoot         `json:"loot"`
	Quests        []NpcQuest        `json:"quests"`
	Abilities     []NpcAbility      `json:"abilities"`
	Spawns        []NpcSpawn        `json:"spawns"`
	Sells         []NpcSellItem     `json:"sells"`
}

// NpcSellItem is an item this NPC sells (reverse of item_vendor).
type NpcSellItem struct {
	ItemID   int    `json:"itemId"`
	Name     string `json:"name"`
	Quality  int    `json:"quality"`
	IconPath string `json:"iconPath"`
	Cost     int    `json:"cost"`
	Stock    int    `json:"stock"`
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

	// Resolve the faction name from the creature's faction template
	// (creature.faction is a FactionTemplate id -> Faction.dbc id -> name).
	if creature.Faction > 0 {
		s.sqlite.QueryRow(`
			SELECT f.id, f.name
			FROM faction_template ft
			JOIN factions f ON ft.faction_id = f.id
			WHERE ft.template_id = ?
		`, creature.Faction).Scan(&details.FactionID, &details.FactionName)
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
					SELECT st.name, st.description, COALESCE(NULLIF(si.icon_name, ''), st.iconName, '')
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

	// Load what this NPC sells (reverse of item_vendor; item info from our DB).
	sellRows, err := s.sqlite.Query(`
		SELECT iv.item_entry, COALESCE(i.name, ''), COALESCE(i.quality, 0),
		       COALESCE(idi.icon, ''), iv.cost, iv.stock
		FROM item_vendor iv
		LEFT JOIN item_template i ON iv.item_entry = i.entry
		LEFT JOIN item_display_info idi ON i.display_id = idi.ID
		WHERE iv.npc_entry = ?
		ORDER BY i.quality DESC, i.name
	`, entry)
	if err == nil {
		defer sellRows.Close()
		for sellRows.Next() {
			var it NpcSellItem
			if err := sellRows.Scan(&it.ItemID, &it.Name, &it.Quality, &it.IconPath, &it.Cost, &it.Stock); err == nil {
				details.Sells = append(details.Sells, it)
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

	// creature_spawn is created once in ensureSchema. Replace this creature's rows.
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

// loadZoneBounds lazily reads data/zones.json (client WorldMapArea-derived).
// This is the complete, authoritative geometry — it includes Mount Hyjal and
// the custom octo zones that the live aowow_zones import is missing.
func (s *NpcService) loadZoneBounds() {
	s.zonesOnce.Do(func() {
		b, err := os.ReadFile(filepath.Join(s.dataDir, "zones.json"))
		if err != nil {
			fmt.Printf("[NpcService] could not read zones.json for coord conversion: %v\n", err)
			return
		}
		if err := json.Unmarshal(b, &s.zoneBounds); err != nil {
			fmt.Printf("[NpcService] could not parse zones.json: %v\n", err)
			return
		}
		// Index by areatableID so area-grid lookups can find a zone's map bounds.
		// Prefer bounded entries; an instance/zeroed entry shouldn't shadow one.
		s.zoneByArea = make(map[int]*zoneBound, len(s.zoneBounds))
		for i := range s.zoneBounds {
			z := &s.zoneBounds[i]
			if z.AreatableID == 0 {
				continue
			}
			if cur, ok := s.zoneByArea[z.AreatableID]; ok && (cur.XMin != 0 || cur.XMax != 0) {
				continue
			}
			s.zoneByArea[z.AreatableID] = z
		}
	})
}

// loadAreaGrid lazily loads the client-derived area grid (data/area_grid.bin). It
// is the authoritative source for which zone a world coordinate sits in — read
// from the same ADT terrain the game uses — so it resolves spawns correctly where
// zones' axis-aligned bounding boxes overlap (e.g. the Barrens vs Dustwallow
// Marsh). Absent until generated via Tools; resolution then falls back to bounds.
func (s *NpcService) loadAreaGrid() {
	s.areaOnce.Do(func() {
		g, err := datatools.LoadAreaGrid(filepath.Join(s.dataDir, "area_grid.bin"))
		if err != nil {
			fmt.Printf("[NpcService] could not load area_grid.bin: %v\n", err)
			return
		}
		s.areaGrid = g
	})
}

// zoneFromAreaGrid resolves a world point to its in-game zone via the area grid,
// returning that zone's map name and 0-100 coords projected into its bounds.
func (s *NpcService) zoneFromAreaGrid(mapId int, worldX, worldY float64) (name string, mapX, mapY float64, ok bool) {
	s.loadAreaGrid()
	s.loadZoneBounds() // builds zoneByArea, which we index below
	if s.areaGrid == nil || s.zoneByArea == nil {
		return "", 0, 0, false
	}
	areaID, found := s.areaGrid.ZoneAt(mapId, worldX, worldY)
	if !found {
		return "", 0, 0, false
	}
	z := s.zoneByArea[int(areaID)]
	if z == nil || (z.XMin == 0 && z.XMax == 0) {
		return "", 0, 0, false
	}
	mapX = clampPct((z.YMax - worldY) / (z.YMax - z.YMin) * 100)
	mapY = clampPct((z.XMax - worldX) / (z.XMax - z.XMin) * 100)
	return z.Name, mapX, mapY, true
}

// zoneFromJSON finds the smallest-area zone in zones.json that contains the
// world point on the given map, and converts to 0-100 map percentage. Returns
// the matched zone's name, coords, its world area (for specificity comparison),
// and whether a match was found.
func (s *NpcService) zoneFromJSON(mapId int, worldX, worldY float64) (name string, mapX, mapY, area float64, ok bool) {
	s.loadZoneBounds()
	bestArea := math.MaxFloat64
	var best *zoneBound
	for i := range s.zoneBounds {
		z := &s.zoneBounds[i]
		if z.MapID != mapId || z.Name == "" {
			continue
		}
		if z.XMin == 0 && z.XMax == 0 { // instance / no bounds
			continue
		}
		if z.XMin < worldX && z.XMax > worldX && z.YMin < worldY && z.YMax > worldY {
			a := (z.XMax - z.XMin) * (z.YMax - z.YMin)
			if a > 0 && a < bestArea {
				bestArea = a
				best = z
			}
		}
	}
	if best == nil {
		return "", 0, 0, 0, false
	}
	mapX = clampPct((best.YMax - worldY) / (best.YMax - best.YMin) * 100)
	mapY = clampPct((best.XMax - worldX) / (best.XMax - best.XMin) * 100)
	return best.Name, mapX, mapY, bestArea, true
}

// continentBound returns the largest-area zone on a map — i.e. the continent
// itself (Azeroth on map 0, Kalimdor on map 1), which dwarfs every subzone.
func (s *NpcService) continentBound(mapID int) *zoneBound {
	var best *zoneBound
	var bestArea float64
	for i := range s.zoneBounds {
		z := &s.zoneBounds[i]
		if z.MapID != mapID || (z.XMin == 0 && z.XMax == 0) {
			continue
		}
		if a := (z.XMax - z.XMin) * (z.YMax - z.YMin); a > bestArea {
			bestArea = a
			best = z
		}
	}
	return best
}

// resolveContinentSpawn converts a continent-map percentage — what octowow
// reports (under zone 0) for spawns it can't pin to a subzone — back to world
// coordinates and finds the specific zone containing it. It tries both
// continents and keeps the most specific (smallest-area) match, which recovers
// custom octo zones (e.g. ThalassianHighlands) that aowow lumps into "Azeroth".
// Returns the zone's folder name and local 0-100 coords, or ok=false if no
// subzone contains the point on either continent.
func (s *NpcService) resolveContinentSpawn(cx, cy float64) (zoneName string, x, y float64, ok bool) {
	s.loadZoneBounds()
	bestArea := math.MaxFloat64
	for _, mapID := range []int{0, 1} {
		c := s.continentBound(mapID)
		if c == nil {
			continue
		}
		// Invert the continent projection (note the WoW axis swap: map X comes
		// from world Y and vice-versa).
		worldY := c.YMax - (cx/100)*(c.YMax-c.YMin)
		worldX := c.XMax - (cy/100)*(c.XMax-c.XMin)
		name, mx, my, area, found := s.zoneFromJSON(mapID, worldX, worldY)
		// Reject matches that resolve to the continent itself (no real subzone).
		if found && name != c.Name && area < bestArea {
			bestArea = area
			zoneName, x, y, ok = name, mx, my, true
		}
	}
	return zoneName, x, y, ok
}

func clampPct(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 100 {
		return 100
	}
	return v
}

// convertWorldToMapCoords converts world coordinates to map percentage coordinates (0-100)
// Using the aowow_zones table boundaries similar to the PHP coord_db2wow function
func (s *NpcService) convertWorldToMapCoords(mapId, zoneId int, worldX, worldY float64) (zoneName string, mapX, mapY float64) {
	// Most authoritative: the client area grid (per-chunk ADT areaIds). It resolves
	// the exact in-game zone even where zones' bounding boxes overlap, which the
	// box-containment heuristics below cannot. Falls through when the grid isn't
	// generated or the point has no terrain area (ocean / WMO-only instance map).
	if name, gx, gy, ok := s.zoneFromAreaGrid(mapId, worldX, worldY); ok {
		return name, gx, gy
	}

	// Primary geometry source: client-authoritative zones.json. It contains
	// zones the live aowow_zones import lacks (Mount Hyjal, custom octo zones),
	// so we prefer it when it finds a more-specific (smaller) zone than MySQL.
	jName, jX, jY, jArea, jOk := s.zoneFromJSON(mapId, worldX, worldY)

	if s.mysql == nil {
		if jOk {
			return jName, jX, jY
		}
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
		mArea := (xMax - xMin) * (yMax - yMin)

		// Prefer the JSON match when it found a meaningfully more-specific zone
		// than MySQL — e.g. a Mount Hyjal spawn that MySQL snaps to the much
		// larger Felwood box. The 0.9 factor keeps MySQL's nicer display names
		// when both resolve to the same zone (bounds may jitter slightly).
		if jOk && jArea < mArea*0.9 {
			return jName, jX, jY
		}

		// Convert coordinates
		// WoW World (MySQL) -> Map Percentage (0-100)
		// Standard Formula:
		// MapX = (y_max - worldY) / (y_max - y_min) * 100
		// MapY = (x_max - worldX) / (x_max - x_min) * 100

		mapX = clampPct((yMax - worldY) / (yMax - yMin) * 100)
		mapY = clampPct((xMax - worldX) / (xMax - xMin) * 100)

		return name, mapX, mapY
	}

	// MySQL found no bounded match — fall back to the JSON geometry if it had one.
	if jOk {
		return jName, jX, jY
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

// syncGameObjectSpawnsFromMySQL syncs game-object spawn coordinates from the
// MySQL `gameobject` table (id = gameobject_template.entry), converting world
// coords to 0-100 map percentages — the same pipeline as creature spawns.
func (s *NpcService) syncGameObjectSpawnsFromMySQL(entry int) {
	if s.mysql == nil {
		return
	}

	// Replace existing spawns for this object.
	s.sqlite.Exec("DELETE FROM gameobject_spawn WHERE gameobject_entry = ?", entry)

	rows, err := s.mysql.DB().Query(`
		SELECT map, AVG(position_x) as avg_x, AVG(position_y) as avg_y, AVG(position_z) as avg_z
		FROM gameobject
		WHERE id = ?
		GROUP BY map, ROUND(position_x, -1), ROUND(position_y, -1)
		LIMIT 50
	`, entry)
	if err != nil {
		fmt.Printf("Warning: Could not query gameobject spawns from MySQL: %v\n", err)
		return
	}
	defer rows.Close()

	spawnCount := 0
	for rows.Next() {
		var mapId int
		var worldX, worldY, z float64
		if err := rows.Scan(&mapId, &worldX, &worldY, &z); err != nil {
			continue
		}
		zoneName, mapX, mapY := s.convertWorldToMapCoords(mapId, 0, worldX, worldY)
		if _, err := s.sqlite.Exec(`
			INSERT INTO gameobject_spawn (gameobject_entry, map_id, zone_id, zone_name, position_x, position_y, position_z)
			VALUES (?, ?, ?, ?, ?, ?, ?)
		`, entry, mapId, 0, zoneName, mapX, mapY, z); err == nil {
			spawnCount++
		}
	}
	if spawnCount > 0 {
		fmt.Printf("  ✓ Synced %d spawn points for gameobject %d\n", spawnCount, entry)
	}
}

// SyncObjectSpawnsFromWeb scrapes a game object's spawn points from octowow.st
// and replaces its gameobject_spawn rows. Unlike the MySQL path this needs no
// coordinate conversion — aowow already provides per-zone map percentages — and
// it groups by the authoritative zone areatableID, which we resolve to a folder
// name. This is the spawn source for users without a MySQL connection.
func (s *NpcService) SyncObjectSpawnsFromWeb(entry int) (int, error) {
	if s.scraper == nil {
		return 0, fmt.Errorf("no scraper available")
	}
	points, err := s.scraper.ScrapeObjectSpawns(entry)
	if err != nil {
		return 0, err
	}

	s.sqlite.Exec("DELETE FROM gameobject_spawn WHERE gameobject_entry = ?", entry)

	n := 0
	for _, p := range points {
		zoneName := s.zoneNameByID(p.ZoneID)
		x, y := p.X, p.Y
		// octowow buckets spawns it can't map to a subzone under zone 0 (the
		// continent) with continent-level coords. Recover the real zone + local
		// coords from the geometry — aowow lacks the custom octo zones, we don't.
		if p.ZoneID == 0 {
			if zn, zx, zy, ok := s.resolveContinentSpawn(p.X, p.Y); ok {
				zoneName, x, y = zn, zx, zy
			}
		}
		if _, err := s.sqlite.Exec(`
			INSERT OR IGNORE INTO gameobject_spawn (gameobject_entry, map_id, zone_id, zone_name, position_x, position_y, position_z)
			VALUES (?, ?, ?, ?, ?, ?, ?)
		`, entry, 0, p.ZoneID, zoneName, x, y, 0); err == nil {
			n++
		}
	}
	fmt.Printf("✓ Web-synced %d spawn points for gameobject %d\n", n, entry)
	return n, nil
}

// zoneNameByID resolves an areatableID to its client texture-folder name via
// quest_categories_enhanced (populated from zones.json). Zone 0 is the
// continent/world bucket aowow uses for unzoned spawns.
func (s *NpcService) zoneNameByID(zoneID int) string {
	if zoneID == 0 {
		return "Azeroth"
	}
	var name string
	s.sqlite.QueryRow("SELECT name FROM quest_categories_enhanced WHERE id = ?", zoneID).Scan(&name)
	return name
}

// SyncAllGameObjectSpawns syncs spawn points for all game objects.
func (s *NpcService) SyncAllGameObjectSpawns(progressCb func(current, total int, id int)) error {
	if s.mysql == nil {
		return fmt.Errorf("no mysql connection")
	}

	rows, err := s.sqlite.Query("SELECT entry FROM gameobject_template ORDER BY entry")
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
		s.syncGameObjectSpawnsFromMySQL(entry)
		if progressCb != nil && i%10 == 0 {
			progressCb(i+1, total, entry)
		}
	}
	if progressCb != nil {
		progressCb(total, total, 0)
	}
	return nil
}

// FullSyncNpcs performs a full sync (scrape + DB) for all NPCs starting from a
// specific ID. NPC sync is network-bound (web scrape + MySQL), so it runs over a
// worker pool like the item sync; sql.DB is safe for concurrent use and mu guards
// the shared progress counter. Honors the stop flag.
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
	const numWorkers = 10
	jobs := make(chan int, total)
	var wg sync.WaitGroup
	var mu sync.Mutex
	processed := 0

	worker := func() {
		defer wg.Done()
		for entry := range jobs {
			if s.IsStopped() {
				return
			}
			s.SyncNpcData(entry)
			mu.Lock()
			processed++
			if progressCb != nil {
				progressCb(processed, total, entry)
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
	for _, entry := range entries {
		jobs <- entry
	}
	close(jobs)
	wg.Wait()

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
	scrapeOK := err == nil
	if err != nil {
		fmt.Printf("Scrape failed: %v\n", err)
		scrapedData = &ScrapedNpcData{Infobox: make(map[string]string)}
	}

	// Store what this NPC sells into item_vendor — the same table the item
	// "sold by" scrape writes, so syncing either side fills the relationship.
	// Only refresh on a successful scrape, so a failed fetch can't wipe a real
	// vendor's inventory.
	if scrapeOK {
		s.sqlite.Exec("DELETE FROM item_vendor WHERE npc_entry = ?", entry)
		for _, sale := range scrapedData.Sells {
			s.sqlite.Exec(
				`INSERT OR REPLACE INTO item_vendor (item_entry, npc_entry, cost, stock) VALUES (?, ?, ?, ?)`,
				sale.ItemEntry, entry, sale.Cost, sale.Stock)
		}
	}

	// Model and map images are not handled here. Model renders are produced
	// locally from the client MPQs on demand (and via the bulk render job) keyed
	// by CreatureDisplayInfo id — see RenderModelOnDemand / useNpcModel — so a
	// per-entry model_<id>.png would be both redundant and wrongly keyed. The NPC
	// view's map is a locally-generated zone map. We keep only scraped metadata.
	localModelPath := ""
	localMapPath := ""

	// Store Metadata to SQLite (columns are ensured once in ensureSchema).
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
	// A. Scrape + metadata (no creature_template changes). Non-fatal: a failure
	// here (e.g. scrape hiccup, transient write contention) must NOT skip the
	// MySQL stats + spawn sync below — that's how a full sync could leave an NPC
	// without its spawn/zone while a manual re-sync fixed it.
	if err := s.syncNpcImages(entry); err != nil {
		fmt.Printf("[SyncNpcData] metadata step failed for %d (continuing to MySQL sync): %v\n", entry, err)
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

// resolveCreatureWeapons builds the held-weapon attachments for a creature from
// its equipment template: equipentry1 = main hand (right), equipentry2 = off hand
// (left). Each equipped item's display id resolves to a weapon model + texture
// via the client DBCs. Returns nil if the creature has no equipment or no item
// resolves to a model.
func (s *NpcService) resolveCreatureWeapons(cf datatools.ClientFiles, entry int) []datatools.AttachedItem {
	var eq1, eq2 int
	err := s.sqlite.QueryRow(`
		SELECT et.equipentry1, et.equipentry2
		FROM creature_template ct
		JOIN creature_equip_template et ON ct.equipment_id = et.entry
		WHERE ct.entry = ?`, entry).Scan(&eq1, &eq2)
	if err != nil {
		return nil
	}
	var out []datatools.AttachedItem
	// item class: 2 = weapon, 4 = armor (subclass 6 = shield).
	const classArmor = 4
	resolve := func(itemEntry int, offHand bool) {
		if itemEntry <= 0 {
			return
		}
		var displayID, class int
		if s.sqlite.QueryRow("SELECT display_id, class FROM item_template WHERE entry = ?", itemEntry).Scan(&displayID, &class) != nil || displayID == 0 {
			return
		}
		if offHand && class == classArmor {
			if sh, ok := datatools.ResolveShield(cf, displayID); ok {
				out = append(out, sh)
			}
			return
		}
		attach := uint32(1) // main hand → right
		if offHand {
			attach = 2 // off-hand weapon → left hand
		}
		if w, ok := datatools.ResolveWeapon(cf, displayID, attach); ok {
			out = append(out, w)
		}
	}
	resolve(eq1, false)
	resolve(eq2, true)
	return out
}

// renderJob is one model-render task: a creatureEntry of 0 means a display-keyed
// render (model_<displayID>.png, body + display armor); a non-zero creatureEntry
// means a per-creature render (model_creature_<entry>.png) that additionally
// draws the creature's held weapons.
type renderJob struct {
	creatureEntry int
	displayID     int
}

// RenderAllNpcModels renders creature models from the client MPQs (at
// <baseDir>/Data) into data/npc_images — a "warm the whole cache" pass, since
// pages also render on demand. It runs in two phases over one worker pool: (1)
// one render per distinct display into model_<displayId>.png (body + display
// armor incl. shoulders); (2) one render per creature that has held weapons into
// model_creature_<entry>.png (body + armor + weapons). All renders are local;
// displays the software renderer can't handle simply produce no file (the UI
// shows a placeholder). Existing files are skipped; honors the stop flag.
// delayMs is unused (kept for API compatibility) now that nothing downloads.
func (s *NpcService) RenderAllNpcModels(baseDir string, startFrom, delayMs int, progressCb func(current, total, displayID int)) error {
	dataPath := filepath.Join(baseDir, "Data")
	// Validate the client path up front so the caller gets a clear error.
	if probe, err := datatools.NewMpqSource(dataPath); err != nil {
		return fmt.Errorf("open client MPQs at %s: %w", baseDir, err)
	} else {
		probe.Close()
	}

	rows, err := s.sqlite.Query(
		"SELECT DISTINCT display_id1 FROM creature_template WHERE display_id1 >= ? AND display_id1 > 0 ORDER BY display_id1",
		startFrom)
	if err != nil {
		return err
	}
	var allJobs []renderJob
	for rows.Next() {
		var d int
		if rows.Scan(&d) == nil {
			allJobs = append(allJobs, renderJob{displayID: d})
		}
	}
	rows.Close()

	// Phase 2 jobs: creatures whose equipment template has at least one item.
	eqRows, err := s.sqlite.Query(`
		SELECT ct.entry, ct.display_id1
		FROM creature_template ct
		JOIN creature_equip_template et ON ct.equipment_id = et.entry
		WHERE ct.display_id1 > 0
		  AND (et.equipentry1 > 0 OR et.equipentry2 > 0 OR et.equipentry3 > 0)
		ORDER BY ct.entry`)
	if err == nil {
		for eqRows.Next() {
			var entry, disp int
			if eqRows.Scan(&entry, &disp) == nil {
				allJobs = append(allJobs, renderJob{creatureEntry: entry, displayID: disp})
			}
		}
		eqRows.Close()
	}

	npcImagesDir := filepath.Join(s.dataDir, "npc_images")
	if err := os.MkdirAll(npcImagesDir, 0755); err != nil {
		return err
	}
	opt := datatools.DefaultRenderOptions()
	total := len(allJobs)

	// Rasterizing is CPU-bound, so fan out across cores. Each worker opens its
	// OWN MPQ set — model reads are hash-table lookups, so no listfile scan and
	// no shared mutable state between workers.
	workers := runtime.NumCPU() - 1
	if workers < 1 {
		workers = 1
	}
	if workers > 8 {
		workers = 8
	}
	jobs := make(chan renderJob, 256)
	var done int64
	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			src, err := datatools.NewMpqSource(dataPath)
			if err != nil {
				return
			}
			defer src.Close()
			for j := range jobs {
				if !s.IsStopped() {
					if j.creatureEntry == 0 {
						s.renderDisplayJob(src, npcImagesDir, j.displayID, opt)
					} else {
						s.renderCreatureWeaponJob(src, npcImagesDir, j.creatureEntry, j.displayID, opt)
					}
				}
				cur := atomic.AddInt64(&done, 1)
				if progressCb != nil {
					progressCb(int(cur), total, j.displayID)
				}
			}
		}()
	}
	for _, j := range allJobs {
		if s.IsStopped() {
			break
		}
		jobs <- j
	}
	close(jobs)
	wg.Wait()
	return nil
}

// renderDisplayJob renders one display to model_<displayID>.png (skipping if it
// exists). Local render only — a failure (e.g. a humanoid without a baked skin)
// leaves no file and the UI shows a placeholder. No remote fallback.
func (s *NpcService) renderDisplayJob(src datatools.ClientFiles, dir string, displayID int, opt datatools.RenderOptions) {
	out := filepath.Join(dir, fmt.Sprintf("model_%d.png", displayID))
	if _, statErr := os.Stat(out); statErr == nil {
		return
	}
	_ = datatools.RenderCreatureModelToFile(src, displayID, out, opt)
}

// renderCreatureWeaponJob renders a creature's body + armor + held weapons to
// model_creature_<entry>.png. It's a no-op when the creature resolves no weapon
// models (the display-keyed image already covers body + armor) or the model
// can't be rendered locally.
func (s *NpcService) renderCreatureWeaponJob(src datatools.ClientFiles, dir string, entry, displayID int, opt datatools.RenderOptions) {
	out := filepath.Join(dir, fmt.Sprintf("model_creature_%d.png", entry))
	if _, statErr := os.Stat(out); statErr == nil {
		return
	}
	weapons := s.resolveCreatureWeapons(src, entry)
	if len(weapons) == 0 {
		return
	}
	cm, err := datatools.ResolveCreatureModel(src, displayID)
	if err != nil {
		return
	}
	cm.Attachments = append(cm.Attachments, weapons...)
	img, err := datatools.RenderResolvedModel(src, cm, opt)
	if err != nil {
		return
	}
	f, err := os.Create(out)
	if err != nil {
		return
	}
	defer f.Close()
	_ = png.Encode(f, img)
}

// RenderModelOnDemand renders a single creature's model into data/npc_images if
// not already cached: a per-creature image (with held weapons) when the creature
// has equipment, plus the shared display image. Local-only — no network. The
// caller must serialize calls (the MPQ source is not concurrency-safe).
func (s *NpcService) RenderModelOnDemand(cf datatools.ClientFiles, entry, displayID int) {
	if displayID <= 0 {
		return
	}
	dir := filepath.Join(s.dataDir, "npc_images")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return
	}
	opt := datatools.DefaultRenderOptions()
	if entry > 0 {
		s.renderCreatureWeaponJob(cf, dir, entry, displayID, opt)
	}
	out := filepath.Join(dir, fmt.Sprintf("model_%d.png", displayID))
	if _, statErr := os.Stat(out); statErr != nil {
		_ = datatools.RenderCreatureModelToFile(cf, displayID, out, opt)
	}
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
