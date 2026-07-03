package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"inklab/backend/database"
	"inklab/backend/datatools"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// ImportReport is the generic result of a Tools-tab data import.
type ImportReport struct {
	Success bool     `json:"success"`
	Title   string   `json:"title"`
	Lines   []string `json:"lines"`
}

// DataItem is one client-derived dataset's current state, so the Import tab can
// show exactly what's present or missing — and, for a missing one, which client
// file (DBC / interface art) feeds it.
type DataItem struct {
	Key    string `json:"key"`
	Label  string `json:"label"`
	Source string `json:"source"` // client file(s) this comes from
	Kind   string `json:"kind"`   // "image" (file count) | "table" (DB rows)
	Count  int    `json:"count"`
}

// DataStatus reports how much locally-built data InkLab currently has, so the UI
// can tell the user when a category is empty and needs importing. The flat image
// counts are kept for the summary badges; Datasets is the full per-dataset
// inventory (images + DBC-derived reference tables) for the "what's missing" view.
type DataStatus struct {
	Icons     int        `json:"icons"`
	Maps      int        `json:"maps"`
	NpcImages int        `json:"npcImages"`
	TalentBgs int        `json:"talentBgs"`
	Datasets  []DataItem `json:"datasets"`
}

// GetDataStatus reports the local image counts plus a full inventory of every
// client-derived dataset (image dirs and DBC-fed reference tables) with the
// source file each comes from, so a user can see precisely what's missing.
func (a *App) GetDataStatus() DataStatus {
	icons := countFiles(filepath.Join(a.DataDir, "icons"), ".jpg", ".png", ".jpeg")
	maps := countFiles(filepath.Join(a.DataDir, "maps"), ".jpg", ".png")
	npcImages := countFiles(filepath.Join(a.DataDir, "npc_images"), ".jpg", ".png", ".jpeg")
	talentBgs := countFiles(filepath.Join(a.DataDir, "talent_bg"), ".png", ".jpg")

	datasets := []DataItem{
		// Image data (file counts under data/).
		{"icons", "Item / spell icons", `Interface\Icons\*.blp`, "image", icons},
		{"maps", "Zone maps", `Interface\WorldMap + WorldMapArea.dbc`, "image", maps},
		{"minimaps", "Zone minimaps (terrain)", `textures\Minimap + WorldMapArea.dbc`, "image", countFiles(filepath.Join(a.DataDir, "minimaps"), ".jpg", ".png")},
		{"talentBgs", "Talent backgrounds", `Interface\TalentFrame art`, "image", talentBgs},
		{"raceIcons", "Race / gender icons", `UI-CharacterCreate-Races.blp`, "image", countFiles(filepath.Join(a.DataDir, "race_icons"), ".png")},
		{"coinIcons", "Coin icons (g/s/c)", `UI-MoneyIcons.blp`, "image", countFiles(filepath.Join(a.DataDir, "coin_icons"), ".png")},
		// DBC-derived reference tables (row counts).
		{"itemIcons", "Item icon mappings", "ItemDisplayInfo.dbc", "table", a.countRows("item_display_info")},
		{"spellIcons", "Spell icon mappings", "Spell.dbc + SpellIcon.dbc", "table", a.countRows("spell_icons")},
		{"skills", "Skills", "SkillLine.dbc", "table", a.countRows("spell_skills")},
		{"itemTypeNames", "Item type names (localized)", "ItemClass + ItemSubClass.dbc", "table", a.countRows("item_subclass_names")},
		{"slotNames", "Equip-slot names (localized)", "GlobalStrings.lua (INVTYPE)", "table", a.countRows("inventory_type_names")},
		{"creatureTypeNames", "Creature type names (localized)", "CreatureType.dbc", "table", a.countRows("creature_type_names")},
		{"clientStrings", "UI strings (quality/bind/trigger)", "GlobalStrings.lua", "table", a.countRows("client_strings")},
		{"proficiencies", "Class proficiencies (weapon/armor usability)", "SkillLineAbility.dbc", "table", a.countRowsWhere("spell_skill_spells", "classmask > 0")},
		{"zones", "Zones (localized names)", "WorldMapArea.dbc + AreaTable.dbc", "table", a.countRowsWhere("quest_categories_enhanced", "display_name != ''")},
		{"talents", "Talents", "Talent.dbc + TalentTab.dbc", "table", a.countRows("talent")},
		{"itemSets", "Item sets", "ItemSet.dbc", "table", a.countRows("itemsets")},
		{"factions", "Factions", "Faction.dbc", "table", a.countRows("factions")},
		{"factionTemplates", "Faction templates (NPC reactions)", "FactionTemplate.dbc", "table", a.countRows("faction_template")},
		{"areaGrid", "Spawn zone grid", `World\Maps\*.adt terrain`, "table", a.areaGridTiles()},
		{"taxi", "Flight (taxi) nodes", "TaxiNodes.dbc", "table", a.countRows("taxi_node")},
		{"creatureFamilies", "Creature families", "CreatureFamily.dbc", "table", a.countRows("creature_family")},
		{"locks", "Locks", "Lock.dbc", "table", a.countRows("locks")},
		{"classes", "Classes", "ChrClasses.dbc", "table", a.countRows("class_info")},
		{"races", "Races", "ChrRaces.dbc + CharBaseInfo.dbc + glue strings", "table", a.countRows("races")},
		{"spellSchools", "Spell school names", "GlobalStrings.lua", "table", a.countRows("spell_schools")},
		{"spellMechanics", "Spell mechanics", "SpellMechanic.dbc", "table", a.countRows("spell_mechanics")},
		{"dispelTypes", "Dispel types", "SpellDispelType.dbc", "table", a.countRows("spell_dispel_types")},
		{"statTypes", "Item stat names", "GlobalStrings.lua", "table", a.countRows("stat_types")},
	}

	return DataStatus{
		Icons:     icons,
		Maps:      maps,
		NpcImages: npcImages,
		TalentBgs: talentBgs,
		Datasets:  datasets,
	}
}

