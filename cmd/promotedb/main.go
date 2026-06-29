// Command promotedb promotes the maintainer's locally-scraped rows to
// 'official' in data/inklab.db, in preparation for shipping it as the embedded
// baseline. Run this before bumping embeddedDBVersion and building a release:
// your accumulated scrapes (e.g. custom-zone spawns like Balor) then ship to
// new users as official data — losslessly, with no octowow re-scraping — and
// on the user side they become replaceable baseline rather than masquerading
// as that user's own local rows.
//
// It's the inverse-direction counterpart to the per-user provenance: locally
// scraped data is 'local' on a user's machine, but 'official' once you publish
// it.
//
// Usage:
//
//	go run ./cmd/promotedb [dataDir]   (defaults to "data")
package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

// provenanceTables carry an `origin` column (kept in sync with the Stage-1
// schema). Add new scrape-populated tables here as they gain provenance.
var provenanceTables = []string{"creature_spawn", "gameobject_spawn"}

func main() {
	dataDir := "data"
	if len(os.Args) > 1 {
		dataDir = os.Args[1]
	}
	dbPath := filepath.Join(dataDir, "inklab.db")

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		fatal("open sqlite", err)
	}
	defer db.Close()

	total := 0
	for _, t := range provenanceTables {
		res, err := db.Exec(fmt.Sprintf("UPDATE %s SET origin = 'official' WHERE origin = 'local'", t))
		if err != nil {
			// Column/table absent just means nothing to promote there.
			fmt.Printf("  (skip %s: %v)\n", t, err)
			continue
		}
		n, _ := res.RowsAffected()
		if n > 0 {
			fmt.Printf("  ✓ promoted %d local row(s) in %s\n", n, t)
		}
		total += int(n)
	}

	fmt.Printf("✓ Promoted %d local row(s) to official in %s\n", total, dbPath)
	if total > 0 {
		fmt.Println("  Next: bump embeddedDBVersion in embedded_data.go, then build/release.")
	}
}

func fatal(ctx string, err error) {
	fmt.Fprintf(os.Stderr, "error (%s): %v\n", ctx, err)
	os.Exit(1)
}
