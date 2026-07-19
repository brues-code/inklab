import { useEffect, useState } from 'react'
import { PageLayout } from '../../components/ui'
import {
    RAID_TIMERS,
    MISC_TIMERS,
    nextOccurrence,
    battlegroundState,
    edgeOfMadnessState,
    dmfSchedule,
    formatCountdown,
    formatLocal,
} from '../../utils/turtleTimers'

function useNow() {
    const [now, setNow] = useState(() => Date.now())
    useEffect(() => {
        const id = setInterval(() => setNow(Date.now()), 1000)
        return () => clearInterval(id)
    }, [])
    return now
}

function TimerCard({ title, subtitle, value, valueClass = '', countdownMs, footer, note }) {
    return (
        <div className="flex flex-col gap-1 rounded border border-border-dark bg-bg-panel p-4">
            <div className="text-sm font-bold uppercase text-wow-gold">{title}</div>
            {subtitle && <div className="text-xs text-white/50">{subtitle}</div>}
            {value && <div className={`mt-1 text-lg font-bold ${valueClass}`}>{value}</div>}
            <div className="mt-1 text-3xl font-bold tabular-nums text-white">
                {formatCountdown(countdownMs)}
            </div>
            {footer && <div className="mt-1 text-xs text-white/60">{footer}</div>}
            {note && <div className="mt-2 text-xs italic text-white/40">{note}</div>}
        </div>
    )
}

function SectionTitle({ children }) {
    return (
        <h2 className="mb-3 mt-6 text-sm font-bold uppercase tracking-wide text-white/60 first:mt-0">
            {children}
        </h2>
    )
}

export default function TimersPage() {
    const now = useNow()

    const bg = battlegroundState(now)
    const eom = edgeOfMadnessState(now)
    const dmf = dmfSchedule(now)

    const dmfFactionClass =
        dmf.current.faction === 'Horde' ? 'text-wow-horde' : 'text-wow-alliance'

    return (
        <PageLayout>
            <div className="flex-1 overflow-y-auto p-6">
                <div className="mx-auto max-w-5xl">
                    <p className="mb-4 text-xs text-white/40">
                        Octo WoW reset schedule, computed locally — raids reset at 04:00 UTC,
                        the Darkmoon Faire moves at midnight UTC, and all times below are
                        shown in your local timezone.
                    </p>

                    <SectionTitle>Raid Resets</SectionTitle>
                    <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
                        {RAID_TIMERS.map((t) => {
                            const resetMs = nextOccurrence(t, now)
                            return (
                                <TimerCard
                                    key={t.id}
                                    title={t.name}
                                    subtitle={`${t.detail ? `${t.detail} — ` : ''}every ${t.periodDays} days`}
                                    countdownMs={resetMs - now}
                                    footer={`Resets ${formatLocal(resetMs)}`}
                                />
                            )
                        })}
                    </div>

                    <SectionTitle>World Events</SectionTitle>
                    <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
                        <TimerCard
                            title="Darkmoon Faire"
                            subtitle="Moves every Wednesday · closed while it moves"
                            value={
                                dmf.current.setupDay
                                    ? `Setup day — reopens in ${dmf.current.location}`
                                    : `${dmf.current.location} (${dmf.current.town})`
                            }
                            valueClass={dmfFactionClass}
                            // On the setup Wednesday the countdown is to tomorrow's
                            // reopening; otherwise to the next relocation.
                            countdownMs={
                                (dmf.current.setupDay ? dmf.reopensMs : dmf.moveMs) - now
                            }
                            footer={
                                dmf.current.setupDay
                                    ? `Reopens ${formatLocal(dmf.reopensMs)}`
                                    : `Moves to ${dmf.moveState.location} ${formatLocal(dmf.moveMs)}`
                            }
                        />
                        <TimerCard
                            title="Battleground of the Day"
                            subtitle="Rotates daily"
                            value={bg.current}
                            countdownMs={bg.nextChangeMs - now}
                            footer={`Next: ${bg.next}, ${formatLocal(bg.nextChangeMs)}`}
                        />
                        <TimerCard
                            title="Edge of Madness"
                            subtitle="Zul'Gurub — rotates every 14 days"
                            value={eom.current}
                            countdownMs={eom.nextChangeMs - now}
                            footer={`Next: ${eom.next}, ${formatLocal(eom.nextChangeMs)}`}
                        />
                        {MISC_TIMERS.map((t) => {
                            const resetMs = nextOccurrence(t, now)
                            return (
                                <TimerCard
                                    key={t.id}
                                    title={t.name}
                                    subtitle={`Every ${t.periodDays} days`}
                                    countdownMs={resetMs - now}
                                    footer={`Resets ${formatLocal(resetMs)}`}
                                    note={t.note}
                                />
                            )
                        })}
                    </div>
                </div>
            </div>
        </PageLayout>
    )
}