// countRows returns the row count of a reference table, or 0 if the table is
// missing or unreadable.
func (a *App) countRows(table string) int {
	if a.db == nil {
		return 0
	}
	var n int
	// table is a fixed literal from GetDataStatus, never user input.
	if err := a.db.DB().QueryRow("SELECT COUNT(*) FROM " + table).Scan(&n); err != nil {
		return 0
	}
	return n
}

// countRowsWhere is countRows with a fixed WHERE clause (both literals from
// GetDataStatus, never user input) — e.g. counting only zones that have a
// localized display name.
func (a *App) countRowsWhere(table, where string) int {
	if a.db == nil {
		return 0
	}
	var n int
	if err := a.db.DB().QueryRow("SELECT COUNT(*) FROM " + table + " WHERE " + where).Scan(&n); err != nil {
		return 0
	}
	return n
}

// areaGridTiles returns the number of ADT tiles in data/area_grid.bin (the
// authoritative spawn->zone source), or 0 when the grid hasn't been generated.
func (a *App) areaGridTiles() int {
	g, err := datatools.LoadAreaGrid(filepath.Join(a.DataDir, "area_grid.bin"))
	if err != nil || g == nil {
		return 0
	}
	return g.TileCount()
}

// countFiles returns the number of files in dir with one of the given
// extensions (case-insensitive). Missing dir counts as 0.
func countFiles(dir string, exts ...string) int {
	ents, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}
	n := 0
	for _, e := range ents {
		if e.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(e.Name()))
		for _, want := range exts {
			if ext == want {
				n++
				break
			}
		}
	}
	return n
}

// openClientMPQ returns an in-memory, read-only ClientFiles over the client's
// MPQ archives under <baseDir>/Data, or (nil, false) if there are none (callers
// then fall back to loose extracted folders). The client directory is only
// read, never written.
func openClientMPQ(baseDir string) (datatools.ClientFiles, bool) {
	dataDir := filepath.Join(baseDir, "Data")
	ents, err := os.ReadDir(dataDir)
	if err != nil {
		return nil, false
	}
	hasMPQ := false
	for _, e := range ents {
		if !e.IsDir() && strings.EqualFold(filepath.Ext(e.Name()), ".mpq") {
			hasMPQ = true
			break
		}
	}
	if !hasMPQ {
		return nil, false
	}
	src, err := datatools.NewMpqSource(dataDir)
	if err != nil {
		return nil, false
	}
	return src, true
}

