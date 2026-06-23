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

	fmt.Println("--- Schema for atlasloot_items ---")
	rows, err := db.Query("PRAGMA table_info(atlasloot_items)")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dfltValue interface{}
		rows.Scan(&cid, &name, &ctype, &notnull, &dfltValue, &pk)
		fmt.Printf("Col: %s | Type: %s | NotNull: %d\n", name, ctype, notnull)
	}

	fmt.Println("\n--- Foreign Keys ---")
	fkRows, err := db.Query("PRAGMA foreign_key_list(atlasloot_items)")
	if err != nil {
		log.Fatal(err)
	}
	defer fkRows.Close()
	for fkRows.Next() {
		var id, seq int
		var table, from, to, onUpdate, onDelete, match string
		fkRows.Scan(&id, &seq, &table, &from, &to, &onUpdate, &onDelete, &match)
		fmt.Printf("FK: %s(%s) -> %s(%s)\n", from, table, to) // Args might represent slightly different columns, checking logic
	}
}
