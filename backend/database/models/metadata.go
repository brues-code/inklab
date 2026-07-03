package models

// ZoneEntry represents a zone for JSON import
type ZoneEntry struct {
	AreaID int    `json:"areatableID"`
	MapID  int    `json:"mapID"`
	Name   string `json:"name_loc0"`     // map-texture folder name (the map-image key)
	DisplayName string `json:"displayName"` // official localized name (AreaTable.dbc)
	// InstanceType from Map.dbc: 0 continent, 1 dungeon, 2 raid, 3 battleground.
	InstanceType int `json:"instanceType"`
}

// QuestSortEntry represents a QuestSort.dbc category for JSON import. Quests
// reference these via a NEGATIVE ZoneOrSort (e.g. ZoneOrSort -61 -> sortID 61
// "Warlock"); they have no AreaTable entry.
type QuestSortEntry struct {
	SortID int    `json:"sortID"`
	Name   string `json:"name_loc0"`
}

// SkillEntry represents a skill for JSON import
type SkillEntry struct {
	ID         int    `json:"skillID"`
	CategoryID int    `json:"categoryID"`
	Name       string `json:"name_loc0"`
}

// SkillLineAbilityEntry represents a skill-spell relationship for JSON import
type SkillLineAbilityEntry struct {
	SkillID       int `json:"skillID"`
	SpellID       int `json:"spellID"`
	ClassMask     int `json:"classmask"`
	ReqSkillValue int `json:"req_skill_value"`
}

// StatFilter requires an item to have at least Min of a given stat_type.
type StatFilter struct {
	Stat int `json:"stat"` // stat_type id (matches GetItemStatTypes)
	Min  int `json:"min"`  // minimum summed value across the item's stat slots
}

// ResistFilter requires an item to have at least Min of a given resistance
// school (1=Holy 2=Fire 3=Nature 4=Frost 5=Shadow 6=Arcane, matching spell_schools).
type ResistFilter struct {
	School int `json:"school"`
	Min    int `json:"min"`
}

// SearchFilter defines criteria for advanced item search. Translated to SQL by
// the item filter service (item_filter.go).
type SearchFilter struct {
	Query         string `json:"query"`
	Quality       []int  `json:"quality,omitempty"`
	Class         []int  `json:"class,omitempty"`
	SubClass      []int  `json:"subClass,omitempty"`
	InventoryType []int  `json:"inventoryType,omitempty"`
	MinLevel      int    `json:"minLevel,omitempty"`
	MaxLevel      int    `json:"maxLevel,omitempty"`
	MinReqLevel   int    `json:"minReqLevel,omitempty"`
	MaxReqLevel   int    `json:"maxReqLevel,omitempty"`

	// Stats: each entry requires the item to carry at least Min of that stat.
	Stats []StatFilter `json:"stats,omitempty"`
	// UsableByClass (1..N, 0 = any) keeps only items the given class can equip.
	UsableByClass int `json:"usableByClass,omitempty"`
	// Sources: keep items obtainable via any of these — "drop","object",
	// "container","vendor","quest","crafted","disenchant".
	Sources []string `json:"sources,omitempty"`

	// Item properties.
	Bonding       []int `json:"bonding,omitempty"` // bonding values to include
	OnlyUnique    bool  `json:"onlyUnique,omitempty"`
	ClassSpecific bool  `json:"classSpecific,omitempty"`
	RaceSpecific  bool  `json:"raceSpecific,omitempty"`
	StartsQuest   bool  `json:"startsQuest,omitempty"`
	HasEffect     bool  `json:"hasEffect,omitempty"` // has an on-use/equip spell
	// HasRandomSuffix keeps only items that roll a random suffix ("of the
	// Monkey"), i.e. random_property > 0.
	HasRandomSuffix bool `json:"hasRandomSuffix,omitempty"`

	// Requirements & economy.
	RequiresProf      bool `json:"requiresProf,omitempty"` // required_skill > 0
	MinSkillRank      int  `json:"minSkillRank,omitempty"`
	MaxSkillRank      int  `json:"maxSkillRank,omitempty"`
	RequiresRep       bool `json:"requiresRep,omitempty"` // required_reputation_faction > 0
	RequiredRepFaction int `json:"requiredRepFaction,omitempty"`
	MinBuyPrice       int  `json:"minBuyPrice,omitempty"`
	MaxBuyPrice       int  `json:"maxBuyPrice,omitempty"`
	MinSellPrice      int  `json:"minSellPrice,omitempty"`
	MaxSellPrice      int  `json:"maxSellPrice,omitempty"`
	MinDurability     int  `json:"minDurability,omitempty"`
	MaxDurability     int  `json:"maxDurability,omitempty"`

	// Weapon & armor stats.
	MinDps       float64        `json:"minDps,omitempty"`
	MaxDps       float64        `json:"maxDps,omitempty"`
	MinSpeed     float64        `json:"minSpeed,omitempty"`
	MaxSpeed     float64        `json:"maxSpeed,omitempty"`
	DamageSchool int            `json:"damageSchool,omitempty"` // dmg_type1; <0 = any
	MinArmor     int            `json:"minArmor,omitempty"`
	MaxArmor     int            `json:"maxArmor,omitempty"`
	MinBlock     int            `json:"minBlock,omitempty"`
	MaxBlock     int            `json:"maxBlock,omitempty"`
	Resists      []ResistFilter `json:"resists,omitempty"`

	// Sort field ("name","itemLevel","requiredLevel","quality") + direction.
	Sort    string `json:"sort,omitempty"`
	SortDir string `json:"sortDir,omitempty"` // "asc" | "desc"

	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

// SearchResult represents the search output
type SearchResult struct {
	Items      []*Item       `json:"items"`
	Creatures  []*Creature   `json:"creatures,omitempty"`
	Quests     []*Quest      `json:"quests,omitempty"`
	Spells     []*Spell      `json:"spells,omitempty"`
	Objects    []*GameObject `json:"objects,omitempty"`
	TotalCount int           `json:"totalCount"`
}
