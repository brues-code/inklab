// Command dbc2json regenerates the data/*.json files InkLab imports from the
// client DBC files. The generators live in backend/datatools so the app's
// Tools tab runs the same logic in-process.
//
// Usage:
//
//	go run ./cmd/dbc2json gen-all <DBFilesClient dir> <data dir>
package main

import (
	"fmt"
	"os"

	"inklab/backend/datatools"
)

func main() {
	if len(os.Args) < 4 || os.Args[1] != "gen-all" {
		fmt.Println("usage: dbc2json gen-all <DBFilesClient dir> <data dir>")
		os.Exit(2)
	}
	if err := datatools.GenerateDBCJSON(os.Args[2], os.Args[3]); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	fmt.Println("✓ regenerated data/*.json from", os.Args[2])
}
