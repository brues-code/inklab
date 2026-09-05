package main

import (
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

// makeDB creates a database with the provenance-carrying tables and runs the
// given seed statements.
func makeDB(t *testing.T, path string, seed ...string) {
	t.Helper()
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(`
		CREATE TABLE creature_spawn (
			id INTEGER PRIMARY KEY AUTOINCREMENT, creature_entry INTEGER, map_id INTEGER,
			zone_id INTEGER, zone_name TEXT, position_x REAL, position_y REAL, position_z REAL,
			origin TEXT NOT NULL DEFAULT 'official',
			UNIQUE(creature_entry, map_id, position_x, position_y)
		);
		CREATE TABLE gameobject_spawn (
			id INTEGER PRIMARY KEY AUTOINCREMENT, gameobject_entry INTEGER, map_id INTEGER,
			zone_id INTEGER, zone_name TEXT, position_x REAL, position_y REAL, position_z REAL,
			origin TEXT NOT NULL DEFAULT 'official',
			UNIQUE(gameobject_entry, map_id, position_x, position_y)
		);
		CREATE TABLE creature_loot_template (
			entry INTEGER, item INTEGER, ChanceOrQuestChance REAL DEFAULT 0, groupid INTEGER DEFAULT 0,
			mincountOrRef INTEGER DEFAULT 1, maxcount INTEGER DEFAULT 1,
			origin TEXT NOT NULL DEFAULT 'official',
			PRIMARY KEY (entry, item)
		)`); err != nil {
		t.Fatal(err)
	}
	for _, s := range seed {
		if _, err := db.Exec(s); err != nil {
			t.Fatalf("seed %q: %v", s, err)
		}
	}
}

func query(t *testing.T, path, q string) [][2]string {
	t.Helper()
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	rows, err := db.Query(q)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var out [][2]string
	for rows.Next() {
		var a, b string
		if err := rows.Scan(&a, &b); err != nil {
			t.Fatal(err)
		}
		out = append(out, [2]string{a, b})
	}
	return out
}

// TestGraftLocalRows covers the two merge shapes an upgrade has to handle: spawn
// points accumulate alongside the new baseline, while a scraped loot table
// replaces the baseline's version of that same creature outright.
func TestGraftLocalRows(t *testing.T) {
	dir := t.TempDir()
	newPath := filepath.Join(dir, "new.db")
	oldPath := filepath.Join(dir, "old.db")

	// New baseline: one official spawn and a three-row loot table for creature 42.
	makeDB(t, newPath,
		`INSERT INTO creature_spawn (creature_entry, map_id, zone_name, position_x, position_y, origin)
		 VALUES (42, 0, 'Elwynn Forest', 10, 10, 'official')`,
		`INSERT INTO creature_loot_template (entry, item, ChanceOrQuestChance, origin) VALUES
			(42, 111, 5, 'official'), (42, 222, 5, 'official'), (42, 333, 5, 'official'),
			(99, 777, 1, 'official')`,
	)
	// The user's DB: a scraped spawn elsewhere, and a scraped loot table for 42
	// that shares one item with the baseline and adds another.
	makeDB(t, oldPath,
		`INSERT INTO creature_spawn (creature_entry, map_id, zone_name, position_x, position_y, origin)
		 VALUES (42, 0, 'Westfall', 55, 55, 'local')`,
		`INSERT INTO creature_loot_template (entry, item, ChanceOrQuestChance, origin) VALUES
			(42, 111, 42.5, 'local'), (42, 444, 12, 'local')`,
	)

	if err := graftLocalRows(newPath, oldPath); err != nil {
		t.Fatalf("graftLocalRows: %v", err)
	}

	// Spawns accumulate: both the official and the scraped point survive.
	spawns := query(t, newPath, "SELECT zone_name, origin FROM creature_spawn WHERE creature_entry = 42 ORDER BY zone_name")
	if len(spawns) != 2 {
		t.Errorf("creature_spawn rows = %d, want 2 (official + local): %v", len(spawns), spawns)
	}

	// Loot replaces: creature 42's table is exactly the two scraped rows. The
	// baseline's 222/333 must be gone — keeping them would merge two servers'
	// loot tables into one list.
	loot := query(t, newPath, "SELECT item, origin FROM creature_loot_template WHERE entry = 42 ORDER BY item")
	want := [][2]string{{"111", "local"}, {"444", "local"}}
	if len(loot) != len(want) {
		t.Fatalf("creature 42 loot = %v, want %v", loot, want)
	}
	for i := range want {
		if loot[i] != want[i] {
			t.Errorf("creature 42 loot[%d] = %v, want %v", i, loot[i], want[i])
		}
	}
	// The scraped chance wins for the shared item, rather than the baseline's.
	chance := query(t, newPath, "SELECT ChanceOrQuestChance, origin FROM creature_loot_template WHERE entry = 42 AND item = 111")
	if len(chance) != 1 || chance[0][0] != "42.5" {
		t.Errorf("item 111 chance = %v, want the scraped 42.5", chance)
	}

	// A creature the user never scraped keeps the shipped table untouched.
	untouched := query(t, newPath, "SELECT item, origin FROM creature_loot_template WHERE entry = 99")
	if len(untouched) != 1 || untouched[0] != [2]string{"777", "official"} {
		t.Errorf("creature 99 loot = %v, want the shipped row", untouched)
	}
}

// An older DB without the origin column must not break the upgrade: the graft
// skips what it can't read and the user still gets the fresh baseline.
func TestGraftLocalRowsPreOriginDB(t *testing.T) {
	dir := t.TempDir()
	newPath := filepath.Join(dir, "new.db")
	oldPath := filepath.Join(dir, "old.db")

	makeDB(t, newPath, `INSERT INTO creature_loot_template (entry, item, origin) VALUES (42, 111, 'official')`)

	db, err := sql.Open("sqlite", oldPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`CREATE TABLE creature_loot_template (entry INTEGER, item INTEGER, PRIMARY KEY (entry, item))`); err != nil {
		t.Fatal(err)
	}
	db.Close()

	if err := graftLocalRows(newPath, oldPath); err != nil {
		t.Fatalf("graftLocalRows on a pre-origin DB: %v", err)
	}
	if got := query(t, newPath, "SELECT item, origin FROM creature_loot_template WHERE entry = 42"); len(got) != 1 {
		t.Errorf("baseline loot = %v, want the shipped row left intact", got)
	}
}
