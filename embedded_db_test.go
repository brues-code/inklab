package main

import (
	"bytes"
	"compress/gzip"
	"database/sql"
	"io"
	"testing"

	_ "modernc.org/sqlite"
)

// TestEmbeddedDBIsAValidArchive guards the shipped artifact: the build embeds
// data/inklab.db.gz, and a truncated or wrongly-packed archive would only
// surface as a failed extraction on a user's first run. Checking the gzip header
// and SQLite magic costs nothing and catches both.
func TestEmbeddedDBIsAValidArchive(t *testing.T) {
	if len(embeddedDBGz) == 0 {
		t.Fatal("embedded database archive is empty")
	}

	zr, err := gzip.NewReader(bytes.NewReader(embeddedDBGz))
	if err != nil {
		t.Fatalf("embedded archive is not valid gzip: %v", err)
	}
	defer zr.Close()

	const magic = "SQLite format 3\x00"
	head := make([]byte, len(magic))
	if _, err := io.ReadFull(zr, head); err != nil {
		t.Fatalf("reading the archive: %v", err)
	}
	if string(head) != magic {
		t.Fatalf("decompressed header = %q; want a SQLite database", head)
	}
}

// The extraction path itself must produce a database that opens.
func TestWriteEmbeddedDB(t *testing.T) {
	if testing.Short() {
		t.Skip("decompresses the whole baseline")
	}
	path := t.TempDir() + "/inklab.db"
	if err := writeEmbeddedDB(path); err != nil {
		t.Fatalf("writeEmbeddedDB: %v", err)
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("opening the extracted database: %v", err)
	}
	defer db.Close()
	var n int
	if err := db.QueryRow("SELECT COUNT(*) FROM item_template").Scan(&n); err != nil {
		t.Fatalf("querying the extracted database: %v", err)
	}
	if n == 0 {
		t.Error("extracted database has no items")
	}
}
