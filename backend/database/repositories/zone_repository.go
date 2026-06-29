package repositories

import (
	"database/sql"
	"strconv"
	"strings"
	"unicode"

	"inklab/backend/database/helpers"
	"inklab/backend/database/models"
)

// ZoneRepository serves the "Zone" database category. A zone is a row in
// quest_categories_enhanced with a positive id (the areatable id); negative ids
// are quest sorts (class/profession), not real zones.
//
// Quests link to a zone cleanly by id (quest_template.ZoneOrSort = qce.id).
// NPCs are messier: creature_spawn.zone_name holds the aowow display name
// ("The Barrens", "Elwynn Forrest", typos and all) while qce.name is the short
// client texture-folder name ("Barrens", "Elwynn"). There is no shared id on
// the spawn rows, so we bridge the two with normalized + longest-prefix name
// matching (see zoneKey / matchZone), which resolves 82/87 distinct spawn zones
// correctly; the rest are continents (excluded) or one-off rarities.
type ZoneRepository struct {
	db *sql.DB
}

// NewZoneRepository creates a new zone repository.
func NewZoneRepository(db *sql.DB) *ZoneRepository {
	return &ZoneRepository{db: db}
}

// zoneAliases patches known aowow spelling errors so their spawns still resolve.
// Keyed and valued by zoneKey output (lowercase alphanumerics).
var zoneAliases = map[string]string{
	"ogrimmar": "orgrimmar", // aowow misspells the Horde capital
}

// zoneKey normalizes a zone name for matching: lowercase, drop parenthetical
// segments ("(Dungeon)"), keep only alphanumerics, strip a leading "the", then
// apply alias fixes.
func zoneKey(s string) string {
	s = strings.ToLower(s)
	for {
		i := strings.IndexAny(s, "([")
		if i < 0 {
			break
		}
		j := strings.IndexAny(s[i:], ")]")
		if j < 0 {
			break
		}
		s = s[:i] + s[i+j+1:]
	}
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	k := strings.TrimPrefix(b.String(), "the")
	if alias, ok := zoneAliases[k]; ok {
		k = alias
	}
	return k
}

// humanizeZoneName turns a client texture-folder name ("BlackrockMountain",
// "EasternPlaguelands") into a readable display name ("Blackrock Mountain",
// "Eastern Plaguelands") by inserting spaces at camelCase / letter-digit
// boundaries. The raw folder name is still used for the map lookup and matching.
func humanizeZoneName(s string) string {
	if s == "" {
		return s
	}
	runes := []rune(s)
	var b strings.Builder
	for i, r := range runes {
		if i > 0 {
			prev := runes[i-1]
			switch {
			case unicode.IsUpper(r) && (unicode.IsLower(prev) || unicode.IsDigit(prev)):
				b.WriteRune(' ')
			case unicode.IsDigit(r) && unicode.IsLetter(prev):
				b.WriteRune(' ')
			}
		}
		b.WriteRune(r)
	}
	return b.String()
}

// zoneInfo is an internal zone row carrying its match key. name is the raw
// texture-folder name (map-image key / matching identifier); displayName is the
// official localized name (AreaTable.dbc) shown to the user.
type zoneInfo struct {
	id          int
	groupID     int
	name        string
	displayName string
	groupName   string
	key         string
}

