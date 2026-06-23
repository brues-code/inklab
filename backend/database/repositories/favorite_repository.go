package repositories

import (
	"database/sql"
	"fmt"
	"time"

	"inklab/backend/database/models"
)

// FavoriteRepository handles favorite item operations
type FavoriteRepository struct {
	db *sql.DB
}

// NewFavoriteRepository creates a new FavoriteRepository
func NewFavoriteRepository(db *sql.DB) *FavoriteRepository {
	return &FavoriteRepository{db: db}
}

// InitSchema creates the favorites table if not exists
func (r *FavoriteRepository) InitSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS favorites (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		item_entry INTEGER NOT NULL,
		category TEXT DEFAULT '',
		added_at TEXT NOT NULL,
		UNIQUE(item_entry)
	);
	CREATE INDEX IF NOT EXISTS idx_favorites_item ON favorites(item_entry);
	CREATE INDEX IF NOT EXISTS idx_favorites_category ON favorites(category);
	`
	_, err := r.db.Exec(schema)
	if err != nil {
		return err
	}

	// Try to add status column if it doesn't exist
	// We ignore the error "duplicate column name" if it already exists
	_, _ = r.db.Exec(`ALTER TABLE favorites ADD COLUMN status INTEGER DEFAULT 0`)

	return nil
}

// AddFavorite adds an item to favorites
func (r *FavoriteRepository) AddFavorite(itemEntry int, category string) error {
	now := time.Now().Format(time.RFC3339)
	_, err := r.db.Exec(`
		INSERT OR REPLACE INTO favorites (item_entry, category, added_at, status)
		VALUES (?, ?, ?, 0)
	`, itemEntry, category, now)
	return err
}

// RemoveFavorite removes an item from favorites
func (r *FavoriteRepository) RemoveFavorite(itemEntry int) error {
	_, err := r.db.Exec(`DELETE FROM favorites WHERE item_entry = ?`, itemEntry)
	return err
}

// IsFavorite checks if an item is in favorites
func (r *FavoriteRepository) IsFavorite(itemEntry int) (bool, error) {
	var count int
	err := r.db.QueryRow(`SELECT COUNT(*) FROM favorites WHERE item_entry = ?`, itemEntry).Scan(&count)
	return count > 0, err
}

// GetFavorite gets a single favorite by item entry
func (r *FavoriteRepository) GetFavorite(itemEntry int) (*models.FavoriteItem, error) {
	row := r.db.QueryRow(`
		SELECT f.id, f.item_entry, f.category, f.added_at,
		       COALESCE(i.name, ''), COALESCE(i.quality, 0), 
		       COALESCE(di.icon, ''), COALESCE(i.item_level, 0),
		       COALESCE(f.status, 0)
		FROM favorites f
		LEFT JOIN item_template i ON f.item_entry = i.entry
		LEFT JOIN item_display_info di ON i.display_id = di.id
		WHERE f.item_entry = ?
	`, itemEntry)

	fav := &models.FavoriteItem{}
	err := row.Scan(
		&fav.ID, &fav.ItemEntry, &fav.Category, &fav.AddedAt,
		&fav.ItemName, &fav.ItemQuality, &fav.IconPath, &fav.ItemLevel,
		&fav.Status,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return fav, err
}

// GetAllFavorites returns all favorites with item details
func (r *FavoriteRepository) GetAllFavorites() ([]*models.FavoriteItem, error) {
	rows, err := r.db.Query(`
		SELECT f.id, f.item_entry, f.category, f.added_at,
		       COALESCE(i.name, ''), COALESCE(i.quality, 0), 
		       COALESCE(di.icon, ''), COALESCE(i.item_level, 0),
		       COALESCE(f.status, 0)
		FROM favorites f
		LEFT JOIN item_template i ON f.item_entry = i.entry
		LEFT JOIN item_display_info di ON i.display_id = di.id
		ORDER BY f.category, f.added_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*models.FavoriteItem
	for rows.Next() {
		fav := &models.FavoriteItem{}
		if err := rows.Scan(
			&fav.ID, &fav.ItemEntry, &fav.Category, &fav.AddedAt,
			&fav.ItemName, &fav.ItemQuality, &fav.IconPath, &fav.ItemLevel,
			&fav.Status,
		); err != nil {
			fmt.Printf("Error scanning favorite: %v\n", err)
			continue
		}
		items = append(items, fav)
	}
	return items, nil
}

// GetFavoritesByCategory returns favorites filtered by category
func (r *FavoriteRepository) GetFavoritesByCategory(category string) ([]*models.FavoriteItem, error) {
	rows, err := r.db.Query(`
		SELECT f.id, f.item_entry, f.category, f.added_at,
		       COALESCE(i.name, ''), COALESCE(i.quality, 0), 
		       COALESCE(di.icon, ''), COALESCE(i.item_level, 0),
		       COALESCE(f.status, 0)
		FROM favorites f
		LEFT JOIN item_template i ON f.item_entry = i.entry
		LEFT JOIN item_display_info di ON i.display_id = di.id
		WHERE f.category = ?
		ORDER BY f.added_at DESC
	`, category)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*models.FavoriteItem
	for rows.Next() {
		fav := &models.FavoriteItem{}
		if err := rows.Scan(
			&fav.ID, &fav.ItemEntry, &fav.Category, &fav.AddedAt,
			&fav.ItemName, &fav.ItemQuality, &fav.IconPath, &fav.ItemLevel,
			&fav.Status,
		); err != nil {
			continue
		}
		items = append(items, fav)
	}
	return items, nil
}

// GetCategories returns all distinct categories with item counts
func (r *FavoriteRepository) GetCategories() ([]*models.FavoriteCategory, error) {
	rows, err := r.db.Query(`
		SELECT COALESCE(category, '') as cat, COUNT(*) as cnt
		FROM favorites
		GROUP BY category
		ORDER BY category
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cats []*models.FavoriteCategory
	for rows.Next() {
		cat := &models.FavoriteCategory{}
		if err := rows.Scan(&cat.Name, &cat.Count); err != nil {
			continue
		}
		cats = append(cats, cat)
	}
	return cats, nil
}

// UpdateCategory updates the category of a favorite
func (r *FavoriteRepository) UpdateCategory(itemEntry int, category string) error {
	_, err := r.db.Exec(`UPDATE favorites SET category = ? WHERE item_entry = ?`, category, itemEntry)
	return err
}

// UpdateStatus updates the status of a favorite
func (r *FavoriteRepository) UpdateStatus(itemEntry int, status int) error {
	_, err := r.db.Exec(`UPDATE favorites SET status = ? WHERE item_entry = ?`, status, itemEntry)
	return err
}

// GetFavoriteCount returns the total number of favorites
func (r *FavoriteRepository) GetFavoriteCount() (int, error) {
	var count int
	err := r.db.QueryRow(`SELECT COUNT(*) FROM favorites`).Scan(&count)
	return count, err
}
