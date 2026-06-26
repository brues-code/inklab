package models

// Creature represents a WoW NPC
type Creature struct {
	Entry      int     `json:"entry"`
	Name       string  `json:"name"`
	Subname    string  `json:"subname,omitempty"`
	LevelMin   int     `json:"levelMin"`
	LevelMax   int     `json:"levelMax"`
	HealthMin  int     `json:"healthMin"`
	HealthMax  int     `json:"healthMax"`
	ManaMin    int     `json:"manaMin"`
	ManaMax    int     `json:"manaMax"`
	GoldMin    int     `json:"goldMin"`
	GoldMax    int     `json:"goldMax"`
	Type       int     `json:"type"`
	TypeName   string  `json:"typeName"`
	Rank       int     `json:"rank"`
	RankName   string  `json:"rankName"`
	Faction    int     `json:"faction"`
	NPCFlags   int     `json:"npcFlags"`
	MinDmg     float64 `json:"minDmg"`
	MaxDmg     float64 `json:"maxDmg"`
	Armor      int     `json:"armor"`
	HolyRes    int     `json:"holyRes"`
	FireRes    int     `json:"fireRes"`
	NatureRes  int     `json:"natureRes"`
	FrostRes   int     `json:"frostRes"`
	ShadowRes  int     `json:"shadowRes"`
	ArcaneRes  int     `json:"arcaneRes"`
	DisplayID1 int     `json:"displayId1"`
}

// CreatureType represents a creature type category
type CreatureType struct {
	Type  int    `json:"type"`
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// BeastFamily represents a beast family (CreatureFamily.dbc) with its NPC count,
// used to sub-filter the Beast creature type on the NPCs page.
type BeastFamily struct {
	Family int    `json:"family"`
	Name   string `json:"name"`
	Count  int    `json:"count"`
}

// CreatureDetail includes a creature with its loot and quests
type CreatureDetail struct {
	*Creature
	Loot         []*LootItem        `json:"loot"`
	StartsQuests []*QuestRelation   `json:"startsQuests"`
	EndsQuests   []*QuestRelation   `json:"endsQuests"`
	Abilities    []*CreatureAbility `json:"abilities"`
	Spawns       []*CreatureSpawn   `json:"spawns"`
}

type CreatureAbility struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Icon        string `json:"icon"`
	Description string `json:"description"`
}

type CreatureSpawn struct {
	MapID int     `json:"mapId"`
	X     float64 `json:"x"`
	Y     float64 `json:"y"`
	Z     float64 `json:"z"`
}

// CreatureTemplateEntry represents a creature for JSON import
type CreatureTemplateEntry struct {
	Entry            int    `json:"entry"`
	Name             string `json:"name"`
	Subname          string `json:"subname"`
	LevelMin         int    `json:"level_min"`
	LevelMax         int    `json:"level_max"`
	HealthMin        int    `json:"health_min"`
	HealthMax        int    `json:"health_max"`
	ManaMin          int    `json:"mana_min"`
	ManaMax          int    `json:"mana_max"`
	CreatureType     int    `json:"creature_type"`
	CreatureRank     int    `json:"creature_rank"`
	Faction          int    `json:"faction"`
	NPCFlags         int    `json:"npc_flags"`
	LootID           int    `json:"loot_id"`
	SkinLootID       int    `json:"skinning_loot_id"`
	PickpocketLootID int    `json:"pickpocket_loot_id"`
}
