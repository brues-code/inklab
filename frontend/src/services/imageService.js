/**
 * Unified Image Service
 * Loads images from the local store only (no network). Model renders are
 * generated on demand from the client MPQs; icons come from the local client set.
 */
import { getClientBasePath } from '../utils/constants';

// Image cache to avoid repeated API calls
const imageCache = new Map();

/**
 * Load an image from the LOCAL store only. Images are never fetched from the
 * network — models are rendered locally from the client MPQs and icons come from
 * the locally imported client icon set. Returns null when not present locally.
 * @param {string} imageType - 'icon' | 'npc_model' | 'npc_map'
 * @param {string} name - Image name without extension (e.g., 'inv_sword_01', 'model_15114')
 * @returns {Promise<string|null>} - Data URL that can be used as img src, or null
 */
export const loadImage = async (imageType, name) => {
    const cacheKey = `${imageType}:${name}`;

    if (imageCache.has(cacheKey)) {
        return imageCache.get(cacheKey);
    }

    if (window?.go?.main?.App?.GetLocalImage) {
        try {
            const result = await window.go.main.App.GetLocalImage(imageType, name);
            if (result && result.data && !result.error) {
                const dataUrl = `data:${result.mimeType};base64,${result.data}`;
                imageCache.set(cacheKey, dataUrl);
                return dataUrl;
            }
        } catch (e) {
            console.log(`[ImageService] Local not found: ${name}`);
        }
    }

    return null;
};

// Default placeholder icon name (ships locally in data/icons).
const QUESTIONMARK = 'inv_misc_questionmark';

/**
 * Load the generic questionmark placeholder (ships locally in data/icons) as a
 * data URL, so callers always have a usable src instead of a broken image.
 * @returns {Promise<string|null>}
 */
export const loadQuestionmarkIcon = async () => {
    return loadImage('icon', QUESTIONMARK);
};

/**
 * Load an icon from the local client icon set, falling back to the questionmark
 * placeholder when the named icon isn't present locally. No network fetching.
 * @param {string} iconName - Icon name (e.g., 'inv_sword_01')
 * @returns {Promise<string|null>} - Image data URL
 */
export const loadIcon = async (iconName) => {
    if (!iconName) return loadQuestionmarkIcon();

    const result = await loadImage('icon', iconName.toLowerCase());
    if (result) return result;

    // Named icon not in the local set — fall back to the placeholder.
    return loadQuestionmarkIcon();
};

/**
 * Load an NPC model render, fully locally: a per-creature render (with held
 * weapons) when present, else the shared display render, else generated on
 * demand from the client MPQs. No network fetching — returns null when nothing
 * can be produced (e.g. no client configured, or a humanoid without a baked
 * skin), and the UI shows a placeholder.
 * @param {number} displayId - creature display id (display_id1)
 * @param {number} creatureEntry - creature entry (for the weaponed per-creature render)
 * @returns {Promise<string|null>} - Image data URL or null
 */
export const loadNpcModel = async (displayId, creatureEntry = 0) => {
    // 1. Per-creature render (body + armor + held weapons) — weapons are
    //    creature-specific, so they can't be display-keyed.
    if (creatureEntry) {
        const c = await loadImage('npc_model', `model_creature_${creatureEntry}`);
        if (c) return c;
    }
    // 2. Shared display render (cached on disk).
    const d = await loadImage('npc_model', `model_${displayId}`);
    if (d) return d;

    // 3. Generate on demand from the client MPQs. The backend writes the file(s)
    //    to disk; we then re-read via the local path so each cache key holds its
    //    own correct image. Uses the configured client path or its default.
    const baseDir = getClientBasePath();
    if (baseDir && window?.go?.main?.App?.RenderNpcModel) {
        try {
            const rendered = await window.go.main.App.RenderNpcModel(creatureEntry || 0, displayId, baseDir);
            if (rendered) {
                if (creatureEntry) {
                    const c2 = await loadImage('npc_model', `model_creature_${creatureEntry}`);
                    if (c2) return c2;
                }
                const d2 = await loadImage('npc_model', `model_${displayId}`);
                if (d2) return d2;
            }
        } catch (e) {
            // nothing renderable — fall through to null (UI placeholder)
        }
    }

    return null;
};

/**
 * Load an NPC portrait (head shot) render, fully locally. Portraits are
 * display-keyed (model_portrait_<displayId>.png); they're written alongside the
 * full-body render, so an on-demand RenderNpcModel produces both. Returns the
 * portrait data URL, or null when nothing renderable exists (UI placeholder).
 * @param {number} displayId - creature display id (display_id1)
 * @param {number} creatureEntry - creature entry (only used to trigger render)
 * @returns {Promise<string|null>} - Image data URL or null
 */
export const loadNpcPortrait = async (displayId, creatureEntry = 0, generate = true) => {
    if (!displayId) return null;
    // 1. Cached portrait on disk.
    const p = await loadImage('npc_model', `model_portrait_${displayId}`);
    if (p) return p;

    // 2. Generate on demand from the client MPQs (writes body + portrait), then
    //    re-read the portrait file. Skipped for list thumbnails (generate=false)
    //    so scrolling a long list doesn't kick off hundreds of renders.
    if (!generate) return null;
    const baseDir = getClientBasePath();
    if (baseDir && window?.go?.main?.App?.RenderNpcModel) {
        try {
            const rendered = await window.go.main.App.RenderNpcModel(creatureEntry || 0, displayId, baseDir);
            if (rendered) {
                const p2 = await loadImage('npc_model', `model_portrait_${displayId}`);
                if (p2) return p2;
            }
        } catch (e) {
            // nothing renderable — fall through to null (UI placeholder)
        }
    }
    return null;
};

/**
 * Load a locally-generated zone map by zone name (texture-folder name).
 * Local-only: there is no remote fallback.
 * @param {string} zoneName - e.g. "Elwynn", "EasternPlaguelands"
 * @returns {Promise<string|null>} - data URL or null
 */
export const loadZoneMap = async (zoneName) => {
    if (!zoneName) return null;
    return loadImage('zone_map', zoneName, null);
};

/**
 * Clear image cache (useful for forcing refresh)
 */
export const clearImageCache = () => {
    imageCache.clear();
};

/**
 * Evict a single cached image so the next load re-reads local/remote.
 * @param {string} imageType - 'icon' | 'npc_model' | 'npc_map'
 * @param {string} name - Image name (e.g. 'model_15114')
 */
export const evictImage = (imageType, name) => {
    imageCache.delete(`${imageType}:${name}`);
};

/**
 * Evict an NPC's cached model and map images.
 * @param {number} npcId
 */
export const evictNpcImages = (npcId) => {
    imageCache.delete(`npc_model:model_${npcId}`);
    imageCache.delete(`npc_map:map_${npcId}`);
};

/**
 * Preload multiple images in background
 * @param {Array<{type: string, name: string, url: string}>} images
 */
export const preloadImages = async (images) => {
    await Promise.all(
        images.map(img => loadImage(img.type, img.name, img.url))
    );
};
