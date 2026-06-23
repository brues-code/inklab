// Package database - Re-exports for backward compatibility
// This file re-exports types and functions from sub-packages to maintain
// compatibility with existing code that imports "inklab/backend/database"
package database

import (
	"database/sql"
	"inklab/backend/database/helpers"
	"inklab/backend/database/importers"
	"inklab/backend/database/models"
	"inklab/backend/database/repositories"
)

// === Type Aliases for Models ===

type Item = models.Item
type ItemTemplate = models.ItemTemplate
type ItemDef = models.ItemDef
type SetBonus = models.SetBonus
type ItemSet = models.ItemSet
type ItemSetInfo = models.ItemSetInfo
type TooltipData = models.TooltipData
type ItemClass = models.ItemClass
type ItemSubClass = models.ItemSubClass
type InventorySlot = models.InventorySlot
type ItemDetail = models.ItemDetail
type CreatureDrop = models.CreatureDrop
type QuestReward = models.QuestReward
type ItemSetBrowse = models.ItemSetBrowse
type ItemSetDetail = models.ItemSetDetail
type ItemTemplateEntry = models.ItemTemplateEntry
type ItemSetEntry = models.ItemSetEntry

type Creature = models.Creature
type CreatureType = models.CreatureType
type CreatureDetail = models.CreatureDetail
type CreatureTemplateEntry = models.CreatureTemplateEntry

type Quest = models.Quest
type QuestCategory = models.QuestCategory
type QuestDetail = models.QuestDetail
type QuestSeriesItem = models.QuestSeriesItem
type QuestItem = models.QuestItem
type QuestReputation = models.QuestReputation
type QuestRelation = models.QuestRelation
type QuestCategoryGroup = models.QuestCategoryGroup
type QuestCategoryEnhanced = models.QuestCategoryEnhanced
type QuestTemplateEntry = models.QuestTemplateEntry

type Spell = models.Spell
type SpellSkillCategory = models.SpellSkillCategory
type SpellSkill = models.SpellSkill
type SpellEntry = models.SpellEntry
type SpellDetail = models.SpellDetail
type SpellTemplateFull = models.SpellTemplateFull

type LootItem = models.LootItem
type LootEntry = models.LootEntry
type LootTemplateEntry = models.LootTemplateEntry

type GameObject = models.GameObject
type ObjectType = models.ObjectType
type LockEntry = models.LockEntry
type GameObjectDetail = models.GameObjectDetail

type Faction = models.Faction
type FactionEntry = models.FactionEntry
type FactionDetail = models.FactionDetail

type Category = models.Category
type CategoryItem = models.CategoryItem

type AtlasLootCategory = models.AtlasLootCategory
type AtlasLootModule = models.AtlasLootModule
type AtlasLootTable = models.AtlasLootTable
type AtlasLootItem = models.AtlasLootItem
type AtlasTable = models.AtlasTable
type AtlasLootImportItem = models.AtlasLootImportItem
type AtlasLootImportTable = models.AtlasLootImportTable
type AtlasLootImportModule = models.AtlasLootImportModule
type AtlasLootImportCategory = models.AtlasLootImportCategory

type FavoriteItem = models.FavoriteItem
type FavoriteCategory = models.FavoriteCategory
type FavoriteResult = models.FavoriteResult

type ZoneEntry = models.ZoneEntry
type SkillEntry = models.SkillEntry
type SkillLineAbilityEntry = models.SkillLineAbilityEntry
type SearchFilter = models.SearchFilter
type SearchResult = models.SearchResult

// === Repository Types ===

type ItemRepository = repositories.ItemRepository
type CreatureRepository = repositories.CreatureRepository
type QuestRepository = repositories.QuestRepository
type SpellRepository = repositories.SpellRepository
type LootRepository = repositories.LootRepository
type FactionRepository = repositories.FactionRepository
type GameObjectRepository = repositories.GameObjectRepository
type CategoryRepository = repositories.CategoryRepository
type AtlasLootRepository = repositories.AtlasLootRepository
type LocaleRepository = repositories.LocaleRepository
type FavoriteRepository = repositories.FavoriteRepository

// === Factory Functions ===

func NewItemRepository(db *SQLiteDB) *ItemRepository {
	return repositories.NewItemRepository(db.DB())
}

func NewCreatureRepository(db *SQLiteDB) *CreatureRepository {
	return repositories.NewCreatureRepository(db.DB())
}

func NewQuestRepository(db *SQLiteDB) *QuestRepository {
	return repositories.NewQuestRepository(db.DB())
}

func NewSpellRepository(db *SQLiteDB) *SpellRepository {
	return repositories.NewSpellRepository(db.DB())
}

func NewLootRepository(db *SQLiteDB) *LootRepository {
	return repositories.NewLootRepository(db.DB())
}

func NewFactionRepository(db *SQLiteDB) *FactionRepository {
	return repositories.NewFactionRepository(db.DB())
}

func NewGameObjectRepository(db *SQLiteDB) *GameObjectRepository {
	return repositories.NewGameObjectRepository(db.DB())
}

func NewCategoryRepository(db *SQLiteDB) *CategoryRepository {
	return repositories.NewCategoryRepository(db.DB())
}

func NewAtlasLootRepository(db *SQLiteDB) *AtlasLootRepository {
	return repositories.NewAtlasLootRepository(db.DB())
}

func NewLocaleRepository(db *SQLiteDB) *LocaleRepository {
	return repositories.NewLocaleRepository(db.DB())
}

func NewFavoriteRepository(db *SQLiteDB) *FavoriteRepository {
	return repositories.NewFavoriteRepository(db.DB())
}

// === Helper Function Exports ===

var GetClassName = helpers.GetClassName
var GetSubClassName = helpers.GetSubClassName
var GetInventoryTypeName = helpers.GetInventoryTypeName
var GetBondingName = helpers.GetBondingName
var GetQualityName = helpers.GetQualityName
var GetCreatureTypeName = helpers.GetCreatureTypeName
var GetCreatureRankName = helpers.GetCreatureRankName
var GetTriggerPrefix = helpers.GetTriggerPrefix
var CleanName = helpers.CleanName
var CleanItemName = helpers.CleanItemName
var FormatSpellDesc = helpers.FormatSpellDesc

// === Importer Factory Functions ===

func NewFactionImporter(db *SQLiteDB) *importers.FactionImporter {
	return importers.NewFactionImporter(db.DB())
}

func NewItemSetImporter(db *SQLiteDB) *importers.ItemSetImporter {
	return importers.NewItemSetImporter(db.DB())
}
func NewAtlasLootImporter(db *SQLiteDB) *importers.AtlasLootImporter {
	return importers.NewAtlasLootImporter(db.DB())
}

func NewMetadataImporter(db *SQLiteDB) *importers.MetadataImporter {
	return importers.NewMetadataImporter(db.DB())
}

// GeneratedImporter for 1:1 MySQL tables
type GeneratedImporter = importers.GeneratedImporter

func NewGeneratedImporter(db *sql.DB) *GeneratedImporter {
	return importers.NewGeneratedImporter(db)
}

// MySQLImporter for direct MySQL -> SQLite import
type MySQLImporter = importers.MySQLImporter

func NewMySQLImporter(sqliteDB *sql.DB, mysqlDB *sql.DB) *MySQLImporter {
	return importers.NewMySQLImporter(sqliteDB, mysqlDB)
}
