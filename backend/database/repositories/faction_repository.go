package repositories

import (
	"database/sql"

	"inklab/backend/database/models"
)

// FactionRepository handles faction-related database operations
type FactionRepository struct {
	db *sql.DB
}

// NewFactionRepository creates a new faction repository
func NewFactionRepository(db *sql.DB) *FactionRepository {
	return &FactionRepository{db: db}
}

// GetFactions returns all factions ordered by side and name
func (r *FactionRepository) GetFactions() ([]*models.Faction, error) {
	rows, err := r.db.Query(`
		SELECT id, name, description, side, category_id
		FROM factions
		ORDER BY side, name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var factions []*models.Faction
	for rows.Next() {
		f := &models.Faction{}
		var desc *string
		if err := rows.Scan(&f.ID, &f.Name, &desc, &f.Side, &f.CategoryId); err != nil {
			continue
		}
		if desc != nil {
			f.Description = *desc
		}
		factions = append(factions, f)
	}
	return factions, nil
}

// GetFactionDetail returns detailed information about a faction
func (r *FactionRepository) GetFactionDetail(id int) (*models.FactionDetail, error) {
	f := &models.FactionDetail{}

	var desc *string
	err := r.db.QueryRow(`
		SELECT id, name, description, side, category_id
		FROM factions WHERE id = ?
	`, id).Scan(&f.ID, &f.Name, &desc, &f.Side, &f.CategoryId)
	if err != nil {
		return nil, err
	}
	if desc != nil {
		f.Description = *desc
	}

	// Side name mapping
	switch f.Side {
	case 1:
		f.SideName = "Alliance"
	case 2:
		f.SideName = "Horde"
	default:
		f.SideName = "Neutral"
	}

	// Get quests that reward reputation with this faction
	questRows, _ := r.db.Query(`
		SELECT entry, Title, QuestLevel
		FROM quest_template
		WHERE RewRepFaction1 = ? OR RewRepFaction2 = ? OR RewRepFaction3 = ? OR RewRepFaction4 = ?
		ORDER BY QuestLevel
		LIMIT 100
	`, id, id, id, id)
	if questRows != nil {
		defer questRows.Close()
		for questRows.Next() {
			qr := &models.QuestRelation{}
			if err := questRows.Scan(&qr.Entry, &qr.Title, &qr.Level); err == nil {
				f.Quests = append(f.Quests, qr)
			}
		}
	}

	return f, nil
}
