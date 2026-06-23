package services

import (
	"fmt"
	"inklab/backend/database/models"
)

// UpsertItemSet inserts or updates an item set in the database
func (s *SyncService) UpsertItemSet(set *models.ItemSetEntry) error {
	_, err := s.db.Exec(`
		INSERT INTO itemsets (
			itemset_id, name, 
			item1, item2, item3, item4, item5, item6, item7, item8, item9, item10,
			bonus1, bonus2, bonus3, bonus4, bonus5, bonus6, bonus7, bonus8,
			spell1, spell2, spell3, spell4, spell5, spell6, spell7, spell8
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(itemset_id) DO UPDATE SET
			name=excluded.name,
			item1=excluded.item1, item2=excluded.item2, item3=excluded.item3, item4=excluded.item4, item5=excluded.item5,
			item6=excluded.item6, item7=excluded.item7, item8=excluded.item8, item9=excluded.item9, item10=excluded.item10,
			bonus1=excluded.bonus1, bonus2=excluded.bonus2, bonus3=excluded.bonus3, bonus4=excluded.bonus4,
			bonus5=excluded.bonus5, bonus6=excluded.bonus6, bonus7=excluded.bonus7, bonus8=excluded.bonus8,
			spell1=excluded.spell1, spell2=excluded.spell2, spell3=excluded.spell3, spell4=excluded.spell4,
			spell5=excluded.spell5, spell6=excluded.spell6, spell7=excluded.spell7, spell8=excluded.spell8
	`,
		set.ID, set.Name,
		set.Item1, set.Item2, set.Item3, set.Item4, set.Item5, set.Item6, set.Item7, set.Item8, set.Item9, set.Item10,
		set.Bonus1, set.Bonus2, set.Bonus3, set.Bonus4, set.Bonus5, set.Bonus6, set.Bonus7, set.Bonus8,
		set.Spell1, set.Spell2, set.Spell3, set.Spell4, set.Spell5, set.Spell6, set.Spell7, set.Spell8,
	)
	if err != nil {
		return fmt.Errorf("failed to upsert item set %d: %w", set.ID, err)
	}
	return nil
}
