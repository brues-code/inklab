package models

// RaceClass is a class available to a race (id + display name from class_info).
type RaceClass struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// RacialSpell is a race's racial trait resolved to a real spell entry, so the UI
// can link through to the spell page.
type RacialSpell struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Icon string `json:"icon"`
}

// Race is a playable race assembled entirely from client data.
type Race struct {
	ID         int           `json:"id"`
	Name       string        `json:"name"`
	FileString string        `json:"fileString"`
	Prefix     string        `json:"prefix"`
	Faction    string        `json:"faction"`
	Info       string        `json:"info"`
	Abilities  []string      `json:"abilities"` // character-create racial blurbs
	Classes    []RaceClass   `json:"classes"`
	Racials    []RacialSpell `json:"racials"`
}
