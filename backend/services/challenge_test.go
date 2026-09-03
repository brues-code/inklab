package services

import (
	"database/sql"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"inklab/backend/parsers"

	_ "modernc.org/sqlite"
)

// blazingFastPage is the interstitial octowow.st serves behind BlazingFast,
// trimmed to the parts that identify it. Note the HTTP 200 in the tests below:
// that is what made this page indistinguishable from a real one. Detection
// itself is covered by parsers.TestIsChallengePage.
const blazingFastPage = `<!DOCTYPE HTML>
<html lang="en-US">
<head>
	<title>Just a moment please...</title>
	<script src="/bf.jquery.max.js"></script>
</head>
<body>
	<form id="bf-form" action="/blzgfst-shark/" method="get"></form>
</body>
</html>`

// realObjectPage stands in for an aowow object page: the spawn data lives in a
// myMapper.update call.
const realObjectPage = `<html><head><title>Copper Vein - Object</title></head><body>
<a href="#" onclick="myMapper.update({zone: 1519,coords: [[39.95,84.36,0]]})">Stormwind City</a>
</body></html>`

// stubClient serves one canned response to every request.
type stubClient struct {
	status int
	body   string
}

func (c *stubClient) Do(*http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: c.status,
		Body:       io.NopCloser(strings.NewReader(c.body)),
	}, nil
}

func (c *stubClient) Get(string) (*http.Response, error) { return c.Do(nil) }

func TestFetchPageRejectsChallenge(t *testing.T) {
	s := &ScraperService{Client: &stubClient{status: 200, body: blazingFastPage}}
	if _, err := s.fetchPage("https://example.test/db/?object=1731", scrapeUA); !errors.Is(err, ErrChallenge) {
		t.Fatalf("fetchPage error = %v, want ErrChallenge", err)
	}

	// A real page still comes back whole.
	s = &ScraperService{Client: &stubClient{status: 200, body: realObjectPage}}
	body, err := s.fetchPage("https://example.test/db/?object=1731", scrapeUA)
	if err != nil {
		t.Fatalf("fetchPage on a real page: %v", err)
	}
	if !strings.Contains(string(body), "myMapper.update") {
		t.Fatalf("body not returned intact: %q", body)
	}
}

// TestScrapeObjectRejectsChallenge covers the whole path that used to lose data:
// a 200 interstitial must surface as an error, not as an object with no spawns.
func TestScrapeObjectRejectsChallenge(t *testing.T) {
	s := &ScraperService{Client: &stubClient{status: 200, body: blazingFastPage}}
	obj, err := s.ScrapeObject(1731)
	if !errors.Is(err, ErrChallenge) {
		t.Fatalf("ScrapeObject error = %v, want ErrChallenge", err)
	}
	if obj != nil {
		t.Fatalf("ScrapeObject returned data for a challenge page: %+v", obj)
	}
}

// spawnTestDB is an in-memory database holding just the spawn table, with one
// shipped ('official') row for object 1731.
func spawnTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if _, err := db.Exec(`
		CREATE TABLE gameobject_spawn (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			gameobject_entry INTEGER,
			map_id INTEGER,
			zone_id INTEGER,
			zone_name TEXT,
			position_x REAL,
			position_y REAL,
			position_z REAL,
			origin TEXT NOT NULL DEFAULT 'official',
			UNIQUE(gameobject_entry, map_id, position_x, position_y)
		)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`
		INSERT INTO gameobject_spawn
			(gameobject_entry, map_id, zone_id, zone_name, position_x, position_y, position_z, origin)
		VALUES (1731, 0, 12, 'Elwynn Forest', 42.5, 63.1, 0, 'official')`); err != nil {
		t.Fatal(err)
	}
	return db
}

func spawnCount(t *testing.T, db *sql.DB) int {
	t.Helper()
	var n int
	if err := db.QueryRow("SELECT COUNT(*) FROM gameobject_spawn WHERE gameobject_entry = 1731").Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// TestWriteObjectSpawnsKeepsRowsOnEmptyScrape is the regression for the wipe: a
// response that parses to no spawn points (an interstitial, a page shape change)
// must leave the existing rows alone rather than deleting them and writing
// nothing back.
func TestWriteObjectSpawnsKeepsRowsOnEmptyScrape(t *testing.T) {
	db := spawnTestDB(t)
	s := &NpcService{sqlite: db}

	if n := s.writeObjectSpawns(1731, nil); n != 0 {
		t.Fatalf("writeObjectSpawns wrote %d rows for an empty scrape", n)
	}
	if got := spawnCount(t, db); got != 1 {
		t.Fatalf("official spawn count = %d after an empty scrape, want 1 (kept)", got)
	}
}

// A real scrape stays authoritative: its points replace what was there.
func TestWriteObjectSpawnsReplacesOnRealScrape(t *testing.T) {
	db := spawnTestDB(t)
	s := &NpcService{sqlite: db}

	n := s.writeObjectSpawns(1731, []parsers.SpawnPoint{
		{ZoneID: 1519, ZoneName: "Stormwind City", X: 39.95, Y: 84.36},
	})
	if n != 1 {
		t.Fatalf("writeObjectSpawns wrote %d rows, want 1", n)
	}
	if got := spawnCount(t, db); got != 1 {
		t.Fatalf("spawn count = %d after a real scrape, want 1", got)
	}
	var origin string
	if err := db.QueryRow("SELECT origin FROM gameobject_spawn WHERE gameobject_entry = 1731").Scan(&origin); err != nil {
		t.Fatal(err)
	}
	if origin != "local" {
		t.Fatalf("origin = %q after a scrape, want \"local\"", origin)
	}
}
