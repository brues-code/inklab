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

// RunMapImport stitches the client world-map art into data/maps/<zone>.jpg.
// It reads directly from the client's MPQ archives (in memory) when present,
// otherwise from loose <baseDir>/BlizzardInterfaceArt/WorldMap +
// DBFilesClient/WorldMapOverlay.dbc. Output only ever goes to data/maps.
func (a *App) RunMapImport(baseDir string) ImportReport {
	out := filepath.Join(a.DataDir, "maps")
	var (
		res    *datatools.MapGenResult
		err    error
		source string
	)
	if src, ok := openClientMPQ(baseDir); ok {
		defer src.Close()
		source = "client MPQs"
		res, err = datatools.GenerateZoneMapsFrom(src, out, nil)
	} else {
		worldMap := filepath.Join(baseDir, "BlizzardInterfaceArt", "WorldMap")
		overlay := filepath.Join(baseDir, "DBFilesClient", "WorldMapOverlay.dbc")
		source = "loose folder"
		res, err = datatools.GenerateZoneMaps(worldMap, overlay, out, nil)
	}
	if err != nil {
		return ImportReport{Title: "Map generation failed", Lines: []string{err.Error(), "source: " + source}}
	}
	rep := ImportReport{Success: true, Title: "Maps generated",
		Lines: []string{fmt.Sprintf("%d zone maps written to data/maps (from %s)", res.Generated, source)}}
	if res.Skipped > 0 {
		rep.Lines = append(rep.Lines, fmt.Sprintf("%d skipped", res.Skipped))
	}
	return rep
}

// RunIconImport decodes client icon art into data/icons/<name>.jpg (lowercased).
// It reads directly from the client's MPQ archives (in memory) when present,
// otherwise from loose <baseDir>/BlizzardInterfaceArt/Icons.
func (a *App) RunIconImport(baseDir string) ImportReport {
	out := filepath.Join(a.DataDir, "icons")
	var (
		res    *datatools.IconGenResult
		err    error
		source string
	)
	if src, ok := openClientMPQ(baseDir); ok {
		defer src.Close()
		source = "client MPQs"
		res, err = datatools.GenerateIconsFrom(src, out, nil)
	} else {
		iconsDir := filepath.Join(baseDir, "BlizzardInterfaceArt", "Icons")
		source = "loose folder"
		res, err = datatools.GenerateIcons(iconsDir, out, nil)
	}
	if err != nil {
		return ImportReport{Title: "Icon import failed", Lines: []string{err.Error(), "source: " + source}}
	}
	rep := ImportReport{Success: true, Title: "Icons extracted",
		Lines: []string{fmt.Sprintf("%d icons written to data/icons (from %s)", res.Generated, source)}}
	if res.Skipped > 0 {
		rep.Lines = append(rep.Lines, fmt.Sprintf("%d skipped (unsupported format)", res.Skipped))
	}
	return rep
}

// RunDbcImport regenerates data/*.json from the client DBCs and re-applies the
// JSON-fed reference tables (zones, skills, quest sorts, factions, item sets,
// icons, spell backfill) to the live DB. DBCs are read directly from the
// client's MPQ archives (in memory) when present, otherwise from loose
// <baseDir>/DBFilesClient. It does NOT touch the MySQL-sourced templates.
func (a *App) RunDbcImport(baseDir string) ImportReport {
	var err error
	if src, ok := openClientMPQ(baseDir); ok {
		defer src.Close()
		err = datatools.GenerateDBCJSONFrom(src, a.DataDir)
	} else {
		dbcDir := filepath.Join(baseDir, "DBFilesClient")
		err = datatools.GenerateDBCJSON(dbcDir, a.DataDir)
	}
	if err != nil {
		return ImportReport{Title: "DBC regen failed", Lines: []string{err.Error()}}
	}

	var lines []string
	add := func(label string, err error) {
		if err != nil {
			lines = append(lines, label+": "+err.Error())
		} else {
			lines = append(lines, label+" refreshed")
		}
	}

	// Skills / zones / quest sorts (REPLACE-based, idempotent).
	add("zones / skills / quest sorts", database.NewMetadataImporter(a.db).ImportAll(a.DataDir))
	// Item sets & factions skip when present, so clear then re-import.
	a.db.DB().Exec("DELETE FROM itemsets")
	add("item sets", database.NewItemSetImporter(a.db).CheckAndImport(a.DataDir))
	a.db.DB().Exec("DELETE FROM factions")
	add("factions", database.NewFactionImporter(a.db).CheckAndImport(a.DataDir))
	// Icons + missing-spell backfill.
	gen := database.NewGeneratedImporter(a.db.DB())
	_ = gen.ImportMissingSpells(filepath.Join(a.DataDir, "spells_enhanced.json"))
	_ = gen.ImportItemIcons(filepath.Join(a.DataDir, "item_icons.json"))
	_ = gen.ImportSpellIcons(filepath.Join(a.DataDir, "spells_enhanced.json"))
	lines = append(lines, "icons + spell text refreshed")

	return ImportReport{Success: true, Title: "DBC import complete", Lines: lines}
}
