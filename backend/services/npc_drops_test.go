package services

import (
	"database/sql"
	"testing"

	"inklab/backend/parsers"

	_ "modernc.org/sqlite"
)

// lootTestDB is an in-memory database with a creature and the loot table,
// seeded the way the shipped (leaked Turtle 1.17.2) dump looks: a couple of
// direct rows plus a reference pointer (mincountOrRef < 0) into the world-drop
// table.
func lootTestDB(t *testing.T, entry, lootID int) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if _, err := db.Exec(`
		CREATE TABLE creature_template (entry INTEGER PRIMARY KEY, name TEXT, loot_id INTEGER DEFAULT 0);
		CREATE TABLE creature_loot_template (
			entry INTEGER, item INTEGER, ChanceOrQuestChance REAL DEFAULT 0,
			groupid INTEGER DEFAULT 0, mincountOrRef INTEGER DEFAULT 1, maxcount INTEGER DEFAULT 1,
			origin TEXT NOT NULL DEFAULT 'official',
			PRIMARY KEY (entry, item)
		)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("INSERT INTO creature_template (entry, name, loot_id) VALUES (?, 'Test Mob', ?)", entry, lootID); err != nil {
		t.Fatal(err)
	}
	key := lootID
	if key == 0 {
		key = entry
	}
	if _, err := db.Exec(`
		INSERT INTO creature_loot_template (entry, item, ChanceOrQuestChance, groupid, mincountOrRef, maxcount) VALUES
			(?, 2589, 23.33, 0, 1, 4),
			(?, 30018, 5, 0, -30018, 1)`, key, key); err != nil {
		t.Fatal(err)
	}
	return db
}

func lootRows(t *testing.T, db *sql.DB, key int) map[int][3]float64 {
	t.Helper()
	rows, err := db.Query("SELECT item, ChanceOrQuestChance, mincountOrRef, maxcount FROM creature_loot_template WHERE entry = ?", key)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	out := map[int][3]float64{}
	for rows.Next() {
		var item int
		var chance, min, max float64
		if err := rows.Scan(&item, &chance, &min, &max); err != nil {
			t.Fatal(err)
		}
		out[item] = [3]float64{chance, min, max}
	}
	return out
}

// A scraped drop table is authoritative: it replaces the shipped rows outright,
// reference pointers included, since octowow's list already has that loot
// flattened into it.
func TestWriteNpcDropsReplaces(t *testing.T) {
	const entry = 62172
	db := lootTestDB(t, entry, 0)
	s := &NpcService{sqlite: db}

	n := s.writeNpcDrops(entry, []parsers.ItemDrop{
		{ItemEntry: 857, Chance: 0.0466, MinCount: 1, MaxCount: 1},
		{ItemEntry: 2592, Chance: 17, MinCount: 1, MaxCount: 4},
		{ItemEntry: 3637, Chance: -100, MinCount: 1, MaxCount: 1},
	})
	if n != 3 {
		t.Fatalf("wrote %d rows, want 3", n)
	}

	got := lootRows(t, db, entry)
	if len(got) != 3 {
		t.Fatalf("loot rows = %d, want 3 (old rows should be gone): %+v", len(got), got)
	}
	if _, stale := got[30018]; stale {
		t.Errorf("reference pointer row survived; it would double-count world drops")
	}
	if r := got[2592]; r != [3]float64{17, 1, 4} {
		t.Errorf("Wool Cloth row = %v; want chance 17, stack 1-4", r)
	}
	// The quest drop's negative chance must reach the column unchanged.
	if r := got[3637]; r[0] != -100 {
		t.Errorf("quest drop chance = %v; want -100", r[0])
	}

	// Scraped rows are the user's own data: 'local' provenance is what carries
	// them across an embedded-DB upgrade and what promotedb publishes.
	var local int
	if err := db.QueryRow("SELECT COUNT(*) FROM creature_loot_template WHERE entry = ? AND origin = 'local'", entry).Scan(&local); err != nil {
		t.Fatal(err)
	}
	if local != 3 {
		t.Errorf("rows tagged local = %d, want 3", local)
	}
}

// A scraped drop table outranks the world DB: the MySQL loot sync must leave it
// alone rather than overwriting it with the (different server's) dump.
func TestHasLocalLoot(t *testing.T) {
	const entry = 62172
	db := lootTestDB(t, entry, 0)
	s := &NpcService{sqlite: db}

	if s.hasLocalLoot(entry) {
		t.Fatal("shipped rows reported as local")
	}
	s.writeNpcDrops(entry, []parsers.ItemDrop{{ItemEntry: 857, Chance: 1, MinCount: 1, MaxCount: 1}})
	if !s.hasLocalLoot(entry) {
		t.Fatal("scraped rows not reported as local")
	}
}

// An empty scrape must not touch existing loot — the same rule that stopped an
// interstitial from wiping spawn data.
func TestWriteNpcDropsKeepsRowsOnEmptyScrape(t *testing.T) {
	const entry = 62172
	db := lootTestDB(t, entry, 0)
	s := &NpcService{sqlite: db}

	if n := s.writeNpcDrops(entry, nil); n != 0 {
		t.Fatalf("wrote %d rows for an empty scrape", n)
	}
	if got := lootRows(t, db, entry); len(got) != 2 {
		t.Fatalf("loot rows = %d after an empty scrape, want the original 2", len(got))
	}
}

// Creatures that share a loot table are keyed by loot_id, which is also how
// GetNpcDetails reads loot back — write it anywhere else and the NPC page shows
// nothing.
func TestWriteNpcDropsUsesLootID(t *testing.T) {
	const entry, lootID = 5000, 4321
	db := lootTestDB(t, entry, lootID)
	s := &NpcService{sqlite: db}

	if n := s.writeNpcDrops(entry, []parsers.ItemDrop{{ItemEntry: 765, Chance: 0.16, MinCount: 1, MaxCount: 1}}); n != 1 {
		t.Fatalf("wrote %d rows, want 1", n)
	}
	if got := lootRows(t, db, lootID); len(got) != 1 {
		t.Fatalf("rows under loot_id %d = %d, want 1", lootID, len(got))
	}
	if got := lootRows(t, db, entry); len(got) != 0 {
		t.Fatalf("rows written under the entry instead of loot_id: %+v", got)
	}
}
