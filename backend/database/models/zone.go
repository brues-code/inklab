package models

// ZoneListEntry is a browsable zone row (one per real areatable zone) with the
// counts that make it worth listing.
type ZoneListEntry struct {
	ID         int    `json:"id"`
	Name       string `json:"name"`
	GroupID    int    `json:"groupId"`
	GroupName  string `json:"groupName"`
	NpcCount   int    `json:"npcCount"`
	QuestCount int    `json:"questCount"`
}

// ZoneNpc is a creature that spawns in a zone.
type ZoneNpc struct {
	Entry    int    `json:"entry"`
	Name     string `json:"name"`
	Subname  string `json:"subname"`
	LevelMin int    `json:"levelMin"`
	LevelMax int    `json:"levelMax"`
	Rank     int    `json:"rank"`
	RankName string `json:"rankName"`
	Type     int    `json:"type"`
	TypeName string `json:"typeName"`
}

// ZoneQuest is a quest assigned to a zone (quest_template.ZoneOrSort).
type ZoneQuest struct {
	Entry      int    `json:"entry"`
	Title      string `json:"title"`
	QuestLevel int    `json:"questLevel"`
	MinLevel   int    `json:"minLevel"`
}

// ZoneSpawn is a single creature spawn point in map-percentage coords (0-100),
// used to plot markers on the zone map.
type ZoneSpawn struct {
	Entry int     `json:"entry"`
	X     float64 `json:"x"`
	Y     float64 `json:"y"`
}

// ZoneDetail is the full payload for the Zone detail view: header info, the map
// key, derived level range, and the NPCs / quests / spawn markers in the zone.
type ZoneDetail struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	GroupName string `json:"groupName"`
	MapName   string `json:"mapName"` // texture-folder name fed to useZoneMap
	MinLevel  int    `json:"minLevel"`
	MaxLevel  int    `json:"maxLevel"`

	Npcs   []*ZoneNpc   `json:"npcs"`
	Quests []*ZoneQuest `json:"quests"`
	Spawns []*ZoneSpawn `json:"spawns"`
}
