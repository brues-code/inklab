/**
 * React Hook for loading images with the unified image service
 */
import { useState, useEffect } from 'react';
import { loadImage, loadIcon, loadNpcModel, loadNpcPortrait, loadZoneMap, type ImageType } from './imageService';

/** Async-image hook state shared by every hook in this module. */
export interface ImageState {
    src: string | null;
    loading: boolean;
    error: boolean;
}

export type { ImageType };

/**
 * Hook for loading a single image.
 */
export const useImage = (imageType: ImageType, name?: string | null, remoteUrl: string | null = null): ImageState => {
    const [src, setSrc] = useState<string | null>(null);
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

        loadImage(imageType, name)
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
 * Hook for loading an icon (e.g., 'inv_sword_01').
 */
export const useIcon = (iconName?: string | null): ImageState => {
    const [src, setSrc] = useState<string | null>(null);
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
 */
export const useNpcModel = (displayId: number, reloadKey = 0, creatureEntry = 0): ImageState => {
    const [src, setSrc] = useState<string | null>(null);
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
 */
export const useNpcPortrait = (displayId: number, reloadKey = 0, creatureEntry = 0, generate = true): ImageState => {
    const [src, setSrc] = useState<string | null>(null);
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
 * Hook for loading a locally-generated zone map by zone name (e.g. "Elwynn").
 */
export const useZoneMap = (zoneName?: string | null, reloadKey = 0): ImageState => {
    const [src, setSrc] = useState<string | null>(null);
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
