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
		SELECT entry, Title, QuestLevel, IFNULL(RequiredRaces,0)
		FROM quest_template
		WHERE RewRepFaction1 = ? OR RewRepFaction2 = ? OR RewRepFaction3 = ? OR RewRepFaction4 = ?
		ORDER BY QuestLevel
		LIMIT 100
	`, id, id, id, id)
	if questRows != nil {
		defer questRows.Close()
		for questRows.Next() {
			qr := &models.QuestRelation{}
			var reqRaces int
			if err := questRows.Scan(&qr.Entry, &qr.Title, &qr.Level, &reqRaces); err == nil {
				qr.Side, _ = resolveSideAndRaces(reqRaces)
				f.Quests = append(f.Quests, qr)
			}
		}
	}

	// Quest Givers: NPCs that start or end any of this faction's rep quests.
	giverRows, _ := r.db.Query(`
		SELECT DISTINCT c.entry, c.name, COALESCE(c.subname, ''), c.level_min, c.level_max
		FROM creature_template c
		JOIN (
			SELECT id FROM creature_questrelation WHERE quest IN (
				SELECT entry FROM quest_template
				WHERE RewRepFaction1=? OR RewRepFaction2=? OR RewRepFaction3=? OR RewRepFaction4=? OR RewRepFaction5=?
			)
			UNION
			SELECT id FROM creature_involvedrelation WHERE quest IN (
				SELECT entry FROM quest_template
				WHERE RewRepFaction1=? OR RewRepFaction2=? OR RewRepFaction3=? OR RewRepFaction4=? OR RewRepFaction5=?
			)
		) rel ON c.entry = rel.id
		ORDER BY c.name
		LIMIT 200
	`, id, id, id, id, id, id, id, id, id, id)
	if giverRows != nil {
		defer giverRows.Close()
		for giverRows.Next() {
			n := &models.FactionNpc{}
			if err := giverRows.Scan(&n.Entry, &n.Name, &n.Subname, &n.LevelMin, &n.LevelMax); err == nil {
				f.QuestGivers = append(f.QuestGivers, n)
			}
		}
	}

	// Faction Members: creatures whose faction-template maps to this faction.
	// Requires faction_template (imported from FactionTemplate.dbc); empty until
	// the client import has generated it.
	memberRows, _ := r.db.Query(`
		SELECT entry, name, COALESCE(subname, ''), level_min, level_max
		FROM creature_template
		WHERE faction IN (SELECT template_id FROM faction_template WHERE faction_id = ?)
		ORDER BY name
		LIMIT 300
	`, id)
	if memberRows != nil {
		defer memberRows.Close()
		for memberRows.Next() {
			n := &models.FactionNpc{}
			if err := memberRows.Scan(&n.Entry, &n.Name, &n.Subname, &n.LevelMin, &n.LevelMax); err == nil {
				f.Members = append(f.Members, n)
			}
		}
	}

	return f, nil
}
