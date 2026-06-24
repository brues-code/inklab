import { useState, useEffect } from 'react'

// DataStatusBanner alerts app-wide when InkLab has no locally-built image data
// (icons / zone maps). These ship empty and are built from the user's WoW
// client via the Tools tab, so without this a fresh user just sees placeholder
// icons and missing maps with no explanation.
//
// Dismissal is session-only: if the data is still missing on the next launch,
// the banner returns — it should keep nudging until resolved, but not nag
// within a session.
export function DataStatusBanner({ onGoToTools }) {
    const [missing, setMissing] = useState([])
    const [dismissed, setDismissed] = useState(false)

    useEffect(() => {
        let cancelled = false
        const app = window?.go?.main?.App
        if (!app?.GetDataStatus) return
        app.GetDataStatus()
            .then((s) => {
                if (cancelled || !s) return
                const m = [
                    s.icons === 0 && 'icons',
                    s.maps === 0 && 'zone maps',
                ].filter(Boolean)
                setMissing(m)
            })
            .catch(() => {})
        return () => { cancelled = true }
    }, [])

    if (dismissed || missing.length === 0) return null

    return (
        <div className="flex items-center justify-between gap-4 px-5 py-2 text-sm bg-amber-500/15 border-b border-amber-500/40 text-amber-300">
            <span>
                ⚠️ No <strong>{missing.join(' or ')}</strong> found — InkLab builds these
                from your WoW client. Items will show placeholder icons and NPCs won't
                show a zone map until you import them.
            </span>
            <div className="flex items-center gap-4">
                <button
                    onClick={() => onGoToTools?.()}
                    className="font-semibold underline hover:no-underline"
                >
                    Open Import
                </button>
                <button
                    onClick={() => setDismissed(true)}
                    aria-label="Dismiss"
                    title="Dismiss for this session"
                    className="text-amber-300/70 hover:text-amber-300 text-base leading-none"
                >
                    ✕
                </button>
            </div>
        </div>
    )
}

export default DataStatusBanner
