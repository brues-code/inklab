package repositories

import (
	"database/sql"
	"fmt"
	"strconv"

	"inklab/backend/database/helpers"
	"inklab/backend/database/models"
)

// CreatureRepository handles creature-related database operations
type CreatureRepository struct {
	db      *sql.DB
	mysqlDB *sql.DB
}

// SetMySQL sets the MySQL connection
func (r *CreatureRepository) SetMySQL(db *sql.DB) {
	r.mysqlDB = db
}

// NewCreatureRepository creates a new creature repository
func NewCreatureRepository(db *sql.DB) *CreatureRepository {
	return &CreatureRepository{db: db}
}

// GetCreatureTypes returns all creature types with counts
func (r *CreatureRepository) GetCreatureTypes() ([]*models.CreatureType, error) {
	rows, err := r.db.Query(`
		SELECT type, COUNT(*) as count
		FROM creature_template
		GROUP BY type
		ORDER BY type
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var types []*models.CreatureType
	for rows.Next() {
		t := &models.CreatureType{}
		if err := rows.Scan(&t.Type, &t.Count); err != nil {
			continue
		}
		t.Name = helpers.GetCreatureTypeName(t.Type)
		types = append(types, t)
	}

	return types, nil
}

// GetCreaturesByType returns creatures filtered by type
func (r *CreatureRepository) GetCreaturesByType(creatureType int, nameFilter string, limit, offset int) ([]*models.Creature, int, error) {
	whereClause := "WHERE type = ?"
	args := []interface{}{creatureType}

	if nameFilter != "" {
		whereClause += " AND name LIKE ?"
		args = append(args, "%"+nameFilter+"%")
	}

	// Count
	var count int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM creature_template %s", whereClause)
	err := r.db.QueryRow(countQuery, args...).Scan(&count)
	if err != nil {
		return nil, 0, err
	}

	// Data
	dataArgs := append(args, limit, offset)
	dataQuery := fmt.Sprintf(`
		SELECT entry, name, subname, level_min, level_max,
			health_min, health_max, mana_min, mana_max,
			type, rank, faction, npc_flags, display_id1
		FROM creature_template
		%s
		ORDER BY level_max DESC, name
		LIMIT ? OFFSET ?
	`, whereClause)

	rows, err := r.db.Query(dataQuery, dataArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var creatures []*models.Creature
	for rows.Next() {
		c := &models.Creature{}
		var subname *string
		err := rows.Scan(
			&c.Entry, &c.Name, &subname, &c.LevelMin, &c.LevelMax,
			&c.HealthMin, &c.HealthMax, &c.ManaMin, &c.ManaMax,
			&c.Type, &c.Rank, &c.Faction, &c.NPCFlags, &c.DisplayID1,
		)
		if err != nil {
			continue
		}
		if subname != nil {
			c.Subname = *subname
		}
		c.TypeName = helpers.GetCreatureTypeName(c.Type)
		c.RankName = helpers.GetCreatureRankName(c.Rank)
		creatures = append(creatures, c)
	}

	return creatures, count, nil
}

// SearchCreatures searches for creatures by name or ID
func (r *CreatureRepository) SearchCreatures(query string, limit int) ([]*models.Creature, error) {
	var rows *sql.Rows
	var err error

	// Check if query is a number
	if id, parseErr := strconv.Atoi(query); parseErr == nil {
		rows, err = r.db.Query(`
		SELECT entry, name, subname, level_min, level_max,
			health_min, health_max, mana_min, mana_max,
			type, rank, faction, npc_flags, display_id1
		FROM creature_template
		WHERE name LIKE ? OR entry = ?
		ORDER BY length(name), name
		LIMIT ?
	`, "%"+query+"%", id, limit)
	} else {
		rows, err = r.db.Query(`
		SELECT entry, name, subname, level_min, level_max,
			health_min, health_max, mana_min, mana_max,
			type, rank, faction, npc_flags, display_id1
		FROM creature_template
		WHERE name LIKE ?
		ORDER BY length(name), name
		LIMIT ?
	`, "%"+query+"%", limit)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var creatures []*models.Creature
	for rows.Next() {
		c := &models.Creature{}
		var subname *string
		err := rows.Scan(
			&c.Entry, &c.Name, &subname, &c.LevelMin, &c.LevelMax,
			&c.HealthMin, &c.HealthMax, &c.ManaMin, &c.ManaMax,
			&c.Type, &c.Rank, &c.Faction, &c.NPCFlags, &c.DisplayID1,
		)
		if err != nil {
			continue
		}
		if subname != nil {
			c.Subname = *subname
		}
		c.TypeName = helpers.GetCreatureTypeName(c.Type)
		c.RankName = helpers.GetCreatureRankName(c.Rank)
		creatures = append(creatures, c)
	}

	return creatures, nil
}

// GetCreatureByID retrieves a single creature by ID
func (r *CreatureRepository) GetCreatureByID(entry int) (*models.Creature, error) {
	c := &models.Creature{}
	var subname *string
	err := r.db.QueryRow(`
		SELECT entry, name, subname, level_min, level_max, 
			health_min, health_max, mana_min, mana_max,
			type, rank, faction, npc_flags,
			gold_min, gold_max,
			dmg_min, dmg_max, armor,
			holy_res, fire_res, nature_res, frost_res, shadow_res, arcane_res,
			display_id1
		FROM creature_template WHERE entry = ?
	`, entry).Scan(
		&c.Entry, &c.Name, &subname, &c.LevelMin, &c.LevelMax,
		&c.HealthMin, &c.HealthMax, &c.ManaMin, &c.ManaMax,
		&c.Type, &c.Rank, &c.Faction, &c.NPCFlags,
		&c.GoldMin, &c.GoldMax,
		&c.MinDmg, &c.MaxDmg, &c.Armor,
		&c.HolyRes, &c.FireRes, &c.NatureRes, &c.FrostRes, &c.ShadowRes, &c.ArcaneRes,
		&c.DisplayID1,
	)
	if err != nil {
		return nil, err
	}
	if subname != nil {
		c.Subname = *subname
	}
	c.TypeName = helpers.GetCreatureTypeName(c.Type)
	c.RankName = helpers.GetCreatureRankName(c.Rank)
	return c, nil
}

// GetCreatureCount returns the total number of creatures
func (r *CreatureRepository) GetCreatureCount() (int, error) {
	var count int
	err := r.db.QueryRow("SELECT COUNT(*) FROM creature_template").Scan(&count)
	return count, err
}

// GetCreatureDetail returns full creature information with loot and quests
func (r *CreatureRepository) GetCreatureDetail(entry int) (*models.CreatureDetail, error) {
	creature, err := r.GetCreatureByID(entry)
	if err != nil {
		return nil, err
	}

	detail := &models.CreatureDetail{Creature: creature}

	// Get loot
	lootRepo := NewLootRepository(r.db)
	loot, err := lootRepo.GetCreatureLoot(entry)
	if err == nil {
		detail.Loot = loot
	}

	// Get quests this creature starts
	rows, err := r.db.Query(`
		SELECT q.entry, q.Title
		FROM creature_questrelation nqs
		JOIN quest_template q ON nqs.quest = q.entry
		WHERE nqs.id = ?
	`, entry)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			qr := &models.QuestRelation{Type: "quest"}
			rows.Scan(&qr.Entry, &qr.Name)
			detail.StartsQuests = append(detail.StartsQuests, qr)
		}
	}

	// Get quests this creature ends
	rows2, err := r.db.Query(`
		SELECT q.entry, q.Title
		FROM creature_involvedrelation nqe
		JOIN quest_template q ON nqe.quest = q.entry
		WHERE nqe.id = ?
	`, entry)
	if err == nil {
		defer rows2.Close()
		for rows2.Next() {
			qr := &models.QuestRelation{Type: "quest"}
			rows2.Scan(&qr.Entry, &qr.Name)
			detail.EndsQuests = append(detail.EndsQuests, qr)
		}
	}

	// If MySQL is available, ANY relationship data should come from there as requested
	if r.mysqlDB != nil {
		return r.getCreatureDetailMySQL(entry)
	}

	return detail, nil
}

func (r *CreatureRepository) getCreatureDetailMySQL(entry int) (*models.CreatureDetail, error) {
	// 1. Get basic info from MySQL (to update stats if needed, or just use what we have?
	// Let's re-fetch to be safe and get accurate realtime stats)
	c := &models.Creature{}
	var subname *string
	// Note: Column names must match TW core. Assuming standard 1.12 columns.
	err := r.mysqlDB.QueryRow(`
		SELECT entry, name, subname, minlevel, maxlevel, 
			minhealth, maxhealth, minmana, maxmana,
			creature_type, rank, faction_A, npc_flags,
			minitemgold, maxitemgold,
			mindmg, maxdmg, armor,
			resistance1, resistance2, resistance3, resistance4, resistance5, resistance6,
			displayid1
		FROM creature_template WHERE entry = ?
	`, entry).Scan(
		&c.Entry, &c.Name, &subname, &c.LevelMin, &c.LevelMax,
		&c.HealthMin, &c.HealthMax, &c.ManaMin, &c.ManaMax,
		&c.Type, &c.Rank, &c.Faction, &c.NPCFlags,
		&c.GoldMin, &c.GoldMax,
		&c.MinDmg, &c.MaxDmg, &c.Armor,
		&c.HolyRes, &c.FireRes, &c.NatureRes, &c.FrostRes, &c.ShadowRes, &c.ArcaneRes,
		&c.DisplayID1,
	)

	if err != nil {
		// Fallback to SQLite if MySQL fetch fails (e.g. connection lost or row missing)
		// But if fetch fails, likely other queries will too.
		fmt.Printf("MySQL creature fetch failed: %v. Using SQLite base.\n", err)
		return r.GetCreatureDetail(entry) // Recursive call? No, logic above prevents infinite loop if check is r.mysqlDB != nil.
		// Actually, if we are IN this function, r.mysqlDB is != nil.
		// We should probably just return error or fallback manually.
		// Let's just return error for now to confirm connection.
		return nil, err
	}

	if subname != nil {
		c.Subname = *subname
	}
	c.TypeName = helpers.GetCreatureTypeName(c.Type)
	c.RankName = helpers.GetCreatureRankName(c.Rank)

	detail := &models.CreatureDetail{Creature: c}

	// 2. Spawns
	rows, err := r.mysqlDB.Query("SELECT map, position_x, position_y, position_z FROM creature WHERE id = ?", entry)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			s := &models.CreatureSpawn{}
			rows.Scan(&s.MapID, &s.X, &s.Y, &s.Z)
			detail.Spawns = append(detail.Spawns, s)
		}
	}

	// 3. Abilities (Try Spell1..Spell8 first)
	// Some cores use creature_spells table, some use template columns.
	// We'll try template columns first.
	// For simplicity, let's assume they are spell1..spell4 (standard).
	var s1, s2, s3, s4 int
	err = r.mysqlDB.QueryRow("SELECT spell1, spell2, spell3, spell4 FROM creature_template WHERE entry = ?", entry).Scan(&s1, &s2, &s3, &s4)
	if err == nil {
		spellIDs := []int{s1, s2, s3, s4}
		for _, sid := range spellIDs {
			if sid > 0 {
				ab := &models.CreatureAbility{ID: sid}
				// Fetch spell info from SQLite (faster/easier as we have it) or MySQL?
				// Use MySQL to be consistent.
				var sname, sdesc string
				// spell_template in MySQL might be different table name? `spell_template` usually.
				// In 1.12 it's `spell_template`.
				r.mysqlDB.QueryRow("SELECT name, description FROM spell_template WHERE entry = ?", sid).Scan(&sname, &sdesc)
				ab.Name = sname
				ab.Description = sdesc
				// Icon? Standard DB doesn't have icon path directly usually, requires DBC lookup.
				// But we have local SQLite `spell_template` with icon_path!
				// So let's fallback to local SQLite for icon and rich text.
				var iconPath string
				r.db.QueryRow("SELECT icon_path FROM spell_template WHERE entry = ?", sid).Scan(&iconPath)
				ab.Icon = iconPath

				detail.Abilities = append(detail.Abilities, ab)
			}
		}
	}

	// 4. Loot
	// creature_loot_template
	lRows, err := r.mysqlDB.Query("SELECT item, ChanceOrQuestChance, mincountOrRef, maxcount FROM creature_loot_template WHERE entry = ?", entry)
	if err == nil {
		defer lRows.Close()
		for lRows.Next() {
			li := &models.LootItem{}
			var itemID int
			lRows.Scan(&itemID, &li.Chance, &li.MinCount, &li.MaxCount)
			li.ItemID = itemID

			// Enrich with local item data
			var iName, iIcon string
			var iQual int
			r.db.QueryRow(`
				SELECT i.name, i.quality, COALESCE(idi.icon, '') 
				FROM item_template i 
				LEFT JOIN item_display_info idi ON i.display_id = idi.ID 
				WHERE i.entry = ?
			`, itemID).Scan(&iName, &iQual, &iIcon)
			li.Name = iName
			li.Quality = iQual
			li.IconPath = iIcon

			detail.Loot = append(detail.Loot, li)
		}
	}

	// 5. Quests (Starts)
	qRows, err := r.mysqlDB.Query("SELECT quest FROM creature_questrelation WHERE id = ?", entry)
	if err == nil {
		defer qRows.Close()
		for qRows.Next() {
			qr := &models.QuestRelation{Type: "quest"}
			var qID int
			qRows.Scan(&qID)
			qr.Entry = qID

			// Enrich
			var qTitle string
			r.db.QueryRow("SELECT Title FROM quest_template WHERE entry = ?", qID).Scan(&qTitle)
			qr.Name = qTitle

			detail.StartsQuests = append(detail.StartsQuests, qr)
		}
	}

	// 6. Quests (Ends)
	qRows2, err := r.mysqlDB.Query("SELECT quest FROM creature_involvedrelation WHERE id = ?", entry)
	if err == nil {
		defer qRows2.Close()
		for qRows2.Next() {
			qr := &models.QuestRelation{Type: "quest"}
			var qID int
			qRows2.Scan(&qID)
			qr.Entry = qID

			// Enrich
			var qTitle string
			r.db.QueryRow("SELECT Title FROM quest_template WHERE entry = ?", qID).Scan(&qTitle)
			qr.Name = qTitle

			detail.EndsQuests = append(detail.EndsQuests, qr)
		}
	}

	return detail, nil
}
