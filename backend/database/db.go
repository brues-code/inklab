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
	// Add busy_timeout to prevent lock issues
	db, err := sql.Open("sqlite", dbPath+"?_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Set connection pool settings - allow multiple concurrent reads
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)

	// Enable WAL mode for better concurrency
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		return nil, fmt.Errorf("failed to set WAL mode: %w", err)
	}

	// Enable foreign keys
	if _, err := db.Exec("PRAGMA foreign_keys=ON"); err != nil {
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
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

	return nil
}

// DB returns the underlying sql.DB for direct queries
func (s *SQLiteDB) DB() *sql.DB {
	return s.db
}
