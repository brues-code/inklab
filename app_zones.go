package main

import (
	"fmt"

	"inklab/backend/database"
)

// GetZones returns all browsable zones (those with NPCs or quests) with their
// continent/type group and counts.
func (a *App) GetZones() []*database.ZoneListEntry {
	zones, err := a.zoneRepo.GetZones()
	if err != nil {
		fmt.Printf("[API] Error getting zones: %v\n", err)
		return []*database.ZoneListEntry{}
	}
	return zones
}

// GetZoneNames returns every zone's official display name keyed by its
// normalized match key, so the frontend's <ZoneName> component can resolve any
// raw spawn/folder zone string to the localized name in one place.
func (a *App) GetZoneNames() []*database.ZoneNameInfo {
	names, err := a.zoneRepo.ZoneNames()
	if err != nil {
		fmt.Printf("[API] Error getting zone names: %v\n", err)
		return []*database.ZoneNameInfo{}
	}
	return names
}

// GetZoneDetail returns the map, level range, NPCs, quests and spawn markers
// for a single zone.
func (a *App) GetZoneDetail(id int) (*database.ZoneDetail, error) {
	d, err := a.zoneRepo.GetZoneDetail(id)
	if err != nil {
		fmt.Printf("[API] Error getting zone detail [%d]: %v\n", id, err)
		return nil, err
	}
	return d, nil
}
