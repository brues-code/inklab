// Shared API utilities for database components
// These wrap Wails Go bindings with fallbacks

// Detail APIs (consolidated from services/api.js)
export const GetQuestDetail = (entry: number) => {
    console.log(`[API] Fetching Quest Detail for: ${entry}`);
    if (window?.go?.main?.App?.GetQuestDetail) {
        return window.go.main.App.GetQuestDetail(entry)
            .then(res => {
                console.log(`[API] Received Quest Detail for ${entry}:`, res);
                return res;
            })
            .catch(err => {
                console.error(`[API] Failed to get Quest Detail for ${entry}:`, err);
                throw err;
            });
    }
    console.warn(`[API] GetQuestDetail not found in Wails App!`);
    return Promise.resolve(null)
}

export const GetCreatureDetail = (entry: number) => {
    console.log(`[API] Fetching Creature Detail for: ${entry}`);
    if (window?.go?.main?.App?.GetCreatureDetail) {
        return window.go.main.App.GetCreatureDetail(entry);
    }
    return Promise.resolve(null)
}

export const GetItemDetail = (entry: number) => {
    console.log(`[API] Fetching Item Detail for: ${entry}`);
    if (window?.go?.main?.App?.GetItemDetail) {
        return window.go.main.App.GetItemDetail(entry);
    }
    return Promise.resolve(null)
}

export const GetSpellDetail = (entry: number) => {
    console.log(`[API] Fetching Spell Detail for: ${entry}`);
    if (window?.go?.main?.App?.GetSpellDetail) {
        return window.go.main.App.GetSpellDetail(entry);
    }
    return Promise.resolve(null)
}

// Items APIs
export const BrowseItemsByClassAndSlot = (classId: number, subClass: number, inventoryType: number, nameFilter = '') => {
    if (window?.go?.main?.App?.BrowseItemsByClassAndSlot) {
        return window.go.main.App.BrowseItemsByClassAndSlot(classId, subClass, inventoryType, nameFilter)
    }
    return Promise.resolve([])
}

export const GetItemSets = () => {
    if (window?.go?.main?.App?.GetItemSets) {
        return window.go.main.App.GetItemSets()
    }
    return Promise.resolve([])
}

export const GetItemSetDetail = (itemSetId: number) => {
    if (window?.go?.main?.App?.GetItemSetDetail) {
        return window.go.main.App.GetItemSetDetail(itemSetId)
    }
    return Promise.resolve(null)
}

// Creature/NPC APIs
export const GetCreatureTypes = () => {
    if (window?.go?.main?.App?.GetCreatureTypes) {
        return window.go.main.App.GetCreatureTypes()
    }
    return Promise.resolve([])
}

export const BrowseCreaturesByType = (creatureType: number, nameFilter = '') => {
    if (window?.go?.main?.App?.BrowseCreaturesByType) {
        return window.go.main.App.BrowseCreaturesByType(creatureType, nameFilter)
    }
    return Promise.resolve([])
}

// Paginated version of BrowseCreaturesByType
export const BrowseCreaturesByTypePaged = (creatureType: number, nameFilter = '', limit = 100, offset = 0) => {
    if (window?.go?.main?.App?.BrowseCreaturesByTypePaged) {
        return window.go.main.App.BrowseCreaturesByTypePaged(creatureType, nameFilter, limit, offset)
    }
    return Promise.resolve({ creatures: [], total: 0, hasMore: false })
}

export const GetCreatureLoot = (entry: number) => {
    if (window?.go?.main?.App?.GetCreatureLoot) {
        return window.go.main.App.GetCreatureLoot(entry)
    }
    return Promise.resolve([])
}

// Quest APIs
export const GetQuestCategories = () => {
    if (window?.go?.main?.App?.GetQuestCategories) {
        return window.go.main.App.GetQuestCategories()
    }
    return Promise.resolve([])
}

export const GetQuestsByCategory = (categoryId: number) => {
    if (window?.go?.main?.App?.GetQuestsByCategory) {
        return window.go.main.App.GetQuestsByCategory(categoryId)
    }
    return Promise.resolve([])
}

export const SearchQuests = (query: string) => {
    if (window?.go?.main?.App?.SearchQuests) {
        return window.go.main.App.SearchQuests(query)
    }
    return Promise.resolve([])
}

