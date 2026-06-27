package main

import (
	"fmt"
	"inklab/backend/database"
)

// GetQuestCategories returns all quest categories (zones and sorts)
func (a *App) GetQuestCategories() []*database.QuestCategory {
	fmt.Println("[API] GetQuestCategories called")
	cats, err := a.questRepo.GetQuestCategories()
	if err != nil {
		fmt.Printf("[API] Error getting quest categories: %v\n", err)
		return []*database.QuestCategory{}
	}
	return cats
}

// GetQuestsByCategory returns quests filtered by category
func (a *App) GetQuestsByCategory(categoryID int) ([]*database.Quest, error) {
	fmt.Printf("[API] GetQuestsByCategory called: %d\n", categoryID)
	quests, err := a.questRepo.GetQuestsByCategory(categoryID)
	if err != nil {
		fmt.Printf("[API] Error getting quests: %v\n", err)
		return nil, err
	}
	return quests, nil
}

// SearchQuests searches for quests by title
func (a *App) SearchQuests(query string) ([]*database.Quest, error) {
	quests, err := a.questRepo.SearchQuests(query)
	if err != nil {
		fmt.Printf("Error searching quests: %v\n", err)
		return nil, err
	}
	return quests, nil
}

// GetQuestDetail returns full details for a quest
func (a *App) GetQuestDetail(entry int) (*database.QuestDetail, error) {
	q, err := a.questRepo.GetQuestDetail(entry)
	if err != nil {
		fmt.Printf("Error getting quest detail [%d]: %v\n", entry, err)
		return nil, err
	}
	return q, nil
}

// GetQuestCategoryGroups returns all quest category groups (Eastern Kingdoms, Kalimdor, etc.)
func (a *App) GetQuestCategoryGroups() []*database.QuestCategoryGroup {
	fmt.Println("[API] GetQuestCategoryGroups called")
	groups, err := a.questRepo.GetQuestCategoryGroups()
	if err != nil {
		fmt.Printf("[API] Error getting quest category groups: %v\n", err)
		return []*database.QuestCategoryGroup{}
	}
	fmt.Printf("[API] Returning %d quest category groups\n", len(groups))
	return groups
}

// GetQuestCategoriesByGroup returns categories for a specific group
func (a *App) GetQuestCategoriesByGroup(groupID int) []*database.QuestCategoryEnhanced {
	fmt.Printf("[API] GetQuestCategoriesByGroup called: %d\n", groupID)
	cats, err := a.questRepo.GetQuestCategoriesByGroup(groupID)
	if err != nil {
		fmt.Printf("[API] Error getting categories: %v\n", err)
		return []*database.QuestCategoryEnhanced{}
	}
	fmt.Printf("[API] Returning %d categories\n", len(cats))
	return cats
}

// GetQuestsByEnhancedCategory returns quests for an enhanced category
func (a *App) GetQuestsByEnhancedCategory(categoryID int, nameFilter string) []*database.Quest {
	fmt.Printf("[API] GetQuestsByEnhancedCategory called: %d, filter=%s\n", categoryID, nameFilter)
	quests, err := a.questRepo.GetQuestsByEnhancedCategory(categoryID, nameFilter)
	if err != nil {
		fmt.Printf("[API] Error getting quests: %v\n", err)
		return []*database.Quest{}
	}
	fmt.Printf("[API] Returning %d quests\n", len(quests))
	return quests
}

// SyncQuestData syncs a single quest from the web. It uses the SAME
// fill/set-policy importer as the full sync (FetchAndImportQuest) so WDB-carried
// fields — Title, Details, Objectives, EndText, QuestLevel, ZoneOrSort — are only
// backfilled when empty and never clobbered by an empty scrape. (The old path
// here blanket-overwrote those columns, wiping e.g. the Objectives text the WDB
// provides but octowow's page omits.)
func (a *App) SyncQuestData(entry int) (*database.QuestDetail, error) {
	fmt.Printf("[API] SyncQuestData called: %d\n", entry)
	if a.syncService == nil {
		return nil, fmt.Errorf("sync service not ready")
	}
	res := a.syncService.FetchAndImportQuest(entry)
	if res != nil && !res.Success {
		fmt.Printf("[API] Error syncing quest %d: %s\n", entry, res.Error)
		return nil, fmt.Errorf("%s", res.Error)
	}
	return a.GetQuestDetail(entry)
}
