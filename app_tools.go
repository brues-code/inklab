package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"inklab/backend/database"
	"inklab/backend/datatools"
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
		{"npcImages", "NPC model renders", `Creature\*.m2 (client MPQs)`, "image", npcImages},
		{"talentBgs", "Talent backgrounds", `Interface\TalentFrame art`, "image", talentBgs},
		{"raceIcons", "Race / gender icons", `UI-CharacterCreate-Races.blp`, "image", countFiles(filepath.Join(a.DataDir, "race_icons"), ".png")},
		{"coinIcons", "Coin icons (g/s/c)", `UI-MoneyIcons.blp`, "image", countFiles(filepath.Join(a.DataDir, "coin_icons"), ".png")},
		// DBC-derived reference tables (row counts).
		{"itemIcons", "Item icon mappings", "ItemDisplayInfo.dbc", "table", a.countRows("item_display_info")},
		{"spellIcons", "Spell icon mappings", "Spell.dbc + SpellIcon.dbc", "table", a.countRows("spell_icons")},
		{"skills", "Skills", "SkillLine.dbc", "table", a.countRows("spell_skills")},
		{"talents", "Talents", "Talent.dbc + TalentTab.dbc", "table", a.countRows("talent")},
		{"itemSets", "Item sets", "ItemSet.dbc", "table", a.countRows("itemsets")},
		{"factions", "Factions", "Faction.dbc", "table", a.countRows("factions")},
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
	if a.syncService != nil {
		a.syncService.FullSyncSpells(0, false, "", 0, nil)
	}
	rep.Lines = append(rep.Lines, "spell text (DBC) + talents refreshed")
}
