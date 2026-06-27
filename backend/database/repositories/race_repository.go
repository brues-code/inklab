package repositories

import (
	"database/sql"

	"inklab/backend/database/models"
)

// RaceRepository serves the "Race" database category — playable races and the
// data the character-create screen shows: flavor text, available classes and
// racial traits. Everything here is client-derived (see datatools.genRaces).
type RaceRepository struct {
	db *sql.DB
}

func NewRaceRepository(db *sql.DB) *RaceRepository {
	return &RaceRepository{db: db}
}

// GetRaces returns every race with its racial blurbs, available classes (named
// from class_info) and racial spells (resolved to spell entries with icons).
func (r *RaceRepository) GetRaces() ([]*models.Race, error) {
	rows, err := r.db.Query(`
		SELECT id, name, file_string, prefix, faction, info
		FROM races
		ORDER BY CASE faction WHEN 'Alliance' THEN 0 WHEN 'Horde' THEN 1 ELSE 2 END, name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var races []*models.Race
	for rows.Next() {
		var m models.Race
		if err := rows.Scan(&m.ID, &m.Name, &m.FileString, &m.Prefix, &m.Faction, &m.Info); err != nil {
			continue
		}
		races = append(races, &m)
	}
	for _, m := range races {
		m.Abilities = r.abilities(m.ID)
		m.Classes = r.classes(m.ID)
		m.Racials = r.racials(m.ID)
	}
	return races, nil
}

func (r *RaceRepository) abilities(raceID int) []string {
	rows, err := r.db.Query("SELECT text FROM race_abilities WHERE race_id = ? ORDER BY idx", raceID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var t string
		if rows.Scan(&t) == nil {
			out = append(out, t)
		}
	}
	return out
}

func (r *RaceRepository) classes(raceID int) []models.RaceClass {
	rows, err := r.db.Query(`
		SELECT rc.class_id, COALESCE(ci.name, '')
		FROM race_classes rc
		LEFT JOIN class_info ci ON ci.id = rc.class_id
		WHERE rc.race_id = ?
		ORDER BY ci.name`, raceID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []models.RaceClass
	for rows.Next() {
		var c models.RaceClass
		if rows.Scan(&c.ID, &c.Name) == nil {
			out = append(out, c)
		}
	}
	return out
}

func (r *RaceRepository) racials(raceID int) []models.RacialSpell {
	rows, err := r.db.Query(`
		SELECT sp.entry, sp.name, COALESCE(NULLIF(si.icon_name, ''), sp.iconName, '')
		FROM race_spells rs
		JOIN spell_template sp ON sp.entry = rs.spell_id
		LEFT JOIN spell_icons si ON sp.spellIconId = si.id
		WHERE rs.race_id = ?
		ORDER BY sp.name`, raceID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []models.RacialSpell
	for rows.Next() {
		var s models.RacialSpell
		if rows.Scan(&s.ID, &s.Name, &s.Icon) == nil {
			out = append(out, s)
		}
	}
	return out
}
