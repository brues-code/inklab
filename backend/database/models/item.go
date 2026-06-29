// Package models contains all database entity definitions
package models

// Item represents a database item record
type Item struct {
	Entry          int    `json:"entry"`
	Name           string `json:"name"`
	Description    string `json:"description,omitempty"`
	Quality        int    `json:"quality"`
	ItemLevel      int    `json:"itemLevel"`
	RequiredLevel  int    `json:"requiredLevel"`
	Class          int    `json:"class"`
	SubClass       int    `json:"subClass"`
	InventoryType  int    `json:"inventoryType"`
	// TypeName/SlotName are the client-localized subclass + equip-slot labels,
	// resolved server-side (so 2H weapons read correctly without client knowing
	// the subclass hierarchy). Populated by the item browse query.
	TypeName       string `json:"typeName,omitempty"`
	SlotName       string `json:"slotName,omitempty"`
	IconPath       string `json:"iconPath"`
	SellPrice      int    `json:"sellPrice,omitempty"`
	BuyPrice       int    `json:"buyPrice,omitempty"`
	AllowableClass int    `json:"allowableClass,omitempty"`
	AllowableRace  int    `json:"allowableRace,omitempty"`
	Bonding        int    `json:"bonding,omitempty"`
	MaxDurability  int    `json:"maxDurability,omitempty"`
	MaxCount       int    `json:"maxCount,omitempty"`
	Armor          int    `json:"armor,omitempty"`
	// Stats
	StatType1   int `json:"statType1,omitempty"`
	StatValue1  int `json:"statValue1,omitempty"`
	StatType2   int `json:"statType2,omitempty"`
	StatValue2  int `json:"statValue2,omitempty"`
	StatType3   int `json:"statType3,omitempty"`
	StatValue3  int `json:"statValue3,omitempty"`
	StatType4   int `json:"statType4,omitempty"`
	StatValue4  int `json:"statValue4,omitempty"`
	StatType5   int `json:"statType5,omitempty"`
	StatValue5  int `json:"statValue5,omitempty"`
	StatType6   int `json:"statType6,omitempty"`
	StatValue6  int `json:"statValue6,omitempty"`
	StatType7   int `json:"statType7,omitempty"`
	StatValue7  int `json:"statValue7,omitempty"`
	StatType8   int `json:"statType8,omitempty"`
	StatValue8  int `json:"statValue8,omitempty"`
	StatType9   int `json:"statType9,omitempty"`
	StatValue9  int `json:"statValue9,omitempty"`
	StatType10  int `json:"statType10,omitempty"`
	StatValue10 int `json:"statValue10,omitempty"`
	// Weapon stats
	Delay    int     `json:"delay,omitempty"`
	DmgMin1  float64 `json:"dmgMin1,omitempty"`
	DmgMax1  float64 `json:"dmgMax1,omitempty"`
	DmgType1 int     `json:"dmgType1,omitempty"`
	DmgMin2  float64 `json:"dmgMin2,omitempty"`
	DmgMax2  float64 `json:"dmgMax2,omitempty"`
	DmgType2 int     `json:"dmgType2,omitempty"`
	// Resistances
	HolyRes   int `json:"holyRes,omitempty"`
	FireRes   int `json:"fireRes,omitempty"`
	NatureRes int `json:"natureRes,omitempty"`
	FrostRes  int `json:"frostRes,omitempty"`
	ShadowRes int `json:"shadowRes,omitempty"`
	ArcaneRes int `json:"arcaneRes,omitempty"`
	// Spells
	SpellID1      int `json:"spellId1,omitempty"`
	SpellTrigger1 int `json:"spellTrigger1,omitempty"`
	SpellID2      int `json:"spellId2,omitempty"`
	SpellTrigger2 int `json:"spellTrigger2,omitempty"`
	SpellID3      int `json:"spellId3,omitempty"`
	SpellTrigger3 int `json:"spellTrigger3,omitempty"`
	// Set
	SetID    int    `json:"setId,omitempty"`
	DropRate string `json:"dropRate,omitempty"`
	// Container (bag) slot count; 0 for non-containers.
	ContainerSlots int `json:"containerSlots,omitempty"`
}

