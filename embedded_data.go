package main

import (
	"bytes"
	"compress/gzip"
	"database/sql"
	_ "embed" // for //go:embed on embeddedDBGz
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	_ "modernc.org/sqlite"
)

// localMergeTables are the tables that carry an `origin` column. On an official
// DB upgrade their user-scraped ('local') rows are grafted into the new
// baseline. `id` (AUTOINCREMENT) is intentionally excluded so it re-assigns;
// INSERT OR IGNORE drops a local row when the new official already has the same
// row (unique key), so a newer official always wins.
//
// replaceKey marks a table whose rows form one logical record per key rather
// than independent facts. Spawn points accumulate — a local point next to an
// official one is simply another place the creature stands — but a loot table
// does not: octowow's scraped drop list is the creature's WHOLE table with
// reference loot already flattened in, so grafting it alongside the baseline's
// rows would merge two servers' tables and double-count world drops. For those,
// every key the user scraped has its baseline rows dropped first, so the record
// comes entirely from one source.
var localMergeTables = []struct {
	name, cols, replaceKey string
}{
	{name: "creature_spawn", cols: "creature_entry,map_id,zone_id,zone_name,position_x,position_y,position_z,origin"},
	{name: "gameobject_spawn", cols: "gameobject_entry,map_id,zone_id,zone_name,position_x,position_y,position_z,origin"},
	{
		name:       "creature_loot_template",
		cols:       "entry,item,ChanceOrQuestChance,groupid,mincountOrRef,maxcount,origin",
		replaceKey: "entry",
	},
}

// refreshDBPreservingLocal replaces the extracted DB with the (newer) embedded
// official baseline while carrying over the user's local scrapes. It writes the
// embedded DB to a temp file, grafts local rows from the existing DB into it,
// then atomically swaps it into place. If the graft fails (e.g. an old DB with
// no `origin` column — nothing was tagged local anyway), it still ships the
// fresh official baseline rather than failing the launch.
func refreshDBPreservingLocal(dbPath string) error {
	tmpPath := dbPath + ".new"
	_ = os.Remove(tmpPath)
	if err := writeEmbeddedDB(tmpPath); err != nil {
		return err
	}

	if err := graftLocalRows(tmpPath, dbPath); err != nil {
		log.Printf("  ⚠ could not preserve local scrapes (%v); shipping fresh official data", err)
	}

	// Swap the new baseline in via a backup, so a failed rename never leaves the
	// user without a database (Windows can't rename over an existing file).
	bak := dbPath + ".bak"
	_ = os.Remove(bak)
	if err := os.Rename(dbPath, bak); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, dbPath); err != nil {
		_ = os.Rename(bak, dbPath) // restore the original on failure
		return err
	}
	_ = os.Remove(bak)
	return nil
}

// graftLocalRows copies origin='local' rows from oldPath into the new baseline
// at newPath (INSERT OR IGNORE, so newer official rows win on key collision).
// For a table with a replaceKey, the baseline's rows for every key the user
// scraped are cleared first, so that record comes wholly from the local copy
// instead of being merged with the shipped one.
func graftLocalRows(newPath, oldPath string) error {
	db, err := sql.Open("sqlite", newPath)
	if err != nil {
		return err
	}
	defer db.Close()
	if _, err := db.Exec("ATTACH DATABASE ? AS old", oldPath); err != nil {
		return err
	}
	defer db.Exec("DETACH DATABASE old")

	for _, t := range localMergeTables {
		if t.replaceKey != "" {
			clear := fmt.Sprintf(
				"DELETE FROM main.%s WHERE %s IN (SELECT DISTINCT %s FROM old.%s WHERE origin='local')",
				t.name, t.replaceKey, t.replaceKey, t.name)
			if _, err := db.Exec(clear); err != nil {
				// Old DB predates the origin column — nothing was tagged local, so
				// there is nothing to make room for either.
				log.Printf("  (skip %s local graft: %v)", t.name, err)
				continue
			}
		}
		q := fmt.Sprintf(
			"INSERT OR IGNORE INTO main.%s (%s) SELECT %s FROM old.%s WHERE origin='local'",
			t.name, t.cols, t.cols, t.name)
		if res, err := db.Exec(q); err != nil {
			// Old DB predates the origin column (pre-Stage-1) — nothing tagged local.
			log.Printf("  (skip %s local graft: %v)", t.name, err)
		} else if n, _ := res.RowsAffected(); n > 0 {
			log.Printf("  ✓ preserved %d local %s row(s)", n, t.name)
		}
	}
	return nil
}

// The baseline database ships gzipped: raw it is ~195 MB, past GitHub's 100 MB
// per-file limit, and every byte of it also rides in the user's download. The
// archive is produced by cmd/packdb (VACUUM + gzip) and is the file the repo
// commits — data/inklab.db itself is gitignored, being a local working copy.
//
//go:embed data/inklab.db.gz
var embeddedDBGz []byte

