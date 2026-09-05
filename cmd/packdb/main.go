// Command packdb prepares the shipped database artifact. It VACUUMs
// data/inklab.db — reclaiming the page churn a release's imports and scrapes
// leave behind — then writes data/inklab.db.gz, which is what embedded_data.go
// embeds and what the repository actually commits.
//
// The raw database is far past GitHub's 100 MB per-file limit (a full world-DB
// loot import alone pushed it to ~195 MB) and would keep growing. Gzipped it is
// roughly a quarter of that, which also cuts the same weight off every user's
// download, since the blob is embedded in the binary.
//
// Run it after cmd/promotedb and before committing; scripts/release.ps1 does
// both in order.
//
// Usage:
//
//	go run ./cmd/packdb [dataDir]   (defaults to "data")
package main

import (
	"compress/gzip"
	"database/sql"
	"fmt"
	"io"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

func main() {
	dataDir := "data"
	if len(os.Args) > 1 {
		dataDir = os.Args[1]
	}
	dbPath := filepath.Join(dataDir, "inklab.db")
	gzPath := dbPath + ".gz"

	before := mustSize(dbPath)

	// VACUUM rewrites the file compactly. Releases follow bulk imports and
	// deletes, which leave partially-filled pages behind.
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		fatal("open sqlite", err)
	}
	if _, err := db.Exec("VACUUM"); err != nil {
		db.Close()
		fatal("vacuum", err)
	}
	db.Close()
	vacuumed := mustSize(dbPath)

	in, err := os.Open(dbPath)
	if err != nil {
		fatal("open db", err)
	}
	defer in.Close()

	out, err := os.Create(gzPath)
	if err != nil {
		fatal("create gz", err)
	}
	zw, err := gzip.NewWriterLevel(out, gzip.BestCompression)
	if err != nil {
		out.Close()
		fatal("gzip writer", err)
	}
	if _, err := io.Copy(zw, in); err != nil {
		zw.Close()
		out.Close()
		fatal("compress", err)
	}
	if err := zw.Close(); err != nil {
		out.Close()
		fatal("finish gzip", err)
	}
	if err := out.Close(); err != nil {
		fatal("close gz", err)
	}

	packed := mustSize(gzPath)
	fmt.Printf("✓ Packed %s\n", gzPath)
	fmt.Printf("  %s → %s vacuumed → %s gzipped (%.0f%% of original)\n",
		mb(before), mb(vacuumed), mb(packed), float64(packed)/float64(before)*100)

	// GitHub rejects any file over 100 MB, and the push only fails after the
	// upload — better to hear it here.
	const githubLimit = 100 << 20
	if packed > githubLimit {
		fmt.Printf("\n⚠ %s exceeds GitHub's 100 MB file limit — the push will be rejected.\n", mb(packed))
		os.Exit(1)
	}
}

func mb(n int64) string { return fmt.Sprintf("%.1f MB", float64(n)/(1<<20)) }

func mustSize(p string) int64 {
	fi, err := os.Stat(p)
	if err != nil {
		fatal("stat "+p, err)
	}
	return fi.Size()
}

func fatal(ctx string, err error) {
	fmt.Fprintf(os.Stderr, "error (%s): %v\n", ctx, err)
	os.Exit(1)
}
