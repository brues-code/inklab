package main

import (
	"fmt"
	"path/filepath"

	"inklab/backend/database"
	"inklab/backend/datatools"
)

// ImportReport is the generic result of a Tools-tab data import.
type ImportReport struct {
	Success bool     `json:"success"`
	Title   string   `json:"title"`
	Lines   []string `json:"lines"`
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
		} else {
			rep.Lines = append(rep.Lines, fmt.Sprintf("%s → %s: %d updated, %d new (%d records)",
				r.File, r.Table, r.Updated, r.Inserted, r.Records))
		}
	}
	return rep
}

// RunMapImport stitches the client world-map art into data/maps/<zone>.jpg,
// reading tiles from <baseDir>/BlizzardInterfaceArt/WorldMap and overlay
// placements from <baseDir>/DBFilesClient/WorldMapOverlay.dbc.
func (a *App) RunMapImport(baseDir string) ImportReport {
	worldMap := filepath.Join(baseDir, "BlizzardInterfaceArt", "WorldMap")
	overlay := filepath.Join(baseDir, "DBFilesClient", "WorldMapOverlay.dbc")
	out := filepath.Join(a.DataDir, "maps")
	res, err := datatools.GenerateZoneMaps(worldMap, overlay, out, nil)
	if err != nil {
		return ImportReport{Title: "Map generation failed", Lines: []string{err.Error(), "looked in: " + worldMap}}
	}
	rep := ImportReport{Success: true, Title: "Maps generated",
		Lines: []string{fmt.Sprintf("%d zone maps written to data/maps", res.Generated)}}
	if res.Skipped > 0 {
		rep.Lines = append(rep.Lines, fmt.Sprintf("%d skipped", res.Skipped))
	}
	return rep
}

// RunIconImport extracts client icon art from <baseDir>/BlizzardInterfaceArt/Icons
// into data/icons/<name>.jpg (lowercased), so icons resolve locally without
// downloading.
func (a *App) RunIconImport(baseDir string) ImportReport {
	iconsDir := filepath.Join(baseDir, "BlizzardInterfaceArt", "Icons")
	out := filepath.Join(a.DataDir, "icons")
	res, err := datatools.GenerateIcons(iconsDir, out, nil)
	if err != nil {
		return ImportReport{Title: "Icon import failed", Lines: []string{err.Error(), "looked in: " + iconsDir}}
	}
	rep := ImportReport{Success: true, Title: "Icons extracted",
		Lines: []string{fmt.Sprintf("%d icons written to data/icons", res.Generated)}}
	if res.Skipped > 0 {
		rep.Lines = append(rep.Lines, fmt.Sprintf("%d skipped (unsupported format)", res.Skipped))
	}
	return rep
}

// RunDbcImport regenerates data/*.json from the client DBCs in
// <baseDir>/DBFilesClient and re-applies the JSON-fed reference tables (zones,
// skills, quest sorts, factions, item sets, icons, spell backfill) to the live
// DB. It does NOT touch the MySQL-sourced templates.
func (a *App) RunDbcImport(baseDir string) ImportReport {
	dbcDir := filepath.Join(baseDir, "DBFilesClient")
	if err := datatools.GenerateDBCJSON(dbcDir, a.DataDir); err != nil {
		return ImportReport{Title: "DBC regen failed", Lines: []string{err.Error(), "looked in: " + dbcDir}}
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
