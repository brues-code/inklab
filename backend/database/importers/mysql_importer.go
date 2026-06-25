package importers

import (
	"database/sql"
	"fmt"
	"log"
)

// MySQLImporter handles importing data directly from MySQL to SQLite
type MySQLImporter struct {
	sqliteDB *sql.DB
	mysqlDB  *sql.DB
}

// NewMySQLImporter creates a new MySQL importer
func NewMySQLImporter(sqliteDB *sql.DB, mysqlDB *sql.DB) *MySQLImporter {
	return &MySQLImporter{
		sqliteDB: sqliteDB,
		mysqlDB:  mysqlDB,
	}
}

// ImportTable copies a table from MySQL to SQLite
// querySelect: MySQL SELECT query
// queryInsert: SQLite INSERT query
// batchSize: number of rows to commit at a time
func (i *MySQLImporter) ImportTable(tableName, querySelect, queryInsert string, batchSize int) error {
	if i.mysqlDB == nil {
		return fmt.Errorf("mysql connection is nil")
	}

	log.Printf("📥 Importing %s from MySQL...", tableName)

	// 1. Query MySQL
	rows, err := i.mysqlDB.Query(querySelect)
	if err != nil {
		return fmt.Errorf("failed to query mysql table %s: %w", tableName, err)
	}
	defer rows.Close()

	// Get column information to create efficient scanners
	cols, err := rows.Columns()
	if err != nil {
		return err
	}
	colCount := len(cols)

	// 2. Prepare SQLite Transaction
	tx, err := i.sqliteDB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(queryInsert)
	if err != nil {
		return fmt.Errorf("failed to prepare insert for %s: %w", tableName, err)
	}
	defer stmt.Close()

	// 3. Iterate and Insert
	count := 0
	values := make([]interface{}, colCount)
	valuePtrs := make([]interface{}, colCount)
	for i := range values {
		valuePtrs[i] = &values[i]
	}

	for rows.Next() {
		if err := rows.Scan(valuePtrs...); err != nil {
			return fmt.Errorf("failed to scan row from %s: %w", tableName, err)
		}

		// The MySQL driver returns TEXT/VARCHAR columns as []byte. Inserting a
		// []byte into SQLite stores it with BLOB affinity, and a BLOB never
		// compares equal to a TEXT literal — silently breaking name lookups and
		// LIKE search. Convert to string so the column keeps TEXT affinity.
		for j, v := range values {
			if b, ok := v.([]byte); ok {
				values[j] = string(b)
			}
		}

		if _, err := stmt.Exec(values...); err != nil {
			return fmt.Errorf("failed to insert into %s: %w", tableName, err)
		}

		count++
		if count%batchSize == 0 {
			if err := tx.Commit(); err != nil {
				return fmt.Errorf("failed to commit batch for %s: %w", tableName, err)
			}
			// Start new transaction
			tx, err = i.sqliteDB.Begin()
			if err != nil {
				return err
			}
			stmt, err = tx.Prepare(queryInsert)
			if err != nil {
				return err
			}
		}

		if count%1000 == 0 {
			fmt.Printf("\r  Processing %s: %d rows...", tableName, count)
		}
	}

	// Final commit
	if err := tx.Commit(); err != nil {
		return err
	}
	stmt.Close() // Explicit close for the last one

	fmt.Printf("\r✓ Imported %d rows into %s            \n", count, tableName)
	return nil
}

