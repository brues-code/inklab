package models

// SpellUsedByItem represents an item that uses a spell
type SpellUsedByItem struct {
	Entry       int    `json:"entry"`
	Name        string `json:"name"`
	Quality     int    `json:"quality"`
	IconPath    string `json:"iconPath"`
	TriggerType int    `json:"triggerType"` // 0=Use, 1=Equip, 2=ChanceOnHit
}

// SpellEffectInfo is one decoded spell effect for the detail view. Effect/aura
// names come from the server-source enums (helpers.SpellEffectNames/AuraTypeNames);
// values come straight from spell_template.
type SpellEffectInfo struct {
	Index        int    `json:"index"`
	Effect       string `json:"effect"`
	AuraName     string `json:"auraName,omitempty"`
	Value        string `json:"value,omitempty"`
	Radius       string `json:"radius,omitempty"`
	Mechanic     string `json:"mechanic,omitempty"`
	TriggerSpell int    `json:"triggerSpell,omitempty"`
	// CreatedItem is set for Create Item effects (24): the item this spell crafts.
	CreatedItem *SpellUsedByItem `json:"createdItem,omitempty"`
}

type SpellDetail struct {
	*SpellTemplateFull
	Icon        string             `json:"icon"`
	SchoolName  string             `json:"schoolName"` // localized, from spell_schools (else "")
	ToolTip     string             `json:"toolTip"`
	CastTime    string             `json:"castTime"`
	Range       string             `json:"range"`
	Duration    string             `json:"duration"`
	Power       string             `json:"power"`
	Cooldown    string             `json:"cooldown"`
	GCD         string             `json:"gcd"`
	Proc        string             `json:"proc"` // real proc rate (PPM / %) from the world DB proc tables
	MechanicName string            `json:"mechanicName"` // decoded from spell_mechanics (client)
	DispelType  string             `json:"dispelType"`   // decoded from spell_dispel_types (client)
	Effects     []SpellEffectInfo  `json:"effects,omitempty"`
	// MountDisplayID is the creature display id for a Mounted (aura 78) spell —
	// its misc value is a creature_template entry whose display_id1 is the model.
	// Lets the spell page render the mount model (no Turtle collection data needed).
	MountDisplayID int               `json:"mountDisplayId,omitempty"`
	Flags       []string           `json:"flags,omitempty"`
	UsedByItems []*SpellUsedByItem `json:"usedByItems,omitempty"`
	TaughtByNpcs   []*SpellTrainerNpc `json:"taughtByNpcs,omitempty"`   // trainers (scraped)
	TaughtByItems  []*SpellUsedByItem `json:"taughtByItems,omitempty"`  // recipe items that teach it
	TaughtByQuests []*SpellRewardQuest `json:"taughtByQuests,omitempty"` // quests that reward/teach it
}

// SpellTrainerNpc is an NPC that trains this spell (from npc_trainer_spell).
type SpellTrainerNpc struct {
	Entry    int    `json:"entry"`
	Name     string `json:"name"`
	LevelMin int    `json:"levelMin"`
	LevelMax int    `json:"levelMax"`
}

// SpellRewardQuest is a quest whose reward teaches this spell.
type SpellRewardQuest struct {
	Entry int    `json:"entry"`
	Title string `json:"title"`
	Level int    `json:"level"`
	Side  string `json:"side"` // "Alliance", "Horde", or "Both" (from RequiredRaces)
}
