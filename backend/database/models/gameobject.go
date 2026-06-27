package models

// GameObject represents a WoW game object
type GameObject struct {
	Entry     int     `json:"entry"`
	Name      string  `json:"name"`
	Type      int     `json:"type"`
	TypeName  string  `json:"typeName"`
	DisplayID int     `json:"displayId"`
	Size      float64 `json:"size"`
	Data      []int   `json:"data,omitempty"` // For JSON import
}

// ObjectType represents a GO category
type ObjectType struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// LockEntry represents a lock record
type LockEntry struct {
	ID    int `json:"lockID"`
	Type1 int `json:"type1"`
	Type2 int `json:"type2"`
	Type3 int `json:"type3"`
	Type4 int `json:"type4"`
	Type5 int `json:"type5"`
	Prop1 int `json:"lockproperties1"`
	Prop2 int `json:"lockproperties2"`
	Prop3 int `json:"lockproperties3"`
	Prop4 int `json:"lockproperties4"`
	Prop5 int `json:"lockproperties5"`
	Req1  int `json:"requiredskill1"`
	Req2  int `json:"requiredskill2"`
	Req3  int `json:"requiredskill3"`
	Req4  int `json:"requiredskill4"`
	Req5  int `json:"requiredskill5"`
}

// GameObjectDetail represents detailed object info for detail view
type GameObjectDetail struct {
	Entry        int              `json:"entry"`
	Name         string           `json:"name"`
	Type         int              `json:"type"`
	TypeName     string           `json:"typeName"`
	DisplayID    int              `json:"displayId"`
	Faction      int              `json:"faction"`
	Flags        int              `json:"flags"`
	Size         float64          `json:"size"`
	Data0        int              `json:"data0"` // Often loot_id or quest_id
	Data1        int              `json:"data1"`
	GatherSkill    string         `json:"gatherSkill,omitempty"`    // herb/ore/fishing node skill
	GatherSkillReq int            `json:"gatherSkillReq,omitempty"` // required skill level
	StartsQuests []*QuestRelation `json:"startsQuests,omitempty"`
	EndsQuests   []*QuestRelation `json:"endsQuests,omitempty"`
	Contains     []*LootItem      `json:"contains,omitempty"`
	Spawns       []*ObjectSpawn   `json:"spawns,omitempty"`
}

// ObjectSpawn is a single game-object spawn point in map-percentage coords
// (0-100). JSON shape matches the NPC spawn payload so the map-marker UI can be
// shared between the NPC and Object detail views.
type ObjectSpawn struct {
	MapID    int     `json:"mapId"`
	ZoneName string  `json:"zoneName"`
	X        float64 `json:"x"`
	Y        float64 `json:"y"`
}
