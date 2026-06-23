package main

import (
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	dbPath := "data/inklab.db"
	imgDir := "data/npc_images"

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	rows, err := db.Query("SELECT entry, map_url, model_image_url, map_image_local, model_image_local FROM creature_metadata")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	type Update struct {
		entry      int
		mapLocal   string
		modelLocal string
	}
	var updates []Update

	fmt.Println("Starting deduplication and migration...")

	for rows.Next() {
		var entry int
		var mapUrl, modelUrl, mapLocal, modelLocal sql.NullString
		if err := rows.Scan(&entry, &mapUrl, &modelUrl, &mapLocal, &modelLocal); err != nil {
			continue
		}

		newMapLocal := mapLocal.String
		newModelLocal := modelLocal.String

		// Process Map Image
		if mapUrl.Valid && mapUrl.String != "" {
			expectedPath := getHashPath(mapUrl.String, imgDir)
			if mapLocal.String != "" && mapLocal.String != expectedPath {
				migrate(mapLocal.String, expectedPath)
				newMapLocal = expectedPath
			}
		}

		// Process Model Image
		if modelUrl.Valid && modelUrl.String != "" {
			expectedPath := getHashPath(modelUrl.String, imgDir)
			if modelLocal.String != "" && modelLocal.String != expectedPath {
				migrate(modelLocal.String, expectedPath)
				newModelLocal = expectedPath
			}
		}

		if newMapLocal != mapLocal.String || newModelLocal != modelLocal.String {
			updates = append(updates, Update{entry, newMapLocal, newModelLocal})
		}
	}

	fmt.Printf("Updating %d database records...\n", len(updates))
	for _, u := range updates {
		_, err := db.Exec("UPDATE creature_metadata SET map_image_local = ?, model_image_local = ? WHERE entry = ?", u.mapLocal, u.modelLocal, u.entry)
		if err != nil {
			fmt.Printf("Failed to update entry %d: %v\n", u.entry, err)
		}
	}

	fmt.Println("Cleaning up orphaned 'map_*' and 'model_*' files...")
	cleanupOrphaned(imgDir)

	fmt.Println("Done!")
}

func getHashPath(url, dir string) string {
	hash := md5.Sum([]byte(url))
	hashName := hex.EncodeToString(hash[:])
	ext := ".jpg"
	lowerUrl := strings.ToLower(url)
	if strings.Contains(lowerUrl, ".png") {
		ext = ".png"
	} else if strings.Contains(lowerUrl, ".gif") {
		ext = ".gif"
	} else if strings.Contains(lowerUrl, ".webp") {
		ext = ".webp"
	}
	return filepath.Join(dir, hashName+ext)
}

func migrate(oldPath, newPath string) {
	if oldPath == "" || oldPath == newPath {
		return
	}
	// Normalise paths (handle potential absolute/relative mix)
	if _, err := os.Stat(oldPath); os.IsNotExist(err) {
		return
	}

	if _, err := os.Stat(newPath); err == nil {
		// New path already exists, just delete old one
		os.Remove(oldPath)
		// fmt.Printf("Deleted duplicate: %s\n", oldPath)
	} else {
		// Rename old to new
		os.Rename(oldPath, newPath)
		// fmt.Printf("Migrated: %s -> %s\n", oldPath, newPath)
	}
}

func cleanupOrphaned(dir string) {
	files, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	for _, f := range files {
		name := f.Name()
		if strings.HasPrefix(name, "map_") || strings.HasPrefix(name, "model_") {
			path := filepath.Join(dir, name)
			os.Remove(path)
			// fmt.Printf("Removed orphaned: %s\n", name)
		}
	}
}
