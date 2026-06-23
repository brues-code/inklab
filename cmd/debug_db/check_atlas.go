package main

import (
	"database/sql"
	"fmt"
	"log"

	"inklab/backend/database/importers"
	"inklab/backend/database/schema"

	_ "modernc.org/sqlite"
)

func main() {
	db, err := sql.Open("sqlite", "./data/inklab.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Apply migrations
	fmt.Println("Applying Migrations...")
	schema.MigrateAtlasLoot(db)

	// Run Importer
	fmt.Println("Running Importer Check...")
	importer := importers.NewAtlasLootImporter(db)
	err = importer.CheckAndImport("./data")
	if err != nil {
		log.Printf("Import failed: %v", err)
	} else {
		fmt.Println("Import logic executed (or skipped if already valid).")
	}

	// 1. Check Categories
	rows, err := db.Query("SELECT id, display_name FROM atlasloot_categories")
	if err != nil {
		log.Fatal(err)
	}
	var catID int
	var catName string

	fmt.Println("\nCategories:")
	for rows.Next() {
		rows.Scan(&catID, &catName)
		fmt.Printf("ID: %d, Name: %s\n", catID, catName)
	}
	rows.Close()

	// 2. Check T0 Sets
	fmt.Println("\n--- Checking T0 Sets structure ---")
	rows, err = db.Query(`
        SELECT m.display_name as Module, t.display_name as TableName, t.table_key, COUNT(i.item_id) as ItemCount
        FROM atlasloot_tables t 
        JOIN atlasloot_modules m ON t.module_id = m.id
        LEFT JOIN atlasloot_items i ON t.id = i.table_id
        WHERE m.name = 'T0SET'
        GROUP BY t.id
    `)
	if err != nil {
		log.Fatal(err)
	}

	for rows.Next() {
		var mod, tbl, key string
		var count int
		rows.Scan(&mod, &tbl, &key, &count)
		fmt.Printf("Module: %s | Table: %s (%s) | Items: %d\n", mod, tbl, key, count)
	}
	rows.Close()

	// 3. Check Alchemy
	fmt.Println("\n--- Checking Alchemy structure ---")
	rows, err = db.Query(`
        SELECT m.display_name as Module, t.display_name as TableName, t.table_key, COUNT(i.item_id) as ItemCount
        FROM atlasloot_tables t 
        JOIN atlasloot_modules m ON t.module_id = m.id
        LEFT JOIN atlasloot_items i ON t.id = i.table_id
        WHERE m.name = 'ALCHEMYMENU'
        GROUP BY t.id
    `)
	if err != nil {
		log.Fatal(err)
	}

	for rows.Next() {
		var mod, tbl, key string
		var count int
		rows.Scan(&mod, &tbl, &key, &count)
		fmt.Printf("Module: %s | Table: %s (%s) | Items: %d\n", mod, tbl, key, count)
	}
	rows.Close()
}
