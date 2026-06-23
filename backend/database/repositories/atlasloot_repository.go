package repositories

import (
	"database/sql"
	"fmt"

	"inklab/backend/database/models"
)

// AtlasLootRepository handles AtlasLoot data queries
type AtlasLootRepository struct {
	db *sql.DB
}

// NewAtlasLootRepository creates a new repository
func NewAtlasLootRepository(db *sql.DB) *AtlasLootRepository {
	return &AtlasLootRepository{db: db}
}

// GetCategories returns all category names
func (r *AtlasLootRepository) GetCategories() ([]string, error) {
	rows, err := r.db.Query(`
		SELECT display_name FROM atlasloot_categories 
		ORDER BY sort_order, display_name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		categories = append(categories, name)
	}
	return categories, nil
}

// GetModules returns module names for a category
func (r *AtlasLootRepository) GetModules(categoryName string) ([]string, error) {
	rows, err := r.db.Query(`
		SELECT m.display_name
		FROM atlasloot_modules m
		JOIN atlasloot_categories c ON m.category_id = c.id
		WHERE c.display_name = ?
		ORDER BY m.sort_order, m.display_name
	`, categoryName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var modules []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		modules = append(modules, name)
	}
	return modules, nil
}

// GetTables returns table references for a module
func (r *AtlasLootRepository) GetTables(categoryName, moduleName string) ([]models.AtlasTable, error) {
	rows, err := r.db.Query(`
		SELECT t.table_key, t.display_name
		FROM atlasloot_tables t
		JOIN atlasloot_modules m ON t.module_id = m.id
		JOIN atlasloot_categories c ON m.category_id = c.id
		WHERE c.display_name = ? AND m.display_name = ?
		ORDER BY t.sort_order, t.display_name
	`, categoryName, moduleName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []models.AtlasTable
	for rows.Next() {
		var t models.AtlasTable
		if err := rows.Scan(&t.Key, &t.DisplayName); err != nil {
			return nil, err
		}
		tables = append(tables, t)
	}
	return tables, nil
}

// GetLootItems returns items for a specific table
func (r *AtlasLootRepository) GetLootItems(catName, modName, tableKey string) ([]*models.LootEntry, error) {
	rows, err := r.db.Query(`
		SELECT al.item_id, al.drop_chance, al.sort_order, 
		       COALESCE(NULLIF(al.override_name, ''), i.name, ''), 
		       COALESCE(NULLIF(al.override_icon, ''), idi.icon, ''), 
		       COALESCE(i.quality, al.quality, 0),
		       al.spell_id
		FROM atlasloot_items al
		JOIN atlasloot_tables t ON al.table_id = t.id
		JOIN atlasloot_modules m ON t.module_id = m.id
		JOIN atlasloot_categories c ON m.category_id = c.id
		LEFT JOIN item_template i ON al.item_id = i.entry
		LEFT JOIN item_display_info idi ON i.display_id = idi.ID
		WHERE c.display_name = ? AND m.display_name = ? AND t.table_key = ?
		ORDER BY al.sort_order, al.item_id
	`, catName, modName, tableKey)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*models.LootEntry
	for rows.Next() {
		entry := &models.LootEntry{}
		var sortOrder int
		if err := rows.Scan(&entry.ItemID, &entry.DropChance, &sortOrder, &entry.Name, &entry.IconPath, &entry.Quality, &entry.SpellID); err != nil {
			return nil, err
		}
		items = append(items, entry)
	}
	return items, nil
}

// ClearAllData removes all AtlasLoot data
func (r *AtlasLootRepository) ClearAllData() error {
	tables := []string{"atlasloot_items", "atlasloot_tables", "atlasloot_modules", "atlasloot_categories"}
	for _, table := range tables {
		r.db.Exec(fmt.Sprintf("DELETE FROM %s", table))
	}
	return nil
}

// GetStats returns statistics about AtlasLoot data
func (r *AtlasLootRepository) GetStats() (map[string]int, error) {
	stats := make(map[string]int)
	tables := map[string]string{
		"categories": "atlasloot_categories",
		"modules":    "atlasloot_modules",
		"tables":     "atlasloot_tables",
		"items":      "atlasloot_items",
	}
	for key, table := range tables {
		var count int
		r.db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", table)).Scan(&count)
		stats[key] = count
	}
	return stats, nil
}
