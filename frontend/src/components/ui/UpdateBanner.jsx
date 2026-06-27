import { useState } from 'react'
import { useUpdateCheck } from '../../hooks/queries/app'

const SNOOZE_KEY = 'inklab_update_snooze'
const SNOOZE_MS = 7 * 24 * 60 * 60 * 1000 // one week

// Whether the banner for `latest` is currently snoozed. A dismissal hides the
// banner for a week, but a newer release than the one dismissed re-alerts.
function isSnoozed(latest) {
    try {
        const raw = localStorage.getItem(SNOOZE_KEY)
        if (!raw) return false
        const { version, at } = JSON.parse(raw)
        return version === latest && Date.now() - at < SNOOZE_MS
    } catch {
        return false
    }
}

export function UpdateBanner() {
    const [dismissed, setDismissed] = useState(false)
    // One-shot remote version check, cached for the session.
    const { data: info } = useUpdateCheck()

    if (dismissed || !info?.updateAvailable || isSnoozed(info.latest)) return null

    const dismiss = () => {
        try {
            localStorage.setItem(
                SNOOZE_KEY,
                JSON.stringify({ version: info.latest, at: Date.now() }),
            )
        } catch {
            // ignore storage failures; just hide for this session
        }
        setDismissed(true)
    }

    return (
        <div className="flex items-center justify-between gap-4 border-b border-wow-gold/40 bg-wow-gold/15 px-5 py-2 text-sm text-wow-gold">
            <span>
                A new version <strong>{info.latest}</strong> is available — you're on {info.current}
                .
            </span>
            <div className="flex items-center gap-4">
                <a
                    href={info.url}
                    target="_blank"
                    rel="noreferrer"
                    className="font-semibold underline hover:no-underline"
                >
                    Download
                </a>
                <button
                    onClick={dismiss}
                    aria-label="Dismiss for a week"
                    title="Dismiss for a week"
                    className="text-base leading-none text-wow-gold/70 hover:text-wow-gold"
                >
                    ✕
                </button>
            </div>
        </div>
    )
}

export default UpdateBanner
