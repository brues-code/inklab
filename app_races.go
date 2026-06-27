package main

import (
	"fmt"

	"inklab/backend/database"
)

// GetRaces returns every playable race with its flavor text, available classes
// and racial spells — all assembled from client data. Drives the Races tab.
func (a *App) GetRaces() []*database.Race {
	races, err := a.raceRepo.GetRaces()
	if err != nil {
		fmt.Printf("[API] Error getting races: %v\n", err)
		return []*database.Race{}
	}
	return races
}
