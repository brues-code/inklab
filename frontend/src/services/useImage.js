/**
 * React Hook for loading images with the unified image service
 */
import { useState, useEffect } from 'react';
import { loadImage, loadIcon, loadNpcModel, loadZoneMap } from './imageService';

/**
 * Hook for loading a single image
 * @param {string} imageType - 'icon' | 'npc_model' | 'npc_map'
 * @param {string} name - Image name
 * @param {string} remoteUrl - Fallback URL
 * @returns {{ src: string | null, loading: boolean, error: boolean }}
 */
export const useImage = (imageType, name, remoteUrl = null) => {
    const [src, setSrc] = useState(null);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState(false);

    useEffect(() => {
        if (!name && !remoteUrl) {
            setLoading(false);
            setError(true);
            return;
        }

        setLoading(true);
        setError(false);

        loadImage(imageType, name, remoteUrl)
            .then(result => {
                if (result) {
                    setSrc(result);
                } else {
                    setError(true);
                }
            })
            .catch(() => {
                setError(true);
            })
            .finally(() => {
                setLoading(false);
            });
    }, [imageType, name, remoteUrl]);

    return { src, loading, error };
};

/**
 * Hook for loading an icon
 * @param {string} iconName - Icon name (e.g., 'inv_sword_01')
 * @returns {{ src: string | null, loading: boolean, error: boolean }}
 */
export const useIcon = (iconName) => {
    const [src, setSrc] = useState(null);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState(false);

    useEffect(() => {
        if (!iconName) {
            setLoading(false);
            setError(true);
            return;
        }

        setLoading(true);
        setError(false);

        loadIcon(iconName)
            .then(result => {
                if (result) {
                    setSrc(result);
                } else {
                    setError(true);
                }
            })
            .catch(() => {
                setError(true);
            })
            .finally(() => {
                setLoading(false);
            });
    }, [iconName]);

    return { src, loading, error };
};

/**
 * Hook for loading an NPC model render, keyed by creature display id.
 * @param {number} displayId - creature display id (display_id1)
 * @param {string} remoteUrl - octowow model URL
 * @returns {{ src: string | null, loading: boolean, error: boolean }}
 */
export const useNpcModel = (displayId, remoteUrl, reloadKey = 0, creatureEntry = 0) => {
    const [src, setSrc] = useState(null);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState(false);

    useEffect(() => {
        if (!displayId) {
            setLoading(false);
            setError(true);
            return;
        }

        setLoading(true);
        setError(false);

        loadNpcModel(displayId, remoteUrl, creatureEntry)
            .then(result => {
                if (result) {
                    setSrc(result);
                } else {
                    setError(true);
                }
            })
            .catch(() => {
                setError(true);
            })
            .finally(() => {
                setLoading(false);
            });
    }, [displayId, remoteUrl, reloadKey, creatureEntry]);

    return { src, loading, error };
};

/**
 * Hook for loading a locally-generated zone map by zone name.
 * @param {string} zoneName - texture-folder name (e.g. "Elwynn")
 * @param {number} reloadKey - bump to force a reload
 * @returns {{ src: string | null, loading: boolean, error: boolean }}
 */
export const useZoneMap = (zoneName, reloadKey = 0) => {
    const [src, setSrc] = useState(null);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState(false);

    useEffect(() => {
        if (!zoneName) {
            setSrc(null);
            setLoading(false);
            setError(true);
            return;
        }
        setLoading(true);
        setError(false);
        loadZoneMap(zoneName)
            .then(result => {
                if (result) setSrc(result);
                else { setSrc(null); setError(true); }
            })
            .catch(() => { setSrc(null); setError(true); })
            .finally(() => setLoading(false));
    }, [zoneName, reloadKey]);

    return { src, loading, error };
};
