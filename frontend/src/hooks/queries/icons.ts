import { useQuery } from '@tanstack/react-query'
import { queryKeys } from './keys'

export interface IconEntry {
    name: string
    itemCount: number
    spellCount: number
}

export interface IconUsageItem {
    entry: number
    name: string
    quality: number
}

export interface IconUsageSpell {
    entry: number
    name: string
    rank: string
}

export interface IconUsage {
    name: string
    items: IconUsageItem[]
    spells: IconUsageSpell[]
}

// window.go guards (rather than wailsjs imports) so the frontend still builds
// before the generated bindings are refreshed — same pattern as services/api.ts.
const ListLocalIcons = (): Promise<IconEntry[]> =>
    window?.go?.main?.App?.ListLocalIcons ? window.go.main.App.ListLocalIcons() : Promise.resolve([])

const GetIconUsage = (name: string): Promise<IconUsage | null> =>
    window?.go?.main?.App?.GetIconUsage
        ? window.go.main.App.GetIconUsage(name)
        : Promise.resolve(null)

/** All unique icons in the local set, with item/spell usage counts. */
export const useLocalIcons = () =>
    useQuery({ queryKey: queryKeys.icons, queryFn: ListLocalIcons, staleTime: Infinity })

/** Every item and spell that renders the named icon. */
export const useIconUsage = (name?: string | null) =>
    useQuery({
        queryKey: queryKeys.iconUsage(name),
        queryFn: () => GetIconUsage(name as string),
        enabled: !!name,
        staleTime: Infinity,
    })
