/**
 * Unified Image Service
 * Handles loading images from local storage with remote fallback
 * Works consistently in both dev and production modes
 */

// Image cache to avoid repeated API calls
const imageCache = new Map();

/**
 * Load an image with local-first strategy
 * @param {string} imageType - 'icon' | 'npc_model' | 'npc_map'
 * @param {string} name - Image name without extension (e.g., 'inv_sword_01', 'model_15114')
 * @param {string} remoteUrl - Fallback remote URL
 * @returns {Promise<string>} - Data URL that can be used as img src
 */
export const loadImage = async (imageType, name, remoteUrl = null) => {
    const cacheKey = `${imageType}:${name}`;
    
    // Check cache first
    if (imageCache.has(cacheKey)) {
        return imageCache.get(cacheKey);
    }

    // Try local first
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

    // Fallback to remote
    if (remoteUrl) {
        if (window?.go?.main?.App?.FetchRemoteImage) {
            try {
                const result = await window.go.main.App.FetchRemoteImage(remoteUrl, imageType, name);
                if (result && result.data && !result.error) {
                    const dataUrl = `data:${result.mimeType};base64,${result.data}`;
                    imageCache.set(cacheKey, dataUrl);
                    return dataUrl;
                }
            } catch (e) {
                console.log(`[ImageService] Remote fetch failed: ${remoteUrl}`);
            }
            // Binding is present and the fetch genuinely failed (e.g. 404 on an
            // octo-custom icon). Return null so callers can fall back rather
            // than handing back a dead URL that renders as a broken image.
            return null;
        }

        // Binding unavailable (pure-browser dev): return the URL so the browser
        // can attempt to load it directly.
        return remoteUrl;
    }

    return null;
};

// Default placeholder icon name (ships locally in data/icons).
const QUESTIONMARK = 'inv_misc_questionmark';

/**
 * Load the generic questionmark placeholder as a real data URL. Used when a
 * requested icon resolves nowhere (e.g. octo-custom icons not local and 404 on
 * the CDN) so callers never have to point at a non-existent /local-icons route.
 * @returns {Promise<string|null>}
 */
export const loadQuestionmarkIcon = async () => {
    const cdnUrl = `https://wow.zamimg.com/images/wow/icons/medium/${QUESTIONMARK}.jpg`;
    return loadImage('icon', QUESTIONMARK, cdnUrl);
};

/**
 * Load an icon with fallback chain. Resolves to the questionmark placeholder
 * (as a data URL) when the named icon can't be found locally or remotely, so
 * the UI always has a usable src instead of a broken image.
 * @param {string} iconName - Icon name (e.g., 'inv_sword_01')
 * @returns {Promise<string|null>} - Image data URL
 */
export const loadIcon = async (iconName) => {
    if (!iconName) return loadQuestionmarkIcon();

    const name = iconName.toLowerCase();
    const cdnUrl = `https://wow.zamimg.com/images/wow/icons/medium/${name}.jpg`;

    const result = await loadImage('icon', name, cdnUrl);
    if (result) return result;

    // Named icon resolved nowhere — fall back to the placeholder.
    return loadQuestionmarkIcon();
};

/**
 * Load an NPC model render, keyed by CreatureDisplayInfo id (octowow serves
 * these at /images/models/<displayId>.png and many NPCs share a display).
 * @param {number} displayId - creature display id (display_id1)
 * @param {string} remoteUrl - octowow model URL
 * @returns {Promise<string>} - Image URL
 */
export const loadNpcModel = async (displayId, remoteUrl, creatureEntry = 0) => {
    // Prefer a per-creature render (body + armor + held weapons) when one exists
    // locally — weapons are creature-specific, so they can't be display-keyed.
    // Falls back to the shared display render (and then octowow) otherwise.
    if (creatureEntry) {
        const local = await loadImage('npc_model', `model_creature_${creatureEntry}`, null);
        if (local) return local;
    }
    return loadImage('npc_model', `model_${displayId}`, remoteUrl);
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
