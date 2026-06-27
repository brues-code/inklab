/**
 * Unified Image Service
 * Loads images from the local store only (no network). Model renders are
 * generated on demand from the client MPQs; icons come from the local client set.
 */
import { getClientBasePath } from '../utils/constants'

/** Local image categories understood by the backend GetLocalImage call. */
export type ImageType = 'icon' | 'npc_model' | 'npc_map' | 'zone_map' | 'race_icon'

// Image cache to avoid repeated API calls
const imageCache = new Map<string, string>()

/**
 * Load an image from the LOCAL store only. Images are never fetched from the
 * network — models are rendered locally from the client MPQs and icons come from
 * the locally imported client icon set. Returns null when not present locally.
 */
export const loadImage = async (
    imageType: ImageType,
    name?: string | null,
): Promise<string | null> => {
    const cacheKey = `${imageType}:${name}`

    if (imageCache.has(cacheKey)) {
        return imageCache.get(cacheKey) ?? null
    }

    if (name && window?.go?.main?.App?.GetLocalImage) {
        try {
            const result = await window.go.main.App.GetLocalImage(imageType, name)
            if (result && result.data && !result.error) {
                const dataUrl = `data:${result.mimeType};base64,${result.data}`
                imageCache.set(cacheKey, dataUrl)
                return dataUrl
            }
        } catch (e) {
            console.log(`[ImageService] Local not found: ${name}`)
        }
    }

    return null
}

// Default placeholder icon name (ships locally in data/icons).
const QUESTIONMARK = 'inv_misc_questionmark'

/**
 * Load the generic questionmark placeholder (ships locally in data/icons) as a
 * data URL, so callers always have a usable src instead of a broken image.
 */
export const loadQuestionmarkIcon = async (): Promise<string | null> => {
    return loadImage('icon', QUESTIONMARK)
}

/**
 * Load an icon from the local client icon set, falling back to the questionmark
 * placeholder when the named icon isn't present locally. No network fetching.
 */
export const loadIcon = async (iconName?: string | null): Promise<string | null> => {
    if (!iconName) return loadQuestionmarkIcon()

    const result = await loadImage('icon', iconName.toLowerCase())
    if (result) return result

    // Named icon not in the local set — fall back to the placeholder.
    return loadQuestionmarkIcon()
}

/**
 * Load an NPC model render, fully locally: a per-creature render (with held
 * weapons) when present, else the shared display render, else generated on
 * demand from the client MPQs. No network fetching — returns null when nothing
 * can be produced (e.g. no client configured, or a humanoid without a baked
 * skin), and the UI shows a placeholder.
 */
export const loadNpcModel = async (
    displayId: number,
    creatureEntry = 0,
): Promise<string | null> => {
    // 1. Per-creature render (body + armor + held weapons) — weapons are
    //    creature-specific, so they can't be display-keyed.
    if (creatureEntry) {
        const c = await loadImage('npc_model', `model_creature_${creatureEntry}`)
        if (c) return c
    }
    // 2. Shared display render (cached on disk).
    const d = await loadImage('npc_model', `model_${displayId}`)
    if (d) return d

    // 3. Generate on demand from the client MPQs. The backend writes the file(s)
    //    to disk; we then re-read via the local path so each cache key holds its
    //    own correct image. Uses the configured client path or its default.
    const baseDir = getClientBasePath()
    if (baseDir && window?.go?.main?.App?.RenderNpcModel) {
        try {
            const rendered = await window.go.main.App.RenderNpcModel(
                creatureEntry || 0,
                displayId,
                baseDir,
            )
            if (rendered) {
                if (creatureEntry) {
                    const c2 = await loadImage('npc_model', `model_creature_${creatureEntry}`)
                    if (c2) return c2
                }
                const d2 = await loadImage('npc_model', `model_${displayId}`)
                if (d2) return d2
            }
        } catch (e) {
            // nothing renderable — fall through to null (UI placeholder)
        }
    }

    return null
}

/**
 * Load an NPC portrait (head shot) render, fully locally. Portraits are
 * display-keyed (model_portrait_<displayId>.png); they're written alongside the
 * full-body render, so an on-demand RenderNpcModel produces both. Returns the
 * portrait data URL, or null when nothing renderable exists (UI placeholder).
 */
export const loadNpcPortrait = async (
    displayId: number,
    creatureEntry = 0,
    generate = true,
): Promise<string | null> => {
    if (!displayId) return null
    // 1. Cached portrait on disk.
    const p = await loadImage('npc_model', `model_portrait_${displayId}`)
    if (p) return p

    // 2. Generate on demand from the client MPQs (writes body + portrait), then
    //    re-read the portrait file. Skipped for list thumbnails (generate=false)
    //    so scrolling a long list doesn't kick off hundreds of renders.
    if (!generate) return null
    const baseDir = getClientBasePath()
    if (baseDir && window?.go?.main?.App?.RenderNpcModel) {
        try {
            const rendered = await window.go.main.App.RenderNpcModel(
                creatureEntry || 0,
                displayId,
                baseDir,
            )
            if (rendered) {
                const p2 = await loadImage('npc_model', `model_portrait_${displayId}`)
                if (p2) return p2
            }
        } catch (e) {
            // nothing renderable — fall through to null (UI placeholder)
        }
    }
    return null
}

/**
 * Load a locally-generated zone map by zone name (texture-folder name).
 * Local-only: there is no remote fallback.
 */
export const loadZoneMap = async (zoneName?: string | null): Promise<string | null> => {
    if (!zoneName) return null
    return loadImage('zone_map', zoneName)
}

/**
 * Clear image cache (useful for forcing refresh)
 */
export const clearImageCache = (): void => {
    imageCache.clear()
}

/**
 * Evict a single cached image so the next load re-reads local/remote.
 */
export const evictImage = (imageType: ImageType, name: string): void => {
    imageCache.delete(`${imageType}:${name}`)
}

/**
 * Evict an NPC's cached model and map images.
 */
export const evictNpcImages = (npcId: number): void => {
    imageCache.delete(`npc_model:model_${npcId}`)
    imageCache.delete(`npc_map:map_${npcId}`)
}

/**
 * Preload multiple images in background.
 */
export const preloadImages = async (
    images: Array<{ type: ImageType; name: string; url?: string }>,
): Promise<void> => {
    await Promise.all(images.map((img) => loadImage(img.type, img.name)))
}
