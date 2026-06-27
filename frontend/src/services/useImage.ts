/**
 * React hooks for loading images via the unified image service, backed by
 * TanStack Query. Images are data URLs produced locally (icons, NPC model/
 * portrait renders, zone maps); caching them in Query dedupes identical loads
 * across the UI (e.g. the same icon in many list rows fetches once) and removes
 * the per-hook fetch effect.
 */
import { useQuery } from '@tanstack/react-query'
import {
    loadImage,
    loadIcon,
    loadNpcModel,
    loadNpcPortrait,
    loadZoneMap,
    loadZoneMinimap,
    type ImageType,
} from './imageService'
import { queryKeys } from '../hooks/queries/keys'

/** Async-image hook state shared by every hook in this module. */
export interface ImageState {
    src: string | null
    loading: boolean
    error: boolean
}

export type { ImageType }

// The loaders resolve to a data URL or null (they catch and never throw), so
// "error" means the query settled with no src.
const imageState = (q: { data?: string | null; isLoading: boolean }): ImageState => {
    const src = q.data ?? null
    return { src, loading: q.isLoading, error: !q.isLoading && !src }
}

/** Load a single image of the given type by name. */
export const useImage = (
    imageType: ImageType,
    name?: string | null,
    remoteUrl: string | null = null,
): ImageState => {
    const q = useQuery({
        queryKey: queryKeys.image(imageType, name),
        queryFn: () => loadImage(imageType, name),
        enabled: !!(name || remoteUrl),
        staleTime: Infinity,
    })
    return imageState(q)
}

/** Load an icon (e.g. 'inv_sword_01'). */
export const useIcon = (iconName?: string | null): ImageState => {
    const q = useQuery({
        queryKey: queryKeys.icon(iconName),
        queryFn: () => loadIcon(iconName),
        enabled: !!iconName,
        staleTime: Infinity,
    })
    return imageState(q)
}

/**
 * Load an NPC model render (fully local: rendered from the client MPQs, no
 * network). Prefers a per-creature render (with weapons) when given an entry.
 * Bumping reloadKey produces a fresh key after a forced image refresh.
 */
export const useNpcModel = (displayId: number, reloadKey = 0, creatureEntry = 0): ImageState => {
    const q = useQuery({
        queryKey: queryKeys.npcModel(displayId, creatureEntry, reloadKey),
        queryFn: () => loadNpcModel(displayId, creatureEntry),
        enabled: !!displayId,
        staleTime: Infinity,
    })
    return imageState(q)
}

/** Load an NPC portrait (head shot) render, fully local. Mirrors useNpcModel. */
export const useNpcPortrait = (
    displayId: number,
    reloadKey = 0,
    creatureEntry = 0,
    generate = true,
): ImageState => {
    const q = useQuery({
        queryKey: queryKeys.npcPortrait(displayId, creatureEntry, generate, reloadKey),
        queryFn: () => loadNpcPortrait(displayId, creatureEntry, generate),
        enabled: !!displayId,
        staleTime: Infinity,
    })
    return imageState(q)
}

/** Load a locally-generated zone map by zone name (e.g. "Elwynn"). */
export const useZoneMap = (zoneName?: string | null, reloadKey = 0): ImageState => {
    const q = useQuery({
        queryKey: queryKeys.zoneMap(zoneName, reloadKey),
        queryFn: () => loadZoneMap(zoneName),
        enabled: !!zoneName,
        staleTime: Infinity,
    })
    return imageState(q)
}

/**
 * Load a locally-generated terrain minimap by zone name. Mirrors useZoneMap;
 * settles with no src when the zone has no minimap, so the UI can fall back to
 * the atlas map.
 */
export const useZoneMinimap = (zoneName?: string | null, reloadKey = 0): ImageState => {
    const q = useQuery({
        queryKey: queryKeys.zoneMinimap(zoneName, reloadKey),
        queryFn: () => loadZoneMinimap(zoneName),
        enabled: !!zoneName,
        staleTime: Infinity,
    })
    return imageState(q)
}
