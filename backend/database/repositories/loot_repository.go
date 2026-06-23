package repositories

import (
	"database/sql"
	"math"

	"inklab/backend/database/models"
)

// LootRepository handles loot-related database operations
type LootRepository struct {
	db *sql.DB
}

// NewLootRepository creates a new loot repository
func NewLootRepository(db *sql.DB) *LootRepository {
	return &LootRepository{db: db}
}

// GetCreatureLoot returns the flattened loot table for a creature
func (r *LootRepository) GetCreatureLoot(creatureEntry int) ([]*models.LootItem, error) {
	// 1. Get loot_id FROM creature_template table
	var lootID int
	err := r.db.QueryRow("SELECT loot_id FROM creature_template WHERE entry = ?", creatureEntry).Scan(&lootID)
	if err != nil {
		if err == sql.ErrNoRows {
			return []*models.LootItem{}, nil
		}
		return nil, err
	}

	if lootID == 0 {
		return []*models.LootItem{}, nil
	}

	// 2. Process loot recursively
	lootMap := make(map[int]*models.LootItem)
	if err := r.processLoot(lootID, 1.0, false, lootMap, 0); err != nil {
		return nil, err
	}

	// 3. Convert map to slice and enrich with item info
	var lootList []*models.LootItem
	for itemID, item := range lootMap {
		// Enrich with name, icon, quality
		var name, icon string
		var quality int
		err := r.db.QueryRow(`
			SELECT i.name, i.quality, COALESCE(idi.icon, '') 
			FROM item_template i 
			LEFT JOIN item_display_info idi ON i.display_id = idi.ID 
			WHERE i.entry = ?
		`, itemID).Scan(&name, &quality, &icon)
		if err == nil {
			item.Name = name
			item.Quality = quality
			item.IconPath = icon
			lootList = append(lootList, item)
		}
	}

	return lootList, nil
}

// processLoot recursively processes loot tables
func (r *LootRepository) processLoot(entry int, multiplier float64, isRef bool, results map[int]*models.LootItem, depth int) error {
	if depth > 10 {
		return nil // Prevent infinite recursion
	}

	tableName := "creature_loot_template"
	if isRef {
		tableName = "reference_loot_template"
	}

	rows, err := r.db.Query(`
		SELECT item, ChanceOrQuestChance, mincountOrRef, maxcount, groupid
		FROM `+tableName+` 
		WHERE entry = ?
	`, entry)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var itemID, minOrRef, maxCount, groupID int
		var chance float64
		if err := rows.Scan(&itemID, &chance, &minOrRef, &maxCount, &groupID); err != nil {
			continue
		}

		// Calculate actual chance
		absChance := math.Abs(chance)
		currentChance := absChance * multiplier

		if minOrRef < 0 {
			// Reference
			refID := -minOrRef
			if err := r.processLoot(refID, currentChance/100.0, true, results, depth+1); err != nil {
				return err
			}
		} else {
			// Item
			if currentChance < 0.0001 {
				currentChance = 0.0001
			}

			if existing, ok := results[itemID]; ok {
				existing.Chance += currentChance
			} else {
				results[itemID] = &models.LootItem{
					ItemID:   itemID,
					Chance:   currentChance,
					MinCount: minOrRef,
					MaxCount: maxCount,
				}
			}
		}
	}
	return nil
}
