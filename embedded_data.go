package main

import (
	"embed"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

//go:embed data/inklab.db
var embeddedDB []byte

// embeddedDBVersion identifies the embedded database's data revision. Bump it
// whenever data/inklab.db is regenerated with fixes so production builds
// overwrite a previously-extracted (stale) copy instead of keeping it forever.
const embeddedDBVersion = 1

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

//go:embed data/npc_images/*
var embeddedNpcImages embed.FS

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
	if missing || stale {
		if stale {
			log.Printf("Embedded database is newer (v%d); refreshing extracted copy...", embeddedDBVersion)
		} else {
			log.Println("Extracting embedded database...")
		}
		if err := os.WriteFile(dbPath, embeddedDB, 0644); err != nil {
			return "", false, fmt.Errorf("failed to write database: %w", err)
		}
		writeExtractedDBVersion(dataDir, embeddedDBVersion)
		log.Println("✓ Database ready at", dbPath)
	} else {
		log.Println("✓ Using existing database:", dbPath)
	}

	// Icons live in data/icons (extracted from client art via the Tools tab, or
	// downloaded on demand). Nothing to extract from the binary.

	// Extract NPC images if directory is empty
	npcImagesDir := filepath.Join(dataDir, "npc_images")
	if err := os.MkdirAll(npcImagesDir, 0755); err != nil {
		log.Printf("Warning: Failed to create npc_images directory: %v", err)
	} else {
		npcEntries, _ := os.ReadDir(npcImagesDir)
		if len(npcEntries) < 5 { // Only extract if very few images exist
			log.Println("Extracting embedded NPC images...")
			extractAssets(embeddedNpcImages, "data/npc_images", npcImagesDir)
		}
	}

	return dataDir, isDevMode, nil
}

// extractAssets extracts files from an embedded FS to a destination directory
func extractAssets(embedFs embed.FS, srcDir string, destDir string) error {
	count := 0
	err := fs.WalkDir(embedFs, srcDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}

		// Read from embedded FS
		content, err := embedFs.ReadFile(path)
		if err != nil {
			return err
		}

		// Write to the correct dataDir location
		relPath, _ := filepath.Rel(srcDir, path)
		localPath := filepath.Join(destDir, relPath)

		// Create parent directory if needed (e.g. for nested resources)
		if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
			return err
		}

		// Only write if file doesn't exist (preserve user updates)
		if _, err := os.Stat(localPath); os.IsNotExist(err) {
			if err := os.WriteFile(localPath, content, 0644); err != nil {
				return err
			}
			count++
			if count%100 == 0 {
				log.Printf("  Extracted %d files...", count)
			}
		}
		return nil
	})

	if err != nil {
		log.Printf("Error extracting assets: %v", err)
		return err
	}
	log.Printf("✓ Extracted %d files to %s", count, destDir)
	return nil
}