// RunCacheImport patches the WoW WDB caches under <baseDir>/WDB into inklab.db.
// Only the freshest server-delivered values are overlaid; no data is otherwise
// replaced.
func (a *App) RunCacheImport(baseDir string) ImportReport {
	wdbDir := filepath.Join(baseDir, "WDB")
	dbPath := filepath.Join(a.DataDir, "inklab.db")
	results, err := datatools.PatchAllCaches(wdbDir, dbPath)
	if err != nil {
		return ImportReport{Title: "Cache import failed", Lines: []string{err.Error(), "looked in: " + wdbDir}}
	}
	rep := ImportReport{Success: true, Title: "Cache import complete"}
	for _, r := range results {
		if r.Error != "" {
			rep.Lines = append(rep.Lines, fmt.Sprintf("%s — error: %s", r.File, r.Error))
			continue
		}
		rep.Lines = append(rep.Lines, fmt.Sprintf("%s → %s: %d updated, %d new (%d records)",
			r.File, r.Table, r.Updated, r.Inserted, r.Records))
		// List the actual new entries so you can see exactly what was added.
		const maxShown = 25
		for i, e := range r.NewEntries {
			if i >= maxShown {
				rep.Lines = append(rep.Lines, fmt.Sprintf("    … and %d more", len(r.NewEntries)-maxShown))
				break
			}
			name := e.Name
			if name == "" {
				name = "(unnamed)"
			}
			rep.Lines = append(rep.Lines, fmt.Sprintf("    + [%d] %s", e.ID, name))
		}
	}
	return rep
}

// RunClientImport regenerates everything sourced from the WoW client in one
// pass — DBC reference data (regenerated JSON + re-applied to the DB), icons,
// and zone maps. It opens the client's MPQ archives once (in memory) when
// present and reuses them for all three steps; otherwise it falls back to the
// loose extracted folders per category. Output only ever goes to InkLab's
// data/ dir; the client directory is never written.
// SelectClientFolder opens a native folder picker for the WoW client directory
// and returns the chosen path. Returns "" if the user cancels (the caller keeps
// the existing value). The optional current path seeds the dialog's start
// location so it opens where the user last pointed it.
func (a *App) SelectClientFolder(current string) (string, error) {
	return runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title:                "Select your WoW client folder",
		DefaultDirectory:     current,
		CanCreateDirectories: false,
	})
}

