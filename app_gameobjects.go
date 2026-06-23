package main

import (
	"fmt"
	"inklab/backend/database"
)

// GetObjectTypes returns all object types
func (a *App) GetObjectTypes() []*database.ObjectType {
	fmt.Println("[API] GetObjectTypes called")
	types, err := a.objectRepo.GetObjectTypes()
	if err != nil {
		fmt.Printf("[API] Error getting object types: %v\n", err)
		return []*database.ObjectType{}
	}
	fmt.Printf("[API] GetObjectTypes returning %d types\n", len(types))
	return types
}

// GetObjectsByType returns objects filtered by type
func (a *App) GetObjectsByType(typeID int, nameFilter string) []*database.GameObject {
	fmt.Printf("[API] GetObjectsByType called: type=%d, filter='%s'\n", typeID, nameFilter)
	objects, err := a.objectRepo.GetObjectsByType(typeID, nameFilter)
	if err != nil {
		fmt.Printf("[API] Error browsing objects: %v\n", err)
		return []*database.GameObject{}
	}
	fmt.Printf("[API] GetObjectsByType returning %d objects\n", len(objects))
	return objects
}

// SearchObjects searches for objects by name
func (a *App) SearchObjects(query string) []*database.GameObject {
	objects, err := a.objectRepo.SearchObjects(query)
	if err != nil {
		fmt.Printf("Error searching objects: %v\n", err)
		return []*database.GameObject{}
	}
	return objects
}

// GetObjectDetail returns detailed information about a game object
func (a *App) GetObjectDetail(entry int) *database.GameObjectDetail {
	fmt.Printf("[API] GetObjectDetail called: %d\n", entry)
	detail, err := a.objectRepo.GetObjectDetail(entry)
	if err != nil {
		fmt.Printf("[API] Error getting object detail: %v\n", err)
		return nil
	}
	return detail
}