// ItemTemplate represents a complete item from item_template.json
type ItemTemplate struct {
	Entry          int    `json:"entry"`
	Class          int    `json:"class"`
	SubClass       int    `json:"subclass"`
	Name           string `json:"name"`
	DisplayID      int    `json:"displayid"`
	Quality        int    `json:"quality"`
	Flags          int    `json:"flags"`
	BuyPrice       int    `json:"buyPrice"`
	SellPrice      int    `json:"sellPrice"`
	InventoryType  int    `json:"inventoryType"`
	AllowableClass int    `json:"allowableClass"`
	AllowableRace  int    `json:"allowableRace"`
	ItemLevel      int    `json:"itemLevel"`
	RequiredLevel  int    `json:"requiredLevel"`
	MaxCount       int    `json:"maxCount"`
	Stackable      int    `json:"stackable"`
	ContainerSlots int    `json:"containerSlots"`
	// Stats
	StatType1   int `json:"statType1"`
	StatValue1  int `json:"statValue1"`
	StatType2   int `json:"statType2"`
	StatValue2  int `json:"statValue2"`
	StatType3   int `json:"statType3"`
	StatValue3  int `json:"statValue3"`
	StatType4   int `json:"statType4"`
	StatValue4  int `json:"statValue4"`
	StatType5   int `json:"statType5"`
	StatValue5  int `json:"statValue5"`
	StatType6   int `json:"statType6"`
	StatValue6  int `json:"statValue6"`
	StatType7   int `json:"statType7"`
	StatValue7  int `json:"statValue7"`
	StatType8   int `json:"statType8"`
	StatValue8  int `json:"statValue8"`
	StatType9   int `json:"statType9"`
	StatValue9  int `json:"statValue9"`
	StatType10  int `json:"statType10"`
	StatValue10 int `json:"statValue10"`
	// Damage
	DmgMin1        float64 `json:"dmgMin1"`
	DmgMax1        float64 `json:"dmgMax1"`
	DmgType1       int     `json:"dmgType1"`
	DmgMin2        float64 `json:"dmgMin2"`
	DmgMax2        float64 `json:"dmgMax2"`
	DmgType2       int     `json:"dmgType2"`
	Armor          int     `json:"armor"`
	HolyRes        int     `json:"holyRes"`
	FireRes        int     `json:"fireRes"`
	NatureRes      int     `json:"natureRes"`
	FrostRes       int     `json:"frostRes"`
	ShadowRes      int     `json:"shadowRes"`
	ArcaneRes      int     `json:"arcaneRes"`
	Delay          int     `json:"delay"`
	RangedModRange float64 `json:"rangedModRange"`
	// Spells
	SpellID1      int `json:"spellId1"`
	SpellTrigger1 int `json:"spellTrigger1"`
	SpellID2      int `json:"spellId2"`
	SpellTrigger2 int `json:"spellTrigger2"`
	SpellID3      int `json:"spellId3"`
	SpellTrigger3 int `json:"spellTrigger3"`
	Bonding       int `json:"bonding"`
	MaxDurability int `json:"maxDurability"`
	SetID         int `json:"setId"`
	Material      int `json:"material"`
}

// ItemDef represents a single item's metadata (simplified for AtlasLoot)
type ItemDef struct {
	Entry         int    `json:"id"`
	Name          string `json:"name"`
	Description   string `json:"description"`
	Quality       int    `json:"quality"`
	ItemLevel     int    `json:"itemlevel"`
	RequiredLevel int    `json:"requiredLevel"`
	InventoryType int    `json:"inventoryType"`
	IconPath      string `json:"icon"`
	SellPrice     int    `json:"sellPrice"`
	ItemLink      string `json:"itemLink"`
	Class         string `json:"class"`
	SubClass      string `json:"subClass"`
	EquipSlot     string `json:"equipSlot"`
	MaxStack      int    `json:"maxStack"`
}

// SetBonus represents a set bonus
type SetBonus struct {
	Threshold   int    `json:"threshold"`
	SpellID     int    `json:"spellId"`
	Description string `json:"description"`
}

// ItemSet represents an item set
type ItemSet struct {
	ID      int        `json:"id"`
	Name    string     `json:"name"`
	ItemIDs []int      `json:"itemIds"`
	Bonuses []SetBonus `json:"bonuses"`
}

// ItemSetInfo represents set info for tooltip
type ItemSetInfo struct {
	Name    string   `json:"name"`
	Items   []string `json:"items"`
	Bonuses []string `json:"bonuses"`
}

// TooltipData represents data for rendering a tooltip
// TooltipEffect is one item spell effect line (e.g. "Use: ...") plus the spell
// id it comes from, so the UI can link to the spell page.
type TooltipEffect struct {
	Text    string `json:"text"`
	SpellID int    `json:"spellId"`
}