func (a *App) RunClientImport(baseDir string) ImportReport {
	mpqSrc, useMPQ := openClientMPQ(baseDir)
	if useMPQ {
		defer mpqSrc.Close()
	}
	source := "loose folders"
	if useMPQ {
		source = "client MPQs"
	}

	// pick returns the shared MPQ source, or the per-category loose source when
	// no MPQ archives are present.
	pick := func(loose datatools.ClientFiles) datatools.ClientFiles {
		if useMPQ {
			return mpqSrc
		}
		return loose
	}

	rep := ImportReport{Success: true, Title: "Client data imported", Lines: []string{"Source: " + source}}
	fail := func(msg string) {
		rep.Success = false
		rep.Lines = append(rep.Lines, msg)
	}

	// 1. DBC reference data: regenerate data/*.json, then re-apply to the DB.
	// Client-rooted so the DBC gen can also read loose interface files (e.g.
	// FrameXML/Fonts.xml for RAID_CLASS_COLORS), not just DBFilesClient.
	dbcSrc := pick(datatools.NewDirSourceClient(baseDir))
	if err := datatools.GenerateDBCJSONFrom(dbcSrc, a.DataDir); err != nil {
		fail("DBC reference data: " + err.Error())
	} else {
		a.reapplyReferenceData(&rep)
	}

	// 2. Icons.
	iconSrc := pick(datatools.NewDirSourceIcons(filepath.Join(baseDir, "BlizzardInterfaceArt", "Icons")))
	if res, err := datatools.GenerateIconsFrom(iconSrc, filepath.Join(a.DataDir, "icons"), nil); err != nil {
		fail("Icons: " + err.Error())
	} else {
		line := fmt.Sprintf("%d icons", res.Generated)
		if res.Skipped > 0 {
			line += fmt.Sprintf(" (%d skipped)", res.Skipped)
		}
		rep.Lines = append(rep.Lines, line)
	}

	// 3. Zone maps.
	mapSrc := pick(datatools.NewDirSourceMaps(filepath.Join(baseDir, "BlizzardInterfaceArt", "WorldMap"), filepath.Join(baseDir, "DBFilesClient")))
	if res, err := datatools.GenerateZoneMapsFrom(mapSrc, filepath.Join(a.DataDir, "maps"), nil); err != nil {
		fail("Zone maps: " + err.Error())
	} else {
		line := fmt.Sprintf("%d zone maps", res.Generated)
		if res.Skipped > 0 {
			line += fmt.Sprintf(" (%d skipped)", res.Skipped)
		}
		rep.Lines = append(rep.Lines, line)
	}

	// 4. Talent-frame background art (320x384 composited per tree). Built from
	//    the same MPQ/loose source; powers the in-game look of the calculator.
	bgSrc := pick(datatools.NewDirSourceClient(baseDir))
	if res, err := datatools.GenerateTalentBackgrounds(bgSrc, filepath.Join(a.DataDir, "talent_bg"), nil); err != nil {
		fail("Talent backgrounds: " + err.Error())
	} else {
		line := fmt.Sprintf("%d talent backgrounds", res.Generated)
		if res.Skipped > 0 {
			line += fmt.Sprintf(" (%d skipped)", res.Skipped)
		}
		rep.Lines = append(rep.Lines, line)
	}

	// 5. Race/gender icons cropped from the character-create sprite sheet.
	raceSrc := pick(datatools.NewDirSourceClient(baseDir))
	if res, err := datatools.GenerateRaceIcons(raceSrc, filepath.Join(a.DataDir, "race_icons")); err != nil {
		fail("Race icons: " + err.Error())
	} else {
		rep.Lines = append(rep.Lines, fmt.Sprintf("%d race icons", res.Generated))
	}

	// 5b. Coin icons (gold/silver/copper) cropped from the money-icons sprite.
	coinSrc := pick(datatools.NewDirSourceClient(baseDir))
	if res, err := datatools.GenerateCoinIcons(coinSrc, filepath.Join(a.DataDir, "coin_icons")); err != nil {
		fail("Coin icons: " + err.Error())
	} else {
		rep.Lines = append(rep.Lines, fmt.Sprintf("%d coin icons", res.Generated))
	}

	// 5c. Minimaps (terrain view): the in-game minimap tiles stitched per zone and
	//     cropped to each WorldMapArea rect — the same rect the painted atlas maps
	//     cover, so spawn pins line up on either view. Lets the UI toggle a zone
	//     between its atlas painting and the terrain minimap.
	mmSrc := pick(datatools.NewDirSourceClient(baseDir))
	if res, err := datatools.GenerateMinimaps(mmSrc, filepath.Join(a.DataDir, "minimaps"), nil); err != nil {
		fail("Minimaps: " + err.Error())
	} else {
		line := fmt.Sprintf("%d minimaps", res.Generated)
		if res.Skipped > 0 {
			line += fmt.Sprintf(" (%d skipped)", res.Skipped)
		}
		rep.Lines = append(rep.Lines, line)
	}

	// 6. Area grid: per-chunk ADT zone data for authoritative spawn->zone
	//    resolution. Needs the MPQ/loose World\Maps terrain; non-fatal.
	gridSrc := pick(datatools.NewDirSourceClient(baseDir))
	if err := datatools.GenerateAreaGrid(gridSrc, filepath.Join(a.DataDir, "area_grid.bin"), nil); err != nil {
		fail("Area grid: " + err.Error())
	} else if g, _ := datatools.LoadAreaGrid(filepath.Join(a.DataDir, "area_grid.bin")); g != nil {
		rep.Lines = append(rep.Lines, fmt.Sprintf("%d area-grid tiles", g.TileCount()))
	}

	if !rep.Success {
		rep.Title = "Client import finished with errors"
	}
	return rep
}

