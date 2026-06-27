// Package repositories contains database access layer implementations
package repositories

import (
	"database/sql"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"inklab/backend/database/helpers"
	"inklab/backend/database/models"
)

// ItemRepository handles item-related database operations
type ItemRepository struct {
	db *sql.DB
}

// NewItemRepository creates a new item repository
func NewItemRepository(db *sql.DB) *ItemRepository {
	return &ItemRepository{db: db}
}

// SearchItems searches for items by name
func (r *ItemRepository) SearchItems(query string, limit int) ([]*models.Item, error) {
	rows, err := r.db.Query(`
		SELECT t.entry, t.name, t.quality, t.item_level, t.required_level, 
			t.class, t.subclass, t.inventory_type, COALESCE(d.icon, '')
		FROM item_template t
		LEFT JOIN item_display_info d ON t.display_id = d.ID
		WHERE t.name LIKE ?
		ORDER BY length(t.name), t.name
		LIMIT ?
	`, "%"+query+"%", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*models.Item
	for rows.Next() {
		item := &models.Item{}
		err := rows.Scan(
			&item.Entry, &item.Name, &item.Quality, &item.ItemLevel,
			&item.RequiredLevel, &item.Class, &item.SubClass, &item.InventoryType, &item.IconPath,
		)
		if err != nil {
			continue
		}
		items = append(items, item)
	}
	return items, nil
}

// GetItemByID retrieves a single item by ID
func (r *ItemRepository) GetItemByID(id int) (*models.Item, error) {
	item := &models.Item{}
	err := r.db.QueryRow(`
		SELECT t.entry, t.name, COALESCE(t.description, ''), t.quality, t.item_level, t.required_level,
			t.class, t.subclass, t.inventory_type, COALESCE(d.icon, ''), t.sell_price,
			t.allowable_class, t.allowable_race, t.bonding, t.max_durability, t.max_count, t.armor,
			t.stat_type1, t.stat_value1, t.stat_type2, t.stat_value2, t.stat_type3, t.stat_value3,
			t.stat_type4, t.stat_value4, t.stat_type5, t.stat_value5, t.stat_type6, t.stat_value6,
			t.stat_type7, t.stat_value7, t.stat_type8, t.stat_value8, t.stat_type9, t.stat_value9,
			t.stat_type10, t.stat_value10,
			t.delay, t.dmg_min1, t.dmg_max1, t.dmg_type1,
			t.dmg_min2, t.dmg_max2, t.dmg_type2,
			t.holy_res, t.fire_res, t.nature_res, t.frost_res, t.shadow_res, t.arcane_res,
			t.spellid_1, t.spelltrigger_1, t.spellid_2, t.spelltrigger_2, t.spellid_3, t.spelltrigger_3,
			t.set_id, t.container_slots
		FROM item_template t
		LEFT JOIN item_display_info d ON t.display_id = d.ID
		WHERE t.entry = ?
	`, id).Scan(
		&item.Entry, &item.Name, &item.Description, &item.Quality, &item.ItemLevel, &item.RequiredLevel,
		&item.Class, &item.SubClass, &item.InventoryType, &item.IconPath, &item.SellPrice,
		&item.AllowableClass, &item.AllowableRace, &item.Bonding, &item.MaxDurability, &item.MaxCount, &item.Armor,
		&item.StatType1, &item.StatValue1, &item.StatType2, &item.StatValue2, &item.StatType3, &item.StatValue3,
		&item.StatType4, &item.StatValue4, &item.StatType5, &item.StatValue5, &item.StatType6, &item.StatValue6,
		&item.StatType7, &item.StatValue7, &item.StatType8, &item.StatValue8, &item.StatType9, &item.StatValue9,
		&item.StatType10, &item.StatValue10,
		&item.Delay, &item.DmgMin1, &item.DmgMax1, &item.DmgType1,
		&item.DmgMin2, &item.DmgMax2, &item.DmgType2,
		&item.HolyRes, &item.FireRes, &item.NatureRes, &item.FrostRes, &item.ShadowRes, &item.ArcaneRes,
		&item.SpellID1, &item.SpellTrigger1, &item.SpellID2, &item.SpellTrigger2, &item.SpellID3, &item.SpellTrigger3,
		&item.SetID, &item.ContainerSlots,
	)
	if err != nil {
		return nil, err
	}
	return item, nil
}

// GetItemCount returns the total number of items
func (r *ItemRepository) GetItemCount() (int, error) {
	var count int
	err := r.db.QueryRow("SELECT COUNT(*) FROM item_template").Scan(&count)
	return count, err
}

// GetItemClasses returns all item classes with their subclasses and inventory slots
func (r *ItemRepository) GetItemClasses() ([]*models.ItemClass, error) {
	rows, err := r.db.Query(`
		SELECT DISTINCT class, subclass, inventory_type
		FROM item_template
		WHERE class IN (0,1,2,4,5,6,7,9,11,12,13,15)
		ORDER BY class, subclass, inventory_type
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	classMap := make(map[int]*models.ItemClass)
	subclassMap := make(map[string]*models.ItemSubClass)

	// Mapping from Two-Handed subclasses to their base types
	// subclass 1 (Two-Handed Axe) -> subclass 0 (Axe)
	// subclass 5 (Two-Handed Mace) -> subclass 4 (Mace)
	// subclass 8 (Two-Handed Sword) -> subclass 7 (Sword)
	twoHandedToBase := map[int]int{
		1: 0,
		5: 4,
		8: 7,
	}

	for rows.Next() {
		var class, subclass, invType int
		if err := rows.Scan(&class, &subclass, &invType); err != nil {
			continue
		}

		// Ensure class exists
		if _, exists := classMap[class]; !exists {
			classMap[class] = &models.ItemClass{
				Class:      class,
				Name:       helpers.GetClassName(class),
				SubClasses: []*models.ItemSubClass{},
			}
		}

		// For weapons (class 2), merge Two-Handed subclasses into base types
		displaySubclass := subclass
		if class == 2 {
			if baseSubclass, isTwoHanded := twoHandedToBase[subclass]; isTwoHanded {
				// This is a Two-Handed subclass, merge into base type
				displaySubclass = baseSubclass
			}
		}

		// Ensure subclass exists (use displaySubclass for key)
		subKey := fmt.Sprintf("%d-%d", class, displaySubclass)
		if _, exists := subclassMap[subKey]; !exists {
			sc := &models.ItemSubClass{
				Class:          class,
				SubClass:       displaySubclass,
				Name:           helpers.GetSubClassName(class, displaySubclass),
				InventorySlots: []*models.InventorySlot{},
			}
			subclassMap[subKey] = sc
			classMap[class].SubClasses = append(classMap[class].SubClasses, sc)
		}

		// Add inventory slot if applicable (mainly for armor/weapons)
		// For weapons, add slots from both base and Two-Handed subclasses
		if (class == 2 || class == 4) && invType > 0 {
			// Check if this slot already exists
			slotExists := false
			for _, existingSlot := range subclassMap[subKey].InventorySlots {
				if existingSlot.InventoryType == invType {
					slotExists = true
					break
				}
			}
			if !slotExists {
				slot := &models.InventorySlot{
					Class:         class,
					SubClass:      displaySubclass,
					InventoryType: invType,
					Name:          helpers.GetInventoryTypeName(invType),
				}
				subclassMap[subKey].InventorySlots = append(subclassMap[subKey].InventorySlots, slot)
			}
		}
	}

	// Convert map to slice and sort
	var classes []*models.ItemClass
	for _, c := range classMap {
		classes = append(classes, c)
	}
	sort.Slice(classes, func(i, j int) bool {
		return classes[i].Class < classes[j].Class
	})

	return classes, nil
}

// GetStatTypes returns the distinct item stat types that actually appear on at
// least one item, each with its display name from stat_types. This drives the
// item filter's stat dropdown so it adapts to whatever stats the data uses.
// Resistances/armor are dedicated columns (not stat_typeN) and are handled
// separately on the frontend.
func (r *ItemRepository) GetStatTypes() ([]*models.StatType, error) {
	rows, err := r.db.Query(`
		WITH used AS (
			SELECT stat_type1 AS id FROM item_template WHERE stat_type1 > 0
			UNION SELECT stat_type2 FROM item_template WHERE stat_type2 > 0
			UNION SELECT stat_type3 FROM item_template WHERE stat_type3 > 0
			UNION SELECT stat_type4 FROM item_template WHERE stat_type4 > 0
			UNION SELECT stat_type5 FROM item_template WHERE stat_type5 > 0
			UNION SELECT stat_type6 FROM item_template WHERE stat_type6 > 0
			UNION SELECT stat_type7 FROM item_template WHERE stat_type7 > 0
			UNION SELECT stat_type8 FROM item_template WHERE stat_type8 > 0
			UNION SELECT stat_type9 FROM item_template WHERE stat_type9 > 0
			UNION SELECT stat_type10 FROM item_template WHERE stat_type10 > 0
		)
		SELECT used.id, COALESCE(st.name, '')
		FROM used
		LEFT JOIN stat_types st ON st.id = used.id
		ORDER BY used.id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []*models.StatType
	for rows.Next() {
		var s models.StatType
		if err := rows.Scan(&s.ID, &s.Name); err != nil {
			continue
		}
		if s.Name == "" {
			if n := helpers.GetStatName(s.ID); n != "" {
				s.Name = n
			} else {
				s.Name = fmt.Sprintf("Stat %d", s.ID)
			}
		}
		stats = append(stats, &s)
	}
	return stats, nil
}

// GetItemsByClass returns items filtered by class and subclass
func (r *ItemRepository) GetItemsByClass(class, subClass int, nameFilter string, limit, offset int) ([]*models.Item, int, error) {
	// For weapons (class 2), when querying base types, also include the dedicated Two-Handed subclass
	// subclass 0 (Axe) -> also include subclass 1 (Two-Handed Axe)
	// subclass 4 (Mace) -> also include subclass 5 (Two-Handed Mace)
	// subclass 7 (Sword) -> also include subclass 8 (Two-Handed Sword)
	baseToTwoHanded := map[int]int{
		0: 1,
		4: 5,
		7: 8,
	}

	var whereClause string
	var args []interface{}

	if class == 2 {
		if twoHandedSubclass, hasTwoHanded := baseToTwoHanded[subClass]; hasTwoHanded {
			// Include both the base subclass AND the dedicated Two-Handed subclass
			whereClause = "WHERE class = ? AND subclass IN (?, ?)"
			args = []interface{}{class, subClass, twoHandedSubclass}
		} else {
			whereClause = "WHERE class = ? AND subclass = ?"
			args = []interface{}{class, subClass}
		}
	} else {
		whereClause = "WHERE class = ? AND subclass = ?"
		args = []interface{}{class, subClass}
	}

	if nameFilter != "" {
		whereClause += " AND name LIKE ?"
		args = append(args, "%"+nameFilter+"%")
	}

	// Count
	var count int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM item_template %s", whereClause)
	err := r.db.QueryRow(countQuery, args...).Scan(&count)
	if err != nil {
		return nil, 0, err
	}

	// Data
	dataArgs := append(args, limit, offset)
	dataQuery := fmt.Sprintf(`
		SELECT entry, name, quality, item_level, required_level, class, subclass, inventory_type, COALESCE(d.icon, ''),
			armor,
			stat_type1, stat_value1, stat_type2, stat_value2, stat_type3, stat_value3, stat_type4, stat_value4, stat_type5, stat_value5,
			stat_type6, stat_value6, stat_type7, stat_value7, stat_type8, stat_value8, stat_type9, stat_value9, stat_type10, stat_value10,
			holy_res, fire_res, nature_res, frost_res, shadow_res, arcane_res
		FROM item_template t
		LEFT JOIN item_display_info d ON t.display_id = d.ID
		%s
		ORDER BY quality DESC, item_level DESC
		LIMIT ? OFFSET ?
	`, whereClause)

	rows, err := r.db.Query(dataQuery, dataArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var items []*models.Item
	for rows.Next() {
		item := &models.Item{}
		err := rows.Scan(
			&item.Entry, &item.Name, &item.Quality, &item.ItemLevel,
			&item.RequiredLevel, &item.Class, &item.SubClass, &item.InventoryType, &item.IconPath,
			&item.Armor,
			&item.StatType1, &item.StatValue1, &item.StatType2, &item.StatValue2, &item.StatType3, &item.StatValue3,
			&item.StatType4, &item.StatValue4, &item.StatType5, &item.StatValue5, &item.StatType6, &item.StatValue6,
			&item.StatType7, &item.StatValue7, &item.StatType8, &item.StatValue8, &item.StatType9, &item.StatValue9,
			&item.StatType10, &item.StatValue10,
			&item.HolyRes, &item.FireRes, &item.NatureRes, &item.FrostRes, &item.ShadowRes, &item.ArcaneRes,
		)
		if err != nil {
			continue
		}
		items = append(items, item)
	}

	return items, count, nil
}

// GetItemsByClassAndSlot returns items filtered by class, subclass, and inventory type
func (r *ItemRepository) GetItemsByClassAndSlot(class, subClass, inventoryType int, nameFilter string, limit, offset int) ([]*models.Item, int, error) {
	// For weapons (class 2), when querying Two-Hand slot (17), also include the dedicated Two-Handed subclass
	// subclass 0 (Axe) + inv 17 -> also include subclass 1 (Two-Handed Axe)
	// subclass 4 (Mace) + inv 17 -> also include subclass 5 (Two-Handed Mace)
	// subclass 7 (Sword) + inv 17 -> also include subclass 8 (Two-Handed Sword)
	baseToTwoHanded := map[int]int{
		0: 1,
		4: 5,
		7: 8,
	}

	var whereClause string
	var args []interface{}

	if class == 2 && inventoryType == 17 {
		if twoHandedSubclass, hasTwoHanded := baseToTwoHanded[subClass]; hasTwoHanded {
			// Include both the base subclass with Two-Hand slot AND the dedicated Two-Handed subclass
			whereClause = "WHERE class = ? AND ((subclass = ? AND inventory_type = ?) OR subclass = ?)"
			args = []interface{}{class, subClass, inventoryType, twoHandedSubclass}
		} else {
			whereClause = "WHERE class = ? AND subclass = ? AND inventory_type = ?"
			args = []interface{}{class, subClass, inventoryType}
		}
	} else {
		whereClause = "WHERE class = ? AND subclass = ? AND inventory_type = ?"
		args = []interface{}{class, subClass, inventoryType}
	}

	if nameFilter != "" {
		whereClause += " AND name LIKE ?"
		args = append(args, "%"+nameFilter+"%")
	}

	// Count
	var count int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM item_template %s", whereClause)
	err := r.db.QueryRow(countQuery, args...).Scan(&count)
	if err != nil {
		return nil, 0, err
	}

	// Data
	dataArgs := append(args, limit, offset)
	dataQuery := fmt.Sprintf(`
		SELECT entry, name, quality, item_level, required_level, class, subclass, inventory_type, COALESCE(d.icon, ''),
			armor,
			stat_type1, stat_value1, stat_type2, stat_value2, stat_type3, stat_value3, stat_type4, stat_value4, stat_type5, stat_value5,
			stat_type6, stat_value6, stat_type7, stat_value7, stat_type8, stat_value8, stat_type9, stat_value9, stat_type10, stat_value10,
			holy_res, fire_res, nature_res, frost_res, shadow_res, arcane_res
		FROM item_template t
		LEFT JOIN item_display_info d ON t.display_id = d.ID
		%s
		ORDER BY quality DESC, item_level DESC
		LIMIT ? OFFSET ?
	`, whereClause)

	rows, err := r.db.Query(dataQuery, dataArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var items []*models.Item
	for rows.Next() {
		item := &models.Item{}
		err := rows.Scan(
			&item.Entry, &item.Name, &item.Quality, &item.ItemLevel,
			&item.RequiredLevel, &item.Class, &item.SubClass, &item.InventoryType, &item.IconPath,
			&item.Armor,
			&item.StatType1, &item.StatValue1, &item.StatType2, &item.StatValue2, &item.StatType3, &item.StatValue3,
			&item.StatType4, &item.StatValue4, &item.StatType5, &item.StatValue5, &item.StatType6, &item.StatValue6,
			&item.StatType7, &item.StatValue7, &item.StatType8, &item.StatValue8, &item.StatType9, &item.StatValue9,
			&item.StatType10, &item.StatValue10,
			&item.HolyRes, &item.FireRes, &item.NatureRes, &item.FrostRes, &item.ShadowRes, &item.ArcaneRes,
		)
		if err != nil {
			continue
		}
		items = append(items, item)
	}

	return items, count, nil
}

// AdvancedSearch performs a multi-dimensional search on items
func (r *ItemRepository) AdvancedSearch(filter models.SearchFilter) (*models.SearchResult, error) {
	if filter.Limit <= 0 {
		filter.Limit = 50
	}
	if filter.Limit > 200 {
		filter.Limit = 200
	}

	var conditions []string
	var args []interface{}

	// Name or ID filter
	if filter.Query != "" {
		// Check if query is a number (ID search)
		if id, err := strconv.Atoi(filter.Query); err == nil {
			conditions = append(conditions, "entry = ?")
			args = append(args, id)
		} else {
			// Text search by name
			conditions = append(conditions, "name LIKE ?")
			args = append(args, "%"+filter.Query+"%")
		}
	}

	// Quality filter
	if len(filter.Quality) > 0 {
		placeholders := make([]string, len(filter.Quality))
		for i, q := range filter.Quality {
			placeholders[i] = "?"
			args = append(args, q)
		}
		conditions = append(conditions, fmt.Sprintf("quality IN (%s)", strings.Join(placeholders, ",")))
	}

	// Class filter
	if len(filter.Class) > 0 {
		placeholders := make([]string, len(filter.Class))
		for i, c := range filter.Class {
			placeholders[i] = "?"
			args = append(args, c)
		}
		conditions = append(conditions, fmt.Sprintf("class IN (%s)", strings.Join(placeholders, ",")))
	}

	// SubClass filter
	if len(filter.SubClass) > 0 {
		placeholders := make([]string, len(filter.SubClass))
		for i, sc := range filter.SubClass {
			placeholders[i] = "?"
			args = append(args, sc)
		}
		conditions = append(conditions, fmt.Sprintf("subclass IN (%s)", strings.Join(placeholders, ",")))
	}

	// InventoryType filter
	if len(filter.InventoryType) > 0 {
		placeholders := make([]string, len(filter.InventoryType))
		for i, it := range filter.InventoryType {
			placeholders[i] = "?"
			args = append(args, it)
		}
		conditions = append(conditions, fmt.Sprintf("inventory_type IN (%s)", strings.Join(placeholders, ",")))
	}

	// Level Range
	if filter.MinLevel > 0 {
		conditions = append(conditions, "item_level >= ?")
		args = append(args, filter.MinLevel)
	}
	if filter.MaxLevel > 0 {
		conditions = append(conditions, "item_level <= ?")
		args = append(args, filter.MaxLevel)
	}

	// Required Level Range
	if filter.MinReqLevel > 0 {
		conditions = append(conditions, "required_level >= ?")
		args = append(args, filter.MinReqLevel)
	}
	if filter.MaxReqLevel > 0 {
		conditions = append(conditions, "required_level <= ?")
		args = append(args, filter.MaxReqLevel)
	}

	// Build WHERE clause
	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Count query
	countQuery := "SELECT COUNT(*) FROM item_template " + whereClause
	var totalCount int
	err := r.db.QueryRow(countQuery, args...).Scan(&totalCount)
	if err != nil {
		return nil, fmt.Errorf("search count error: %w", err)
	}

	// Data query
	dataQuery := fmt.Sprintf(`
		SELECT entry, name, quality, item_level, required_level, class, subclass, inventory_type, COALESCE(d.icon, '')
		FROM item_template t
		LEFT JOIN item_display_info d ON t.display_id = d.ID
		%s
		ORDER BY quality DESC, item_level DESC
		LIMIT ? OFFSET ?
	`, whereClause)

	// Add limit/offset args
	args = append(args, filter.Limit, filter.Offset)

	rows, err := r.db.Query(dataQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("search data error: %w", err)
	}
	defer rows.Close()

	var items []*models.Item
	for rows.Next() {
		item := &models.Item{}
		err := rows.Scan(
			&item.Entry, &item.Name, &item.Quality, &item.ItemLevel,
			&item.RequiredLevel, &item.Class, &item.SubClass, &item.InventoryType, &item.IconPath,
		)
		if err != nil {
			continue
		}
		items = append(items, item)
	}

	return &models.SearchResult{
		Items:      items,
		TotalCount: totalCount,
	}, nil
}

// GetItemSets returns all item sets for browsing
func (r *ItemRepository) GetItemSets() ([]*models.ItemSetBrowse, error) {
	rows, err := r.db.Query(`
		SELECT 
			itemset_id, name,
			item1, item2, item3, item4, item5, item6, item7, item8, item9, item10,
			skill_id, skill_level
		FROM itemsets
		ORDER BY name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Set of item entries that actually exist, so counts and visibility reflect
	// only displayable pieces. Some sets (octo placeholders / future content)
	// reference item ids that exist nowhere in our data; those would otherwise
	// show as a bare "0 items" entry.
	exists := map[int]bool{}
	if er, e := r.db.Query("SELECT entry FROM item_template"); e == nil {
		for er.Next() {
			var id int
			if er.Scan(&id) == nil {
				exists[id] = true
			}
		}
		er.Close()
	}

	// Derive each set's class restriction from its items' allowable_class.
	// allowable_class is a class bitmask (same bits as class_mask); -1/0 or
	// "all real classes set" means unrestricted. OR the restricted bits across a
	// set's items so a class-specific set (e.g. Tier pieces) carries that class.
	const classAll = 1503 // 1+2+4+8+16+64+128+256+1024 (the 9 playable classes)
	classMaskBySet := map[int]int{}
	if cr, e := r.db.Query("SELECT set_id, allowable_class FROM item_template WHERE set_id > 0"); e == nil {
		for cr.Next() {
			var sid, ac int
			if cr.Scan(&sid, &ac) == nil && ac > 0 && (ac&classAll) != classAll {
				classMaskBySet[sid] |= ac & classAll
			}
		}
		cr.Close()
	}

	var sets []*models.ItemSetBrowse
	for rows.Next() {
		set := &models.ItemSetBrowse{}
		var items [10]int
		err := rows.Scan(
			&set.ItemSetID, &set.Name,
			&items[0], &items[1], &items[2], &items[3], &items[4],
			&items[5], &items[6], &items[7], &items[8], &items[9],
			&set.SkillID, &set.SkillLevel,
		)
		if err != nil {
			continue
		}

		// Keep only item IDs that resolve to a real item.
		for _, itemID := range items {
			if itemID > 0 && exists[itemID] {
				set.ItemIDs = append(set.ItemIDs, itemID)
			}
		}
		set.ItemCount = len(set.ItemIDs)

		// Hide sets with no displayable pieces (they reappear once cached).
		if set.ItemCount == 0 {
			continue
		}

		set.ClassMask = classMaskBySet[set.ItemSetID]
		sets = append(sets, set)
	}

	return sets, nil
}

// GetItemSetDetail returns detailed information about an item set
func (r *ItemRepository) GetItemSetDetail(itemSetID int) (*models.ItemSetDetail, error) {
	row := r.db.QueryRow(`
		SELECT 
			itemset_id, name,
			item1, item2, item3, item4, item5, item6, item7, item8, item9, item10,
			spell1, spell2, spell3, spell4, spell5, spell6, spell7, spell8,
			bonus1, bonus2, bonus3, bonus4, bonus5, bonus6, bonus7, bonus8
		FROM itemsets
		WHERE itemset_id = ?
	`, itemSetID)

	var name string
	var items [10]int
	var spells [8]int
	var bonuses [8]int
	var setID int

	err := row.Scan(
		&setID, &name,
		&items[0], &items[1], &items[2], &items[3], &items[4],
		&items[5], &items[6], &items[7], &items[8], &items[9],
		&spells[0], &spells[1], &spells[2], &spells[3],
		&spells[4], &spells[5], &spells[6], &spells[7],
		&bonuses[0], &bonuses[1], &bonuses[2], &bonuses[3],
		&bonuses[4], &bonuses[5], &bonuses[6], &bonuses[7],
	)
	if err != nil {
		return nil, err
	}

	detail := &models.ItemSetDetail{
		ItemSetID: setID,
		Name:      name,
	}

	// Get item details for each item in the set
	for _, itemID := range items {
		if itemID > 0 {
			item, err := r.GetItemByID(itemID)
			if err == nil && item != nil {
				detail.Items = append(detail.Items, item)
			}
		}
	}

	// Build bonuses list
	var setBonuses []models.SetBonus
	for i := 0; i < 8; i++ {
		if spells[i] > 0 && bonuses[i] > 0 {
			setBonuses = append(setBonuses, models.SetBonus{
				Threshold: bonuses[i],
				SpellID:   spells[i],
			})
		}
	}

	// Sort bonuses by threshold (asc)
	sort.Slice(setBonuses, func(i, j int) bool {
		return setBonuses[i].Threshold < setBonuses[j].Threshold
	})

	// Resolve descriptions
	for i := range setBonuses {
		setBonuses[i].Description = r.resolveSpellText(setBonuses[i].SpellID)
	}

	detail.Bonuses = setBonuses

	return detail, nil
}

// GetTooltipData generates tooltip information for an item
func (r *ItemRepository) GetTooltipData(itemID int) (*models.TooltipData, error) {
	item, err := r.GetItemByID(itemID)
	if err != nil {
		return nil, err
	}

	tooltip := &models.TooltipData{
		Entry:         item.Entry,
		Name:          item.Name,
		Quality:       item.Quality,
		ItemLevel:     item.ItemLevel,
		RequiredLevel: item.RequiredLevel,
		SellPrice:     item.SellPrice,
		Description:   item.Description,
	}

	// Unique
	if item.MaxCount == 1 {
		tooltip.Unique = true
	}

	// Binding
	tooltip.Binding = helpers.GetBondingName(item.Bonding)

	// Item Type and Slot
	itemType := helpers.GetSubClassName(item.Class, item.SubClass)
	itemType = strings.ReplaceAll(itemType, " (One-Handed)", "")
	itemType = strings.ReplaceAll(itemType, " (Two-Handed)", "")
	// Containers (class 1) show their capacity, e.g. "18 Slot Bag".
	if item.Class == 1 && item.ContainerSlots > 0 {
		itemType = fmt.Sprintf("%d Slot %s", item.ContainerSlots, itemType)
	}
	tooltip.ItemType = itemType
	tooltip.Slot = helpers.GetInventoryTypeName(item.InventoryType)

	// Armor
	if item.Armor > 0 {
		tooltip.Armor = item.Armor
	}

	// Weapon damage
	if item.DmgMin1 > 0 || item.DmgMax1 > 0 {
		tooltip.DamageRange = fmt.Sprintf("%.0f - %.0f Damage", item.DmgMin1, item.DmgMax1)
		if item.Delay > 0 {
			speed := float64(item.Delay) / 1000.0
			tooltip.AttackSpeed = fmt.Sprintf("Speed %.2f", speed)
			dps := (item.DmgMin1 + item.DmgMax1) / 2.0 / speed
			// Round half up for DPS
			dpsRounded := math.Round(dps*10) / 10
			tooltip.DPS = fmt.Sprintf("(%.1f damage per second)", dpsRounded)
		}
	}

	// Bonus Damage (e.g. Shadow Damage)
	if item.DmgMin2 > 0 || item.DmgMax2 > 0 {
		typeName := helpers.GetSchoolName(item.DmgType2)
		tooltip.Stats = append(tooltip.Stats, fmt.Sprintf("+%.0f - %.0f %s Damage", item.DmgMin2, item.DmgMax2, typeName))
	}

	// Stats
	statPairs := []struct{ t, v int }{
		{item.StatType1, item.StatValue1}, {item.StatType2, item.StatValue2},
		{item.StatType3, item.StatValue3}, {item.StatType4, item.StatValue4},
		{item.StatType5, item.StatValue5}, {item.StatType6, item.StatValue6},
		{item.StatType7, item.StatValue7}, {item.StatType8, item.StatValue8},
		{item.StatType9, item.StatValue9}, {item.StatType10, item.StatValue10},
	}
	for _, sp := range statPairs {
		if sp.t > 0 && sp.v != 0 {
			tooltip.Stats = append(tooltip.Stats, r.formatStat(sp.t, sp.v))
		}
	}

	// Resistances
	if item.HolyRes > 0 {
		tooltip.Resistances = append(tooltip.Resistances, fmt.Sprintf("+%d Holy Resistance", item.HolyRes))
	}
	if item.FireRes > 0 {
		tooltip.Resistances = append(tooltip.Resistances, fmt.Sprintf("+%d Fire Resistance", item.FireRes))
	}
	if item.NatureRes > 0 {
		tooltip.Resistances = append(tooltip.Resistances, fmt.Sprintf("+%d Nature Resistance", item.NatureRes))
	}
	if item.FrostRes > 0 {
		tooltip.Resistances = append(tooltip.Resistances, fmt.Sprintf("+%d Frost Resistance", item.FrostRes))
	}
	if item.ShadowRes > 0 {
		tooltip.Resistances = append(tooltip.Resistances, fmt.Sprintf("+%d Shadow Resistance", item.ShadowRes))
	}
	if item.ArcaneRes > 0 {
		tooltip.Resistances = append(tooltip.Resistances, fmt.Sprintf("+%d Arcane Resistance", item.ArcaneRes))
	}

	// Durability
	if item.MaxDurability > 0 {
		tooltip.Durability = fmt.Sprintf("Durability %d / %d", item.MaxDurability, item.MaxDurability)
	}

	// Spell Effects. The per-item spell cooldown (the category cooldown is the
	// global 1-sec item-use throttle, not a real shown cooldown), used to show
	// "(N cooldown)" on Use effects.
	var icd [3]int
	r.db.QueryRow(`SELECT spellcooldown_1, spellcooldown_2, spellcooldown_3
		FROM item_template WHERE entry = ?`, itemID).Scan(&icd[0], &icd[1], &icd[2])

	spellPairs := []struct{ id, trigger, cd int }{
		{item.SpellID1, item.SpellTrigger1, icd[0]},
		{item.SpellID2, item.SpellTrigger2, icd[1]},
		{item.SpellID3, item.SpellTrigger3, icd[2]},
	}
	for _, sp := range spellPairs {
		if sp.id > 0 {
			effect := r.formatSpellEffect(sp.id, sp.trigger, sp.cd)
			if effect != "" {
				tooltip.Effects = append(tooltip.Effects, models.TooltipEffect{Text: effect, SpellID: sp.id})
			}
		}
	}

	// Set Info
	if item.SetID > 0 {
		var setInfo models.ItemSetInfo

		var setID, skillID, skillLevel int
		var item1, item2, item3, item4, item5, item6, item7, item8, item9, item10 int
		var spell1, spell2, spell3, spell4, spell5, spell6, spell7, spell8 int
		var bonus1, bonus2, bonus3, bonus4, bonus5, bonus6, bonus7, bonus8 int

		err := r.db.QueryRow(`
			SELECT itemset_id, COALESCE(name, ''),
				item1, item2, item3, item4, item5, item6, item7, item8, item9, item10,
				spell1, spell2, spell3, spell4, spell5, spell6, spell7, spell8,
				bonus1, bonus2, bonus3, bonus4, bonus5, bonus6, bonus7, bonus8,
				skill_id, skill_level
			FROM itemsets WHERE itemset_id = ?
		`, item.SetID).Scan(
			&setID, &setInfo.Name,
			&item1, &item2, &item3, &item4, &item5, &item6, &item7, &item8, &item9, &item10,
			&spell1, &spell2, &spell3, &spell4, &spell5, &spell6, &spell7, &spell8,
			&bonus1, &bonus2, &bonus3, &bonus4, &bonus5, &bonus6, &bonus7, &bonus8,
			&skillID, &skillLevel,
		)

		if err == nil {
			// Process items
			itemIDs := []int{item1, item2, item3, item4, item5, item6, item7, item8, item9, item10}
			for _, id := range itemIDs {
				if id > 0 {
					var itemName string
					r.db.QueryRow("SELECT name FROM item_template WHERE entry = ?", id).Scan(&itemName)
					setInfo.Items = append(setInfo.Items, itemName)
				}
			}

			// Process bonuses
			bonuses := []struct{ spell, threshold int }{
				{spell1, bonus1}, {spell2, bonus2}, {spell3, bonus3}, {spell4, bonus4},
				{spell5, bonus5}, {spell6, bonus6}, {spell7, bonus7}, {spell8, bonus8},
			}
			// Sort bonuses by threshold (asc)
			sort.Slice(bonuses, func(i, j int) bool {
				return bonuses[i].threshold < bonuses[j].threshold
			})

			for _, b := range bonuses {
				if b.spell > 0 && b.threshold > 0 {
					description := r.resolveSpellText(b.spell)
					if description != "" {
						setInfo.Bonuses = append(setInfo.Bonuses, fmt.Sprintf("(%d) Set: %s", b.threshold, description))
					}
				}
			}

			tooltip.SetInfo = &setInfo
		}
	}

	return tooltip, nil
}

// SpellData holds all spell-related data for variable replacement
type SpellData struct {
	BasePoints         [3]int
	DieSides           [3]int
	Amplitude          [3]int
	ChainTarget        [3]int
	MiscValue          [3]int
	RadiusIndex        [3]int
	ProcChance         int
	ProcCharges        int
	DurationIndex      int
	RangeID            int
	DmgMultiplier1     float64
	MaxAffectedTargets int
}

// resolveSpellText fetches and formats spell description with parameters
// Implements complete WoW spell variable replacement system
func (r *ItemRepository) resolveSpellText(spellID int) string {
	var name, description string
	var data SpellData

	// Query all needed spell data
	err := r.db.QueryRow(`
		SELECT 
			COALESCE(name, ''), COALESCE(description, ''),
			effectBasePoints1, effectBasePoints2, effectBasePoints3,
			effectDieSides1, effectDieSides2, effectDieSides3,
			effectAmplitude1, effectAmplitude2, effectAmplitude3,
			effectChainTarget1, effectChainTarget2, effectChainTarget3,
			effectMiscValue1, effectMiscValue2, effectMiscValue3,
			effectRadiusIndex1, effectRadiusIndex2, effectRadiusIndex3,
			procChance, procCharges, durationIndex, rangeIndex,
			COALESCE(dmgMultiplier1, 0), COALESCE(maxAffectedTargets, 0)
		FROM spell_template WHERE entry = ?
	`, spellID).Scan(
		&name, &description,
		&data.BasePoints[0], &data.BasePoints[1], &data.BasePoints[2],
		&data.DieSides[0], &data.DieSides[1], &data.DieSides[2],
		&data.Amplitude[0], &data.Amplitude[1], &data.Amplitude[2],
		&data.ChainTarget[0], &data.ChainTarget[1], &data.ChainTarget[2],
		&data.MiscValue[0], &data.MiscValue[1], &data.MiscValue[2],
		&data.RadiusIndex[0], &data.RadiusIndex[1], &data.RadiusIndex[2],
		&data.ProcChance, &data.ProcCharges, &data.DurationIndex, &data.RangeID,
		&data.DmgMultiplier1, &data.MaxAffectedTargets,
	)

	if err != nil {
		return ""
	}

	// Use description if available, otherwise use name
	text := description
	if text == "" {
		text = name
	}
	if text == "" {
		return ""
	}

	// Replace all variable types
	text = r.replaceSpellVariables(text, spellID, &data)

	return text
}

// replaceSpellVariables replaces all WoW spell variables in text
func (r *ItemRepository) replaceSpellVariables(text string, spellID int, data *SpellData) string {
	// Calculate values (base + 1, or base + diesides for ranges)
	v := [3]int{}
	for i := 0; i < 3; i++ {
		if data.DieSides[i] > 1 {
			v[i] = data.BasePoints[i] + data.DieSides[i]
		} else {
			v[i] = data.BasePoints[i] + 1
		}
	}

	// Get duration text
	durationText := r.getSpellDuration(data.DurationIndex)

	// --- Math Expression Handling (e.g. $/1000;s1) ---
	reMath := regexp.MustCompile(`\$([/*+-])([\d\.]+);([a-z]\d?)`)
	if reMath.MatchString(text) {
		// Create variable map for lookups
		vars := make(map[string]float64)
		vars["s1"] = float64(v[0])
		vars["s2"] = float64(v[1])
		vars["s3"] = float64(v[2])
		vars["s"] = float64(v[0])
		vars["m1"] = float64(v[0])
		vars["m2"] = float64(v[1])
		vars["m3"] = float64(v[2])

		// Map other variables if needed (t - amplitude)
		for i := 0; i < 3; i++ {
			if data.Amplitude[i] > 0 {
				ticks := float64(data.Amplitude[i] / 1000)
				vars[fmt.Sprintf("t%d", i+1)] = ticks
			}
		}

		text = reMath.ReplaceAllStringFunc(text, func(match string) string {
			parts := reMath.FindStringSubmatch(match)
			op := parts[1]
			valStr := parts[2]
			varName := parts[3]

			operand, err := strconv.ParseFloat(valStr, 64)
			if err != nil {
				return match
			}

			varValue, ok := vars[varName]
			if !ok {
				return match
			}

			var result float64
			switch op {
			case "/":
				if operand != 0 {
					result = varValue / operand
				}
			case "*":
				result = varValue * operand
			case "+":
				result = varValue + operand
			case "-":
				result = varValue - operand
			}

			// Format: if integer, no decimals; otherwise up to 2 decimals
			if result == math.Trunc(result) {
				return fmt.Sprintf("%.0f", result)
			}
			return fmt.Sprintf("%.2f", result)
		})
	}

	// Simple variable replacements (no cross-spell references)
	// $s1, $s2, $s3 - spell values
	text = strings.ReplaceAll(text, "$s1", fmt.Sprintf("%d", v[0]))
	text = strings.ReplaceAll(text, "$s2", fmt.Sprintf("%d", v[1]))
	text = strings.ReplaceAll(text, "$s3", fmt.Sprintf("%d", v[2]))
	text = strings.ReplaceAll(text, "$s", fmt.Sprintf("%d", v[0])) // $s = $s1

	// $o1, $o2, $o3 - over-time values
	text = r.replaceOvertimeValues(text, data)

	// $d - duration
	text = strings.ReplaceAll(text, "$d", durationText)

	// $h - proc chance
	text = strings.ReplaceAll(text, "$h", fmt.Sprintf("%d", data.ProcChance))

	// $n - proc charges
	text = strings.ReplaceAll(text, "$n", fmt.Sprintf("%d", data.ProcCharges))

	// $i - max affected targets
	if data.MaxAffectedTargets > 0 {
		text = strings.ReplaceAll(text, "$i", fmt.Sprintf("%d", data.MaxAffectedTargets))
	}

	// $t1, $t2, $t3 - ticks/amplitude
	for i := 0; i < 3; i++ {
		if data.Amplitude[i] > 0 {
			ticks := data.Amplitude[i] / 1000
			text = strings.ReplaceAll(text, fmt.Sprintf("$t%d", i+1), fmt.Sprintf("%d", ticks))
		}
	}

	// $x1, $x2, $x3 - chain targets
	for i := 0; i < 3; i++ {
		text = strings.ReplaceAll(text, fmt.Sprintf("$x%d", i+1), fmt.Sprintf("%d", data.ChainTarget[i]))
	}

	// $q1, $q2, $q3 and $u1, $u2, $u3 - misc values
	for i := 0; i < 3; i++ {
		text = strings.ReplaceAll(text, fmt.Sprintf("$q%d", i+1), fmt.Sprintf("%d", data.MiscValue[i]))
		text = strings.ReplaceAll(text, fmt.Sprintf("$u%d", i+1), fmt.Sprintf("%d", data.MiscValue[i]))
	}
	text = strings.ReplaceAll(text, "$q", fmt.Sprintf("%d", data.MiscValue[0]))
	text = strings.ReplaceAll(text, "$u", fmt.Sprintf("%d", data.MiscValue[0]))

	// $m1, $m2, $m3 - multiplier/max values (using base points as fallback)
	for i := 0; i < 3; i++ {
		text = strings.ReplaceAll(text, fmt.Sprintf("$m%d", i+1), fmt.Sprintf("%d", v[i]))
	}

	// $a1, $a2, $a3 - area/radius
	for i := 0; i < 3; i++ {
		if data.RadiusIndex[i] > 0 {
			var radius int
			r.db.QueryRow("SELECT radiusBase FROM spell_radius WHERE id = ?", data.RadiusIndex[i]).Scan(&radius)
			text = strings.ReplaceAll(text, fmt.Sprintf("$a%d", i+1), fmt.Sprintf("%d", radius))
		}
	}

	// $r - range
	if data.RangeID > 0 {
		var rangeMax int
		r.db.QueryRow("SELECT rangeMax FROM spell_range WHERE id = ?", data.RangeID).Scan(&rangeMax)
		text = strings.ReplaceAll(text, "$r", fmt.Sprintf("%d", rangeMax))
	}

	// $f1 - damage multiplier
	if data.DmgMultiplier1 > 0 {
		text = strings.ReplaceAll(text, "$f1", fmt.Sprintf("%.1f", data.DmgMultiplier1))
	}

	// Handle ${} bracket format for all variables
	text = r.replaceBracketVariables(text, v, data, durationText)

	// Handle cross-spell references (must be last)
	text = r.replaceCrossSpellReferences(text)

	// Handle $l variables for pluralization (e.g., $leffect:effects;)
	// This chooses singular or plural form based on preceding number
	text = r.replacePluralVariables(text)

	return text
}

// replaceOvertimeValues calculates and replaces $o values based on duration and amplitude
func (r *ItemRepository) replaceOvertimeValues(text string, data *SpellData) string {
	// Get duration in milliseconds
	var durationBase int
	if data.DurationIndex > 0 {
		r.db.QueryRow("SELECT duration_base FROM spell_durations WHERE id = ?", data.DurationIndex).Scan(&durationBase)
	}
	if durationBase < 0 {
		durationBase = -durationBase
	}

	for i := 0; i < 3; i++ {
		baseVal := data.BasePoints[i] + 1

		// Calculate total over-time value
		var otValue int
		if data.Amplitude[i] > 0 && durationBase > 0 {
			// ticks = duration / amplitude
			ticks := durationBase / data.Amplitude[i]
			// total = base_value * ticks
			otValue = baseVal * ticks
		} else {
			otValue = baseVal
		}

		text = strings.ReplaceAll(text, fmt.Sprintf("$o%d", i+1), fmt.Sprintf("%d", otValue))
	}

	return text
}

// replaceBracketVariables handles ${variable} format
func (r *ItemRepository) replaceBracketVariables(text string, v [3]int, data *SpellData, durationText string) string {
	text = strings.ReplaceAll(text, "${s1}", fmt.Sprintf("%d", v[0]))
	text = strings.ReplaceAll(text, "${s2}", fmt.Sprintf("%d", v[1]))
	text = strings.ReplaceAll(text, "${s3}", fmt.Sprintf("%d", v[2]))
	text = strings.ReplaceAll(text, "${d}", durationText)
	text = strings.ReplaceAll(text, "${h}", fmt.Sprintf("%d", data.ProcChance))
	text = strings.ReplaceAll(text, "${n}", fmt.Sprintf("%d", data.ProcCharges))
	return text
}

// replaceCrossSpellReferences handles $XXXXXd, $XXXXXs1, etc.
func (r *ItemRepository) replaceCrossSpellReferences(text string) string {
	// $XXXXXd - duration of spell XXXXX
	re := regexp.MustCompile(`\$(\d+)d`)
	text = re.ReplaceAllStringFunc(text, func(match string) string {
		spellIDStr := match[1 : len(match)-1]
		refSpellID, err := strconv.Atoi(spellIDStr)
		if err != nil {
			return match
		}

		var refDurIndex int
		err = r.db.QueryRow("SELECT durationIndex FROM spell_template WHERE entry = ?", refSpellID).Scan(&refDurIndex)
		if err != nil || refDurIndex == 0 {
			return match
		}

		return r.getSpellDuration(refDurIndex)
	})

	// $XXXXXs1, $XXXXXs2, $XXXXXs3 - spell values from other spells
	re = regexp.MustCompile(`\$(\d+)s(\d)`)
	text = re.ReplaceAllStringFunc(text, func(match string) string {
		parts := re.FindStringSubmatch(match)
		if len(parts) < 3 {
			return match
		}

		refSpellID, _ := strconv.Atoi(parts[1])
		effectNum, _ := strconv.Atoi(parts[2])
		if effectNum < 1 || effectNum > 3 {
			return match
		}

		var basePoints, dieSides int
		query := fmt.Sprintf("SELECT effectBasePoints%d, effectDieSides%d FROM spell_template WHERE entry = ?", effectNum, effectNum)
		err := r.db.QueryRow(query, refSpellID).Scan(&basePoints, &dieSides)
		if err != nil {
			return match
		}

		value := basePoints + 1
		if dieSides > 1 {
			value = basePoints + dieSides
		}

		return fmt.Sprintf("%d", value)
	})

	// $XXXXXo1, $XXXXXo2, $XXXXXo3 - over-time values from other spells
	re = regexp.MustCompile(`\$(\d+)o(\d)`)
	text = re.ReplaceAllStringFunc(text, func(match string) string {
		parts := re.FindStringSubmatch(match)
		if len(parts) < 3 {
			return match
		}

		refSpellID, _ := strconv.Atoi(parts[1])
		effectNum, _ := strconv.Atoi(parts[2])
		if effectNum < 1 || effectNum > 3 {
			return match
		}

		var basePoints int
		query := fmt.Sprintf("SELECT effectBasePoints%d FROM spell_template WHERE entry = ?", effectNum)
		r.db.QueryRow(query, refSpellID).Scan(&basePoints)

		return fmt.Sprintf("%d", basePoints+1)
	})

	return text
}

// replacePluralVariables handles $lsingular:plural; format
// Example: "Removes 1 poison $leffect:effects;" becomes "Removes 1 poison effect"
func (r *ItemRepository) replacePluralVariables(text string) string {
	pluralRegex := regexp.MustCompile(`\$l([^:]+):([^;]+);`)
	matches := pluralRegex.FindAllStringSubmatchIndex(text, -1)

	if len(matches) == 0 {
		return text
	}

	var sb strings.Builder
	lastIndex := 0

	for _, match := range matches {
		// match[0]-match[1] is the full match range
		// match[2]-match[3] is singular
		// match[4]-match[5] is plural

		start := match[0]
		end := match[1]

		// Append content before match
		sb.WriteString(text[lastIndex:start])

		// Look back for number in text[0:start]
		// We want the *last* number before this match
		preceding := text[0:start]

		// Find all numbers
		numRegex := regexp.MustCompile(`(\d+)`)
		numMatches := numRegex.FindAllStringSubmatch(preceding, -1)

		count := 0
		if len(numMatches) > 0 {
			// Take the last one
			lastNumStr := numMatches[len(numMatches)-1][1]
			fmt.Sscanf(lastNumStr, "%d", &count)
		}

		singular := text[match[2]:match[3]]
		plural := text[match[4]:match[5]]

		if count == 1 {
			sb.WriteString(singular)
		} else {
			sb.WriteString(plural)
		}

		lastIndex = end
	}

	sb.WriteString(text[lastIndex:])
	return sb.String()
}

// getSpellDuration returns formatted duration text
func (r *ItemRepository) getSpellDuration(durationIndex int) string {
	if durationIndex == 0 {
		return "duration"
	}

	var durationBase int
	r.db.QueryRow("SELECT duration_base FROM spell_durations WHERE id = ?", durationIndex).Scan(&durationBase)
	if durationBase <= 0 {
		return "duration"
	}

	if durationBase < 0 {
		durationBase = -durationBase
	}

	seconds := durationBase / 1000
	if seconds < 60 {
		return fmt.Sprintf("%d sec", seconds)
	} else if seconds < 3600 {
		return fmt.Sprintf("%d min", seconds/60)
	} else {
		return fmt.Sprintf("%d hr", seconds/3600)
	}
}

// formatSpellEffect returns a formatted spell effect string with trigger prefix
func (r *ItemRepository) formatSpellEffect(spellID, trigger, itemCooldownMs int) string {
	text := r.resolveSpellText(spellID)
	if text == "" {
		return ""
	}

	// Format based on trigger type
	var prefix string
	isUse := false
	switch trigger {
	case 0: // Use
		prefix = "Use:"
		isUse = true
	case 1: // On Equip
		prefix = "Equip:"
	case 2: // Chance on Hit
		prefix = "Chance on hit:"
	case 4: // Soulstone
		prefix = "Use:"
		isUse = true
	case 5: // Use with no delay
		prefix = "Use:"
		isUse = true
	case 6: // Learn spell
		prefix = "Use:"
	default:
		prefix = "Equip:"
	}

	result := fmt.Sprintf("%s %s", prefix, text)

	// Cooldown on Use effects: prefer the per-item cooldown, else the spell's own
	// recovery. Ignore the 1-sec global item-use throttle.
	if isUse {
		cd := itemCooldownMs
		if cd <= 0 {
			r.db.QueryRow("SELECT recoveryTime FROM spell_template WHERE entry = ?", spellID).Scan(&cd)
		}
		if cd > 1000 {
			result += " (" + formatCooldown(cd) + " cooldown)"
		}
	}

	return result
}

// formatCooldown renders a millisecond cooldown as "30 sec" / "2 min" / "1 hr".
func formatCooldown(ms int) string {
	switch {
	case ms >= 3600000:
		return fmt.Sprintf("%g hr", float64(ms)/3600000.0)
	case ms >= 60000:
		return fmt.Sprintf("%g min", float64(ms)/60000.0)
	default:
		return fmt.Sprintf("%g sec", float64(ms)/1000.0)
	}
}

// GetItemDetail returns full item information with drop sources
func (r *ItemRepository) GetItemDetail(entry int) (*models.ItemDetail, error) {
	item, err := r.GetItemByID(entry)
	if err != nil {
		return nil, err
	}

	detail := &models.ItemDetail{Item: item}

	// Get dropped by creatures (including reference loot)
	// Note: We assume c.loot_id matches creature_loot_template.entry.
	rows, err := r.db.Query(`
		SELECT c.entry, c.name, c.level_min, c.level_max, cl.ChanceOrQuestChance
		FROM creature_loot_template cl
		JOIN creature_template c ON cl.entry = c.loot_id
		WHERE cl.item = ?
		
		UNION
		
		SELECT c.entry, c.name, c.level_min, c.level_max, cl.ChanceOrQuestChance
		FROM reference_loot_template rl
		JOIN creature_loot_template cl ON cl.mincountOrRef = -rl.entry
		JOIN creature_template c ON cl.entry = c.loot_id
		WHERE rl.item = ?

		ORDER BY ChanceOrQuestChance DESC
		LIMIT 50
	`, entry, entry)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			drop := &models.CreatureDrop{}
			rows.Scan(&drop.Entry, &drop.Name, &drop.LevelMin, &drop.LevelMax, &drop.Chance)
			detail.DroppedBy = append(detail.DroppedBy, drop)
		}
	}

	// Get quest rewards
	rows2, err := r.db.Query(`
		SELECT entry, Title, QuestLevel, 0 as is_choice
		FROM quest_template
		WHERE RewItemId1 = ? OR RewItemId2 = ? OR RewItemId3 = ? OR RewItemId4 = ?
		UNION
		SELECT entry, Title, QuestLevel, 1 as is_choice
		FROM quest_template
		WHERE RewChoiceItemId1 = ? OR RewChoiceItemId2 = ? OR RewChoiceItemId3 = ? 
		   OR RewChoiceItemId4 = ? OR RewChoiceItemId5 = ? OR RewChoiceItemId6 = ?
		LIMIT 20
	`, entry, entry, entry, entry, entry, entry, entry, entry, entry, entry)
	if err == nil {
		defer rows2.Close()
		for rows2.Next() {
			reward := &models.QuestReward{}
			var isChoice int
			rows2.Scan(&reward.Entry, &reward.Title, &reward.Level, &isChoice)
			reward.IsChoice = isChoice == 1
			detail.RewardFrom = append(detail.RewardFrom, reward)
		}
	}

	// Get contains (if item is a container)
	rows3, err := r.db.Query(`
		SELECT i.entry, i.name, i.quality, COALESCE(idi.icon, ''), il.ChanceOrQuestChance, il.mincountOrRef, il.maxcount
		FROM item_loot_template il
		JOIN item_template i ON il.item = i.entry
		LEFT JOIN item_display_info idi ON i.display_id = idi.ID
		WHERE il.entry = ?
		ORDER BY il.ChanceOrQuestChance DESC
	`, entry)
	if err == nil {
		defer rows3.Close()
		for rows3.Next() {
			drop := &models.ItemDrop{}
			rows3.Scan(&drop.Entry, &drop.Name, &drop.Quality, &drop.IconPath, &drop.Chance, &drop.MinCount, &drop.MaxCount)
			detail.Contains = append(detail.Contains, drop)
		}
	}

	// Get vendors that sell this item ("sold by"). Prefer the live
	// creature_template name/level, falling back to what we scraped.
	rows4, err := r.db.Query(`
		SELECT iv.npc_entry,
		       COALESCE(NULLIF(c.name, ''), iv.npc_name),
		       COALESCE(NULLIF(c.level_min, 0), iv.level_min),
		       COALESCE(NULLIF(c.level_max, 0), iv.level_max),
		       iv.cost, iv.stock
		FROM item_vendor iv
		LEFT JOIN creature_template c ON iv.npc_entry = c.entry
		WHERE iv.item_entry = ?
		ORDER BY iv.npc_name
	`, entry)
	if err == nil {
		defer rows4.Close()
		for rows4.Next() {
			v := &models.ItemVendor{}
			rows4.Scan(&v.Entry, &v.Name, &v.LevelMin, &v.LevelMax, &v.Cost, &v.Stock)
			detail.SoldBy = append(detail.SoldBy, v)
		}
	}

	// Get crafting recipes that create this item ("Created By").
	detail.CreatedBy = r.getCreatedBy(entry)

	// Get containers (gameobject chests + container items) whose loot holds this
	// item — the reverse of Contains.
	detail.ContainedIn, detail.GatheredFrom = r.getContainedIn(entry)

	// Quests this item is an objective of (a required turn-in item).
	objRows, err := r.db.Query(`
		SELECT entry, Title, QuestLevel
		FROM quest_template
		WHERE ReqItemId1 = ? OR ReqItemId2 = ? OR ReqItemId3 = ? OR ReqItemId4 = ?
		LIMIT 20
	`, entry, entry, entry, entry)
	if err == nil {
		defer objRows.Close()
		for objRows.Next() {
			q := &models.QuestReward{}
			if objRows.Scan(&q.Entry, &q.Title, &q.Level) == nil {
				detail.ObjectiveOf = append(detail.ObjectiveOf, q)
			}
		}
	}

	// The quest this item starts (item_template.start_quest), if any.
	var startQuestID int
	r.db.QueryRow("SELECT start_quest FROM item_template WHERE entry = ?", entry).Scan(&startQuestID)
	if startQuestID > 0 {
		sq := &models.QuestReward{}
		if r.db.QueryRow("SELECT entry, Title, QuestLevel FROM quest_template WHERE entry = ?", startQuestID).
			Scan(&sq.Entry, &sq.Title, &sq.Level) == nil {
			detail.StartsQuest = sq
		}
	}

	return detail, nil
}

// loadGatheringSkillNames returns the lowercased names of every Profession (11)
// or Secondary (9) skill from the client's own skill taxonomy. A lock requiring
// one of these (Herbalism, Mining, Survival, Fishing, …) marks a gathering node
// ("Gathered From"); a Class Skill like Lockpicking (category 7) or any lock with
// no matching skill is a chest ("Contained In"). This replaces a hardcoded skill
// list — new gathering/secondary skills classify automatically.
func loadGatheringSkillNames(db interface {
	Query(string, ...interface{}) (*sql.Rows, error)
}) map[string]bool {
	out := map[string]bool{}
	// 11 = Professions, 9 = Secondary Skills (the gathering/secondary categories).
	rows, err := db.Query("SELECT LOWER(name) FROM spell_skills WHERE category_id IN (9, 11)")
	if err != nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		if rows.Scan(&name) == nil && name != "" {
			out[name] = true
		}
	}
	return out
}

// lockSkillRequirement resolves a lock's required skill to (name, level) from
// lock_types, picking the first skill slot (type 2) that actually requires a
// level (req > 0) — which excludes the req-0 "Open"/"Treasure" auto-open
// mechanics but includes Lockpicking, Herbalism, Mining, Survival, Disarm Trap,
// etc. Returns ("", 0) when the lock has no skill requirement.
func lockSkillRequirement(db interface {
	QueryRow(string, ...interface{}) *sql.Row
}, lockID int, lockNames map[int]string) (skill string, level int) {
	if lockID <= 0 {
		return "", 0
	}
	var typ, prop, req [5]int
	err := db.QueryRow(`
		SELECT type1, type2, type3, type4, type5,
		       prop1, prop2, prop3, prop4, prop5,
		       req1, req2, req3, req4, req5
		FROM locks WHERE id = ?`, lockID).Scan(
		&typ[0], &typ[1], &typ[2], &typ[3], &typ[4],
		&prop[0], &prop[1], &prop[2], &prop[3], &prop[4],
		&req[0], &req[1], &req[2], &req[3], &req[4])
	if err != nil {
		return "", 0
	}
	for i := 0; i < 5; i++ {
		if typ[i] == 2 && req[i] > 0 {
			return lockNames[prop[i]], req[i]
		}
	}
	return "", 0
}

// loadLockTypeNames returns LockType id -> name from lock_types (small table).
func loadLockTypeNames(db interface {
	Query(string, ...interface{}) (*sql.Rows, error)
}) map[int]string {
	out := map[int]string{}
	rows, err := db.Query("SELECT id, name FROM lock_types")
	if err != nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var id int
		var name string
		if rows.Scan(&id, &name) == nil {
			out[id] = name
		}
	}
	return out
}

// getContainedIn returns the gameobjects and container items whose loot includes
// this item (the reverse of Contains), split into two lists: gathering nodes
// (herb/ore/fishing objects, by their Lock skill) and everything else (chests +
// container items).
func (r *ItemRepository) getContainedIn(entry int) (contained, gathered []*models.ItemContainer) {
	lockNames := loadLockTypeNames(r.db)
	gatherSkill := loadGatheringSkillNames(r.db)

	// Gameobject chests/nodes (type 3): data1 is the loot template entry, data0
	// the Lock id. The lock's required skill (name from lock_types, level from req)
	// is shown on every entry; a gathering/secondary skill routes the object to
	// "Gathered From", anything else (Lockpicking, Disarm Trap, no lock) stays
	// "Contained In".
	goRows, err := r.db.Query(`
		SELECT DISTINCT o.entry, o.name, gl.ChanceOrQuestChance,
		       l.type1, l.type2, l.type3, l.type4, l.type5,
		       l.prop1, l.prop2, l.prop3, l.prop4, l.prop5,
		       l.req1, l.req2, l.req3, l.req4, l.req5
		FROM gameobject_loot_template gl
		JOIN gameobject_template o ON o.data1 = gl.entry
		LEFT JOIN locks l ON l.id = o.data0
		WHERE gl.item = ? AND o.type = 3
		ORDER BY gl.ChanceOrQuestChance DESC
		LIMIT 50
	`, entry)
	if err == nil {
		for goRows.Next() {
			c := &models.ItemContainer{Kind: "object"}
			var typ, prop, req [5]int
			if goRows.Scan(&c.Entry, &c.Name, &c.Chance,
				&typ[0], &typ[1], &typ[2], &typ[3], &typ[4],
				&prop[0], &prop[1], &prop[2], &prop[3], &prop[4],
				&req[0], &req[1], &req[2], &req[3], &req[4]) != nil {
				continue
			}
			// First skill slot (type 2) that requires a level (req > 0): its
			// LockType name + level. Excludes req-0 auto-open mechanics; includes
			// Lockpicking, Herbalism, Mining, Survival, Disarm Trap, etc.
			for i := 0; i < 5; i++ {
				if typ[i] == 2 && req[i] > 0 {
					c.Skill = lockNames[prop[i]]
					c.SkillReq = req[i]
					break
				}
			}
			if c.Skill != "" && gatherSkill[strings.ToLower(c.Skill)] {
				gathered = append(gathered, c)
			} else {
				contained = append(contained, c)
			}
		}
		goRows.Close()
	}

	// Container items: item_loot_template.entry is the container item's entry.
	itRows, err := r.db.Query(`
		SELECT i.entry, i.name, i.quality, COALESCE(idi.icon, ''), il.ChanceOrQuestChance
		FROM item_loot_template il
		JOIN item_template i ON il.entry = i.entry
		LEFT JOIN item_display_info idi ON i.display_id = idi.ID
		WHERE il.item = ?
		ORDER BY il.ChanceOrQuestChance DESC
		LIMIT 50
	`, entry)
	if err == nil {
		for itRows.Next() {
			c := &models.ItemContainer{Kind: "item"}
			if itRows.Scan(&c.Entry, &c.Name, &c.Quality, &c.IconPath, &c.Chance) == nil {
				contained = append(contained, c)
			}
		}
		itRows.Close()
	}

	return contained, gathered
}

// getCreatedBy returns the tradeskill spells whose Create Item effect (effect
// type 24) produces this item, with the produced count, profession requirement,
// and reagents. Empty for items that aren't crafted.
func (r *ItemRepository) getCreatedBy(entry int) []*models.ItemCraftSource {
	const createItemEffect = 24 // SPELL_EFFECT_CREATE_ITEM
	rows, err := r.db.Query(`
		SELECT st.entry, st.name, COALESCE(NULLIF(si.icon_name, ''), st.iconName, ''),
		       st.effect1, st.effect2, st.effect3,
		       st.effectItemType1, st.effectItemType2, st.effectItemType3,
		       st.effectBasePoints1, st.effectBasePoints2, st.effectBasePoints3,
		       st.reagent1, st.reagent2, st.reagent3, st.reagent4,
		       st.reagent5, st.reagent6, st.reagent7, st.reagent8,
		       st.reagentCount1, st.reagentCount2, st.reagentCount3, st.reagentCount4,
		       st.reagentCount5, st.reagentCount6, st.reagentCount7, st.reagentCount8
		FROM spell_template st
		LEFT JOIN spell_icons si ON st.spellIconId = si.id
		WHERE (st.effect1 = ? AND st.effectItemType1 = ?)
		   OR (st.effect2 = ? AND st.effectItemType2 = ?)
		   OR (st.effect3 = ? AND st.effectItemType3 = ?)
	`, createItemEffect, entry, createItemEffect, entry, createItemEffect, entry)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var sources []*models.ItemCraftSource
	for rows.Next() {
		var spellID int
		var name, icon string
		var eff [3]int
		var effItem [3]int
		var effBase [3]int
		var reagent [8]int
		var reagentCount [8]int
		if rows.Scan(&spellID, &name, &icon,
			&eff[0], &eff[1], &eff[2],
			&effItem[0], &effItem[1], &effItem[2],
			&effBase[0], &effBase[1], &effBase[2],
			&reagent[0], &reagent[1], &reagent[2], &reagent[3],
			&reagent[4], &reagent[5], &reagent[6], &reagent[7],
			&reagentCount[0], &reagentCount[1], &reagentCount[2], &reagentCount[3],
			&reagentCount[4], &reagentCount[5], &reagentCount[6], &reagentCount[7]) != nil {
			continue
		}

		src := &models.ItemCraftSource{SpellID: spellID, SpellName: name, SpellIcon: icon, ProducedCount: 1}
		// The Create Item effect slot for this item gives the produced count
		// (basePoints is amount-1, so +1).
		for i := 0; i < 3; i++ {
			if eff[i] == createItemEffect && effItem[i] == entry {
				if c := effBase[i] + 1; c > 0 {
					src.ProducedCount = c
				}
				break
			}
		}

		for i := 0; i < 8; i++ {
			if reagent[i] > 0 {
				src.Reagents = append(src.Reagents, &models.CraftReagent{Entry: reagent[i], Count: reagentCount[i]})
			}
		}
		r.fillReagentInfo(src.Reagents)
		src.SkillName, src.ReqSkill = r.craftSkillRequirement(spellID)

		sources = append(sources, src)
	}
	return sources
}

// craftSkillRequirement returns the profession name and required skill rank for
// a crafting spell. The authoritative requirement lives on the recipe item
// (pattern/plans/...) that teaches the craft via a learn-spell (effect 36): its
// required_skill/required_skill_rank. Trainer-taught crafts have no recipe item,
// so we fall back to the SkillLineAbility min rank for the skill name (and rank
// where it's meaningful).
func (r *ItemRepository) craftSkillRequirement(spellID int) (string, int) {
	const learnSpellEffect = 36 // SPELL_EFFECT_LEARN_SPELL

	var skillID, rank int
	err := r.db.QueryRow(`
		SELECT it.required_skill, it.required_skill_rank
		FROM spell_template ls
		JOIN item_template it ON it.spellid_1 = ls.entry OR it.spellid_2 = ls.entry
			OR it.spellid_3 = ls.entry OR it.spellid_4 = ls.entry OR it.spellid_5 = ls.entry
		WHERE (ls.effect1 = ? AND ls.effectTriggerSpell1 = ?)
		   OR (ls.effect2 = ? AND ls.effectTriggerSpell2 = ?)
		   OR (ls.effect3 = ? AND ls.effectTriggerSpell3 = ?)
		ORDER BY it.required_skill_rank DESC
		LIMIT 1
	`, learnSpellEffect, spellID, learnSpellEffect, spellID, learnSpellEffect, spellID).Scan(&skillID, &rank)
	if err == nil && skillID > 0 {
		var name string
		r.db.QueryRow("SELECT name FROM spell_skills WHERE id = ?", skillID).Scan(&name)
		return name, rank
	}

	// Fallback: trainer-taught crafts — skill line + min rank from SkillLineAbility.
	var name string
	var req int
	r.db.QueryRow(`
		SELECT ss.name, sss.req_skill_value
		FROM spell_skill_spells sss
		JOIN spell_skills ss ON sss.skill_id = ss.id
		WHERE sss.spell_id = ?
		ORDER BY sss.req_skill_value DESC
		LIMIT 1
	`, spellID).Scan(&name, &req)
	return name, req
}

// fillReagentInfo resolves each reagent's name/quality/icon from item_template.
func (r *ItemRepository) fillReagentInfo(reagents []*models.CraftReagent) {
	for _, rg := range reagents {
		r.db.QueryRow(`
			SELECT i.name, i.quality, COALESCE(idi.icon, '')
			FROM item_template i
			LEFT JOIN item_display_info idi ON i.display_id = idi.ID
			WHERE i.entry = ?
		`, rg.Entry).Scan(&rg.Name, &rg.Quality, &rg.IconPath)
	}
}

// formatStat returns a formatted stat string. The display name comes from the
// stat_types table (base stats localized from the client); it falls back to the
// built-in canonical names, then to a placeholder for unknown ids.
func (r *ItemRepository) formatStat(statType, value int) string {
	var name string
	r.db.QueryRow("SELECT name FROM stat_types WHERE id = ?", statType).Scan(&name)
	if name == "" {
		name = helpers.GetStatName(statType)
	}
	if name == "" {
		name = fmt.Sprintf("Unknown Stat %d", statType)
	}
	if value > 0 {
		return fmt.Sprintf("+%d %s", value, name)
	}
	return fmt.Sprintf("%d %s", value, name)
}
