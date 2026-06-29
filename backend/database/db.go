// Package database provides SQLite database operations for InkLab
package database

import (
	"database/sql"
	"fmt"
	"sync"

	"inklab/backend/database/schema"

	_ "modernc.org/sqlite"
)

// SQLiteDB wraps the SQLite database connection
type SQLiteDB struct {
	db *sql.DB
	mu sync.RWMutex
}

// NewSQLiteDB creates a new SQLite database connection
func NewSQLiteDB(dbPath string) (*SQLiteDB, error) {
	// modernc.org/sqlite uses the "_pragma=" DSN form — NOT mattn's
	// "_busy_timeout=" param, which it silently ignores. These pragmas must be
	// set via the DSN so they apply to EVERY pooled connection: a connection-only
	// `PRAGMA busy_timeout` (via db.Exec) leaves the other pooled connections at
	// timeout 0, so concurrent writers (e.g. the parallel sync workers) fail
	// immediately with SQLITE_BUSY instead of waiting for the write lock.
	dsn := dbPath + "?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Set connection pool settings - allow multiple concurrent reads
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)

	// Verify the connection (and apply the DSN pragmas on the first conn).
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	return &SQLiteDB{db: db}, nil
}

// Close closes the database connection
func (s *SQLiteDB) Close() error {
	return s.db.Close()
}

// InitSchema creates the database schema if it doesn't exist
func (s *SQLiteDB) InitSchema() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Create generated 1:1 MySQL tables FIRST (item_template, creature_template, etc.)
	if _, err := s.db.Exec(schema.GeneratedSchema()); err != nil {
		return fmt.Errorf("failed to create generated schema: %w", err)
	}

	// Create core tables (depends on 1:1 tables for indexes)
	if _, err := s.db.Exec(schema.CoreSchema()); err != nil {
		return fmt.Errorf("failed to create core schema: %w", err)
	}

	// Create AtlasLoot tables
	if _, err := s.db.Exec(schema.AtlasLootSchema()); err != nil {
		return fmt.Errorf("failed to create atlasloot schema: %w", err)
	}

	// Create locale tables
	if _, err := s.db.Exec(schema.LocaleSchema()); err != nil {
		return fmt.Errorf("failed to create locale schema: %w", err)
	}

	// Apply Migrations
	schema.MigrateV2(s.db)
	schema.MigrateAtlasLoot(s.db)
	schema.MigratePerformance(s.db)
	schema.MigrateTextBlobs(s.db)
	s.db.Exec("ALTER TABLE spell_skills ADD COLUMN class_id INTEGER DEFAULT 0")         // ignore error if exists
	s.db.Exec("ALTER TABLE spell_skill_spells ADD COLUMN classmask INTEGER DEFAULT 0")  // ignore error if exists
	s.db.Exec("ALTER TABLE class_info ADD COLUMN color TEXT DEFAULT ''")                // ignore error if exists
	s.db.Exec("ALTER TABLE faction_template ADD COLUMN our_mask INTEGER DEFAULT 0")    // ignore error if exists
	s.db.Exec("ALTER TABLE faction_template ADD COLUMN friend_mask INTEGER DEFAULT 0") // ignore error if exists
	s.db.Exec("ALTER TABLE faction_template ADD COLUMN enemy_mask INTEGER DEFAULT 0")  // ignore error if exists
	s.db.Exec("ALTER TABLE quest_categories_enhanced ADD COLUMN display_name TEXT DEFAULT ''") // ignore error if exists
	s.db.Exec("ALTER TABLE taxi_node ADD COLUMN world_x REAL DEFAULT 0")                       // ignore error if exists
	s.db.Exec("ALTER TABLE taxi_node ADD COLUMN world_y REAL DEFAULT 0")                       // ignore error if exists
	s.db.Exec("ALTER TABLE taxi_node ADD COLUMN alliance_npc INTEGER DEFAULT 0")               // ignore error if exists
	s.db.Exec("ALTER TABLE taxi_node ADD COLUMN horde_npc INTEGER DEFAULT 0")                  // ignore error if exists

	return nil
}

// DB returns the underlying sql.DB for direct queries
func (s *SQLiteDB) DB() *sql.DB {
	return s.db
}
