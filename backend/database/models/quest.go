package models

// Quest represents a WoW quest
type Quest struct {
	Entry            int    `json:"entry"`
	Title            string `json:"title"`
	QuestLevel       int    `json:"questLevel"`
	MinLevel         int    `json:"minLevel"`
	Type             int    `json:"type"`
	ZoneOrSort       int    `json:"zoneOrSort"`
	CategoryName     string `json:"categoryName"`
	RequiredRaces    int    `json:"requiredRaces"`
	RequiredClasses  int    `json:"requiredClasses"`
	SrcItem          int    `json:"srcItemId"`
	RewardXP         int    `json:"rewardXp"`
	RewardMoney      int    `json:"rewardMoney"`
	PrevQuestID      int    `json:"prevQuestId"`
	NextQuestID      int    `json:"nextQuestId"`
	ExclusiveGroup   int    `json:"exclusiveGroup"`
	NextQuestInChain int    `json:"nextQuestInChain"`
}

// QuestCategory represents a zone or category for quests
type QuestCategory struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// QuestDetail includes full quest information with rewards
type QuestDetail struct {
	Entry           int                `json:"entry"`
	Title           string             `json:"title"`
	Details         string             `json:"details"`
	Objectives      string             `json:"objectives"`
	OfferRewardText string             `json:"offerRewardText,omitempty"`
	EndText         string             `json:"endText,omitempty"`
	QuestLevel      int                `json:"questLevel"`
	MinLevel        int                `json:"minLevel"`
	Type            int                `json:"type"`
	ZoneOrSort      int                `json:"zoneOrSort"`
	CategoryName    string             `json:"categoryName"`
	RequiredRaces   int                `json:"requiredRaces,omitempty"`
	Side            string             `json:"side"`
	RaceNames       string             `json:"raceNames"`
	RequiredClasses int                `json:"requiredClasses,omitempty"`
	RewardXP        int                `json:"rewardXp"`
	RewardMoney     int                `json:"rewardMoney"`
	RewardSpell     int                `json:"rewardSpell,omitempty"`
	RewardItems     []*QuestItem       `json:"rewardItems"`
	ChoiceItems     []*QuestItem       `json:"choiceItems"`
	RequiredItems      []*QuestItem       `json:"requiredItems"`
	RequiredObjectives []*QuestObjective  `json:"requiredObjectives"`
	Reputation         []*QuestReputation `json:"reputation"`
	Starters        []*QuestRelation   `json:"starters"`
	Enders          []*QuestRelation   `json:"enders"`
	Series          []*QuestSeriesItem `json:"series"`
	PrevQuests      []*QuestSeriesItem `json:"prevQuests"`
	ExclusiveQuests []*QuestSeriesItem `json:"exclusiveQuests"`
}

// QuestSeriesItem represents a quest in a quest chain
type QuestSeriesItem struct {
	Entry int    `json:"entry"`
	Title string `json:"title"`
	Depth int    `json:"depth"`
}

// QuestItem represents an item reward from a quest
type QuestItem struct {
	Entry   int    `json:"entry"`
	Name    string `json:"name"`
	Icon    string `json:"iconPath"` // Frontend expects iconPath
	Count   int    `json:"count"`
	Quality int    `json:"quality"`
}

// QuestObjective is a kill/interact target required by a quest — a creature or
// gameobject — with the required count.
type QuestObjective struct {
	Entry int    `json:"entry"`
	Name  string `json:"name"`
	Count int    `json:"count"`
	Kind  string `json:"kind"` // "npc" or "object"
}

// QuestReputation represents reputation reward from a quest
type QuestReputation struct {
	FactionID int    `json:"factionId"`
	Name      string `json:"name"`
	Value     int    `json:"value"`
}

// QuestRelation represents an NPC or object related to a quest
type QuestRelation struct {
	Entry int    `json:"entry"`
	Name  string `json:"name"`
	Title string `json:"title,omitempty"` // Quest title (alias for Name in some contexts)
	Level int    `json:"level,omitempty"` // Quest level
	Type  string `json:"type"`            // "npc", "object", "quest", "starts", "ends"
}

