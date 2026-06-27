export const CheckForUpdate = () => {
    if (window?.go?.main?.App?.CheckForUpdate) {
        return window.go.main.App.CheckForUpdate().catch((err) => {
            console.error('[API] CheckForUpdate failed:', err)
            return null
        })
    }
    return Promise.resolve(null)
}

export const GetQuestDetail = (entry: number) => {
    console.log(`[API] Fetching Quest Detail for: ${entry}`)
    if (window?.go?.main?.App?.GetQuestDetail) {
        return window.go.main.App.GetQuestDetail(entry)
            .then((res) => {
                console.log(`[API] Received Quest Detail for ${entry}:`, res)
                return res
            })
            .catch((err) => {
                console.error(`[API] Failed to get Quest Detail for ${entry}:`, err)
                throw err
            })
    }
    console.warn(`[API] GetQuestDetail not found in Wails App!`)
    return Promise.resolve(null)
}

export const GetCreatureDetail = (entry: number) => {
    console.log(`[API] Fetching Creature Detail for: ${entry}`)
    if (window?.go?.main?.App?.GetCreatureDetail) {
        return window.go.main.App.GetCreatureDetail(entry)
    }
    return Promise.resolve(null)
}

export const GetNpcFullDetails = (entry: number) => {
    console.log(`[API] Fetching NPC Full Detail for: ${entry}`)
    if (window?.go?.main?.App?.GetNpcDetails) {
        return window.go.main.App.GetNpcDetails(entry)
    }
    return Promise.resolve(null)
}

export const SyncNpcData = (entry: number) => {
    console.log(`[API] Syncing NPC: ${entry}`)
    if (window?.go?.main?.App?.SyncNpcData) {
        return window.go.main.App.SyncNpcData(entry)
    }
    return Promise.resolve(null)
}

// Re-fetch only the model/map images for an NPC (no creature data is replaced).
export const RefreshNpcImages = (entry: number) => {
    console.log(`[API] Refreshing NPC images: ${entry}`)
    if (window?.go?.main?.App?.RefreshNpcImages) {
        return window.go.main.App.RefreshNpcImages(entry)
    }
    return Promise.resolve(null)
}

export const GetItemDetail = (entry: number) => {
    console.log(`[API] Fetching Item Detail for: ${entry}`)
    if (window?.go?.main?.App?.GetItemDetail) {
        return window.go.main.App.GetItemDetail(entry)
    }
    return Promise.resolve(null)
}

export const GetTooltipData = (entry: number) => {
    if (window?.go?.main?.App?.GetTooltipData) {
        return window.go.main.App.GetTooltipData(entry)
    }
    return Promise.resolve(null)
}

export const GetSpellDetail = (entry: number) => {
    console.log(`[API] Fetching Spell Detail for: ${entry}`)
    if (window?.go?.main?.App?.GetSpellDetail) {
        return window.go.main.App.GetSpellDetail(entry)
    }
    return Promise.resolve(null)
}

export const SyncSingleSpell = (spellID: number) => {
    console.log(`[API] Syncing Spell: ${spellID}`)
    if (window?.go?.main?.App?.SyncSingleSpell) {
        return window.go.main.App.SyncSingleSpell(spellID)
    }
    return Promise.resolve(null)
}

// === Favorites API ===

export const AddFavorite = (itemEntry: number, category = '') => {
    console.log(`[API] Adding Favorite: ${itemEntry}, category: ${category}`)
    if (window?.go?.main?.App?.AddFavorite) {
        return window.go.main.App.AddFavorite(itemEntry, category)
    }
    return Promise.resolve({ success: false, message: 'API not available' })
}

export const RemoveFavorite = (itemEntry: number) => {
    console.log(`[API] Removing Favorite: ${itemEntry}`)
    if (window?.go?.main?.App?.RemoveFavorite) {
        return window.go.main.App.RemoveFavorite(itemEntry)
    }
    return Promise.resolve({ success: false, message: 'API not available' })
}

export const IsFavorite = (itemEntry: number) => {
    if (window?.go?.main?.App?.IsFavorite) {
        return window.go.main.App.IsFavorite(itemEntry)
    }
    return Promise.resolve(false)
}

export const GetAllFavorites = () => {
    console.log(`[API] Getting All Favorites`)
    if (window?.go?.main?.App?.GetAllFavorites) {
        return window.go.main.App.GetAllFavorites()
    }
    return Promise.resolve([])
}

export const GetFavoritesByCategory = (category: string) => {
    if (window?.go?.main?.App?.GetFavoritesByCategory) {
        return window.go.main.App.GetFavoritesByCategory(category)
    }
    return Promise.resolve([])
}

export const GetFavoriteCategories = () => {
    if (window?.go?.main?.App?.GetFavoriteCategories) {
        return window.go.main.App.GetFavoriteCategories()
    }
    return Promise.resolve([])
}

export const UpdateFavoriteCategory = (itemEntry: number, category: string) => {
    if (window?.go?.main?.App?.UpdateFavoriteCategory) {
        return window.go.main.App.UpdateFavoriteCategory(itemEntry, category)
    }
    return Promise.resolve({ success: false, message: 'API not available' })
}

export const UpdateFavoriteStatus = (itemEntry: number, status: number) => {
    if (window?.go?.main?.App?.UpdateFavoriteStatus) {
        return window.go.main.App.UpdateFavoriteStatus(itemEntry, status)
    }
    return Promise.resolve({ success: false, message: 'API not available' })
}

export const ToggleFavorite = (itemEntry: number, category = '') => {
    console.log(`[API] Toggle Favorite: ${itemEntry}`)
    if (window?.go?.main?.App?.ToggleFavorite) {
        return window.go.main.App.ToggleFavorite(itemEntry, category)
    }
    return Promise.resolve({ success: false, message: 'API not available' })
}

export const SyncQuestData = (entry: number) => {
    console.log(`[API] Syncing Quest: ${entry}`)
    if (window?.go?.main?.App?.SyncQuestData) {
        return window.go.main.App.SyncQuestData(entry)
    }
    return Promise.resolve(null)
}