// loadZoneInfos returns every real zone (id > 0) with its precomputed match key.
func (r *ZoneRepository) loadZoneInfos() ([]zoneInfo, error) {
	rows, err := r.db.Query(`
		SELECT qce.id, qce.name, COALESCE(NULLIF(qce.display_name, ''), ''), qce.group_id, COALESCE(g.name, '')
		FROM quest_categories_enhanced qce
		LEFT JOIN quest_category_groups g ON g.id = qce.group_id
		WHERE qce.id > 0
		ORDER BY qce.group_id, qce.name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var zones []zoneInfo
	for rows.Next() {
		var z zoneInfo
		if err := rows.Scan(&z.id, &z.name, &z.displayName, &z.groupID, &z.groupName); err != nil {
			continue
		}
		// Fall back to the camelCase-split folder name when AreaTable hasn't been
		// imported yet (older DB / pre-regeneration zones.json).
		if z.displayName == "" {
			z.displayName = humanizeZoneName(z.name)
		}
		z.key = zoneKey(z.name)
		zones = append(zones, z)
	}
	return zones, nil
}

// matchZone maps a spawn zone_name to the best zone: an exact key match, else
// the zone whose key is the longest prefix of the spawn key (so "Elwynn Forrest"
// -> "Elwynn" and "Stormwind City" -> "Stormwind", while "Alterac Valley" still
// beats "Alterac"). Returns -1 when nothing matches (e.g. continent fallbacks).
func matchZone(spawnName string, zones []zoneInfo) int {
	sk := zoneKey(spawnName)
	if sk == "" {
		return -1
	}
	bestID, bestLen := -1, -1
	for i := range zones {
		if zones[i].key == sk {
			return zones[i].id
		}
		if zones[i].key != "" && len(zones[i].key) > bestLen && strings.HasPrefix(sk, zones[i].key) {
			bestID, bestLen = zones[i].id, len(zones[i].key)
		}
	}
	return bestID
}

// ZoneNames returns the official display name for every real zone, keyed by its
// normalized match key, so the frontend can resolve a raw spawn/folder zone
// string to the localized name in one place.
func (r *ZoneRepository) ZoneNames() ([]*models.ZoneNameInfo, error) {
	zones, err := r.loadZoneInfos()
	if err != nil {
		return nil, err
	}
	out := make([]*models.ZoneNameInfo, 0, len(zones))
	for _, z := range zones {
		if z.key == "" {
			continue
		}
		out = append(out, &models.ZoneNameInfo{Key: z.key, ID: z.id, Name: z.displayName})
	}
	return out, nil
}

// GetZones returns every real zone that has at least one NPC or quest, with
// counts and its continent/type group name.
func (r *ZoneRepository) GetZones() ([]*models.ZoneListEntry, error) {
	zones, err := r.loadZoneInfos()
	if err != nil {
		return nil, err
	}

	// Quest counts by zone id.
	questCounts := map[int]int{}
	if qr, err := r.db.Query(`SELECT ZoneOrSort, COUNT(*) FROM quest_template WHERE ZoneOrSort > 0 GROUP BY ZoneOrSort`); err == nil {
		for qr.Next() {
			var id, c int
			if qr.Scan(&id, &c) == nil {
				questCounts[id] = c
			}
		}
		qr.Close()
	}

	// NPC counts by zone id, bridging spawn display names to zone ids.
	npcCounts := map[int]int{}
	if nr, err := r.db.Query(`SELECT zone_name, COUNT(DISTINCT creature_entry) FROM creature_spawn GROUP BY zone_name`); err == nil {
		for nr.Next() {
			var name string
			var c int
			if nr.Scan(&name, &c) != nil {
				continue
			}
			if id := matchZone(name, zones); id > 0 {
				npcCounts[id] += c
			}
		}
		nr.Close()
	}

	var out []*models.ZoneListEntry
	for _, z := range zones {
		nc, qc := npcCounts[z.id], questCounts[z.id]
		if nc == 0 && qc == 0 {
			continue
		}
		out = append(out, &models.ZoneListEntry{
			ID:         z.id,
			Name:       z.displayName,
			GroupID:    z.groupID,
			GroupName:  z.groupName,
			NpcCount:   nc,
			QuestCount: qc,
		})
	}
	return out, nil
}

// GetZoneDetail returns the map key, derived level range, NPCs, quests and
// spawn markers for a single zone.
func (r *ZoneRepository) GetZoneDetail(id int) (*models.ZoneDetail, error) {
	zones, err := r.loadZoneInfos()
	if err != nil {
		return nil, err
	}

	d := &models.ZoneDetail{ID: id}
	rawName, displayName := "", ""
	for _, z := range zones {
		if z.id == id {
			rawName, displayName, d.GroupName = z.name, z.displayName, z.groupName
			break
		}
	}
	if rawName == "" {
		// Fall back to a direct lookup so unusual ids still resolve a name.
		if err := r.db.QueryRow(`SELECT name, COALESCE(NULLIF(display_name,''), name) FROM quest_categories_enhanced WHERE id = ?`, id).Scan(&rawName, &displayName); err != nil {
			return nil, err
		}
	}
	// MapName is the raw texture-folder name (what useZoneMap expects); Name is
	// the official localized display form.
	d.MapName = rawName
	d.Name = displayName

	// All spawn display-names that resolve to this zone.
	spawnNames := r.spawnNamesForZone(id, zones)

	if len(spawnNames) > 0 {
		placeholders := strings.TrimSuffix(strings.Repeat("?,", len(spawnNames)), ",")
		args := make([]interface{}, len(spawnNames))
		for i, n := range spawnNames {
			args[i] = n
		}

		// NPCs spawning in this zone (deduped across spawn points / names).
		npcRows, err := r.db.Query(`
			SELECT DISTINCT ct.entry, ct.name, COALESCE(ct.subname, ''),
				ct.level_min, ct.level_max, ct.rank, ct.type, COALESCE(ct.npc_flags, 0)
			FROM creature_spawn cs
			JOIN creature_template ct ON ct.entry = cs.creature_entry
			WHERE cs.zone_name IN (`+placeholders+`)
			ORDER BY ct.level_max, ct.name
		`, args...)
		if err == nil {
			defer npcRows.Close()
			first := true
			for npcRows.Next() {
				n := &models.ZoneNpc{}
				if err := npcRows.Scan(&n.Entry, &n.Name, &n.Subname, &n.LevelMin, &n.LevelMax, &n.Rank, &n.Type, &n.NpcFlags); err != nil {
					continue
				}
				n.RankName = helpers.GetCreatureRankName(n.Rank)
				n.TypeName = helpers.GetCreatureTypeName(n.Type)
				if n.LevelMin > 0 {
					if first || n.LevelMin < d.MinLevel {
						d.MinLevel = n.LevelMin
					}
					first = false
				}
				if n.LevelMax > d.MaxLevel {
					d.MaxLevel = n.LevelMax
				}
				d.Npcs = append(d.Npcs, n)
			}
		}

		// Spawn markers (map-percentage coords). Capped so dense zones stay responsive.
		sRows, err := r.db.Query(`
			SELECT creature_entry, position_x, position_y
			FROM creature_spawn
			WHERE zone_name IN (`+placeholders+`)
				AND position_x > 0 AND position_x <= 100
				AND position_y > 0 AND position_y <= 100
			LIMIT 3000
		`, args...)
		if err == nil {
			defer sRows.Close()
			for sRows.Next() {
				s := &models.ZoneSpawn{}
				if err := sRows.Scan(&s.Entry, &s.X, &s.Y); err != nil {
					continue
				}
				d.Spawns = append(d.Spawns, s)
			}
		}
	}

	// Quests assigned to this zone (clean id linkage).
	qRows, err := r.db.Query(`
		SELECT entry, IFNULL(Title, ''), IFNULL(QuestLevel, 0), IFNULL(MinLevel, 0)
		FROM quest_template
		WHERE ZoneOrSort = ?
		ORDER BY QuestLevel, Title
	`, id)
	if err == nil {
		defer qRows.Close()
		for qRows.Next() {
			q := &models.ZoneQuest{}
			if err := qRows.Scan(&q.Entry, &q.Title, &q.QuestLevel, &q.MinLevel); err != nil {
				continue
			}
			d.Quests = append(d.Quests, q)
		}
	}

	// Game objects spawning in this zone, plus their spawn markers.
	objNames := r.spawnNamesForZoneTable("gameobject_spawn", id, zones)
	if len(objNames) > 0 {
		ph := strings.TrimSuffix(strings.Repeat("?,", len(objNames)), ",")
		args := make([]interface{}, len(objNames))
		for i, n := range objNames {
			args[i] = n
		}

		objRows, err := r.db.Query(`
			SELECT DISTINCT gt.entry, gt.name, gt.type
			FROM gameobject_spawn gs
			JOIN gameobject_template gt ON gt.entry = gs.gameobject_entry
			WHERE gs.zone_name IN (`+ph+`)
			ORDER BY gt.name
		`, args...)
		if err == nil {
			defer objRows.Close()
			for objRows.Next() {
				o := &models.ZoneObject{}
				if err := objRows.Scan(&o.Entry, &o.Name, &o.Type); err != nil {
					continue
				}
				if name, ok := objectTypeNames[o.Type]; ok {
					o.TypeName = name
				}
				d.Objects = append(d.Objects, o)
			}
		}

		osRows, err := r.db.Query(`
			SELECT gameobject_entry, position_x, position_y
			FROM gameobject_spawn
			WHERE zone_name IN (`+ph+`)
				AND position_x > 0 AND position_x <= 100
				AND position_y > 0 AND position_y <= 100
			LIMIT 3000
		`, args...)
		if err == nil {
			defer osRows.Close()
			for osRows.Next() {
				s := &models.ZoneSpawn{}
				if err := osRows.Scan(&s.Entry, &s.X, &s.Y); err != nil {
					continue
				}
				d.ObjectSpawns = append(d.ObjectSpawns, s)
			}
		}
	}

	return d, nil
}

// GetZoneLoot returns every distinct item directly dropped by a creature or game
// object that spawns in the zone, with the number of distinct sources and the
// best drop chance. Reference/world-drop pools are excluded (see below). Loaded
// lazily (its own binding) so the zone page stays responsive. Sorted by quality
// then item level so the best gear surfaces first.
func (r *ZoneRepository) GetZoneLoot(id int) ([]*models.ZoneLoot, error) {
	zones, err := r.loadZoneInfos()
	if err != nil {
		return nil, err
	}
	creatureNames := r.spawnNamesForZone(id, zones)
	objNames := r.spawnNamesForZoneTable("gameobject_spawn", id, zones)

	// Build a UNION ALL of (item, source-key, chance) tuples, only including the
	// creature/object halves that actually have spawns (an empty IN () is invalid
	// SQL). The source key is prefixed by kind so a creature and an object with
	// the same entry id count as two distinct sources.
	var parts []string
	var args []interface{}
	inClause := func(names []string) string {
		for _, n := range names {
			args = append(args, n)
		}
		return strings.TrimSuffix(strings.Repeat("?,", len(names)), ",")
	}
	// Dedupe spawns to distinct creature/object entries *before* joining loot — a
	// creature has many spawn points, so joining spawns straight to its loot table
	// multiplies rows by spawn count.
	//
	// Loot is the item's direct rows (mincountOrRef >= 0) PLUS the items in any
	// *boss-specific* reference table it points to. Negative rows are pointers
	// into reference_loot_template; those tables come in two flavors: shared
	// world-drop pools (Linen Cloth, world epics) referenced by hundreds of
	// creatures, and per-boss loot pools referenced by only one or two. We expand
	// only the latter (referenced by <= maxSharedRefSources loot tables) — that
	// surfaces a boss's epics (e.g. Solnius) without dragging in the global pools,
	// which are both slow to expand (~40x the rows) and pure noise in a zone list.
	const maxSharedRefSources = 10
	// creatureDist/objDist build the "distinct entries joined to their loot table"
	// FROM clause. Called fresh per part (not shared) so each placeholder set gets
	// its own args appended in the order they appear in the final SQL.
	creatureDist := func() string {
		return `(SELECT DISTINCT creature_entry AS e FROM creature_spawn WHERE zone_name IN (` + inClause(creatureNames) + `)) ze
			JOIN creature_template c ON c.entry = ze.e
			JOIN creature_loot_template cl ON cl.entry = c.loot_id`
	}
	objDist := func() string {
		return `(SELECT DISTINCT gameobject_entry AS e FROM gameobject_spawn WHERE zone_name IN (` + inClause(objNames) + `)) ze
			JOIN gameobject_template gt ON gt.entry = ze.e
			JOIN gameobject_loot_template gl ON gl.entry = gt.data1`
	}
	if len(creatureNames) > 0 {
		parts = append(parts, `
			SELECT cl.item AS item, 'npc' AS kind, ze.e AS sentry, c.name AS sname, cl.ChanceOrQuestChance AS chance
			FROM `+creatureDist()+` AND cl.mincountOrRef >= 0`)
		parts = append(parts, `
			SELECT rl.item AS item, 'npc' AS kind, ze.e AS sentry, c.name AS sname,
			       cl.ChanceOrQuestChance * COALESCE(NULLIF(rl.ChanceOrQuestChance, 0), 100.0 / NULLIF(g.cnt, 0)) / 100.0 AS chance
			FROM `+creatureDist()+` AND cl.mincountOrRef < 0
			JOIN narrow_refs nr ON nr.mref = cl.mincountOrRef
			JOIN reference_loot_template rl ON rl.entry = -cl.mincountOrRef
			LEFT JOIN ref_group_counts g ON g.entry = rl.entry AND g.groupid = rl.groupid`)
	}
	if len(objNames) > 0 {
		parts = append(parts, `
			SELECT gl.item AS item, 'object' AS kind, ze.e AS sentry, gt.name AS sname, gl.ChanceOrQuestChance AS chance
			FROM `+objDist()+` AND gl.mincountOrRef >= 0`)
		parts = append(parts, `
			SELECT rl.item AS item, 'object' AS kind, ze.e AS sentry, gt.name AS sname,
			       gl.ChanceOrQuestChance * COALESCE(NULLIF(rl.ChanceOrQuestChance, 0), 100.0 / NULLIF(g.cnt, 0)) / 100.0 AS chance
			FROM `+objDist()+` AND gl.mincountOrRef < 0
			JOIN narrow_refs nr ON nr.mref = gl.mincountOrRef
			JOIN reference_loot_template rl ON rl.entry = -gl.mincountOrRef
			LEFT JOIN ref_group_counts g ON g.entry = rl.entry AND g.groupid = rl.groupid`)
	}
	if len(parts) == 0 {
		return nil, nil
	}

	// narrow_refs: reference-table pointers (negative mincountOrRef values) used by
	// at most maxSharedRefSources distinct loot tables — i.e. per-boss pools, not
	// shared world-drop pools. Computed once and joined by the reference parts
	// above. Its threshold placeholder is the first arg (it leads the SQL).
	q := `
		WITH narrow_refs AS (
			SELECT mref FROM (
				SELECT DISTINCT entry, mincountOrRef AS mref FROM creature_loot_template WHERE mincountOrRef < 0
				UNION
				SELECT DISTINCT entry, mincountOrRef AS mref FROM gameobject_loot_template WHERE mincountOrRef < 0
			) GROUP BY mref HAVING COUNT(*) <= ?
		),
		-- Within a reference table, items sharing a group with chance 0 split the
		-- group's roll equally (mangos loot semantics), so each item's share is
		-- 100/cnt %. Used to turn a stored chance of 0 into a real drop %.
		ref_group_counts AS (
			SELECT entry, groupid, COUNT(*) AS cnt
			FROM reference_loot_template
			WHERE ChanceOrQuestChance = 0
			GROUP BY entry, groupid
		)
		SELECT it.entry, it.name, it.quality, COALESCE(idi.icon, ''), it.item_level,
		       t.kind, t.sentry, t.sname, t.chance
		FROM (` + strings.Join(parts, "\n\t\t\tUNION ALL\n") + `) t
		JOIN item_template it ON it.entry = t.item
		LEFT JOIN item_display_info idi ON it.display_id = idi.ID
		ORDER BY it.quality DESC, it.item_level DESC, it.name, it.entry`

	rows, err := r.db.Query(q, append([]interface{}{maxSharedRefSources}, args...)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Rows arrive ordered and contiguous per item (one row per source). Fold them
	// into one ZoneLoot per item: collect distinct sources (capped) and track the
	// best chance. The ORDER BY also fixes the display order, so first-seen order
	// is the final order.
	const (
		maxItems         = 2000
		maxSourcesPerRow = 25
	)
	byItem := map[int]*models.ZoneLoot{}
	srcSeen := map[string]bool{}
	var out []*models.ZoneLoot
	for rows.Next() {
		var itemEntry, quality, ilvl, sentry int
		var name, icon, kind, sname string
		var chance sql.NullFloat64
		if err := rows.Scan(&itemEntry, &name, &quality, &icon, &ilvl, &kind, &sentry, &sname, &chance); err != nil {
			continue
		}
		l := byItem[itemEntry]
		if l == nil {
			if len(out) >= maxItems {
				continue
			}
			l = &models.ZoneLoot{Entry: itemEntry, Name: name, Quality: quality, IconPath: icon, ItemLevel: ilvl}
			byItem[itemEntry] = l
			out = append(out, l)
		}
		if chance.Valid && chance.Float64 > l.Chance {
			l.Chance = chance.Float64
		}
		key := strconv.Itoa(itemEntry) + kind + strconv.Itoa(sentry)
		if !srcSeen[key] && len(l.Sources) < maxSourcesPerRow {
			srcSeen[key] = true
			l.Sources = append(l.Sources, &models.ZoneLootSource{Entry: sentry, Name: sname, Kind: kind})
		}
	}
	return out, nil
}

// spawnNamesForZone returns the distinct creature_spawn.zone_name values that
// resolve to the given zone id.
func (r *ZoneRepository) spawnNamesForZone(id int, zones []zoneInfo) []string {
	return r.spawnNamesForZoneTable("creature_spawn", id, zones)
}

// spawnNamesForZoneTable returns the distinct zone_name values in the given
// spawn table that resolve to the given zone id. Both creature_spawn and
// gameobject_spawn store zone names in mixed forms (aowow display names from the
// MySQL sync, clean folder names from the web sync), so we match via matchZone.
func (r *ZoneRepository) spawnNamesForZoneTable(table string, id int, zones []zoneInfo) []string {
	rows, err := r.db.Query(`SELECT DISTINCT zone_name FROM ` + table)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var names []string
	for rows.Next() {
		var name string
		if rows.Scan(&name) != nil || name == "" {
			continue
		}
		if matchZone(name, zones) == id {
			names = append(names, name)
		}
	}
	return names
}
