package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "modernc.org/sqlite"
)

func main() {
	dbPath := "data/inklab.db"
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	fmt.Println("--- Spell Icon Diagnostic ---")

	// 1. Check spell_template iconName values
	var totalSpells int
	db.QueryRow("SELECT COUNT(*) FROM spell_template").Scan(&totalSpells)
	fmt.Printf("Total spells in spell_template: %d\n", totalSpells)

	var withIconName int
	db.QueryRow("SELECT COUNT(*) FROM spell_template WHERE iconName IS NOT NULL AND iconName != ''").Scan(&withIconName)
	fmt.Printf("Spells with iconName: %d\n", withIconName)

	// 2. Check spell_icons table
	var iconCount int
	db.QueryRow("SELECT COUNT(*) FROM spell_icons").Scan(&iconCount)
	fmt.Printf("spell_icons entries: %d\n", iconCount)

	// 3. Sample spell with iconName
	rows, _ := db.Query("SELECT entry, name, iconName, spellIconId FROM spell_template WHERE iconName IS NOT NULL AND iconName != '' LIMIT 5")
	defer rows.Close()
	fmt.Println("\nSample spells with iconName:")
	for rows.Next() {
		var entry, iconId int
		var name, iconName string
		rows.Scan(&entry, &name, &iconName, &iconId)
		fmt.Printf("  - %d: %s -> icon: %s (iconId: %d)\n", entry, name, iconName, iconId)
	}

	// 4. Sample spells without iconName (but have spellIconId)
	rows2, _ := db.Query("SELECT entry, name, spellIconId FROM spell_template WHERE (iconName IS NULL OR iconName = '') AND spellIconId > 0 LIMIT 5")
	defer rows2.Close()
	fmt.Println("\nSample spells WITHOUT iconName but WITH spellIconId:")
	for rows2.Next() {
		var entry, iconId int
		var name string
		rows2.Scan(&entry, &name, &iconId)

		// Check if this iconId exists in spell_icons
		var iconName string
		db.QueryRow("SELECT icon_name FROM spell_icons WHERE id = ?", iconId).Scan(&iconName)
		fmt.Printf("  - %d: %s (iconId: %d) -> spell_icons.icon_name: %s\n", entry, name, iconId, iconName)
	}
}