type TooltipData struct {
	Entry         int             `json:"entry"`
	Name          string          `json:"name"`
	Quality       int             `json:"quality"`
	ItemLevel     int             `json:"itemLevel,omitempty"`
	Binding       string          `json:"binding,omitempty"`
	Unique        bool            `json:"unique,omitempty"`
	ItemType      string          `json:"typeName,omitempty"`
	Slot          string          `json:"slotName,omitempty"`
	Armor         int             `json:"armor,omitempty"`
	DamageRange   string          `json:"damageText,omitempty"`
	AttackSpeed   string          `json:"speedText,omitempty"`
	DPS           string          `json:"dps,omitempty"`
	Stats         []string        `json:"stats,omitempty"`
	Resistances   []string        `json:"resistances,omitempty"`
	Effects       []TooltipEffect `json:"effects,omitempty"`
	RequiredLevel int             `json:"requiredLevel,omitempty"`
	SellPrice     int             `json:"sellPrice,omitempty"`
	Durability    string          `json:"durability,omitempty"`
	Classes       string          `json:"classes,omitempty"`
	ClassReqs     []*ItemClassReq `json:"classReqs,omitempty"` // colored class restriction (from allowable_class)
	Races         string          `json:"races,omitempty"`
	// Reputation requirement (required_reputation_faction/rank), e.g.
	// faction "The League of Arathor" + standing "Revered".
	ReqRepFaction  string `json:"reqRepFaction,omitempty"`
	ReqRepStanding string `json:"reqRepStanding,omitempty"`
	SetInfo       *ItemSetInfo    `json:"setInfo,omitempty"`
	Description   string          `json:"description,omitempty"`
}

// ItemClass represents a WoW item class (Weapon, Armor, etc.)
type ItemClass struct {
	Class      int             `json:"class"`
	Name       string          `json:"name"`
	SubClasses []*ItemSubClass `json:"subClasses,omitempty"`
}

// StatType is an item stat (the ITEM_MOD enum): a stat_type id and its display
// name. Used to build the item filter's stat dropdown dynamically.
type StatType struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// ItemSubClass represents a WoW item subclass (Axe, Bow, etc.)
type ItemSubClass struct {
	Class          int              `json:"class"`
	SubClass       int              `json:"subClass"`
	Name           string           `json:"name"`
	InventorySlots []*InventorySlot `json:"inventorySlots,omitempty"`
}

// InventorySlot represents a WoW inventory type (Head, Chest, etc.)
type InventorySlot struct {
	Class         int    `json:"class"`
	SubClass      int    `json:"subClass"`
	InventoryType int    `json:"inventoryType"`
	Name          string `json:"name"`
}

// ItemDetail includes extended item information with sources
type ItemDetail struct {
	*Item
	// maxCount, containerSlots and dmgMin2/Max2/Type2 are intentionally NOT
	// re-declared here: they already come from the embedded *Item (populated by
	// the repository). Re-declaring them as zero-valued outer fields shadowed
	// the embedded values in JSON, zeroing them out for the frontend.
	DisplayID       int                `json:"displayId"`
	Flags           int                `json:"flags"`
	BuyCount        int                `json:"buyCount"`
	Stackable       int                `json:"stackable"`
	Material        int                `json:"material"`
	DroppedBy       []*CreatureDrop    `json:"droppedBy"`
	RewardFrom      []*QuestReward     `json:"rewardFrom"`
	Contains        []*ItemDrop        `json:"contains"`
	SoldBy          []*ItemVendor      `json:"soldBy"`
	CreatedBy       []*ItemCraftSource `json:"createdBy"`
	ReagentFor      []*ItemReagentUse  `json:"reagentFor"`
	ContainedIn     []*ItemContainer   `json:"containedIn"`     // gameobject chests
	ContainedInItem []*ItemContainer   `json:"containedInItem"` // container items
	GatheredFrom    []*ItemContainer   `json:"gatheredFrom"`
	ObjectiveOf     []*QuestReward     `json:"objectiveOf"`
	StartsQuest     *QuestReward       `json:"startsQuest,omitempty"`
	// Mount: for a mount item, the spell it grants and that spell's creature
	// display id (from the Mounted aura), so the UI can render the mount model.
	MountSpellID   int `json:"mountSpellId,omitempty"`
	MountDisplayID int `json:"mountDisplayId,omitempty"`
}

// ItemClassReq is one class an item is restricted to (from allowable_class),
// with its UI color (#rrggbb) from class_info.
type ItemClassReq struct {
	Name  string `json:"name"`
	Color string `json:"color"`
}

// ItemContainer is a container (gameobject chest or other item) whose loot
// includes this item — the reverse of Contains.
type ItemContainer struct {
	Entry    int     `json:"entry"`
	Name     string  `json:"name"`
	Kind     string  `json:"kind"` // "object" or "item"
	Quality  int     `json:"quality"`
	IconPath string  `json:"iconPath"`
	Chance   float64 `json:"chance"`
	Skill    string  `json:"skill,omitempty"`    // gathering skill for nodes (Herbalism/Mining/Fishing)
	SkillReq int     `json:"skillReq,omitempty"` // required skill level to gather (lock req value)
}

