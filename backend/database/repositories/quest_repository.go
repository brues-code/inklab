package repositories

import (
	"database/sql"
	"fmt"
	"regexp"
	"strings"

	"inklab/backend/database/models"
)

// QuestRepository handles quest-related database operations
type QuestRepository struct {
	db *sql.DB
}

// NewQuestRepository creates a new quest repository
func NewQuestRepository(db *sql.DB) *QuestRepository {
	return &QuestRepository{db: db}
}

// GetQuestCategories returns all quest categories (zones and sorts) with quest counts
func (r *QuestRepository) GetQuestCategories() ([]*models.QuestCategory, error) {
	rows, err := r.db.Query(`
		SELECT id, name FROM quest_categories ORDER BY name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	categories := make(map[int]*models.QuestCategory)
	var catList []*models.QuestCategory

	for rows.Next() {
		cat := &models.QuestCategory{}
		if err := rows.Scan(&cat.ID, &cat.Name); err != nil {
			continue
		}
		categories[cat.ID] = cat
		catList = append(catList, cat)
	}

	// Now count quests per category
	rows2, err := r.db.Query(`
		SELECT ZoneOrSort, COUNT(*) 
		FROM quest_template 
		GROUP BY ZoneOrSort
	`)
	if err != nil {
		return catList, nil
	}
	defer rows2.Close()

	for rows2.Next() {
		var zoneID, count int
		if err := rows2.Scan(&zoneID, &count); err != nil {
			continue
		}
		if cat, exists := categories[zoneID]; exists {
			cat.Count = count
		}
	}

	// Filter out categories with 0 quests
	var activeCats []*models.QuestCategory
	for _, cat := range catList {
		if cat.Count > 0 {
			activeCats = append(activeCats, cat)
		}
	}

	return activeCats, nil
}

// GetQuestsByCategory returns quests filtered by category (zone or sort)
func (r *QuestRepository) GetQuestsByCategory(categoryID int) ([]*models.Quest, error) {
	rows, err := r.db.Query(`
		SELECT entry, IFNULL(Title,''), IFNULL(QuestLevel,0), IFNULL(MinLevel,0), 
			IFNULL(Type,0), IFNULL(ZoneOrSort,0),
			IFNULL(RewXP,0), IFNULL(RewOrReqMoney,0),
			IFNULL(RequiredRaces,0), IFNULL(RequiredClasses,0), IFNULL(SrcItemId,0),
			IFNULL(PrevQuestId,0), IFNULL(NextQuestId,0), IFNULL(ExclusiveGroup,0), IFNULL(NextQuestInChain,0)
		FROM quest_template
		WHERE ZoneOrSort = ?
		ORDER BY QuestLevel, Title
	`, categoryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var quests []*models.Quest
	for rows.Next() {
		q := &models.Quest{}
		err := rows.Scan(
			&q.Entry, &q.Title, &q.QuestLevel, &q.MinLevel,
			&q.Type, &q.ZoneOrSort,
			&q.RewardXP, &q.RewardMoney,
			&q.RequiredRaces, &q.RequiredClasses, &q.SrcItem,
			&q.PrevQuestID, &q.NextQuestID, &q.ExclusiveGroup, &q.NextQuestInChain,
		)
		if err != nil {
			fmt.Printf("Error scanning quest list: %v\n", err)
			continue
		}
		quests = append(quests, q)
	}
	return quests, nil
}

// GetQuestByID retrieves a single quest by ID
func (r *QuestRepository) GetQuestByID(id int) (*models.Quest, error) {
	row := r.db.QueryRow(`
		SELECT q.entry, IFNULL(q.Title,''), IFNULL(q.QuestLevel,0), IFNULL(q.MinLevel,0), 
			IFNULL(q.Type,0), IFNULL(q.ZoneOrSort,0),
			IFNULL(q.RewXP,0), IFNULL(q.RewOrReqMoney,0),
			IFNULL(q.RequiredRaces,0), IFNULL(q.RequiredClasses,0), IFNULL(q.SrcItemId,0),
			IFNULL(q.PrevQuestId,0), IFNULL(q.NextQuestId,0), IFNULL(q.ExclusiveGroup,0), IFNULL(q.NextQuestInChain,0),
			c.name
		FROM quest_template q
		LEFT JOIN quest_categories c ON q.ZoneOrSort = c.id
		WHERE q.entry = ?
	`, id)

	q := &models.Quest{}
	var catName *string
	err := row.Scan(
		&q.Entry, &q.Title, &q.QuestLevel, &q.MinLevel,
		&q.Type, &q.ZoneOrSort,
		&q.RewardXP, &q.RewardMoney,
		&q.RequiredRaces, &q.RequiredClasses, &q.SrcItem,
		&q.PrevQuestID, &q.NextQuestID, &q.ExclusiveGroup, &q.NextQuestInChain,
		&catName,
	)
	if err != nil {
		return nil, err
	}
	if catName != nil {
		q.CategoryName = *catName
	}
	return q, nil
}

// SearchQuests searches for quests by title
func (r *QuestRepository) SearchQuests(query string) ([]*models.Quest, error) {
	rows, err := r.db.Query(`
		SELECT q.entry, IFNULL(q.Title,''), IFNULL(q.QuestLevel,0), IFNULL(q.MinLevel,0), 
			IFNULL(q.Type,0), IFNULL(q.ZoneOrSort,0),
			IFNULL(q.RewXP,0), IFNULL(q.RewOrReqMoney,0),
			IFNULL(q.RequiredRaces,0), IFNULL(q.RequiredClasses,0), IFNULL(q.SrcItemId,0),
			IFNULL(q.PrevQuestId,0), IFNULL(q.NextQuestId,0), IFNULL(q.ExclusiveGroup,0), IFNULL(q.NextQuestInChain,0),
			c.name
		FROM quest_template q
		LEFT JOIN quest_categories c ON q.ZoneOrSort = c.id
		WHERE q.Title LIKE ?
		ORDER BY length(q.Title), q.Title
		LIMIT 50
	`, "%"+query+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var quests []*models.Quest
	for rows.Next() {
		q := &models.Quest{}
		var catName *string
		err := rows.Scan(
			&q.Entry, &q.Title, &q.QuestLevel, &q.MinLevel,
			&q.Type, &q.ZoneOrSort,
			&q.RewardXP, &q.RewardMoney,
			&q.RequiredRaces, &q.RequiredClasses, &q.SrcItem,
			&q.PrevQuestID, &q.NextQuestID, &q.ExclusiveGroup, &q.NextQuestInChain,
			&catName,
		)
		if err != nil {
			continue
		}
		if catName != nil {
			q.CategoryName = *catName
		}
		quests = append(quests, q)
	}
	return quests, nil
}

// GetQuestCategoryGroups returns all quest category groups
func (r *QuestRepository) GetQuestCategoryGroups() ([]*models.QuestCategoryGroup, error) {
	rows, err := r.db.Query(`
		SELECT id, name FROM quest_category_groups ORDER BY id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var groups []*models.QuestCategoryGroup
	for rows.Next() {
		g := &models.QuestCategoryGroup{}
		if err := rows.Scan(&g.ID, &g.Name); err != nil {
			continue
		}
		groups = append(groups, g)
	}
	return groups, nil
}

// GetQuestCategoriesByGroup returns all categories in a group with quest counts
func (r *QuestRepository) GetQuestCategoriesByGroup(groupID int) ([]*models.QuestCategoryEnhanced, error) {
	rows, err := r.db.Query(`
		SELECT qce.id, qce.group_id, qce.name,
			COALESCE((SELECT COUNT(*) FROM quest_template WHERE ZoneOrSort = qce.id), 0) as quest_count
		FROM quest_categories_enhanced qce
		WHERE qce.group_id = ?
			AND (SELECT COUNT(*) FROM quest_template WHERE ZoneOrSort = qce.id) > 0
		ORDER BY quest_count DESC, qce.name
	`, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []*models.QuestCategoryEnhanced
	for rows.Next() {
		c := &models.QuestCategoryEnhanced{}
		if err := rows.Scan(&c.ID, &c.GroupID, &c.Name, &c.QuestCount); err != nil {
			continue
		}
		categories = append(categories, c)
	}
	return categories, nil
}

// GetQuestsByEnhancedCategory returns quests for a given category (ZoneOrSort value)
func (r *QuestRepository) GetQuestsByEnhancedCategory(categoryID int, nameFilter string) ([]*models.Quest, error) {
	whereClause := "WHERE ZoneOrSort = ?"
	args := []interface{}{categoryID}

	if nameFilter != "" {
		whereClause += " AND title LIKE ?"
		args = append(args, "%"+nameFilter+"%")
	}

	query := fmt.Sprintf(`
		SELECT entry, Title, QuestLevel, MinLevel, Type, ZoneOrSort, RewXP
		FROM quest_template 
		%s
		ORDER BY QuestLevel, Title
		LIMIT 10000
	`, whereClause)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var quests []*models.Quest
	for rows.Next() {
		q := &models.Quest{}
		if err := rows.Scan(&q.Entry, &q.Title, &q.QuestLevel, &q.MinLevel, &q.Type, &q.ZoneOrSort, &q.RewardXP); err != nil {
			continue
		}
		quests = append(quests, q)
	}
	return quests, nil
}

// GetQuestCount returns the total number of quests
func (r *QuestRepository) GetQuestCount() (int, error) {
	var count int
	err := r.db.QueryRow("SELECT COUNT(*) FROM quest_template").Scan(&count)
	return count, err
}

// GetQuestDetail returns full quest information
// resolveQuestRewardSpell determines the spell a quest teaches on completion.
// RewSpell is the dedicated learned-spell reward (always shown). RewSpellCast is
// the broad "cast on completion" field — buffs, summons and teleports also live
// there — so it only counts when it's an actual learn-spell (effect 36). A
// learn-spell is followed through to the ability it teaches. Returns nil when
// there's nothing learnable to show.
func (r *QuestRepository) resolveQuestRewardSpell(rewSpell, rewSpellCast int) *models.QuestRewardSpell {
	fromCast := false
	spellID := rewSpell
	if spellID == 0 {
		spellID, fromCast = rewSpellCast, true
	}
	if spellID <= 0 {
		return nil
	}

	var eff, trig [3]int
	err := r.db.QueryRow(`
		SELECT effect1, effect2, effect3, effectTriggerSpell1, effectTriggerSpell2, effectTriggerSpell3
		FROM spell_template WHERE entry = ?
	`, spellID).Scan(&eff[0], &eff[1], &eff[2], &trig[0], &trig[1], &trig[2])
	isLearn := false
	if err == nil {
		for i := 0; i < 3; i++ {
			if eff[i] == 36 && trig[i] > 0 { // SPELL_EFFECT_LEARN_SPELL
				spellID, isLearn = trig[i], true
				break
			}
		}
	}
	// A RewSpellCast that isn't a learn-spell is a completion effect, not a
	// learned reward — don't surface it.
	if fromCast && !isLearn {
		return nil
	}

	out := &models.QuestRewardSpell{SpellID: spellID}
	r.db.QueryRow(`
		SELECT COALESCE(st.name, ''), COALESCE(NULLIF(si.icon_name, ''), st.iconName, '')
		FROM spell_template st
		LEFT JOIN spell_icons si ON st.spellIconId = si.id
		WHERE st.entry = ?
	`, spellID).Scan(&out.Name, &out.IconName)
	return out
}

func (r *QuestRepository) GetQuestDetail(entry int) (*models.QuestDetail, error) {
	row := r.db.QueryRow(`
		SELECT entry, Title, Details, Objectives, OfferRewardText, EndText,
			QuestLevel, MinLevel, Type, ZoneOrSort,
			RequiredRaces, RequiredClasses,
			RewXP, RewOrReqMoney, RewSpell, RewSpellCast,
			RewItemId1, RewItemId2, RewItemId3, RewItemId4,
			RewItemCount1, RewItemCount2, RewItemCount3, RewItemCount4,
			RewChoiceItemId1, RewChoiceItemId2, RewChoiceItemId3, RewChoiceItemId4, RewChoiceItemId5, RewChoiceItemId6,
			RewChoiceItemCount1, RewChoiceItemCount2, RewChoiceItemCount3, RewChoiceItemCount4, RewChoiceItemCount5, RewChoiceItemCount6,
			RewRepFaction1, RewRepFaction2, RewRepFaction3, RewRepFaction4, RewRepFaction5,
			RewRepValue1, RewRepValue2, RewRepValue3, RewRepValue4, RewRepValue5,
			PrevQuestId, NextQuestId, ExclusiveGroup, NextQuestInChain,
			ReqItemId1, ReqItemId2, ReqItemId3, ReqItemId4,
			ReqItemCount1, ReqItemCount2, ReqItemCount3, ReqItemCount4,
			ReqCreatureOrGOId1, ReqCreatureOrGOId2, ReqCreatureOrGOId3, ReqCreatureOrGOId4,
			ReqCreatureOrGOCount1, ReqCreatureOrGOCount2, ReqCreatureOrGOCount3, ReqCreatureOrGOCount4
		FROM quest_template WHERE entry = ?
	`, entry)

	q := &models.QuestDetail{}
	var details, objectives, offerReward, endText *string
	var rewItems [4]int
	var rewItemCounts [4]int
	var rewChoiceItems [6]int
	var rewChoiceItemCounts [6]int
	var repFactions [5]int
	var repValues [5]int
	var rewSpellCast int
	var prevQuestID, nextQuestID, exclusiveGroup, nextQuestInChain int
	var reqItems [4]int
	var reqItemCounts [4]int
	var reqCreatureGO [4]int
	var reqCreatureGOCounts [4]int

	err := row.Scan(
		&q.Entry, &q.Title, &details, &objectives, &offerReward, &endText,
		&q.QuestLevel, &q.MinLevel, &q.Type, &q.ZoneOrSort,
		&q.RequiredRaces, &q.RequiredClasses,
		&q.RewardXP, &q.RewardMoney, &q.RewardSpell, &rewSpellCast,
		&rewItems[0], &rewItems[1], &rewItems[2], &rewItems[3],
		&rewItemCounts[0], &rewItemCounts[1], &rewItemCounts[2], &rewItemCounts[3],
		&rewChoiceItems[0], &rewChoiceItems[1], &rewChoiceItems[2], &rewChoiceItems[3], &rewChoiceItems[4], &rewChoiceItems[5],
		&rewChoiceItemCounts[0], &rewChoiceItemCounts[1], &rewChoiceItemCounts[2], &rewChoiceItemCounts[3], &rewChoiceItemCounts[4], &rewChoiceItemCounts[5],
		&repFactions[0], &repFactions[1], &repFactions[2], &repFactions[3], &repFactions[4],
		&repValues[0], &repValues[1], &repValues[2], &repValues[3], &repValues[4],
		&prevQuestID, &nextQuestID, &exclusiveGroup, &nextQuestInChain,
		&reqItems[0], &reqItems[1], &reqItems[2], &reqItems[3],
		&reqItemCounts[0], &reqItemCounts[1], &reqItemCounts[2], &reqItemCounts[3],
		&reqCreatureGO[0], &reqCreatureGO[1], &reqCreatureGO[2], &reqCreatureGO[3],
		&reqCreatureGOCounts[0], &reqCreatureGOCounts[1], &reqCreatureGOCounts[2], &reqCreatureGOCounts[3],
	)
	if err != nil {
		return nil, err
	}

	// Reward spell the quest teaches (RewSpell, or a learn-type RewSpellCast),
	// resolved through any learn-spell wrapper to the real ability.
	q.RewardSpellInfo = r.resolveQuestRewardSpell(q.RewardSpell, rewSpellCast)

	q.Title = cleanQuestEscapes(q.Title)
	if details != nil {
		q.Details = cleanQuestEscapes(*details)
	}
	if objectives != nil {
		q.Objectives = cleanQuestEscapes(*objectives)
	}
	if offerReward != nil {
		q.OfferRewardText = cleanQuestEscapes(*offerReward)
	}
	if endText != nil {
		q.EndText = cleanQuestEscapes(*endText)
	}

	// Resolve Side and Races
	q.Side, q.RaceNames = resolveSideAndRaces(q.RequiredRaces)
	q.Classes = r.resolveClasses(q.RequiredClasses)

	// Reputation rewards (faction id -> name).
	for i := 0; i < 5; i++ {
		if repFactions[i] > 0 && repValues[i] != 0 {
			rep := &models.QuestReputation{FactionID: repFactions[i], Value: repValues[i]}
			r.db.QueryRow("SELECT name FROM factions WHERE id = ?", repFactions[i]).Scan(&rep.Name)
			if rep.Name == "" {
				rep.Name = fmt.Sprintf("Faction #%d", repFactions[i])
			}
			q.Reputation = append(q.Reputation, rep)
		}
	}

	// Process reward items
	for i := 0; i < 4; i++ {
		if rewItems[i] > 0 {
			item := &models.QuestItem{Entry: rewItems[i], Count: rewItemCounts[i]}
			var name, icon string
			var quality int
			r.db.QueryRow(`
				SELECT i.name, COALESCE(idi.icon, ''), i.quality 
				FROM item_template i 
				LEFT JOIN item_display_info idi ON i.display_id = idi.ID 
				WHERE i.entry = ?
			`, rewItems[i]).Scan(&name, &icon, &quality)
			item.Name = name
			item.Icon = icon
			item.Quality = quality
			q.RewardItems = append(q.RewardItems, item)
		}
	}

	// Process choice items
	for i := 0; i < 6; i++ {
		if rewChoiceItems[i] > 0 {
			item := &models.QuestItem{Entry: rewChoiceItems[i], Count: rewChoiceItemCounts[i]}
			var name, icon string
			var quality int
			r.db.QueryRow(`
				SELECT i.name, COALESCE(idi.icon, ''), i.quality 
				FROM item_template i 
				LEFT JOIN item_display_info idi ON i.display_id = idi.ID 
				WHERE i.entry = ?
			`, rewChoiceItems[i]).Scan(&name, &icon, &quality)
			item.Name = name
			item.Icon = icon
			item.Quality = quality
			q.ChoiceItems = append(q.ChoiceItems, item)
		}
	}

	// Required items (turn-in objectives), with icon/quality for linking.
	for i := 0; i < 4; i++ {
		if reqItems[i] > 0 {
			item := &models.QuestItem{Entry: reqItems[i], Count: reqItemCounts[i]}
			var name, icon string
			var quality int
			r.db.QueryRow(`
				SELECT i.name, COALESCE(idi.icon, ''), i.quality
				FROM item_template i
				LEFT JOIN item_display_info idi ON i.display_id = idi.ID
				WHERE i.entry = ?
			`, reqItems[i]).Scan(&name, &icon, &quality)
			item.Name = name
			item.Icon = icon
			item.Quality = quality
			q.RequiredItems = append(q.RequiredItems, item)
		}
	}

	// Required kill/interact targets: a positive id is a creature, a negative id
	// is a gameobject (stored as its negation).
	for i := 0; i < 4; i++ {
		id := reqCreatureGO[i]
		if id == 0 {
			continue
		}
		o := &models.QuestObjective{Count: reqCreatureGOCounts[i]}
		if id > 0 {
			o.Entry, o.Kind = id, "npc"
			r.db.QueryRow("SELECT name FROM creature_template WHERE entry = ?", id).Scan(&o.Name)
		} else {
			o.Entry, o.Kind = -id, "object"
			r.db.QueryRow("SELECT name FROM gameobject_template WHERE entry = ?", -id).Scan(&o.Name)
		}
		q.RequiredObjectives = append(q.RequiredObjectives, o)
	}

	// Process prev quests
	if prevQuestID != 0 {
		var title string
		r.db.QueryRow("SELECT Title FROM quest_template WHERE entry = ?", prevQuestID).Scan(&title)
		q.PrevQuests = append(q.PrevQuests, &models.QuestSeriesItem{Entry: prevQuestID, Title: title})
	}

	// Build complete quest chain (all quests before and after this one)
	q.Series = r.buildQuestChain(entry, prevQuestID, nextQuestInChain)

	// Query Starters (NPCs that give this quest)
	startersRows, err := r.db.Query(`
		SELECT c.entry, c.name FROM creature_questrelation cq
		JOIN creature_template c ON cq.id = c.entry
		WHERE cq.quest = ?
	`, entry)
	if err == nil {
		defer startersRows.Close()
		for startersRows.Next() {
			var npcEntry int
			var npcName string
			if err := startersRows.Scan(&npcEntry, &npcName); err == nil {
				q.Starters = append(q.Starters, &models.QuestRelation{
					Entry: npcEntry,
					Name:  npcName,
					Type:  "npc",
				})
			}
		}
	}

	// Query Enders (NPCs that complete this quest)
	endersRows, err := r.db.Query(`
		SELECT c.entry, c.name FROM creature_involvedrelation ci
		JOIN creature_template c ON ci.id = c.entry
		WHERE ci.quest = ?
	`, entry)
	if err == nil {
		defer endersRows.Close()
		for endersRows.Next() {
			var npcEntry int
			var npcName string
			if err := endersRows.Scan(&npcEntry, &npcName); err == nil {
				q.Enders = append(q.Enders, &models.QuestRelation{
					Entry: npcEntry,
					Name:  npcName,
					Type:  "npc",
				})
			}
		}
	}

	return q, nil
}

// buildQuestChain builds a complete quest chain by traversing backwards and forwards
func (r *QuestRepository) buildQuestChain(currentEntry int, prevQuestID int, nextQuestInChain int) []*models.QuestSeriesItem {
	var chain []*models.QuestSeriesItem
	visited := make(map[int]bool)

	// Traverse backwards to find all previous quests (returns in chronological order: earliest first)
	prevQuests := r.getQuestChainBackwards(prevQuestID, visited)

	// Add previous quests in order (already in correct chronological order)
	chain = append(chain, prevQuests...)

	// Add current quest
	var currentTitle string
	r.db.QueryRow("SELECT Title FROM quest_template WHERE entry = ?", currentEntry).Scan(&currentTitle)
	chain = append(chain, &models.QuestSeriesItem{Entry: currentEntry, Title: currentTitle, Depth: 0})
	visited[currentEntry] = true

	// Traverse forwards to find all following quests
	// First try NextQuestInChain, then try reverse lookup (quests that have this as PrevQuestId)
	// Children start at Depth 1 relative to current quest
	nextQuests := r.getQuestChainForwards(currentEntry, nextQuestInChain, visited, 0)
	chain = append(chain, nextQuests...)

	// Only return chain if there's more than just the current quest
	if len(chain) <= 1 {
		return nil
	}

	return chain
}

// getQuestChainBackwards recursively gets all preceding quests
func (r *QuestRepository) getQuestChainBackwards(questID int, visited map[int]bool) []*models.QuestSeriesItem {
	if questID == 0 || visited[questID] {
		return nil
	}
	visited[questID] = true

	var title string
	var prevID int
	err := r.db.QueryRow("SELECT Title, IFNULL(PrevQuestId, 0) FROM quest_template WHERE entry = ?", questID).Scan(&title, &prevID)
	if err != nil {
		return nil
	}

	// Get earlier quests first (recursive)
	result := r.getQuestChainBackwards(prevID, visited)

	// Add this quest
	result = append(result, &models.QuestSeriesItem{Entry: questID, Title: title, Depth: 0})

	return result
}

// getQuestChainForwards recursively gets all following quests
// Uses both NextQuestInChain and reverse lookup (quests that have this as PrevQuestId)
func (r *QuestRepository) getQuestChainForwards(currentQuestID int, nextQuestInChain int, visited map[int]bool, parentDepth int) []*models.QuestSeriesItem {
	var result []*models.QuestSeriesItem
	currentDepth := parentDepth + 1

	// Method 1: Use NextQuestInChain if available
	if nextQuestInChain > 0 && !visited[nextQuestInChain] {
		visited[nextQuestInChain] = true
		var title string
		var nextNext int
		err := r.db.QueryRow("SELECT Title, IFNULL(NextQuestInChain, 0) FROM quest_template WHERE entry = ?", nextQuestInChain).Scan(&title, &nextNext)
		if err == nil {
			result = append(result, &models.QuestSeriesItem{Entry: nextQuestInChain, Title: title, Depth: currentDepth})
			// Continue recursively
			result = append(result, r.getQuestChainForwards(nextQuestInChain, nextNext, visited, currentDepth)...)
		}
		return result
	}

	// Method 2: Reverse lookup - find quests that have currentQuestID as their PrevQuestId
	rows, err := r.db.Query("SELECT entry, Title, IFNULL(NextQuestInChain, 0) FROM quest_template WHERE PrevQuestId = ? OR PrevQuestId = ?",
		currentQuestID, -currentQuestID)
	if err != nil {
		return result
	}

	// Struct to hold temporary results to allow closing rows before recursion
	type nextQuestInfo struct {
		entry    int
		title    string
		nextNext int
	}
	var nextQuests []nextQuestInfo

	for rows.Next() {
		var info nextQuestInfo
		if err := rows.Scan(&info.entry, &info.title, &info.nextNext); err != nil {
			continue
		}
		nextQuests = append(nextQuests, info)
	}
	rows.Close() // Close rows immediately after scanning

	// Now process recursions
	for _, info := range nextQuests {
		if visited[info.entry] {
			continue
		}
		visited[info.entry] = true
		result = append(result, &models.QuestSeriesItem{Entry: info.entry, Title: info.title, Depth: currentDepth})

		// Continue recursively for this branch
		result = append(result, r.getQuestChainForwards(info.entry, info.nextNext, visited, currentDepth)...)
	}

	return result
}

// WoW gender escapes: $GmaleText:femaleText; (and $g). Most quest text
// terminates them with ';', but some omit it and rely on the following
// punctuation (e.g. "$Gboy:girl,"). Surrounding spaces also occur
// ("$g lad : lass;"). Viewer gender is unknown, so we render both forms as
// "male/female".
var (
	// Standard, ';'-terminated form (consumes the ';').
	questGenderRe = regexp.MustCompile(`\$[Gg]\s*([^:;]*?)\s*:\s*([^;]*?)\s*;`)
	// Fallback for the ';'-less form: single words, stop at the first
	// non-letter so trailing punctuation/text is preserved.
	questGenderNoSemiRe = regexp.MustCompile(`\$[Gg]\s*([A-Za-z]+)\s*:\s*([A-Za-z']+)`)
)

// cleanQuestEscapes converts WoW quest text escape codes into plain readable
// text. The client substitutes these at display time; our stored text keeps
// them raw, so without this the UI shows literal "$B" etc.
//
//	$B / $b            -> line break
//	$G male:female;    -> "male/female" (viewer gender is unknown)
//	$N / $n            -> "you" (player name)
//	$R $C $r $c        -> stripped (race/class, unknown to us)
func cleanQuestEscapes(s string) string {
	if s == "" {
		return s
	}
	s = questGenderRe.ReplaceAllString(s, "${1}/${2}")
	s = questGenderNoSemiRe.ReplaceAllString(s, "${1}/${2}")
	s = strings.ReplaceAll(s, "$B", "\n")
	s = strings.ReplaceAll(s, "$b", "\n")
	s = strings.ReplaceAll(s, "$N", "you")
	s = strings.ReplaceAll(s, "$n", "you")
	for _, code := range []string{"$R", "$r", "$C", "$c"} {
		s = strings.ReplaceAll(s, code, "")
	}
	return s
}

func resolveSideAndRaces(mask int) (string, string) {
	if mask == 0 {
		return "Both", "All"
	}

	type raceInfo struct {
		bit  int
		name string
		side string
	}

	races := []raceInfo{
		{1, "Human", "Alliance"},
		{2, "Orc", "Horde"},
		{4, "Dwarf", "Alliance"},
		{8, "Night Elf", "Alliance"},
		{16, "Undead", "Horde"},
		{32, "Tauren", "Horde"},
		{64, "Gnome", "Alliance"},
		{128, "Troll", "Horde"},
		{256, "Goblin", "Horde"},
		{512, "High Elf", "Alliance"},
	}

	var raceNames []string
	hasAlliance := false
	hasHorde := false

	for _, r := range races {
		if mask&r.bit != 0 {
			raceNames = append(raceNames, r.name)
			if r.side == "Alliance" {
				hasAlliance = true
			}
			if r.side == "Horde" {
				hasHorde = true
			}
		}
	}

	side := "Both"
	if hasAlliance && !hasHorde {
		side = "Alliance"
	} else if hasHorde && !hasAlliance {
		side = "Horde"
	}

	return side, strings.Join(raceNames, ", ")
}

// resolveClasses turns a quest's RequiredClasses bitmask into the list of
// restricted classes (name + UI color). The mask bit for a class is
// 1<<(classId-1), so names/colors come straight from class_info (ChrClasses.dbc),
// ordered by class id. Mask 0 = no class restriction.
func (r *QuestRepository) resolveClasses(mask int) []*models.QuestClass {
	if mask == 0 {
		return nil
	}

	rows, err := r.db.Query("SELECT id, name, COALESCE(color,'') FROM class_info ORDER BY id")
	if err != nil {
		return nil
	}
	defer rows.Close()

	var classes []*models.QuestClass
	for rows.Next() {
		var id int
		var name, color string
		if rows.Scan(&id, &name, &color) != nil {
			continue
		}
		if mask&(1<<(id-1)) != 0 {
			classes = append(classes, &models.QuestClass{Name: name, Color: color})
		}
	}
	return classes
}