// ImportAllFromMySQL imports all core data from MySQL
// Each table is checked individually - only empty tables are imported
func (i *MySQLImporter) ImportAllFromMySQL() error {
	// 1. Item Template
	// Note: We select explicit columns to match SQLite schema order
	itemCols := "entry,class,subclass,name,description,display_id,quality,flags,buy_count,buy_price,sell_price,inventory_type,allowable_class,allowable_race,item_level,required_level,required_skill,required_skill_rank,required_spell,required_honor_rank,required_city_rank,required_reputation_faction,required_reputation_rank,max_count,stackable,container_slots,stat_type1,stat_value1,stat_type2,stat_value2,stat_type3,stat_value3,stat_type4,stat_value4,stat_type5,stat_value5,stat_type6,stat_value6,stat_type7,stat_value7,stat_type8,stat_value8,stat_type9,stat_value9,stat_type10,stat_value10,delay,range_mod,ammo_type,dmg_min1,dmg_max1,dmg_type1,dmg_min2,dmg_max2,dmg_type2,dmg_min3,dmg_max3,dmg_type3,dmg_min4,dmg_max4,dmg_type4,dmg_min5,dmg_max5,dmg_type5,block,armor,holy_res,fire_res,nature_res,frost_res,shadow_res,arcane_res,spellid_1,spelltrigger_1,spellcharges_1,spellppmrate_1,spellcooldown_1,spellcategory_1,spellcategorycooldown_1,spellid_2,spelltrigger_2,spellcharges_2,spellppmrate_2,spellcooldown_2,spellcategory_2,spellcategorycooldown_2,spellid_3,spelltrigger_3,spellcharges_3,spellppmrate_3,spellcooldown_3,spellcategory_3,spellcategorycooldown_3,spellid_4,spelltrigger_4,spellcharges_4,spellppmrate_4,spellcooldown_4,spellcategory_4,spellcategorycooldown_4,spellid_5,spelltrigger_5,spellcharges_5,spellppmrate_5,spellcooldown_5,spellcategory_5,spellcategorycooldown_5,bonding,page_text,page_language,page_material,start_quest,lock_id,material,sheath,random_property,set_id,max_durability,area_bound,map_bound,duration,bag_family,disenchant_id,food_type,min_money_loot,max_money_loot,wrapped_gift,extra_flags,other_team_entry,script_name"

	if err := i.runImportIfEmpty("item_template", itemCols); err != nil {
		log.Printf("Warning: Failed to import item_template: %v", err)
	}

	// 2. Creature Template
	creatureCols := "entry,display_id1,display_id2,display_id3,display_id4,mount_display_id,name,subname,gossip_menu_id,level_min,level_max,health_min,health_max,mana_min,mana_max,armor,faction,npc_flags,speed_walk,speed_run,scale,detection_range,call_for_help_range,leash_range,`rank`,xp_multiplier,dmg_min,dmg_max,dmg_school,attack_power,dmg_multiplier,base_attack_time,ranged_attack_time,unit_class,unit_flags,dynamic_flags,beast_family,trainer_type,trainer_spell,trainer_class,trainer_race,ranged_dmg_min,ranged_dmg_max,ranged_attack_power,`type`,type_flags,loot_id,pickpocket_loot_id,skinning_loot_id,holy_res,fire_res,nature_res,frost_res,shadow_res,arcane_res,spell_id1,spell_id2,spell_id3,spell_id4,spell_list_id,pet_spell_list_id,spawn_spell_id,auras,gold_min,gold_max,ai_name,movement_type,inhabit_type,civilian,racial_leader,regeneration,equipment_id,trainer_id,vendor_id,mechanic_immune_mask,school_immune_mask,immunity_flags,flags_extra,phase_quest_id,script_name"
	if err := i.runImportIfEmpty("creature_template", creatureCols); err != nil {
		log.Printf("Warning: Failed to import creature_template: %v", err)
	}

	// 2b. Creature Equipment (maps equipment_id -> 3 equipped item entries, used
	// to render held weapons on NPC models).
	if err := i.runImportIfEmpty("creature_equip_template", "entry,equipentry1,equipentry2,equipentry3"); err != nil {
		log.Printf("Warning: Failed to import creature_equip_template: %v", err)
	}

	// 3. Quest Template
	questCols := "entry,Method,ZoneOrSort,MinLevel,MaxLevel,QuestLevel,`Type`,RequiredClasses,RequiredRaces,RequiredSkill,RequiredSkillValue,RequiredCondition,RepObjectiveFaction,RepObjectiveValue,RequiredMinRepFaction,RequiredMinRepValue,RequiredMaxRepFaction,RequiredMaxRepValue,SuggestedPlayers,LimitTime,QuestFlags,SpecialFlags,PrevQuestId,NextQuestId,ExclusiveGroup,NextQuestInChain,SrcItemId,SrcItemCount,SrcSpell,Title,Details,Objectives,OfferRewardText,RequestItemsText,EndText,ObjectiveText1,ObjectiveText2,ObjectiveText3,ObjectiveText4,ReqItemId1,ReqItemId2,ReqItemId3,ReqItemId4,ReqItemCount1,ReqItemCount2,ReqItemCount3,ReqItemCount4,ReqSourceId1,ReqSourceId2,ReqSourceId3,ReqSourceId4,ReqSourceCount1,ReqSourceCount2,ReqSourceCount3,ReqSourceCount4,ReqCreatureOrGOId1,ReqCreatureOrGOId2,ReqCreatureOrGOId3,ReqCreatureOrGOId4,ReqCreatureOrGOCount1,ReqCreatureOrGOCount2,ReqCreatureOrGOCount3,ReqCreatureOrGOCount4,ReqSpellCast1,ReqSpellCast2,ReqSpellCast3,ReqSpellCast4,RewChoiceItemId1,RewChoiceItemId2,RewChoiceItemId3,RewChoiceItemId4,RewChoiceItemId5,RewChoiceItemId6,RewChoiceItemCount1,RewChoiceItemCount2,RewChoiceItemCount3,RewChoiceItemCount4,RewChoiceItemCount5,RewChoiceItemCount6,RewItemId1,RewItemId2,RewItemId3,RewItemId4,RewItemCount1,RewItemCount2,RewItemCount3,RewItemCount4,RewRepFaction1,RewRepFaction2,RewRepFaction3,RewRepFaction4,RewRepFaction5,RewRepValue1,RewRepValue2,RewRepValue3,RewRepValue4,RewRepValue5,RewXP,RewOrReqMoney,RewMoneyMaxLevel,RewSpell,RewSpellCast,RewMailTemplateId,RewMailDelaySecs,RewMailMoney,PointMapId,PointX,PointY,PointOpt,DetailsEmote1,DetailsEmote2,DetailsEmote3,DetailsEmote4,DetailsEmoteDelay1,DetailsEmoteDelay2,DetailsEmoteDelay3,DetailsEmoteDelay4,IncompleteEmote,CompleteEmote,OfferRewardEmote1,OfferRewardEmote2,OfferRewardEmote3,OfferRewardEmote4,OfferRewardEmoteDelay1,OfferRewardEmoteDelay2,OfferRewardEmoteDelay3,OfferRewardEmoteDelay4,StartScript,CompleteScript"

	var questCount int
	i.sqliteDB.QueryRow("SELECT COUNT(*) FROM quest_template").Scan(&questCount)
	if questCount < 500 { // If mostly empty, re-import
		if err := i.runImport("quest_template", questCols); err != nil {
			log.Printf("Warning: Failed to import quest_template: %v", err)
		}
	} else {
		log.Printf("⏭️  quest_template already has %d rows, skipping", questCount)
	}

	// 4.1 Spell Icons (Aowow)Template
	// Note: Fully importing spell_template from MySQL as requested.
	// Identifying missing columns (like iconName) and letting them handle default values or updates later.
	spellCols := "entry,school,category,castUI,dispel,mechanic,attributes,attributesEx,attributesEx2,attributesEx3,attributesEx4,stances,stancesNot,targets,targetCreatureType,requiresSpellFocus,casterAuraState,targetAuraState,castingTimeIndex,recoveryTime,categoryRecoveryTime,interruptFlags,auraInterruptFlags,channelInterruptFlags,procFlags,procChance,procCharges,maxLevel,baseLevel,spellLevel,durationIndex,powerType,manaCost,manCostPerLevel,manaPerSecond,manaPerSecondPerLevel,rangeIndex,speed,modelNextSpell,stackAmount,totem1,totem2,reagent1,reagent2,reagent3,reagent4,reagent5,reagent6,reagent7,reagent8,reagentCount1,reagentCount2,reagentCount3,reagentCount4,reagentCount5,reagentCount6,reagentCount7,reagentCount8,equippedItemClass,equippedItemSubClassMask,equippedItemInventoryTypeMask,effect1,effect2,effect3,effectDieSides1,effectDieSides2,effectDieSides3,effectBaseDice1,effectBaseDice2,effectBaseDice3,effectDicePerLevel1,effectDicePerLevel2,effectDicePerLevel3,effectRealPointsPerLevel1,effectRealPointsPerLevel2,effectRealPointsPerLevel3,effectBasePoints1,effectBasePoints2,effectBasePoints3,effectBonusCoefficient1,effectBonusCoefficient2,effectBonusCoefficient3,effectMechanic1,effectMechanic2,effectMechanic3,effectImplicitTargetA1,effectImplicitTargetA2,effectImplicitTargetA3,effectImplicitTargetB1,effectImplicitTargetB2,effectImplicitTargetB3,effectRadiusIndex1,effectRadiusIndex2,effectRadiusIndex3,effectApplyAuraName1,effectApplyAuraName2,effectApplyAuraName3,effectAmplitude1,effectAmplitude2,effectAmplitude3,effectMultipleValue1,effectMultipleValue2,effectMultipleValue3,effectChainTarget1,effectChainTarget2,effectChainTarget3,effectItemType1,effectItemType2,effectItemType3,effectMiscValue1,effectMiscValue2,effectMiscValue3,effectTriggerSpell1,effectTriggerSpell2,effectTriggerSpell3,effectPointsPerComboPoint1,effectPointsPerComboPoint2,effectPointsPerComboPoint3,spellVisual1,spellVisual2,spellIconId,activeIconId,spellPriority,name,nameFlags,nameSubtext,nameSubtextFlags,description,descriptionFlags,auraDescription,auraDescriptionFlags,manaCostPercentage,startRecoveryCategory,startRecoveryTime,minTargetLevel,maxTargetLevel,spellFamilyName,spellFamilyFlags,maxAffectedTargets,dmgClass,preventionType,stanceBarOrder,dmgMultiplier1,dmgMultiplier2,dmgMultiplier3,minFactionId,minReputation,requiredAuraVision,customFlags"

	// Check if we need to import. Threshold < 100 means we only have initial seed data.
	var spellCount int
	i.sqliteDB.QueryRow("SELECT COUNT(*) FROM spell_template").Scan(&spellCount)
	if spellCount < 100 {
		if err := i.runImport("spell_template", spellCols); err != nil {
			log.Printf("Warning: Failed to import spell_template: %v", err)
		}
	} else {
		log.Printf("⏭️  spell_template already has %d rows, skipping", spellCount)
	}

	// 4b. Spell Aux Tables (Aowow Structure)
	// spell_icons
	if err := i.runCustomImportIfEmpty("spell_icons", "SELECT id, iconname FROM aowow.aowow_spellicons", "INSERT OR REPLACE INTO spell_icons (id, icon_name) VALUES (?, ?)"); err != nil {
		log.Printf("Warning: Failed to import spell_icons: %v", err)
	}
	// spell_range
	if err := i.runCustomImportIfEmpty("spell_range", "SELECT rangeID, rangeMin, rangeMax, name_loc0 FROM aowow.aowow_spellrange", "INSERT OR REPLACE INTO spell_range (id, range_min, range_max, name) VALUES (?, ?, ?, ?)"); err != nil {
		log.Printf("Warning: Failed to import spell_range: %v", err)
	}
	// spell_durations
	if err := i.runCustomImportIfEmpty("spell_durations", "SELECT durationID, durationBase FROM aowow.aowow_spellduration", "INSERT OR REPLACE INTO spell_durations (id, duration_base) VALUES (?, ?)"); err != nil {
		log.Printf("Warning: Failed to import spell_durations: %v", err)
	}
	// spell_radius
	if err := i.runCustomImportIfEmpty("spell_radius", "SELECT radiusID, radiusBase FROM aowow.aowow_spellradius", "INSERT OR REPLACE INTO spell_radius (id, radius_base) VALUES (?, ?)"); err != nil {
		log.Printf("Warning: Failed to import spell_radius: %v", err)
	}
	// spell_cast_times
	if err := i.runCustomImportIfEmpty("spell_cast_times", "SELECT id, base FROM aowow.aowow_spellcasttimes", "INSERT OR REPLACE INTO spell_cast_times (id, base) VALUES (?, ?)"); err != nil {
		log.Printf("Warning: Failed to import spell_cast_times: %v", err)
	}

	// 5. Gameobject Template
	goCols := "entry,`type`,displayId,name,faction,flags,size,data0,data1,data2,data3,data4,data5,data6,data7,data8,data9,data10,data11,data12,data13,data14,data15,data16,data17,data18,data19,data20,data21,data22,data23,mingold,maxgold,phase_quest_id,script_name"
	if err := i.runImportIfEmpty("gameobject_template", goCols); err != nil {
		log.Printf("Warning: Failed to import gameobject_template: %v", err)
	}

	// 6. Loot Templates
	lootTables := []string{
		"creature_loot_template",
		"reference_loot_template",
		"gameobject_loot_template",
		"item_loot_template",
		"disenchant_loot_template",
	}
	// Column names: entry, item, ChanceOrQuestChance, groupid, mincountOrRef, maxcount
	for _, table := range lootTables {
		if err := i.runLootImportIfEmpty(table); err != nil {
			log.Printf("Warning: Failed to import loot table %s: %v", table, err)
		}
	}

	// 7. Item Display Info
	if err := i.runImportIfEmpty("item_display_info", "ID,icon"); err != nil {
		log.Printf("Warning: Failed to import item_display_info: %v", err)
	}

	// 8. NPC Quest Relations
	if err := i.runImportIfEmpty("creature_questrelation", "id,quest"); err != nil {
		log.Printf("Warning: Failed to import creature_questrelation: %v", err)
	}
	if err := i.runImportIfEmpty("creature_involvedrelation", "id,quest"); err != nil {
		log.Printf("Warning: Failed to import creature_involvedrelation: %v", err)
	}

	// 9. GO Quest Relations
	if err := i.runImportIfEmpty("gameobject_questrelation", "id,quest"); err != nil {
		log.Printf("Warning: Failed to import gameobject_questrelation: %v", err)
	}
	if err := i.runImportIfEmpty("gameobject_involvedrelation", "id,quest"); err != nil {
		log.Printf("Warning: Failed to import gameobject_involvedrelation: %v", err)
	}

	// 10. Aowow Zones (for coordinate conversion)
	// Always force import for this table to ensure coordinates are correct, skipping empty check
	// Note: Mapping areatableID to zoneID as well using SELECT areatableID twice or aliasing if driver supports it.
	// We select areatableID as the second column to map to zoneID in the INSERT statement.
	if err := i.ImportTable("aowow_zones",
		"SELECT mapID, areatableID, name_loc0, x_min, x_max, y_min, y_max, areatableID FROM aowow.aowow_zones",
		"INSERT OR REPLACE INTO aowow_zones (mapID, zoneID, name_loc0, x_min, x_max, y_min, y_max, areatableID) VALUES (?, ?, ?, ?, ?, ?, ?, ?)", 1000); err != nil {
		log.Printf("Warning: Failed to import aowow_zones: %v", err)
	}

	return nil
}

