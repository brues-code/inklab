import { useEffect, useState } from 'react'
import { EventsOn, EventsOff } from '../../../wailsjs/runtime/runtime'
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
    // idle | downloading | restarting | error
    const [phase, setPhase] = useState('idle')
    const [progress, setProgress] = useState(null)
    const [error, setError] = useState('')
    // One-shot remote version check, cached for the session.
    const { data: info } = useUpdateCheck()

    useEffect(() => {
        EventsOn('update:progress', (data) => setProgress(data))
        return () => EventsOff('update:progress')
    }, [])

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

    const applyUpdate = async () => {
        const app = window?.go?.main?.App
        if (!app?.ApplyUpdate) return
        setPhase('downloading')
        setError('')
        try {
            await app.ApplyUpdate()
            setPhase('restarting')
            await app.Restart()
        } catch (e) {
            // Failed mid-way is safe: the running binary is only replaced
            // after a verified download, so a retry or manual download works.
            setError(String(e))
            setPhase('error')
        }
    }

    const pct =
        progress?.total > 0 ? Math.min(100, Math.round((progress.downloaded / progress.total) * 100)) : null

    return (
        <div className="flex items-center justify-between gap-4 border-b border-wow-gold/40 bg-wow-gold/15 px-5 py-2 text-sm text-wow-gold">
            <span>
                {phase === 'downloading' && (
                    <>Downloading <strong>{info.latest}</strong>{pct !== null ? ` — ${pct}%` : '…'}</>
                )}
                {phase === 'restarting' && <>Update installed — restarting…</>}
                {phase === 'error' && (
                    <>
                        Update failed: {error} — you can retry or{' '}
                        <a href={info.url} target="_blank" rel="noreferrer" className="underline">
                            download manually
                        </a>
                        .
                    </>
                )}
                {phase === 'idle' && (
                    <>
                        A new version <strong>{info.latest}</strong> is available — you're on{' '}
                        {info.current}.
                    </>
                )}
            </span>
            <div className="flex items-center gap-4">
                {info.selfUpdate && (phase === 'idle' || phase === 'error') && (
                    <button
                        onClick={applyUpdate}
                        className="font-semibold underline hover:no-underline"
                    >
                        {phase === 'error' ? 'Retry update' : 'Update & restart'}
                    </button>
                )}
                {phase === 'idle' && (
                    <a
                        href={info.url}
                        target="_blank"
                        rel="noreferrer"
                        className={
                            info.selfUpdate
                                ? 'text-wow-gold/70 underline hover:text-wow-gold'
                                : 'font-semibold underline hover:no-underline'
                        }
                    >
                        Download
                    </a>
                )}
                {phase !== 'downloading' && phase !== 'restarting' && (
                    <button
                        onClick={dismiss}
                        aria-label="Dismiss for a week"
                        title="Dismiss for a week"
                        className="text-base leading-none text-wow-gold/70 hover:text-wow-gold"
                    >
                        ✕
                    </button>
                )}
            </div>
        </div>
    )
}

export default UpdateBanner
