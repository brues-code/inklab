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

// ZoneNameInfo maps a zone's normalized match key to its id and official
// localized display name. The frontend loads these once so a single <ZoneName>
// component can turn any raw spawn/folder zone string into the proper name.
type ZoneNameInfo struct {
	Key  string `json:"key"`  // zoneKey(folder name) — the normalized lookup key
	ID   int    `json:"id"`
	Name string `json:"name"` // official localized name (AreaTable.dbc)
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
	NpcFlags int    `json:"npcFlags"` // service bitmask (vendor, trainer, banker, …)
}

// ZoneQuest is a quest assigned to a zone (quest_template.ZoneOrSort).
type ZoneQuest struct {
	Entry      int    `json:"entry"`
	Title      string `json:"title"`
	QuestLevel int    `json:"questLevel"`
	MinLevel   int    `json:"minLevel"`
}

// ZoneObject is a game object that spawns in a zone.
type ZoneObject struct {
	Entry    int    `json:"entry"`
	Name     string `json:"name"`
	Type     int    `json:"type"`
	TypeName string `json:"typeName"`
}

// ZoneLoot is one distinct item that drops from any creature or game object
// spawning in the zone, with how many distinct sources drop it and the best
// drop chance across them.
type ZoneLoot struct {
	Entry     int     `json:"entry"`
	Name      string  `json:"name"`
	Quality   int     `json:"quality"`
	IconPath  string  `json:"iconPath"`
	ItemLevel int     `json:"itemLevel"`
	Sources   int     `json:"sources"` // distinct creatures/objects dropping it
	Chance    float64 `json:"chance"`  // best drop chance across sources
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

	Npcs    []*ZoneNpc    `json:"npcs"`
	Quests  []*ZoneQuest  `json:"quests"`
	Objects []*ZoneObject `json:"objects"`

	Spawns       []*ZoneSpawn `json:"spawns"`       // creature spawn markers
	ObjectSpawns []*ZoneSpawn `json:"objectSpawns"` // game-object spawn markers
}