// ItemCraftSource is a recipe/spell that creates this item (a tradeskill spell
// with a Create Item effect), including its profession requirement and reagents.
type ItemCraftSource struct {
	SpellID       int             `json:"spellId"`
	SpellName     string          `json:"spellName"`
	SpellIcon     string          `json:"spellIcon"`
	SkillName     string          `json:"skillName,omitempty"`
	ReqSkill      int             `json:"reqSkill,omitempty"`
	ProducedCount int             `json:"producedCount,omitempty"`
	Reagents      []*CraftReagent `json:"reagents,omitempty"`
}

// ItemReagentUse is a recipe that consumes this item as a reagent: the craft
// spell, the item it produces (if any), the profession requirement, and how many
// of this reagent the recipe needs. The reverse of ItemCraftSource.Reagents.
type ItemReagentUse struct {
	SpellID         int    `json:"spellId"`
	SpellName       string `json:"spellName"`
	SpellIcon       string `json:"spellIcon"`
	SkillName       string `json:"skillName,omitempty"`
	ReqSkill        int    `json:"reqSkill,omitempty"`
	ReagentCount    int    `json:"reagentCount,omitempty"` // how many of this item the recipe uses
	ProducedItem    int    `json:"producedItem,omitempty"`
	ProducedName    string `json:"producedName,omitempty"`
	ProducedIcon    string `json:"producedIcon,omitempty"`
	ProducedQuality int    `json:"producedQuality,omitempty"`
	ProducedCount   int    `json:"producedCount,omitempty"`
}

// CraftReagent is one material consumed by a crafting spell.
type CraftReagent struct {
	Entry    int    `json:"entry"`
	Name     string `json:"name"`
	Quality  int    `json:"quality"`
	IconPath string `json:"iconPath"`
	Count    int    `json:"count"`
}

// ItemVendor represents an NPC that sells an item (from the item page's
// "sold by" list). Cost is the money price in copper (0 when bought with an
// extended-cost currency); Stock is -1 for unlimited.
type ItemVendor struct {
	Entry     int    `json:"entry"`
	Name      string `json:"name"`
	LevelMin  int    `json:"levelMin"`
	LevelMax  int    `json:"levelMax"`
	Cost      int    `json:"cost"`
	Stock     int    `json:"stock"`
	Location  string `json:"location"`  // vendor's spawn zone(s), comma-joined
	ReactionA string `json:"reactionA"` // "friendly"/"hostile"/"neutral" toward Alliance
	ReactionH string `json:"reactionH"` // "friendly"/"hostile"/"neutral" toward Horde
}

// ItemDrop represents an item dropped by another item (e.g. from chest/clam)
type ItemDrop struct {
	Entry    int     `json:"entry"`
	Name     string  `json:"name"`
	Quality  int     `json:"quality"`
	Chance   float64 `json:"chance"`
	MinCount int     `json:"minCount"`
	MaxCount int     `json:"maxCount"`
	IconPath string  `json:"iconPath"`
}

// CreatureDrop represents a creature that drops an item
type CreatureDrop struct {
	Entry    int     `json:"entry"`
	Name     string  `json:"name"`
	LevelMin int     `json:"levelMin"`
	LevelMax int     `json:"levelMax"`
	Chance   float64 `json:"chance"`
}

// QuestReward represents a quest that rewards an item
type QuestReward struct {
	Entry    int    `json:"entry"`
	Title    string `json:"title"`
	Level    int    `json:"level"`
	IsChoice bool   `json:"isChoice"`
}

// ItemSetBrowse represents an item set for browsing list
type ItemSetBrowse struct {
	ItemSetID  int    `json:"itemsetId"`
	Name       string `json:"name"`
	ItemIDs    []int  `json:"itemIds"`
	ItemCount  int    `json:"itemCount"`
	SkillID    int    `json:"skillId"`
	SkillLevel int    `json:"skillLevel"`
	// ClassMask is the class restriction derived from the set's items'
	// allowable_class (class bits, same as class_mask). 0 = no restriction.
	ClassMask int `json:"classMask"`
}

// ItemSetDetail includes items with their details
type ItemSetDetail struct {
	ItemSetID int        `json:"itemsetId"`
	Name      string     `json:"name"`
	Items     []*Item    `json:"items"`
	Bonuses   []SetBonus `json:"bonuses"`
}

