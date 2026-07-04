// Scheduling math for Turtle WoW world timers. Everything is computed
// client-side from fixed anchors — no server involved.
//
// Anchor timestamps and periods come from turtletimers.com; the algorithms
// were verified against the leaked Turtle WoW server core (vmangos):
// DungeonResetScheduler::CalculateNextResetTime resets each raid every
// `reset_delay` days at Instance.ResetTimeHour (04:00 UTC), and
// DarkmoonFaire::GetDarkmoonState alternates faction on Sunday-based
// week-of-year parity with Wednesday as the setup day.

export const DAY_MS = 24 * 60 * 60 * 1000

// Timezone note, empirically settled on 2026-07-04: an in-game
// C_DateAndTime.GetServerTimeLocal() dump on Octo shows a UTC+5 wall clock,
// but the faire was observed still in Elwynn hours past UTC+5's Sunday
// midnight — so that offset is display-only and the daemon's localtime()
// (which drives the Darkmoon Faire day/week boundaries) runs in UTC. All
// scheduling here is therefore UTC: raids at 04:00, DMF days at midnight.

const utc = (y: number, m: number, d: number, h = 0) => Date.UTC(y, m - 1, d, h)

export type PeriodicTimer = {
    id: string
    name: string
    detail?: string
    periodDays: number
    anchorMs: number
    note?: string
}

export const RAID_TIMERS: PeriodicTimer[] = [
    {
        id: 'raid40',
        name: 'Raid 40',
        detail: 'MC · BWL · AQ40 · Naxx · ES',
        periodDays: 7,
        anchorMs: utc(2023, 5, 24, 4),
    },
    { id: 'onyxia', name: "Onyxia's Lair", periodDays: 5, anchorMs: utc(2023, 5, 30, 4) },
    {
        id: 'karazhan',
        name: 'Karazhan',
        detail: 'Lower & Upper Halls',
        periodDays: 5,
        anchorMs: utc(2023, 5, 31, 4),
    },
    {
        id: 'raid20',
        name: 'Raid 20',
        detail: 'ZG · AQ20',
        periodDays: 3,
        anchorMs: utc(2023, 5, 28, 4),
    },
    { id: 'timbermaw', name: 'Timbermaw Hold', periodDays: 7, anchorMs: utc(2026, 3, 20, 4) },
]

export const MISC_TIMERS: PeriodicTimer[] = [
    { id: 'honor', name: 'Honor Reset', periodDays: 7, anchorMs: utc(2023, 5, 23, 23) },
    {
        id: 'weeklyQuests',
        name: 'Weekly Quests',
        periodDays: 7,
        anchorMs: utc(2023, 5, 24, 4),
        note: 'Safe pickup time — quests taken right at the actual reset can vanish again (double-reset bug).',
    },
]

/** Next occurrence strictly after `nowMs` of a fixed-period timer. */
export function nextOccurrence(timer: PeriodicTimer, nowMs: number): number {
    const period = timer.periodDays * DAY_MS
    const periodsPassed = Math.floor((nowMs - timer.anchorMs) / period)
    return timer.anchorMs + (periodsPassed + 1) * period
}

const mod = (n: number, len: number) => ((n % len) + len) % len

// Daily battleground rotation; index 0 falls on the anchor day (00:00 UTC).
export const BG_ROTATION = [
    'Alterac Valley',
    'Warsong Gulch',
    'Arathi Basin',
    'Blood Ring Arena',
    'Thorn Gorge',
]
const BG_ANCHOR_MS = utc(2023, 10, 13)

export function battlegroundState(nowMs: number) {
    const daysSince = Math.floor((nowMs - BG_ANCHOR_MS) / DAY_MS)
    return {
        current: BG_ROTATION[mod(daysSince, BG_ROTATION.length)],
        next: BG_ROTATION[mod(daysSince + 1, BG_ROTATION.length)],
        nextChangeMs: BG_ANCHOR_MS + (daysSince + 1) * DAY_MS,
    }
}

// Edge of Madness (Zul'Gurub) boss, rotating every 14 days.
export const EOM_BOSSES = ["Gri'lek", "Hazza'rah", 'Renataki', 'Wushoolay']
const EOM_ANCHOR_MS = utc(2023, 10, 24)
const EOM_PERIOD_DAYS = 14

export function edgeOfMadnessState(nowMs: number) {
    const periodsPassed = Math.floor((nowMs - EOM_ANCHOR_MS) / (EOM_PERIOD_DAYS * DAY_MS))
    return {
        current: EOM_BOSSES[mod(periodsPassed, EOM_BOSSES.length)],
        next: EOM_BOSSES[mod(periodsPassed + 1, EOM_BOSSES.length)],
        nextChangeMs: EOM_ANCHOR_MS + (periodsPassed + 1) * EOM_PERIOD_DAYS * DAY_MS,
    }
}

export type DmfState = {
    faction: 'Alliance' | 'Horde'
    location: string
    town: string
    setupDay: boolean
}

// Mirrors DarkmoonFaire::GetDarkmoonState (HardcodedEvents.cpp): Sunday-based
// week-of-year, even weeks Horde (Mulgore), odd weeks Alliance (Elwynn).
// Because the week number restarts every January, the faction can hold an
// extra stretch across New Year instead of alternating cleanly — that quirk
// is intentional here, it matches the server.
export function dmfStateAt(ms: number): DmfState {
    const d = new Date(ms)
    const yday = Math.floor((ms - Date.UTC(d.getUTCFullYear(), 0, 1)) / DAY_MS)
    const wday = d.getUTCDay()
    const weekOfYear = Math.floor((yday - wday + 7) / 7) + 1
    const horde = weekOfYear % 2 === 0
    return {
        faction: horde ? 'Horde' : 'Alliance',
        location: horde ? 'Mulgore' : 'Elwynn Forest',
        town: horde ? 'Thunder Bluff' : 'Goldshire',
        setupDay: wday === 3,
    }
}

export function dmfSchedule(nowMs: number) {
    const current = dmfStateAt(nowMs)
    const nextMidnight = Math.floor(nowMs / DAY_MS) * DAY_MS + DAY_MS

    // Scan forward from the next UTC midnight; state only changes at day
    // boundaries. Two weeks always contains both a faction move and a
    // Wednesday, even across the New Year parity quirk.
    let moveMs = 0
    let moveState = current
    let reopensMs = 0
    for (let i = 0; i < 15; i++) {
        const t = nextMidnight + i * DAY_MS
        const s = dmfStateAt(t)
        if (!moveMs && s.faction !== current.faction) {
            moveMs = t
            moveState = s
        }
        if (!reopensMs && !s.setupDay) reopensMs = t
        if (moveMs && reopensMs) break
    }

    return { current, moveMs, moveState, reopensMs }
}

export function formatCountdown(ms: number): string {
    if (ms <= 0) return 'now'
    const total = Math.floor(ms / 1000)
    const days = Math.floor(total / 86400)
    const hours = Math.floor((total % 86400) / 3600)
    const minutes = Math.floor((total % 3600) / 60)
    const seconds = total % 60
    if (days > 0) return `${days}d ${hours}h ${minutes}m`
    if (hours > 0) return `${hours}h ${minutes}m ${String(seconds).padStart(2, '0')}s`
    return `${minutes}m ${String(seconds).padStart(2, '0')}s`
}

/** "Wed 8 Jul, 06:00" in the user's local timezone. */
export function formatLocal(ms: number): string {
    return new Date(ms).toLocaleString(undefined, {
        weekday: 'short',
        day: 'numeric',
        month: 'short',
        hour: '2-digit',
        minute: '2-digit',
    })
}
