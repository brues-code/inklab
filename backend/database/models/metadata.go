package models

// ZoneEntry represents a zone for JSON import
type ZoneEntry struct {
	AreaID int    `json:"areatableID"`
	MapID  int    `json:"mapID"`
	Name   string `json:"name_loc0"`
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
	SkillID   int `json:"skillID"`
	SpellID   int `json:"spellID"`
	ClassMask int `json:"classmask"`
}

// SearchFilter defines criteria for advanced item search
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
	Limit         int    `json:"limit"`
	Offset        int    `json:"offset"`
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
