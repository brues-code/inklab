package repositories

import (
	"database/sql"
	"fmt"
	"sort"

	"inklab/backend/database/models"
)

// objectTypeNames maps GameObject type ids to display labels. Covers the full
// 1.12 set so every type present in the data can be browsed.
var objectTypeNames = map[int]string{
	0: "Doors", 1: "Buttons", 2: "Quest Givers", 3: "Chests", 4: "Binders",
	5: "Generic / Doodads", 6: "Traps", 7: "Chairs", 8: "Spell Focus",
	9: "Books & Texts", 10: "Interactive", 11: "Elevators & Lifts",
	12: "Area Damage", 13: "Cameras", 14: "Map Objects", 15: "Boats & Zeppelins",
	16: "Duel Flags", 17: "Fishing Nodes", 18: "Summoning Rituals", 19: "Mailboxes",
	20: "Auction Houses", 21: "Guard Posts", 22: "Spell Casters", 23: "Meeting Stones",
	24: "Flag Stands", 25: "Fishing Pools", 26: "Flag Drops", 27: "Mini Games",
	28: "Lottery Kiosks", 29: "Capture Points", 30: "Aura Generators",
	33: "Destructible Buildings",
}

// objectTypePriority lists the player-relevant types first; remaining types
// present in the data are appended afterwards (the bulk doodad buckets like
// Generic/Map Objects land at the end).
var objectTypePriority = []int{
	3, 25, 17, 9, 2, 19, 0, 1, 10, 8, 6, 11, 15, 23, 18, 22, 24, 26, 29,
	4, 21, 16, 13, 30, 27, 28, 12, 7, 14, 5, 20,
}

// GameObjectRepository handles game object-related database operations
type GameObjectRepository struct {
	db *sql.DB
}

// NewGameObjectRepository creates a new game object repository
func NewGameObjectRepository(db *sql.DB) *GameObjectRepository {
	return &GameObjectRepository{db: db}
}

// GetObjectTypes returns derived categories based on Turtlehead logic
func (r *GameObjectRepository) GetObjectTypes() ([]*models.ObjectType, error) {
	types := []*models.ObjectType{}

	// Derived gathering/lock categories: one per LockType (Herbalism, Mining,
	// Survival, Pick Lock, ...) actually used by chest objects (type 3). Names
	// come from lock_types (LockType.dbc), so any server-custom type appears
	// automatically. Category id is -(1000 + lockTypeID) to avoid colliding with
	// real gameobject type ids. Only key-skill lock slots (type 2) count.
	lockRows, err := r.db.Query(`
		SELECT lt.id, lt.name, COUNT(DISTINCT o.entry)
		FROM gameobject_template o
		JOIN locks l ON o.data0 = l.id
		JOIN lock_types lt ON lt.id IN (l.prop1, l.prop2, l.prop3, l.prop4, l.prop5)
		WHERE o.type = 3 AND (
			(l.type1 = 2 AND l.prop1 = lt.id) OR (l.type2 = 2 AND l.prop2 = lt.id) OR
			(l.type3 = 2 AND l.prop3 = lt.id) OR (l.type4 = 2 AND l.prop4 = lt.id) OR
			(l.type5 = 2 AND l.prop5 = lt.id))
		GROUP BY lt.id, lt.name
		ORDER BY COUNT(DISTINCT o.entry) DESC
	`)
	if err == nil {
		for lockRows.Next() {
			var id, count int
			var name string
			if lockRows.Scan(&id, &name, &count) == nil && count > 0 {
				types = append(types, &models.ObjectType{ID: -(1000 + id), Name: name, Count: count})
			}
		}
		lockRows.Close()
	}

	// Count every type actually present, then emit one category per type:
	// priority (player-relevant) order first, the rest appended ascending.
	counts := map[int]int{}
	rows, err := r.db.Query("SELECT type, COUNT(*) FROM gameobject_template GROUP BY type")
	if err != nil {
		return types, err
	}
	for rows.Next() {
		var t, c int
		if err := rows.Scan(&t, &c); err == nil {
			counts[t] = c
		}
	}
	rows.Close()

	seen := map[int]bool{}
	add := func(t int) {
		if seen[t] || counts[t] == 0 {
			return
		}
		seen[t] = true
		name, ok := objectTypeNames[t]
		if !ok {
			name = fmt.Sprintf("Type %d", t)
		}
		types = append(types, &models.ObjectType{ID: t, Name: name, Count: counts[t]})
	}

	for _, t := range objectTypePriority {
		add(t)
	}
	rest := make([]int, 0, len(counts))
	for t := range counts {
		if !seen[t] {
			rest = append(rest, t)
		}
	}
	sort.Ints(rest)
	for _, t := range rest {
		add(t)
	}

	// Present the whole list (derived lock categories + gameobject types) alphabetically.
	sort.Slice(types, func(i, j int) bool { return types[i].Name < types[j].Name })

	return types, nil
}

