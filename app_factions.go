package main

import (
	"fmt"
	"inklab/backend/database"
)

// GetFactions returns all factions
func (a *App) GetFactions() []*database.Faction {
	fmt.Println("[API] GetFactions called")
	factions, err := a.factionRepo.GetFactions()
	if err != nil {
		fmt.Printf("[API] Error getting factions: %v\n", err)
		return []*database.Faction{}
	}
	fmt.Printf("[API] GetFactions returning %d factions\n", len(factions))
	return factions
}

// GetFactionDetail returns detailed information about a faction
func (a *App) GetFactionDetail(id int) *database.FactionDetail {
	fmt.Printf("[API] GetFactionDetail called: %d\n", id)
	detail, err := a.factionRepo.GetFactionDetail(id)
	if err != nil {
		fmt.Printf("[API] Error getting faction detail: %v\n", err)
		return nil
	}
	return detail
}
