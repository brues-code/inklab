/**
 * React Hook for loading images with the unified image service
 */
import { useState, useEffect } from 'react';
import { loadImage, loadIcon, loadNpcModel, loadNpcPortrait, loadZoneMap } from './imageService';

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
 * Hook for loading an NPC model render (fully local: rendered from the client
 * MPQs, no network). Prefers a per-creature render (with weapons) when given an
 * entry. Sets error=true when nothing local can be produced (UI shows a fallback).
 * @param {number} displayId - creature display id (display_id1)
 * @param {number} reloadKey - bump to force a reload
 * @param {number} creatureEntry - creature entry (for the weaponed per-creature render)
 * @returns {{ src: string | null, loading: boolean, error: boolean }}
 */
export const useNpcModel = (displayId, reloadKey = 0, creatureEntry = 0) => {
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

        loadNpcModel(displayId, creatureEntry)
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
    }, [displayId, reloadKey, creatureEntry]);

    return { src, loading, error };
};

/**
 * Hook for loading an NPC portrait (head shot) render, fully local. Mirrors
 * useNpcModel but loads the display-keyed portrait image. Sets error=true when
 * nothing renderable exists (callers fall back to the full-body model or a
 * placeholder).
 * @param {number} displayId - creature display id (display_id1)
 * @param {number} reloadKey - bump to force a reload
 * @param {number} creatureEntry - creature entry (only used to trigger a render)
 * @returns {{ src: string | null, loading: boolean, error: boolean }}
 */
export const useNpcPortrait = (displayId, reloadKey = 0, creatureEntry = 0, generate = true) => {
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
        loadNpcPortrait(displayId, creatureEntry, generate)
            .then(result => {
                if (result) setSrc(result);
                else setError(true);
            })
            .catch(() => setError(true))
            .finally(() => setLoading(false));
    }, [displayId, reloadKey, creatureEntry, generate]);

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
