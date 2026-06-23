package repositories

import (
	"database/sql"

	"inklab/backend/database/models"
)

// CategoryRepository handles category-related database operations
type CategoryRepository struct {
	db *sql.DB
}

// NewCategoryRepository creates a new category repository
func NewCategoryRepository(db *sql.DB) *CategoryRepository {
	return &CategoryRepository{db: db}
}

// InsertCategory inserts a new category
func (r *CategoryRepository) InsertCategory(cat *models.Category) (int64, error) {
	result, err := r.db.Exec(`
		INSERT OR REPLACE INTO categories (key, name, parent_id, type, sort_order)
		VALUES (?, ?, ?, ?, ?)
	`, cat.Key, cat.Name, cat.ParentID, cat.Type, cat.SortOrder)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

// InsertCategoryItem links an item to a category
func (r *CategoryRepository) InsertCategoryItem(catID, itemID int, dropRate string, sortOrder int) error {
	_, err := r.db.Exec(`
		INSERT INTO category_items (category_id, item_id, drop_rate, sort_order)
		VALUES (?, ?, ?, ?)
	`, catID, itemID, dropRate, sortOrder)
	return err
}

// GetRootCategories returns all top-level categories
func (r *CategoryRepository) GetRootCategories() ([]*models.Category, error) {
	rows, err := r.db.Query(`
		SELECT id, key, name, parent_id, type, sort_order
		FROM categories
		WHERE parent_id IS NULL
		ORDER BY sort_order, name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return r.scanCategories(rows)
}

// GetChildCategories returns child categories of a parent
func (r *CategoryRepository) GetChildCategories(parentID int) ([]*models.Category, error) {
	rows, err := r.db.Query(`
		SELECT id, key, name, parent_id, type, sort_order
		FROM categories
		WHERE parent_id = ?
		ORDER BY sort_order, name
	`, parentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return r.scanCategories(rows)
}

// GetCategoryByKey retrieves a category by its key
func (r *CategoryRepository) GetCategoryByKey(key string) (*models.Category, error) {
	cat := &models.Category{}
	var parentID *int
	err := r.db.QueryRow(`
		SELECT id, key, name, parent_id, type, sort_order FROM categories WHERE key = ?
	`, key).Scan(&cat.ID, &cat.Key, &cat.Name, &parentID, &cat.Type, &cat.SortOrder)
	if err != nil {
		return nil, err
	}
	cat.ParentID = parentID
	return cat, nil
}

// GetCategoryItems returns all items in a category
func (r *CategoryRepository) GetCategoryItems(categoryID int) ([]*models.Item, error) {
	rows, err := r.db.Query(`
		SELECT i.entry, i.name, i.quality, i.item_level, i.required_level,
			i.class, i.subclass, i.inventory_type, COALESCE(idi.icon, ''), ci.drop_rate
		FROM item_template i
		JOIN category_items ci ON i.entry = ci.item_id
		LEFT JOIN item_display_info idi ON i.display_id = idi.ID
		WHERE ci.category_id = ?
		ORDER BY ci.sort_order, i.quality DESC, i.item_level DESC
	`, categoryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*models.Item
	for rows.Next() {
		item := &models.Item{}
		var dropRate string
		err := rows.Scan(
			&item.Entry, &item.Name, &item.Quality, &item.ItemLevel,
			&item.RequiredLevel, &item.Class, &item.SubClass, &item.InventoryType, &item.IconPath,
			&dropRate,
		)
		if err != nil {
			continue
		}
		item.DropRate = dropRate
		items = append(items, item)
	}
	return items, nil
}

// GetCategoryCount returns the total number of categories
func (r *CategoryRepository) GetCategoryCount() (int, error) {
	var count int
	err := r.db.QueryRow("SELECT COUNT(*) FROM categories").Scan(&count)
	return count, err
}

// Helper to scan category rows
func (r *CategoryRepository) scanCategories(rows *sql.Rows) ([]*models.Category, error) {
	var cats []*models.Category
	for rows.Next() {
		cat := &models.Category{}
		var parentID *int
		err := rows.Scan(&cat.ID, &cat.Key, &cat.Name, &parentID, &cat.Type, &cat.SortOrder)
		if err != nil {
			continue
		}
		cat.ParentID = parentID
		cats = append(cats, cat)
	}
	return cats, nil
}