// runImportIfEmpty imports a table only if it's empty in SQLite
func (i *MySQLImporter) runImportIfEmpty(table string, cols string) error {
	// Check if table is empty
	var count int
	i.sqliteDB.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", table)).Scan(&count)
	if count > 0 {
		log.Printf("⏭️  %s already has %d rows, skipping", table, count)
		return nil
	}

	return i.runImport(table, cols)
}

func (i *MySQLImporter) runImport(table string, cols string) error {
	// Construct SELECT query
	selectQuery := fmt.Sprintf("SELECT %s FROM %s", cols, table)

	// Construct placeholders for INSERT
	// Count commas to determine number of columns
	numCols := 1
	for _, r := range cols {
		if r == ',' {
			numCols++
		}
	}

	placeholders := "?"
	for k := 1; k < numCols; k++ {
		placeholders += ",?"
	}

	insertQuery := fmt.Sprintf("INSERT OR REPLACE INTO %s (%s) VALUES (%s)", table, cols, placeholders)

	return i.ImportTable(table, selectQuery, insertQuery, 1000)
}

// runCustomImportIfEmpty allows specifying custom SELECT/INSERT queries
func (i *MySQLImporter) runCustomImportIfEmpty(table, selectQuery, insertQuery string) error {
	var count int
	i.sqliteDB.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", table)).Scan(&count)
	if count > 0 {
		log.Printf("⏭️  %s already has %d rows, skipping", table, count)
		return nil
	}
	return i.ImportTable(table, selectQuery, insertQuery, 1000)
}

// runLootImportIfEmpty handles loot table import
// Column names now match between MySQL and SQLite
func (i *MySQLImporter) runLootImportIfEmpty(table string) error {
	// Check if table is empty
	var count int
	i.sqliteDB.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", table)).Scan(&count)
	if count > 0 {
		log.Printf("⏭️  %s already has %d rows, skipping", table, count)
		return nil
	}

	cols := "entry, item, ChanceOrQuestChance, groupid, mincountOrRef, maxcount"
	return i.runImport(table, cols)
}
