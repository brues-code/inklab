package models

// Faction represents a WoW reputation faction
type Faction struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Side        int    `json:"side"` // 1=Alliance, 2=Horde, 3=Both
	CategoryId  int    `json:"categoryId"`
}

// FactionEntry represents a faction for JSON import
type FactionEntry struct {
	FactionID   int    `json:"factionID"`
	Name        string `json:"name_loc0"`
	Description string `json:"description1_loc0"`
	Side        int    `json:"side"`
	Team        int    `json:"team"`
}

// FactionDetail represents detailed faction info for detail view
type FactionDetail struct {
	ID          int              `json:"id"`
	Name        string           `json:"name"`
	Description string           `json:"description"`
	Side        int              `json:"side"` // 1=Alliance, 2=Horde, 3=Both
	SideName    string           `json:"sideName"`
	CategoryId  int              `json:"categoryId"`
	Creatures   []*Creature      `json:"creatures,omitempty"`
	Quests      []*QuestRelation `json:"quests,omitempty"`
	QuestGivers []*FactionNpc    `json:"questGivers,omitempty"` // NPCs offering this faction's rep quests
	Members     []*FactionNpc    `json:"members,omitempty"`     // NPCs belonging to this faction (FactionTemplate)
	// Items gated behind reputation with this faction (required_reputation_faction).
	RequiredByItems []*FactionItemReq `json:"requiredByItems,omitempty"`
}

// FactionItemReq is an item that requires reputation with a faction, with the
// standing needed (e.g. "Revered").
type FactionItemReq struct {
	Entry    int    `json:"entry"`
	Name     string `json:"name"`
	Quality  int    `json:"quality"`
	IconPath string `json:"iconPath"`
	Standing string `json:"standing"`
	Rank     int    `json:"rank"` // raw standing index, for sorting
}

// FactionNpc is a creature associated with a faction (quest giver or member).
type FactionNpc struct {
	Entry    int    `json:"entry"`
	Name     string `json:"name"`
	Subname  string `json:"subname"`
	LevelMin int    `json:"levelMin"`
	LevelMax int    `json:"levelMax"`
}
