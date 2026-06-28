import { useState } from 'react'
import { useDataStatus } from '../../hooks/queries/app'

// DataStatusBanner alerts app-wide when InkLab is missing any locally-built
// dataset — image data (icons / zone maps / model renders) or a DBC-derived
// reference table. These ship empty and are built from the user's WoW client
// via the Tools tab, so without this a fresh (or partially-imported) user just
// sees placeholder icons, missing maps, or blank reference data with no
// explanation.
//
// It reads the full per-dataset inventory from GetDataStatus and reports every
// dataset whose count is 0, so it stays accurate as datasets are added without
// needing to hand-maintain a list here. It disappears entirely once nothing is
// missing.
//
// Dismissal is session-only: if anything is still missing on the next launch,
// the banner returns — it should keep nudging until resolved, but not nag
// within a session.
export function DataStatusBanner({ onGoToTools }) {
    const [dismissed, setDismissed] = useState(false)

    // One-shot data-presence check, cached for the session.
    const { data: status } = useDataStatus()

    // Every dataset currently empty. Until the status loads, treat nothing as
    // missing so we don't flash the banner on startup.
    const missing = (status?.datasets ?? []).filter((d) => d.count === 0)

    if (dismissed || missing.length === 0) return null

    const labels = missing.map((d) => d.label)
    const summary =
        labels.length <= 3
            ? labels.join(', ')
            : `${labels.slice(0, 3).join(', ')} and ${labels.length - 3} more`

    return (
        <div className="flex items-center justify-between gap-4 border-b border-amber-500/40 bg-amber-500/15 px-5 py-2 text-sm text-amber-300">
            <span>
                ⚠️ Missing data:{' '}
                <strong title={labels.join(', ')}>{summary}</strong> — InkLab builds these from your
                WoW client. Affected pages will show placeholders until you import them.
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
                    className="text-base leading-none text-amber-300/70 hover:text-amber-300"
                >
                    ✕
                </button>
            </div>
        </div>
    )
}

export default DataStatusBanner
