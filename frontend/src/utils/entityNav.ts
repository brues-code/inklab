import { useNavigate } from '@tanstack/react-router'

/**
 * Maps an entity type to the Database tab that lists it. Used to pick a
 * sensible originating tab when navigation starts from outside the Database
 * page (global search, favorites, tools) so the Back trail returns to a
 * relevant list.
 */
export const TYPE_TO_TAB: Record<string, string> = {
    item: 'items',
    npc: 'npcs',
    quest: 'quests',
    spell: 'spells',
    object: 'objects',
    zone: 'zones',
    faction: 'factions',
}

export function tabForType(type: string): string {
    return TYPE_TO_TAB[type] || 'items'
}

/**
 * Returns a navigate function with the legacy `(type, entry)` signature used
 * throughout the app. It routes to the Database entity-detail URL, choosing the
 * originating tab from the entity type. Drop-in replacement for the old
 * `onNavigate(type, entry)` callbacks that pushed onto a detail stack.
 */
export function useEntityNavigate() {
    const navigate = useNavigate()
    return (type: string, entry: number | string) =>
        navigate({
            to: '/database/$tab/$type/$id',
            params: { tab: tabForType(type), type, id: String(entry) },
        })
}