// ItemTemplateEntry represents an item for JSON import
type ItemTemplateEntry struct {
	Entry          int     `json:"entry"`
	Name           string  `json:"name"`
	Description    string  `json:"description"`
	Quality        int     `json:"quality"`
	ItemLevel      int     `json:"item_level"`
	RequiredLevel  int     `json:"required_level"`
	Class          int     `json:"class"`
	Subclass       int     `json:"subclass"`
	InventoryType  int     `json:"inventory_type"`
	DisplayID      int     `json:"display_id"`
	BuyPrice       int     `json:"buy_price"`
	SellPrice      int     `json:"sell_price"`
	AllowableClass int     `json:"allowable_class"`
	AllowableRace  int     `json:"allowable_race"`
	Stackable      int     `json:"stackable"`
	MaxCount       int     `json:"max_count"`
	Bonding        int     `json:"bonding"`
	MaxDurability  int     `json:"max_durability"`
	ContainerSlots int     `json:"container_slots"`
	StatType1      int     `json:"stat_type1"`
	StatValue1     int     `json:"stat_value1"`
	StatType2      int     `json:"stat_type2"`
	StatValue2     int     `json:"stat_value2"`
	StatType3      int     `json:"stat_type3"`
	StatValue3     int     `json:"stat_value3"`
	StatType4      int     `json:"stat_type4"`
	StatValue4     int     `json:"stat_value4"`
	StatType5      int     `json:"stat_type5"`
	StatValue5     int     `json:"stat_value5"`
	StatType6      int     `json:"stat_type6"`
	StatValue6     int     `json:"stat_value6"`
	StatType7      int     `json:"stat_type7"`
	StatValue7     int     `json:"stat_value7"`
	StatType8      int     `json:"stat_type8"`
	StatValue8     int     `json:"stat_value8"`
	StatType9      int     `json:"stat_type9"`
	StatValue9     int     `json:"stat_value9"`
	StatType10     int     `json:"stat_type10"`
	StatValue10    int     `json:"stat_value10"`
	Delay          int     `json:"delay"`
	DmgMin1        float64 `json:"dmg_min1"`
	DmgMax1        float64 `json:"dmg_max1"`
	DmgType1       int     `json:"dmg_type1"`
	DmgMin2        float64 `json:"dmg_min2"`
	DmgMax2        float64 `json:"dmg_max2"`
	DmgType2       int     `json:"dmg_type2"`
	Armor          int     `json:"armor"`
	HolyRes        int     `json:"holy_res"`
	FireRes        int     `json:"fire_res"`
	NatureRes      int     `json:"nature_res"`
	FrostRes       int     `json:"frost_res"`
	ShadowRes      int     `json:"shadow_res"`
	ArcaneRes      int     `json:"arcane_res"`
	SpellID1       int     `json:"spellid_1"`
	SpellTrigger1  int     `json:"spelltrigger_1"`
	SpellID2       int     `json:"spellid_2"`
	SpellTrigger2  int     `json:"spelltrigger_2"`
	SpellID3       int     `json:"spellid_3"`
	SpellTrigger3  int     `json:"spelltrigger_3"`
	ItemSet        int     `json:"set_id"`
}

// ItemSetEntry represents an item set for JSON import
type ItemSetEntry struct {
	ID         int    `json:"itemsetID"`
	Name       string `json:"name_loc0"`
	Item1      int    `json:"item1"`
	Item2      int    `json:"item2"`
	Item3      int    `json:"item3"`
	Item4      int    `json:"item4"`
	Item5      int    `json:"item5"`
	Item6      int    `json:"item6"`
	Item7      int    `json:"item7"`
	Item8      int    `json:"item8"`
	Item9      int    `json:"item9"`
	Item10     int    `json:"item10"`
	SkillID    int    `json:"skillID"`
	SkillLevel int    `json:"skilllevel"`
	Bonus1     int    `json:"bonus1"`
	Bonus2     int    `json:"bonus2"`
	Bonus3     int    `json:"bonus3"`
	Bonus4     int    `json:"bonus4"`
	Bonus5     int    `json:"bonus5"`
	Bonus6     int    `json:"bonus6"`
	Bonus7     int    `json:"bonus7"`
	Bonus8     int    `json:"bonus8"`
	Spell1     int    `json:"spell1"`
	Spell2     int    `json:"spell2"`
	Spell3     int    `json:"spell3"`
	Spell4     int    `json:"spell4"`
	Spell5     int    `json:"spell5"`
	Spell6     int    `json:"spell6"`
	Spell7     int    `json:"spell7"`
	Spell8     int    `json:"spell8"`
}
