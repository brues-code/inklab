package main

import (
	"embed"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
)

//go:embed data/inklab.db
var embeddedDB []byte

//go:embed data/icons/*
var embeddedIcons embed.FS

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

	// In production, extract database if not exists
	// In dev mode, we don't extract - we use the existing data/inklab.db directly
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		log.Println("Extracting embedded database...")
		if err := os.WriteFile(dbPath, embeddedDB, 0644); err != nil {
			return "", false, fmt.Errorf("failed to write database: %w", err)
		}
		log.Println("✓ Database extracted to", dbPath)
	} else {
		log.Println("✓ Using existing database:", dbPath)
	}

	// Extract icons if directory is empty or mostly empty (< 50 icons)
	entries, _ := os.ReadDir(iconsDir)
	if len(entries) < 50 {
		log.Println("Extracting embedded icons (this may take a moment)...")
		extractAssets(embeddedIcons, "data/icons", iconsDir)
		log.Println("📝 Tip: Downloaded icons will be saved to data/icons/ and take precedence over embedded ones")
	}

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
