package models

// Spell represents a WoW spell
type Spell struct {
	Entry       int    `json:"entry"`
	Name        string `json:"name"`
	SubName     string `json:"subname"` // Rank or subtext
	Description string `json:"description"`
	Icon        string `json:"icon"`
}

// SpellSkillCategory represents a top-level category for spells
type SpellSkillCategory struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// SpellSkill represents a skill that contains spells
type SpellSkill struct {
	ID         int    `json:"id"`
	CategoryID int    `json:"categoryId"`
	Name       string `json:"name"`
	SpellCount int    `json:"spellCount"`
}

// SpellClass represents a class grouping under the Class Skills category.
// ID is the WoW class bitmask (1 Warrior ... 256 Warlock, 1024 Druid; 0 General).
type SpellClass struct {
	ID         int    `json:"id"`
	Name       string `json:"name"`
	SkillCount int    `json:"skillCount"`
	Color      string `json:"color,omitempty"`
}

// SpellEntry represents a spell for JSON import
type SpellEntry struct {
	Entry             int    `json:"entry"`
	Name              string `json:"name"`
	Description       string `json:"description"`
	EffectBasePoints1 int    `json:"effectBasePoints1"`
	EffectBasePoints2 int    `json:"effectBasePoints2"`
	EffectBasePoints3 int    `json:"effectBasePoints3"`
	EffectDieSides1   int    `json:"effectDieSides1"`
	EffectDieSides2   int    `json:"effectDieSides2"`
	EffectDieSides3   int    `json:"effectDieSides3"`
	DurationIndex     int    `json:"durationIndex"`
	IconName          string `json:"iconName"`
}

// SpellDurationEntry represents a spell duration record for JSON import
type SpellDurationEntry struct {
	ID               int `json:"id"`
	DurationBase     int `json:"durationBase"`
	DurationPerLevel int `json:"durationPerLevel"`
	MaxDuration      int `json:"maxDuration"`
}