// Object APIs
export const GetObjectTypes = () => {
    if (window?.go?.main?.App?.GetObjectTypes) {
        return window.go.main.App.GetObjectTypes()
    }
    return Promise.resolve([])
}

export const GetObjectsByType = (typeId: number, nameFilter = '') => {
    if (window?.go?.main?.App?.GetObjectsByType) {
        return window.go.main.App.GetObjectsByType(typeId, nameFilter)
    }
    return Promise.resolve([])
}

export const SearchObjects = (query: string) => {
    if (window?.go?.main?.App?.SearchObjects) {
        return window.go.main.App.SearchObjects(query)
    }
    return Promise.resolve([])
}

// Spells APIs
export const SearchSpells = (query: string) => {
    if (window?.go?.main?.App?.SearchSpells) {
        return window.go.main.App.SearchSpells(query)
    }
    return Promise.resolve([])
}

// Factions APIs
export const GetFactions = () => {
    if (window?.go?.main?.App?.GetFactions) {
        return window.go.main.App.GetFactions()
    }
    return Promise.resolve([])
}

// Spell Skills APIs (3-level navigation)
export const GetSpellSkillCategories = () => {
    if (window?.go?.main?.App?.GetSpellSkillCategories) {
        return window.go.main.App.GetSpellSkillCategories()
    }
    return Promise.resolve([])
}

export const GetSpellSkillsByCategory = (categoryId: number) => {
    if (window?.go?.main?.App?.GetSpellSkillsByCategory) {
        return window.go.main.App.GetSpellSkillsByCategory(categoryId)
    }
    return Promise.resolve([])
}

export const GetSpellsBySkill = (skillId: number, nameFilter = '') => {
    if (window?.go?.main?.App?.GetSpellsBySkill) {
        return window.go.main.App.GetSpellsBySkill(skillId, nameFilter)
    }
    return Promise.resolve([])
}

// Class Skills sub-navigation: Class Skills -> Class -> Skill (spec)
export const GetSpellClasses = () => {
    if (window?.go?.main?.App?.GetSpellClasses) {
        return window.go.main.App.GetSpellClasses()
    }
    return Promise.resolve([])
}

export const GetSpellSkillsByClass = (classId: number) => {
    if (window?.go?.main?.App?.GetSpellSkillsByClass) {
        return window.go.main.App.GetSpellSkillsByClass(classId)
    }
    return Promise.resolve([])
}

// Enhanced Quest Categories APIs (3-level navigation)
export const GetQuestCategoryGroups = () => {
    if (window?.go?.main?.App?.GetQuestCategoryGroups) {
        return window.go.main.App.GetQuestCategoryGroups()
    }
    return Promise.resolve([])
}

export const GetQuestCategoriesByGroup = (groupId: number) => {
    if (window?.go?.main?.App?.GetQuestCategoriesByGroup) {
        return window.go.main.App.GetQuestCategoriesByGroup(groupId)
    }
    return Promise.resolve([])
}

export const GetQuestsByEnhancedCategory = (categoryId: number, nameFilter = '') => {
    if (window?.go?.main?.App?.GetQuestsByEnhancedCategory) {
        return window.go.main.App.GetQuestsByEnhancedCategory(categoryId, nameFilter)
    }
    return Promise.resolve([])
}

// Zone APIs
export const GetZones = () => {
    if (window?.go?.main?.App?.GetZones) {
        return window.go.main.App.GetZones()
    }
    return Promise.resolve([])
}

export const GetZoneDetail = (id: number) => {
    if (window?.go?.main?.App?.GetZoneDetail) {
        return window.go.main.App.GetZoneDetail(id)
    }
    return Promise.resolve(null)
}

// Filter helper function
export const filterItems = (items: any[], filter: string) => {
    if (!filter || !filter.trim()) return items || []
    const searchLower = filter.toLowerCase().trim()
    const searchNum = parseInt(filter)
    const isNumericSearch = !isNaN(searchNum)

    return (items || []).filter(item => {
        if (isNumericSearch) {
            if (item.entry === searchNum || item.id === searchNum || item.itemsetId === searchNum) {
                return true
            }
        }
        const name = (item.name || item.title || item.displayName || '').toLowerCase()
        if (name.includes(searchLower)) return true
        if (item.key && item.key.toLowerCase().includes(searchLower)) return true
        return false
    })
}