// writeEmbeddedDB decompresses the embedded baseline to path, replacing whatever
// is there. The whole file is streamed rather than held twice in memory.
func writeEmbeddedDB(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := gunzipTo(f, embeddedDBGz); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

// gunzipTo streams a gzipped database into w.
func gunzipTo(w io.Writer, gz []byte) error {
	zr, err := gzip.NewReader(bytes.NewReader(gz))
	if err != nil {
		return fmt.Errorf("database archive is not valid gzip: %w", err)
	}
	defer zr.Close()
	_, err = io.Copy(w, zr)
	return err
}

// gunzipToTemp expands a gzipped database into a temp file and returns its path;
// the caller removes it. Used for baselines that only exist as an archive (the
// embedded copy, or one read out of git).
func gunzipToTemp(gz []byte) (string, error) {
	f, err := os.CreateTemp("", "inklab-baseline-*.db")
	if err != nil {
		return "", err
	}
	if err := gunzipTo(f, gz); err != nil {
		f.Close()
		os.Remove(f.Name())
		return "", err
	}
	if err := f.Close(); err != nil {
		os.Remove(f.Name())
		return "", err
	}
	return f.Name(), nil
}

// embeddedDBVersion identifies the embedded database's data revision. Bump it
// whenever data/inklab.db is regenerated with fixes so production builds
// overwrite a previously-extracted (stale) copy instead of keeping it forever.
const embeddedDBVersion = 7

// dbVersionFile is the marker written next to the extracted database recording
// which embeddedDBVersion produced it.
const dbVersionFile = ".dbversion"

func readExtractedDBVersion(dataDir string) int {
	b, err := os.ReadFile(filepath.Join(dataDir, dbVersionFile))
	if err != nil {
		return 0
	}
	v, _ := strconv.Atoi(strings.TrimSpace(string(b)))
	return v
}

func writeExtractedDBVersion(dataDir string, v int) {
	_ = os.WriteFile(filepath.Join(dataDir, dbVersionFile), []byte(strconv.Itoa(v)), 0644)
}

// Icons are not embedded — they're extracted locally from the client art via
// the Tools tab, or downloaded on demand by the icon service.
//
// NPC model/map images are not embedded either — users build their own cache by
// syncing NPCs (scraped from octowow.st), or share a data/npc_images folder.

// InitializeData ensures data directory exists and extracts embedded database on first run
// Icons are NOT embedded - they remain external and can be updated independently
// Returns the absolute path to the data directory and whether we're in dev mode
func InitializeData() (string, bool, error) {
	var baseDir string

	// Detect if running in dev mode (wails dev)
	// In dev mode, the executable is in build/bin/ directory or a temp directory
	// We want to use the current working directory (project root) instead
	exePath, err := os.Executable()
	if err != nil {
		return "", false, fmt.Errorf("failed to get executable path: %w", err)
	}

	// Check if we're running from dev mode locations:
	// - build/bin (wails dev on Windows/Linux)
	// - Temp/tmp (some dev environments)
	isDevMode := strings.Contains(exePath, "Temp") ||
		strings.Contains(exePath, "tmp") ||
		strings.Contains(exePath, "build"+string(os.PathSeparator)+"bin") ||
		strings.Contains(exePath, "build/bin")

	if isDevMode {
		// Dev mode: use current working directory (project root)
		cwd, err := os.Getwd()
		if err != nil {
			return "", false, fmt.Errorf("failed to get working directory: %w", err)
		}
		baseDir = cwd
		log.Println("🔧 Development mode detected, using project root:", baseDir)
	} else {
		// Production mode: use executable directory
		baseDir = filepath.Dir(exePath)
		log.Println("📦 Production mode, using executable directory:", baseDir)
	}

	dataDir := filepath.Join(baseDir, "data")
	iconsDir := filepath.Join(dataDir, "icons")
	dbPath := filepath.Join(dataDir, "inklab.db")

	// Create directories
	if err := os.MkdirAll(iconsDir, 0755); err != nil {
		return "", false, fmt.Errorf("failed to create data directory: %w", err)
	}

	// Extract on first run, and in production also refresh when the embedded DB
	// is a newer revision than the previously-extracted copy — otherwise data
	// fixes never reach users who already have a data/inklab.db. Dev mode always
	// uses the on-disk db as-is (it's managed via git / rebuilddb).
	_, statErr := os.Stat(dbPath)
	missing := os.IsNotExist(statErr)
	stale := !isDevMode && !missing && readExtractedDBVersion(dataDir) < embeddedDBVersion
	switch {
	case missing:
		// First run: nothing to preserve, write the embedded baseline as-is.
		log.Println("Extracting embedded database...")
		if err := writeEmbeddedDB(dbPath); err != nil {
			return "", false, fmt.Errorf("failed to write database: %w", err)
		}
		writeExtractedDBVersion(dataDir, embeddedDBVersion)
		log.Println("✓ Database ready at", dbPath)
	case stale:
		// Newer official data shipped: refresh to it, but graft the user's own
		// scraped ('local') rows into the new baseline so an update never wipes
		// their additions. A newer official row for the same spawn wins (the
		// graft is INSERT OR IGNORE against the spawn's unique key).
		log.Printf("Embedded database is newer (v%d); merging (preserving local scrapes)...", embeddedDBVersion)
		if err := refreshDBPreservingLocal(dbPath); err != nil {
			return "", false, fmt.Errorf("failed to refresh database: %w", err)
		}
		writeExtractedDBVersion(dataDir, embeddedDBVersion)
		log.Println("✓ Database refreshed at", dbPath)
	default:
		log.Println("✓ Using existing database:", dbPath)
	}

	// Icons live in data/icons (extracted from client art via the Tools tab, or
	// downloaded on demand). Nothing to extract from the binary.

	// NPC images are built locally (synced/scraped from octowow.st) — just make
	// sure the directory exists for the sync to write into.
	npcImagesDir := filepath.Join(dataDir, "npc_images")
	if err := os.MkdirAll(npcImagesDir, 0755); err != nil {
		log.Printf("Warning: Failed to create npc_images directory: %v", err)
	}

	return dataDir, isDevMode, nil
}
