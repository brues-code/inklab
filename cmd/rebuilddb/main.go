// Command rebuilddb rebuilds data/inklab.db from scratch using the local
// MySQL world DB (tw_world) plus the DBC-derived data/*.json files. It mirrors
// the import sequence the app runs on first launch, but headless and without
// the embed/extract logic, so it always produces a fresh database.
//
// Requires a running MySQL/MariaDB with the dumps imported and a .env with
// the connection settings (see .env.example).
//
// Usage:
//
//	go run ./cmd/rebuilddb [dataDir]   (defaults to "data")
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"inklab/backend/database"
	"inklab/backend/services"

	"github.com/joho/godotenv"
)

func main() {
	dataDir := "data"
	if len(os.Args) > 1 {
		dataDir = os.Args[1]
	}
	dbPath := filepath.Join(dataDir, "inklab.db")

	// Start clean.
	for _, p := range []string{dbPath, dbPath + "-wal", dbPath + "-shm"} {
		os.Remove(p)
	}

	db, err := database.NewSQLiteDB(dbPath)
	if err != nil {
		fatal("open sqlite", err)
	}
	defer db.Close()
	if err := db.InitSchema(); err != nil {
		fatal("init schema", err)
	}

	// MySQL connection from .env.
	_ = godotenv.Load()
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s",
		os.Getenv("MYSQL_USER"), os.Getenv("MYSQL_PASSWORD"),
		os.Getenv("MYSQL_HOST"), os.Getenv("MYSQL_PORT"), os.Getenv("MYSQL_DATABASE"))
	mysqlConn, err := database.NewMySQLConnection(dsn)
	if err != nil {
		fatal("mysql connect", err)
	}
	defer mysqlConn.Close()
	fmt.Println("✓ MySQL connected")

	// 1. Core tables from MySQL (item_template, creature_template, ...).
	fmt.Println("Importing core tables from MySQL...")
	if err := database.NewMySQLImporter(db.DB(), mysqlConn.DB()).ImportAllFromMySQL(); err != nil {
		fatal("mysql import", err)
	}

	// 2. DBC-derived reference tables from JSON.
	fmt.Println("Importing item sets, factions, metadata...")
	if err := database.NewItemSetImporter(db).CheckAndImport(dataDir); err != nil {
		fmt.Println("  warn item sets:", err)
	}
	if err := database.NewFactionImporter(db).CheckAndImport(dataDir); err != nil {
		fmt.Println("  warn factions:", err)
	}
	if err := database.NewMetadataImporter(db).ImportAll(dataDir); err != nil {
		fmt.Println("  warn metadata:", err)
	}
	if err := database.NewAtlasLootImporter(db).CheckAndImport(dataDir); err != nil {
		fmt.Println("  warn atlasloot:", err)
	}

	// 3. Icon mappings onto the imported templates (the newly wired step).
	fmt.Println("Applying icon mappings...")
	gen := database.NewGeneratedImporter(db.DB())
	if err := gen.ImportSpellsFromDBC(filepath.Join(dataDir, "spells_enhanced.json")); err != nil {
		fmt.Println("  warn spell DBC import:", err)
	}
	if err := gen.ImportItemIcons(filepath.Join(dataDir, "item_icons.json")); err != nil {
		fmt.Println("  warn item icons:", err)
	}
	if err := gen.ImportSpellIcons(filepath.Join(dataDir, "spells_enhanced.json")); err != nil {
		fmt.Println("  warn spell icons:", err)
	}
	if err := gen.ImportTalents(filepath.Join(dataDir, "talents.json")); err != nil {
		fmt.Println("  warn talents:", err)
	}
	if err := gen.ImportTaxi(filepath.Join(dataDir, "taxi.json")); err != nil {
		fmt.Println("  warn taxi:", err)
	}
	if err := gen.ImportCreatureFamilies(filepath.Join(dataDir, "creature_families.json")); err != nil {
		fmt.Println("  warn creature families:", err)
	}
	if err := gen.ImportLocks(filepath.Join(dataDir, "locks.json")); err != nil {
		fmt.Println("  warn locks:", err)
	}
	if err := gen.ImportClasses(filepath.Join(dataDir, "classes.json")); err != nil {
		fmt.Println("  warn classes:", err)
	}

	// Resolve $-placeholders against the just-imported (DBC-authoritative) values.
	fmt.Println("Resolving spell descriptions...")
	services.NewSyncService(db.DB()).FullSyncSpells(0, false, "", 0, nil)

	fmt.Println("✓ Rebuild complete:", dbPath)
}

func fatal(ctx string, err error) {
	fmt.Fprintf(os.Stderr, "error (%s): %v\n", ctx, err)
	os.Exit(1)
}