// RebuildSpawnZones re-resolves every creature and gameobject spawn's zone from
// the authoritative client area grid (data/area_grid.bin), using MySQL world
// coordinates — no octowow scraping. It heals spawns whose zone was assigned by
// the older smallest-bounding-box heuristic, which mislabels points where zone
// boxes overlap (e.g. Westfall's NE corner counted as Elwynn, or Elwynn mobs
// swallowed by Duskwood). Reports the net per-zone change so the move is visible.
// baseDir is unused (spawns come from MySQL + the area grid, not the client
// folder) but kept so the method matches the Tools "run with base path" calling
// convention shared by RunClientImport / RunCacheImport.
func (a *App) RebuildSpawnZones(baseDir string) ImportReport {
	_ = baseDir
	if a.mysqlDB == nil {
		return ImportReport{
			Title: "Spawn rebuild unavailable",
			Lines: []string{"No MySQL connection — the rebuild needs the world DB for spawn coordinates."},
		}
	}

	rep := ImportReport{Success: true, Title: "Spawn zones rebuilt"}
	if g, _ := datatools.LoadAreaGrid(filepath.Join(a.DataDir, "area_grid.bin")); g == nil {
		rep.Lines = append(rep.Lines,
			"⚠ area_grid.bin missing — zones fall back to bounding boxes (the bug). Run Client Import first.")
	}

	before := a.spawnZoneCounts()
	if err := a.npcService.RebuildSpawnZones(nil); err != nil {
		return ImportReport{Title: "Spawn rebuild failed", Lines: []string{err.Error()}}
	}
	after := a.spawnZoneCounts()

	// Net per-zone change (after - before). Only zones that actually moved.
	type delta struct {
		zone        string
		before, now int
	}
	var changes []delta
	seen := map[string]bool{}
	for z := range before {
		seen[z] = true
	}
	for z := range after {
		seen[z] = true
	}
	moved := 0
	for z := range seen {
		b, n := before[z], after[z]
		if b != n {
			changes = append(changes, delta{z, b, n})
			if n > b {
				moved += n - b
			}
		}
	}
	sort.Slice(changes, func(i, j int) bool {
		di, dj := changes[i].now-changes[i].before, changes[j].now-changes[j].before
		if di < 0 {
			di = -di
		}
		if dj < 0 {
			dj = -dj
		}
		return di > dj
	})

	var total int
	for _, n := range after {
		total += n
	}
	rep.Lines = append(rep.Lines,
		fmt.Sprintf("%d spawns re-resolved; %d reassigned across %d zones.", total, moved, len(changes)))
	if len(changes) == 0 {
		rep.Lines = append(rep.Lines, "No zone changes — spawns already match the area grid.")
	}
	const maxShown = 40
	for i, c := range changes {
		if i >= maxShown {
			rep.Lines = append(rep.Lines, fmt.Sprintf("… and %d more zones", len(changes)-maxShown))
			break
		}
		d := c.now - c.before
		sign := "+"
		if d < 0 {
			sign = ""
		}
		name := c.zone
		if name == "" {
			name = "(no zone)"
		}
		rep.Lines = append(rep.Lines, fmt.Sprintf("%s: %d → %d (%s%d)", name, c.before, c.now, sign, d))
	}
	return rep
}

// spawnZoneCounts tallies how many creature + gameobject spawns are assigned to
// each zone_name — the before/after snapshot RebuildSpawnZones diffs.
func (a *App) spawnZoneCounts() map[string]int {
	counts := map[string]int{}
	for _, tbl := range []string{"creature_spawn", "gameobject_spawn"} {
		rows, err := a.db.DB().Query("SELECT COALESCE(zone_name, ''), COUNT(*) FROM " + tbl + " GROUP BY zone_name")
		if err != nil {
			continue
		}
		for rows.Next() {
			var z string
			var n int
			if rows.Scan(&z, &n) == nil {
				counts[z] += n
			}
		}
		rows.Close()
	}
	return counts
}

