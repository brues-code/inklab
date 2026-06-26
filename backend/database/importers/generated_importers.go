package importers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
)

type GeneratedImporter struct {
	db *sql.DB
}

func NewGeneratedImporter(db *sql.DB) *GeneratedImporter {
	return &GeneratedImporter{db: db}
}

// ImportItemIcons loads display-id -> icon mappings from item_icons.json into
// the item_display_info table (the table item queries join against for icons).
// This is the client ItemDisplayInfo.dbc data and is more complete than the
// server-side copy, so it upserts over whatever the MySQL import provided.
func (i *GeneratedImporter) ImportItemIcons(jsonPath string) error {
	fmt.Printf("  -> Reading item icons from %s...\n", jsonPath)
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		return nil // Icons are optional
	}

	var iconMap map[string]string
	if err := json.Unmarshal(data, &iconMap); err != nil {
		fmt.Printf("  ERROR parsing item_icons.json: %v\n", err)
		return nil
	}

	fmt.Printf("  -> Upserting %d icon mappings into item_display_info...\n", len(iconMap))
	tx, err := i.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare("INSERT OR REPLACE INTO item_display_info (ID, icon) VALUES (?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	count := 0
	for displayIDStr, iconName := range iconMap {
		var displayID int
		fmt.Sscanf(displayIDStr, "%d", &displayID)
		if displayID > 0 {
			if _, err := stmt.Exec(displayID, iconName); err != nil {
				continue
			}
			count++
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	fmt.Printf("  ✓ Successfully wrote %d item icon mappings\n", count)
	return nil
}

// SpellEnhanced represents a spell record from spells_enhanced.json. The fields
// mirror exactly what the $-placeholder resolver reads, so the client DBC can be
// the authoritative source for spell text + values.
type SpellEnhanced struct {
	Entry              int    `json:"entry"`
	Name               string `json:"name"`
	Description        string `json:"description"`
	BasePoints1        int    `json:"effectBasePoints1"`
	BasePoints2        int    `json:"effectBasePoints2"`
	BasePoints3        int    `json:"effectBasePoints3"`
	DieSides1          int    `json:"effectDieSides1"`
	DieSides2          int    `json:"effectDieSides2"`
	DieSides3          int    `json:"effectDieSides3"`
	Amplitude1         int    `json:"effectAmplitude1"`
	Amplitude2         int    `json:"effectAmplitude2"`
	Amplitude3         int    `json:"effectAmplitude3"`
	ChainTarget1       int    `json:"effectChainTarget1"`
	ChainTarget2       int    `json:"effectChainTarget2"`
	ChainTarget3       int    `json:"effectChainTarget3"`
	RadiusIndex1       int    `json:"effectRadiusIndex1"`
	RadiusIndex2       int    `json:"effectRadiusIndex2"`
	RadiusIndex3       int    `json:"effectRadiusIndex3"`
	DurationIndex      int    `json:"durationIndex"`
	RangeIndex         int    `json:"rangeIndex"`
	ProcChance         int    `json:"procChance"`
	ProcCharges        int    `json:"procCharges"`
	MaxTargetLevel     int    `json:"maxTargetLevel"`
	MaxAffectedTargets int    `json:"maxAffectedTargets"`
	SpellIconId        int    `json:"spellIconId"`
	IconName           string `json:"iconName"`
}

// ImportSpellsFromDBC makes the client Spell.dbc the authoritative source for
// spell text and the values the $-placeholder resolver reads. On a custom server
// the client DBC is patched to match what the server actually does, so it is the
// latest/correct data — newer than the frozen MySQL world-DB import. For every
// DBC spell it overwrites name/description/effect values/proc/range/icon on the
// existing row (DBC wins), and inserts spells the MySQL import lacked. The raw
// $-templates it writes are turned into readable text by the spell resolver pass
// that runs immediately after (FullSyncSpells). Name/description/icon are only
// overwritten when the DBC actually provides them, so a DBC gap can't blank good
// data.
func (i *GeneratedImporter) ImportSpellsFromDBC(jsonPath string) error {
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		return nil // optional — only present after a client import
	}
	var spells []SpellEnhanced
	if err := json.Unmarshal(data, &spells); err != nil {
		fmt.Printf("  ERROR parsing spells_enhanced.json: %v\n", err)
		return nil
	}

	tx, err := i.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// DBC wins for existing rows; NULLIF('') keeps prior text/icon when the DBC
	// has none.
	upd, err := tx.Prepare(`UPDATE spell_template SET
		name = COALESCE(NULLIF(?,''), name),
		description = COALESCE(NULLIF(?,''), description),
		effectBasePoints1=?, effectBasePoints2=?, effectBasePoints3=?,
		effectDieSides1=?, effectDieSides2=?, effectDieSides3=?,
		effectAmplitude1=?, effectAmplitude2=?, effectAmplitude3=?,
		effectChainTarget1=?, effectChainTarget2=?, effectChainTarget3=?,
		effectRadiusIndex1=?, effectRadiusIndex2=?, effectRadiusIndex3=?,
		durationIndex=?, rangeIndex=?, procChance=?, procCharges=?,
		maxTargetLevel=?, maxAffectedTargets=?,
		spellIconId=?, iconName=COALESCE(NULLIF(?,''), iconName)
		WHERE entry=?`)
	if err != nil {
		return err
	}
	defer upd.Close()
	ins, err := tx.Prepare(`INSERT OR IGNORE INTO spell_template
		(entry, name, description, effectBasePoints1, effectBasePoints2, effectBasePoints3,
		 effectDieSides1, effectDieSides2, effectDieSides3,
		 effectAmplitude1, effectAmplitude2, effectAmplitude3,
		 effectChainTarget1, effectChainTarget2, effectChainTarget3,
		 effectRadiusIndex1, effectRadiusIndex2, effectRadiusIndex3,
		 durationIndex, rangeIndex, procChance, procCharges,
		 maxTargetLevel, maxAffectedTargets, spellIconId, iconName)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`)
	if err != nil {
		return err
	}
	defer ins.Close()

	updated, inserted := 0, 0
	for _, s := range spells {
		if s.Entry <= 0 {
			continue
		}
		icon := s.IconName
		if icon == "temp" {
			icon = ""
		}
		res, err := upd.Exec(s.Name, s.Description,
			s.BasePoints1, s.BasePoints2, s.BasePoints3,
			s.DieSides1, s.DieSides2, s.DieSides3,
			s.Amplitude1, s.Amplitude2, s.Amplitude3,
			s.ChainTarget1, s.ChainTarget2, s.ChainTarget3,
			s.RadiusIndex1, s.RadiusIndex2, s.RadiusIndex3,
			s.DurationIndex, s.RangeIndex, s.ProcChance, s.ProcCharges,
			s.MaxTargetLevel, s.MaxAffectedTargets, s.SpellIconId, icon, s.Entry)
		if err != nil {
			continue
		}
		if n, _ := res.RowsAffected(); n > 0 {
			updated++
			continue
		}
		// No existing row — insert the DBC spell.
		if _, err := ins.Exec(s.Entry, s.Name, s.Description,
			s.BasePoints1, s.BasePoints2, s.BasePoints3,
			s.DieSides1, s.DieSides2, s.DieSides3,
			s.Amplitude1, s.Amplitude2, s.Amplitude3,
			s.ChainTarget1, s.ChainTarget2, s.ChainTarget3,
			s.RadiusIndex1, s.RadiusIndex2, s.RadiusIndex3,
			s.DurationIndex, s.RangeIndex, s.ProcChance, s.ProcCharges,
			s.MaxTargetLevel, s.MaxAffectedTargets, s.SpellIconId, icon); err == nil {
			inserted++
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	fmt.Printf("  ✓ Spell DBC import: %d updated, %d inserted (client DBC wins)\n", updated, inserted)
	return nil
}

// talentTabJSON mirrors a record in data/talents.json (datatools.TalentTabOut).
type talentTabJSON struct {
	ID         int    `json:"id"`
	Name       string `json:"name"`
	Class      string `json:"class"`
	ClassMask  int    `json:"classMask"`
	Order      int    `json:"order"`
	Background string `json:"background"`
	Talents    []struct {
		ID        int   `json:"id"`
		Row       int   `json:"row"`
		Col       int   `json:"col"`
		Ranks     []int `json:"ranks"`
		ReqTalent int   `json:"reqTalent"`
		ReqRank   int   `json:"reqRank"`
	} `json:"talents"`
}

// ImportTalents loads the DBC-derived talent trees from talents.json into the
// talent_tab / talent tables. The per-rank spell ids are stored as a JSON array;
// name/description/icon are resolved at query time from spell_template. Existing
// rows are replaced so a re-import refreshes the data.
func (i *GeneratedImporter) ImportTalents(jsonPath string) error {
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		return nil // optional — talents only present after a client import
	}
	var tabs []talentTabJSON
	if err := json.Unmarshal(data, &tabs); err != nil {
		fmt.Printf("  ERROR parsing talents.json: %v\n", err)
		return nil
	}

	tx, err := i.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Clear and repopulate (small, fully derived data set).
	tx.Exec("DELETE FROM talent")
	tx.Exec("DELETE FROM talent_tab")

	tabStmt, err := tx.Prepare(`INSERT OR REPLACE INTO talent_tab
		(id, name, class, class_mask, order_index, background) VALUES (?,?,?,?,?,?)`)
	if err != nil {
		return err
	}
	defer tabStmt.Close()
	talStmt, err := tx.Prepare(`INSERT OR REPLACE INTO talent
		(id, tab_id, row, col, ranks, req_talent, req_rank) VALUES (?,?,?,?,?,?,?)`)
	if err != nil {
		return err
	}
	defer talStmt.Close()

	tabCount, talCount := 0, 0
	for _, t := range tabs {
		if _, err := tabStmt.Exec(t.ID, t.Name, t.Class, t.ClassMask, t.Order, t.Background); err != nil {
			continue
		}
		tabCount++
		for _, tal := range t.Talents {
			ranks, _ := json.Marshal(tal.Ranks)
			if _, err := talStmt.Exec(tal.ID, t.ID, tal.Row, tal.Col, string(ranks), tal.ReqTalent, tal.ReqRank); err != nil {
				continue
			}
			talCount++
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	fmt.Printf("  ✓ Imported %d talent trees, %d talents\n", tabCount, talCount)
	return nil
}

// taxiJSON mirrors data/taxi.json (datatools.TaxiData).
type taxiJSON struct {
	Continents []struct {
		MapID int    `json:"mapId"`
		Name  string `json:"name"`
	} `json:"continents"`
	Nodes []struct {
		ID       int     `json:"id"`
		MapID    int     `json:"mapId"`
		Name     string  `json:"name"`
		Alliance bool    `json:"alliance"`
		Horde    bool    `json:"horde"`
		PX       float64 `json:"px"`
		PY       float64 `json:"py"`
	} `json:"nodes"`
	Paths []struct {
		ID   int `json:"id"`
		From int `json:"from"`
		To   int `json:"to"`
	} `json:"paths"`
	Waypoints []struct {
		PathID int     `json:"pathId"`
		Idx    int     `json:"idx"`
		PX     float64 `json:"px"`
		PY     float64 `json:"py"`
	} `json:"waypoints"`
	Transports []struct {
		ID    int    `json:"id"`
		Type  string `json:"type"`
		NameA string `json:"nameA"`
		MapA  int    `json:"mapA"`
		NameB string `json:"nameB"`
		MapB  int    `json:"mapB"`
	} `json:"transports"`
	TransportWps []struct {
		RouteID int     `json:"routeId"`
		Idx     int     `json:"idx"`
		MapID   int     `json:"mapId"`
		PX      float64 `json:"px"`
		PY      float64 `json:"py"`
	} `json:"transportWps"`
}

// ImportTaxi loads the DBC-derived flight network from taxi.json into the
// taxi_node / taxi_path / taxi_path_node tables. Clears and repopulates (small,
// fully derived). Continent name is resolved from the continents list by map id.
func (i *GeneratedImporter) ImportTaxi(jsonPath string) error {
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		return nil // optional — only present after a client import
	}
	var t taxiJSON
	if err := json.Unmarshal(data, &t); err != nil {
		fmt.Printf("  ERROR parsing taxi.json: %v\n", err)
		return nil
	}
	contName := map[int]string{}
	for _, c := range t.Continents {
		contName[c.MapID] = c.Name
	}

	tx, err := i.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	tx.Exec("DELETE FROM taxi_node")
	tx.Exec("DELETE FROM taxi_path")
	tx.Exec("DELETE FROM taxi_path_node")
	tx.Exec("DELETE FROM transport_route")
	tx.Exec("DELETE FROM transport_waypoint")

	nodeStmt, _ := tx.Prepare(`INSERT OR REPLACE INTO taxi_node
		(id, map_id, continent, name, alliance, horde, px, py) VALUES (?,?,?,?,?,?,?,?)`)
	defer nodeStmt.Close()
	pathStmt, _ := tx.Prepare(`INSERT OR REPLACE INTO taxi_path (id, from_node, to_node) VALUES (?,?,?)`)
	defer pathStmt.Close()
	wpStmt, _ := tx.Prepare(`INSERT OR REPLACE INTO taxi_path_node (path_id, idx, px, py) VALUES (?,?,?,?)`)
	defer wpStmt.Close()

	b2i := func(b bool) int {
		if b {
			return 1
		}
		return 0
	}
	for _, n := range t.Nodes {
		nodeStmt.Exec(n.ID, n.MapID, contName[n.MapID], n.Name, b2i(n.Alliance), b2i(n.Horde), n.PX, n.PY)
	}
	for _, p := range t.Paths {
		pathStmt.Exec(p.ID, p.From, p.To)
	}
	for _, w := range t.Waypoints {
		wpStmt.Exec(w.PathID, w.Idx, w.PX, w.PY)
	}

	trStmt, _ := tx.Prepare(`INSERT OR REPLACE INTO transport_route
		(id, type, name_a, map_a, name_b, map_b) VALUES (?,?,?,?,?,?)`)
	defer trStmt.Close()
	twStmt, _ := tx.Prepare(`INSERT OR REPLACE INTO transport_waypoint
		(route_id, idx, map_id, px, py) VALUES (?,?,?,?,?)`)
	defer twStmt.Close()
	for _, tr := range t.Transports {
		trStmt.Exec(tr.ID, tr.Type, tr.NameA, tr.MapA, tr.NameB, tr.MapB)
	}
	for _, w := range t.TransportWps {
		twStmt.Exec(w.RouteID, w.Idx, w.MapID, w.PX, w.PY)
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	fmt.Printf("  ✓ Imported %d flight nodes, %d paths, %d waypoints; %d transports\n",
		len(t.Nodes), len(t.Paths), len(t.Waypoints), len(t.Transports))
	return nil
}

// ImportSpellIcons loads spell icons from spells_enhanced.json and updates spell_template
func (i *GeneratedImporter) ImportSpellIcons(jsonPath string) error {
	fmt.Printf("  -> Reading spell icons from %s...\n", jsonPath)
	file, err := os.Open(jsonPath)
	if err != nil {
		return nil // Optional
	}
	defer file.Close()

	var spells []SpellEnhanced
	if err := json.NewDecoder(file).Decode(&spells); err != nil {
		fmt.Printf("  ERROR parsing spells_enhanced.json: %v\n", err)
		return nil
	}

	// Build unique icon map
	iconMap := make(map[int]string)
	for _, s := range spells {
		if s.SpellIconId > 0 && s.IconName != "" && s.IconName != "temp" {
			iconMap[s.SpellIconId] = s.IconName
		}
	}

	fmt.Printf("  -> Updating database with %d spell icon mappings...\n", len(iconMap))
	tx, err := i.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare("UPDATE spell_template SET iconName = ? WHERE spellIconId = ?")
	if err != nil {
		return err
	}
	defer stmt.Close()

	// Also backfill the spell_icons lookup table from the same complete,
	// DBC-derived mapping. It is otherwise sourced from the live
	// aowow.aowow_spellicons table, which is an incomplete import (missing icon
	// ids such as 2164 / Spell_Holy_CrusaderStrike).
	iconStmt, err := tx.Prepare("INSERT OR REPLACE INTO spell_icons (id, icon_name) VALUES (?, ?)")
	if err != nil {
		return err
	}
	defer iconStmt.Close()

	count := 0
	iconCount := 0
	for iconId, iconName := range iconMap {
		res, err := stmt.Exec(iconName, iconId)
		if err == nil {
			if rows, _ := res.RowsAffected(); rows > 0 {
				count++
			}
		}
		if _, err := iconStmt.Exec(iconId, iconName); err == nil {
			iconCount++
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	fmt.Printf("  ✓ Updated %d spells with icons; backfilled %d spell_icons rows\n", count, iconCount)
	return nil
}
