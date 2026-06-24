package main

import (
	"fmt"
	"path/filepath"

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
