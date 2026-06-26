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

// DataStatus reports how much locally-built image data InkLab currently has, so
// the UI can tell the user when a category is empty and needs importing.
type DataStatus struct {
	Icons        int `json:"icons"`
	Maps         int `json:"maps"`
	NpcImages    int `json:"npcImages"`
	TalentBgs    int `json:"talentBgs"`
}

// GetDataStatus counts the local icon / map / npc-image files under the data dir.
func (a *App) GetDataStatus() DataStatus {
	return DataStatus{
		Icons:     countFiles(filepath.Join(a.DataDir, "icons"), ".jpg", ".png", ".jpeg"),
		Maps:      countFiles(filepath.Join(a.DataDir, "maps"), ".jpg", ".png"),
		NpcImages: countFiles(filepath.Join(a.DataDir, "npc_images"), ".jpg", ".png", ".jpeg"),
		TalentBgs: countFiles(filepath.Join(a.DataDir, "talent_bg"), ".png", ".jpg"),
	}
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
	dbcSrc := pick(datatools.NewDirSourceDBC(filepath.Join(baseDir, "DBFilesClient")))
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

	// 5. Area grid: per-chunk ADT zone data for authoritative spawn->zone
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
	if a.syncService != nil {
		a.syncService.FullSyncSpells(0, false, "", 0, nil)
	}
	rep.Lines = append(rep.Lines, "spell text (DBC) + talents refreshed")
}