// GetObjectsByType returns objects filtered by type
func (r *GameObjectRepository) GetObjectsByType(typeID int, nameFilter string) ([]*models.GameObject, error) {
	var query string
	var args []interface{}

	baseSelect := "SELECT entry, name, type, displayId as display_id, size FROM gameobject_template o"

	if typeID < 0 {
		// Derived lock category: id is -(1000 + lockTypeID). Match key-skill slots.
		lockType := -typeID - 1000
		query = baseSelect + `
			JOIN locks l ON o.data0 = l.id
			WHERE o.type = 3 AND (
				(l.type1 = 2 AND l.prop1 = ?) OR (l.type2 = 2 AND l.prop2 = ?) OR
				(l.type3 = 2 AND l.prop3 = ?) OR (l.type4 = 2 AND l.prop4 = ?) OR
				(l.type5 = 2 AND l.prop5 = ?))
		`
		args = append(args, lockType, lockType, lockType, lockType, lockType)
	} else {
		query = baseSelect + " WHERE o.type = ?"
		args = append(args, typeID)
	}

	if nameFilter != "" {
		query += " AND o.name LIKE ?"
		args = append(args, "%"+nameFilter+"%")
	}
	query += " ORDER BY o.name LIMIT 10000"

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var objects []*models.GameObject
	for rows.Next() {
		o := &models.GameObject{}
		if err := rows.Scan(&o.Entry, &o.Name, &o.Type, &o.DisplayID, &o.Size); err != nil {
			continue
		}
		objects = append(objects, o)
	}
	return objects, nil
}

// SearchObjects searches for objects by name
func (r *GameObjectRepository) SearchObjects(query string) ([]*models.GameObject, error) {
	rows, err := r.db.Query(`
		SELECT entry, name, type, displayId as display_id, size FROM gameobject_template
		WHERE name LIKE ? ORDER BY length(name), name LIMIT 50
	`, "%"+query+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var objects []*models.GameObject
	for rows.Next() {
		o := &models.GameObject{}
		if err := rows.Scan(&o.Entry, &o.Name, &o.Type, &o.DisplayID, &o.Size); err != nil {
			continue
		}
		o.TypeName = objectTypeNames[o.Type]
		objects = append(objects, o)
	}
	return objects, nil
}

// GetObjectCount returns total count
func (r *GameObjectRepository) GetObjectCount() (int, error) {
	var count int
	err := r.db.QueryRow("SELECT COUNT(*) FROM gameobject_template").Scan(&count)
	return count, err
}

// GetObjectDetail returns detailed information about a game object
func (r *GameObjectRepository) GetObjectDetail(entry int) (*models.GameObjectDetail, error) {
	obj := &models.GameObjectDetail{}

	err := r.db.QueryRow(`
		SELECT entry, name, type, displayId, faction, flags, size, data0, data1
		FROM gameobject_template WHERE entry = ?
	`, entry).Scan(&obj.Entry, &obj.Name, &obj.Type, &obj.DisplayID, &obj.Faction, &obj.Flags, &obj.Size, &obj.Data0, &obj.Data1)
	if err != nil {
		return nil, err
	}

	// Type name mapping (shared with the category list).
	if name, ok := objectTypeNames[obj.Type]; ok {
		obj.TypeName = name
	}

	// Get quests started by this object
	startsRows, _ := r.db.Query(`
		SELECT q.entry, q.Title, q.QuestLevel
		FROM gameobject_questrelation gq
		JOIN quest_template q ON gq.quest = q.entry
		WHERE gq.id = ?
	`, entry)
	if startsRows != nil {
		defer startsRows.Close()
		for startsRows.Next() {
			qr := &models.QuestRelation{}
			if err := startsRows.Scan(&qr.Entry, &qr.Title, &qr.Level); err == nil {
				qr.Type = "starts"
				obj.StartsQuests = append(obj.StartsQuests, qr)
			}
		}
	}

	// Get quests ended by this object
	endsRows, _ := r.db.Query(`
		SELECT q.entry, q.Title, q.QuestLevel
		FROM gameobject_involvedrelation gi
		JOIN quest_template q ON gi.quest = q.entry
		WHERE gi.id = ?
	`, entry)
	if endsRows != nil {
		defer endsRows.Close()
		for endsRows.Next() {
			qr := &models.QuestRelation{}
			if err := endsRows.Scan(&qr.Entry, &qr.Title, &qr.Level); err == nil {
				qr.Type = "ends"
				obj.EndsQuests = append(obj.EndsQuests, qr)
			}
		}
	}

	// Spawn points (synced from MySQL into gameobject_spawn), in map-percentage
	// coords for plotting on the zone map. The cap is high (not ~50) so every
	// zone the object spawns in is represented — ordering by id then limiting low
	// truncated multi-zone objects to just their first zone or two. The frontend
	// plots one zone at a time, so returning the full set is cheap.
	spawnRows, _ := r.db.Query(`
		SELECT map_id, zone_name, position_x, position_y
		FROM gameobject_spawn
		WHERE gameobject_entry = ?
		ORDER BY id
		LIMIT 2000
	`, entry)
	if spawnRows != nil {
		defer spawnRows.Close()
		for spawnRows.Next() {
			sp := &models.ObjectSpawn{}
			if err := spawnRows.Scan(&sp.MapID, &sp.ZoneName, &sp.X, &sp.Y); err == nil {
				obj.Spawns = append(obj.Spawns, sp)
			}
		}
	}

	// Get loot (if type is Chest - type 3)
	if obj.Type == 3 && obj.Data1 > 0 {
		lootRows, _ := r.db.Query(`
			SELECT gl.item, i.name, i.quality, gl.ChanceOrQuestChance, COALESCE(idi.icon, '')
			FROM gameobject_loot_template gl
			JOIN item_template i ON gl.item = i.entry
			LEFT JOIN item_display_info idi ON i.display_id = idi.ID
			WHERE gl.entry = ?
			ORDER BY gl.ChanceOrQuestChance DESC
			LIMIT 50
		`, obj.Data1)
		if lootRows != nil {
			defer lootRows.Close()
			for lootRows.Next() {
				li := &models.LootItem{}
				if err := lootRows.Scan(&li.ItemID, &li.Name, &li.Quality, &li.Chance, &li.IconPath); err == nil {
					obj.Contains = append(obj.Contains, li)
				}
			}
		}
	}

	return obj, nil
}