// QuestCategoryGroup represents a top-level quest category group
type QuestCategoryGroup struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// QuestCategoryEnhanced represents an enhanced quest category with group info
type QuestCategoryEnhanced struct {
	ID         int    `json:"id"`
	GroupID    int    `json:"groupId"`
	Name       string `json:"name"`
	QuestCount int    `json:"questCount"`
}

// QuestTemplateEntry represents a quest record from JSON (MySQL export)
type QuestTemplateEntry struct {
	Entry               int    `json:"entry"`
	Title               string `json:"Title"`
	MinLevel            int    `json:"MinLevel"`
	QuestLevel          int    `json:"QuestLevel"`
	Type                int    `json:"Type"`
	ZoneOrSort          int    `json:"ZoneOrSort"`
	Details             string `json:"Details"`
	Objectives          string `json:"Objectives"`
	OfferRewardText     string `json:"OfferRewardText"`
	EndText             string `json:"EndText"`
	RewXP               int    `json:"RewXP"`
	RewOrReqMoney       int    `json:"RewOrReqMoney"`
	RewMoneyMaxLevel    int    `json:"RewMoneyMaxLevel"`
	RewSpell            int    `json:"RewSpell"`
	RewItemId1          int    `json:"RewItemId1"`
	RewItemId2          int    `json:"RewItemId2"`
	RewItemId3          int    `json:"RewItemId3"`
	RewItemId4          int    `json:"RewItemId4"`
	RewItemCount1       int    `json:"RewItemCount1"`
	RewItemCount2       int    `json:"RewItemCount2"`
	RewItemCount3       int    `json:"RewItemCount3"`
	RewItemCount4       int    `json:"RewItemCount4"`
	RewChoiceItemId1    int    `json:"RewChoiceItemId1"`
	RewChoiceItemId2    int    `json:"RewChoiceItemId2"`
	RewChoiceItemId3    int    `json:"RewChoiceItemId3"`
	RewChoiceItemId4    int    `json:"RewChoiceItemId4"`
	RewChoiceItemId5    int    `json:"RewChoiceItemId5"`
	RewChoiceItemId6    int    `json:"RewChoiceItemId6"`
	RewChoiceItemCount1 int    `json:"RewChoiceItemCount1"`
	RewChoiceItemCount2 int    `json:"RewChoiceItemCount2"`
	RewChoiceItemCount3 int    `json:"RewChoiceItemCount3"`
	RewChoiceItemCount4 int    `json:"RewChoiceItemCount4"`
	RewChoiceItemCount5 int    `json:"RewChoiceItemCount5"`
	RewChoiceItemCount6 int    `json:"RewChoiceItemCount6"`
	RewRepFaction1      int    `json:"RewRepFaction1"`
	RewRepFaction2      int    `json:"RewRepFaction2"`
	RewRepFaction3      int    `json:"RewRepFaction3"`
	RewRepFaction4      int    `json:"RewRepFaction4"`
	RewRepFaction5      int    `json:"RewRepFaction5"`
	RewRepValue1        int    `json:"RewRepValue1"`
	RewRepValue2        int    `json:"RewRepValue2"`
	RewRepValue3        int    `json:"RewRepValue3"`
	RewRepValue4        int    `json:"RewRepValue4"`
	RewRepValue5        int    `json:"RewRepValue5"`
	PrevQuestId         int    `json:"PrevQuestId"`
	NextQuestId         int    `json:"NextQuestId"`
	ExclusiveGroup      int    `json:"ExclusiveGroup"`
	NextQuestInChain    int    `json:"NextQuestInChain"`
	RequiredRaces       int    `json:"RequiredRaces"`
	RequiredClasses     int    `json:"RequiredClasses"`
	SrcItemId           int    `json:"SrcItemId"`
}