// reapplyReferenceData re-applies the JSON-fed reference tables (zones, skills,
// quest sorts, factions, item sets, icons, spell backfill) to the live DB after
// the DBC JSON has been regenerated. It does NOT touch MySQL-sourced templates.
func (a *App) reapplyReferenceData(rep *ImportReport) {
	add := func(label string, err error) {
		if err != nil {
			rep.Lines = append(rep.Lines, label+": "+err.Error())
		} else {
			rep.Lines = append(rep.Lines, label+" refreshed")
		}
	}
	// Skills / zones / quest sorts (REPLACE-based, idempotent).
	add("zones / skills / quest sorts", database.NewMetadataImporter(a.db).ImportAll(a.DataDir))
	// Item sets & factions skip when present, so clear then re-import.
	a.db.DB().Exec("DELETE FROM itemsets")
	add("item sets", database.NewItemSetImporter(a.db).CheckAndImport(a.DataDir))
	a.db.DB().Exec("DELETE FROM factions")
	add("factions", database.NewFactionImporter(a.db).CheckAndImport(a.DataDir))
	// Faction templates (NPC -> faction membership) — generated only by the
	// client DBC export, so refresh after one.
	a.db.DB().Exec("DELETE FROM faction_template")
	add("faction templates", database.NewFactionImporter(a.db).CheckAndImportTemplates(a.DataDir))
	// Spell data: the client DBC is authoritative (latest patched values), so it
	// overwrites spell text + values; then resolve $-placeholders against them.
	gen := database.NewGeneratedImporter(a.db.DB())
	_ = gen.ImportSpellsFromDBC(filepath.Join(a.DataDir, "spells_enhanced.json"))
	_ = gen.ImportItemIcons(filepath.Join(a.DataDir, "item_icons.json"))
	_ = gen.ImportSpellIcons(filepath.Join(a.DataDir, "spells_enhanced.json"))
	_ = gen.ImportTalents(filepath.Join(a.DataDir, "talents.json"))
	_ = gen.ImportTaxi(filepath.Join(a.DataDir, "taxi.json"))
	a.MatchFlightmasters() // link nodes to flightmaster NPCs (needs MySQL; no-op without)
	_ = gen.ImportCreatureFamilies(filepath.Join(a.DataDir, "creature_families.json"))
	_ = gen.ImportLocks(filepath.Join(a.DataDir, "locks.json"))
	_ = gen.ImportClasses(filepath.Join(a.DataDir, "classes.json"))
	_ = gen.ImportSpellSchools(filepath.Join(a.DataDir, "spell_schools.json"))
	_ = gen.ImportStatNames(filepath.Join(a.DataDir, "stat_names.json"))
	_ = gen.ImportRaces(filepath.Join(a.DataDir, "races.json"))
	_ = gen.ImportSpellMechanics(filepath.Join(a.DataDir, "spell_mechanics.json"))
	_ = gen.ImportDispelTypes(filepath.Join(a.DataDir, "spell_dispel_types.json"))
	_ = gen.ImportEnchantProcSpells(filepath.Join(a.DataDir, "enchant_proc_spells.json"))
	_ = gen.ImportLockTypes(filepath.Join(a.DataDir, "lock_types.json"))

	// Client-localized item/creature type names (item class/subclass, inventory
	// slot, creature type, curated GlobalStrings). Import from the freshly
	// regenerated JSON, then reload the in-memory resolver overrides so a scan's
	// new names take effect immediately (otherwise type/slot labels like the
	// item filter's Container/Recipe subtypes stay stale until the next restart).
	_ = gen.ImportItemClassNames(filepath.Join(a.DataDir, "item_class_names.json"))
	_ = gen.ImportItemSubclassNames(filepath.Join(a.DataDir, "item_subclass_names.json"))
	_ = gen.ImportInventoryTypes(filepath.Join(a.DataDir, "inventory_types.json"))
	_ = gen.ImportCreatureTypeNames(filepath.Join(a.DataDir, "creature_types.json"))
	_ = gen.ImportClientStrings(filepath.Join(a.DataDir, "client_strings.json"))
	gen.LoadItemNameOverrides()

	if a.syncService != nil {
		a.syncService.FullSyncSpells(0, false, "", 0, nil)
	}
	rep.Lines = append(rep.Lines, "spell text (DBC) + talents refreshed")
}
