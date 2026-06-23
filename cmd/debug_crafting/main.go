package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "modernc.org/sqlite"
)

func main() {
	db, err := sql.Open("sqlite", "data/inklab.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Check Categories
	var catID int
	err = db.QueryRow("SELECT id FROM atlasloot_categories WHERE display_name = 'Crafting'").Scan(&catID)
	if err != nil {
		log.Fatalf("Crafting category not found: %v", err)
	}
	fmt.Printf("Crafting Category ID: %d\n", catID)

	// Check Modules
	rows, err := db.Query("SELECT id, display_name FROM atlasloot_modules WHERE category_id = ?", catID)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	fmt.Println("\n--- Modules ---")
	var modules []int
	for rows.Next() {
		var id int
		var name string
		rows.Scan(&id, &name)
		fmt.Printf("%d: %s\n", id, name)
		modules = append(modules, id)
	}

	// Check Tables for first module
	if len(modules) > 0 {
		fmt.Printf("\n--- Tables for Module %d --- \n", modules[0])
		tRows, err := db.Query("SELECT id, table_key, display_name FROM atlasloot_tables WHERE module_id = ?", modules[0])
		if err != nil {
			log.Fatal(err)
		}
		defer tRows.Close()
		for tRows.Next() {
			var tid int
			var tkey, tname string
			tRows.Scan(&tid, &tkey, &tname)
			fmt.Printf("%d: %s (%s)\n", tid, tname, tkey)

			// Check item count
			var count int
			db.QueryRow("SELECT COUNT(*) FROM atlasloot_items WHERE table_id = ?", tid).Scan(&count)
			fmt.Printf("   -> Items: %d\n", count)
		}
	}
}
