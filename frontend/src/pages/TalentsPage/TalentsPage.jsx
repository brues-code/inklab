import { useState, useEffect, useMemo, useCallback } from 'react'
import { useIcon, useImage } from '../../services/useImage'

// Direct Wails bindings (mirrors the codebase convention for app methods).
const GetTalentClasses = () =>
    window?.go?.main?.App?.GetTalentClasses ? window.go.main.App.GetTalentClasses() : Promise.resolve([])
const GetTalentTrees = (cls) =>
    window?.go?.main?.App?.GetTalentTrees ? window.go.main.App.GetTalentTrees(cls) : Promise.resolve(null)

// Class display names + WoW class colors, in the conventional order.
const CLASS_INFO = {
    WARRIOR: { name: 'Warrior', color: '#C79C6E' },
    PALADIN: { name: 'Paladin', color: '#F58CBA' },
    HUNTER: { name: 'Hunter', color: '#ABD473' },
    ROGUE: { name: 'Rogue', color: '#FFF569' },
    PRIEST: { name: 'Priest', color: '#FFFFFF' },
    SHAMAN: { name: 'Shaman', color: '#0070DE' },
    MAGE: { name: 'Mage', color: '#69CCF0' },
    WARLOCK: { name: 'Warlock', color: '#9482C9' },
    DRUID: { name: 'Druid', color: '#FF7D0A' },
}

// Talent tree geometry (a 4-column × up-to-7-row grid drawn over the parchment).
const CELL = 40
const COL_GAP = 16
const ROW_GAP = 14
const PAD_X = 18
const PAD_Y = 16
const GRID_W = 4 * CELL + 3 * COL_GAP + 2 * PAD_X
const ROWS = 7
const GRID_H = ROWS * CELL + (ROWS - 1) * ROW_GAP + 2 * PAD_Y
const MAX_POINTS = 51

const cellLeft = (col) => PAD_X + col * (CELL + COL_GAP)
const cellTop = (row) => PAD_Y + row * (CELL + ROW_GAP)

// ---- small image components -------------------------------------------------

function TalentIcon({ icon }) {
    const { src } = useIcon(icon)
    return (
        <div
            className="w-full h-full bg-cover bg-center"
            style={{ backgroundImage: src ? `url(${src})` : 'none', backgroundColor: '#111' }}
        />
    )
}

// ---- tooltip ----------------------------------------------------------------

function TalentTooltip({ talent, rank, treeName, reqText, x, y }) {
    if (!talent) return null
    const shown = rank > 0 ? rank : 1
    const rankDesc = talent.ranks?.[shown - 1]?.description || ''
    const nextDesc = rank > 0 && rank < talent.maxRank ? talent.ranks?.[rank]?.description : ''
    // Keep the tooltip on-screen.
    const style = {
        left: Math.min(x + 18, (typeof window !== 'undefined' ? window.innerWidth : 1200) - 320),
        top: Math.max(8, y - 20),
    }
    return (
        <div
            className="fixed z-50 w-[300px] rounded border border-zinc-600 bg-black/95 p-2.5 text-sm shadow-xl pointer-events-none"
            style={style}
        >
            <div className="text-wow-gold font-semibold leading-tight">{talent.name || `Talent ${talent.id}`}</div>
            <div className="text-zinc-400 text-xs mb-1">
                Rank {rank}/{talent.maxRank}
            </div>
            {rankDesc && <div className="text-[#ffd100] whitespace-pre-wrap leading-snug">{rankDesc}</div>}
            {nextDesc && (
                <div className="mt-1.5 border-t border-zinc-700 pt-1.5">
                    <div className="text-zinc-400 text-xs">Next rank:</div>
                    <div className="text-zinc-300 whitespace-pre-wrap leading-snug">{nextDesc}</div>
                </div>
            )}
            {reqText && <div className="mt-1.5 text-red-400">{reqText}</div>}
            {!reqText && rank === 0 && (
                <div className="mt-1.5 text-emerald-400/80 text-xs">Click to learn</div>
            )}
        </div>
    )
}

// ---- one talent tree --------------------------------------------------------

function TalentTree({ tree, points, treeSpent, totalSpent, onChange, onHover, onLeave }) {
    const { src: bg } = useImage('talent_bg', tree.background)

    const byId = useMemo(() => {
        const m = {}
        tree.talents.forEach((t) => (m[t.id] = t))
        return m
    }, [tree])

    const prereqMet = (t) => !t.reqTalent || (points[t.reqTalent] || 0) >= t.reqRank + 1
    const cumBelow = (row) =>
        tree.talents.reduce((s, t) => (t.row < row ? s + (points[t.id] || 0) : s), 0)

    const nodeState = (t) => {
        const rank = points[t.id] || 0
        const tierOpen = cumBelow(t.row) >= 5 * t.row
        const ok = prereqMet(t)
        const canInc = rank < t.maxRank && totalSpent < MAX_POINTS && tierOpen && ok
        let cls
        if (rank >= t.maxRank) cls = 'border-emerald-400 shadow-[0_0_8px_rgba(52,211,153,0.55)]'
        else if (rank > 0) cls = 'border-yellow-400'
        else if (canInc) cls = 'border-zinc-300/80'
        else cls = 'border-zinc-700'
        const locked = rank === 0 && !canInc
        return { rank, canInc, locked, cls, tierOpen, ok }
    }

    const reqTextFor = (t, st) => {
        if (st.ok && st.tierOpen) return ''
        const parts = []
        if (!st.tierOpen) parts.push(`Requires ${5 * t.row} points in ${tree.name}`)
        if (!st.ok) {
            const req = byId[t.reqTalent]
            parts.push(`Requires ${req?.name || 'prerequisite'}`)
        }
        return parts.join('\n')
    }

    // Arrows from each prerequisite to its dependent talent.
    const arrows = tree.talents
        .filter((t) => t.reqTalent && byId[t.reqTalent])
        .map((t) => {
            const from = byId[t.reqTalent]
            const active = (points[from.id] || 0) >= t.reqRank + 1
            return { id: t.id, from, to: t, active }
        })

    const arrowColor = (a) => (a.active ? '#ffd100' : '#555')

    return (
        <div className="flex flex-col">
            <div className="flex items-baseline justify-between px-1 mb-1">
                <span className="text-wow-gold font-semibold">{tree.name}</span>
                <span className="text-zinc-300 text-sm tabular-nums">{treeSpent}</span>
            </div>
            <div
                className="relative rounded-md border-2 border-[#6b5a2e] overflow-hidden"
                style={{ width: GRID_W, height: GRID_H }}
            >
                {/* parchment backdrop */}
                <div
                    className="absolute inset-0 bg-cover bg-center"
                    style={{ backgroundImage: bg ? `url(${bg})` : 'none', backgroundColor: '#16130c' }}
                />
                <div className="absolute inset-0 bg-black/35" />

                {/* prerequisite arrows */}
                <svg className="absolute inset-0" width={GRID_W} height={GRID_H}>
                    <defs>
                        <marker id={`ah-on-${tree.id}`} markerWidth="6" markerHeight="6" refX="3" refY="3" orient="auto">
                            <path d="M0,0 L6,3 L0,6 Z" fill="#ffd100" />
                        </marker>
                        <marker id={`ah-off-${tree.id}`} markerWidth="6" markerHeight="6" refX="3" refY="3" orient="auto">
                            <path d="M0,0 L6,3 L0,6 Z" fill="#555" />
                        </marker>
                    </defs>
                    {arrows.map((a) => {
                        const fx = cellLeft(a.from.col) + CELL / 2
                        const fy = cellTop(a.from.row) + CELL / 2
                        const tx = cellLeft(a.to.col) + CELL / 2
                        const ty = cellTop(a.to.row) + CELL / 2
                        // Stop short of the target icon so the arrowhead sits at its edge.
                        const dx = tx - fx
                        const dy = ty - fy
                        const len = Math.hypot(dx, dy) || 1
                        const ex = tx - (dx / len) * (CELL / 2 + 3)
                        const ey = ty - (dy / len) * (CELL / 2 + 3)
                        const c = arrowColor(a)
                        return (
                            <line
                                key={a.id}
                                x1={fx}
                                y1={fy}
                                x2={ex}
                                y2={ey}
                                stroke={c}
                                strokeWidth="2.5"
                                markerEnd={`url(#ah-${a.active ? 'on' : 'off'}-${tree.id})`}
                                opacity={a.active ? 0.95 : 0.5}
                            />
                        )
                    })}
                </svg>

                {/* talent nodes */}
                {tree.talents.map((t) => {
                    const st = nodeState(t)
                    return (
                        <div
                            key={t.id}
                            className="absolute"
                            style={{ left: cellLeft(t.col), top: cellTop(t.row), width: CELL, height: CELL }}
                            onMouseEnter={(e) => onHover(t, st.rank, tree.name, reqTextFor(t, st), e)}
                            onMouseMove={(e) => onHover(t, st.rank, tree.name, reqTextFor(t, st), e)}
                            onMouseLeave={onLeave}
                            onClick={() => st.canInc && onChange(t.id, 1)}
                            onContextMenu={(e) => {
                                e.preventDefault()
                                if (st.rank > 0) onChange(t.id, -1)
                            }}
                        >
                            <div
                                className={`w-full h-full rounded-sm border-2 ${st.cls} overflow-hidden cursor-pointer transition-shadow ${
                                    st.locked ? 'opacity-50 grayscale' : ''
                                }`}
                            >
                                <TalentIcon icon={t.icon} />
                            </div>
                            <div
                                className={`absolute -bottom-1 -right-1 px-1 rounded-sm text-[11px] font-bold leading-tight border border-black/70 tabular-nums ${
                                    st.rank >= t.maxRank ? 'bg-black text-emerald-400' : 'bg-black text-yellow-300'
                                }`}
                            >
                                {st.rank}/{t.maxRank}
                            </div>
                        </div>
                    )
                })}
            </div>
        </div>
    )
}

// ---- page -------------------------------------------------------------------

function TalentsPage() {
    const [classes, setClasses] = useState([])
    const [selected, setSelected] = useState(null)
    const [data, setData] = useState(null)
    const [points, setPoints] = useState({})
    const [loading, setLoading] = useState(false)
    const [tip, setTip] = useState(null)

    useEffect(() => {
        GetTalentClasses().then((list) => {
            const ordered = (list || []).slice().sort(
                (a, b) => Object.keys(CLASS_INFO).indexOf(a) - Object.keys(CLASS_INFO).indexOf(b)
            )
            setClasses(ordered)
            if (ordered.length && !selected) setSelected(ordered[0])
        })
    }, [])

    useEffect(() => {
        if (!selected) return
        setLoading(true)
        GetTalentTrees(selected)
            .then((d) => {
                setData(d)
                setPoints({})
            })
            .finally(() => setLoading(false))
    }, [selected])

    // Per-tree and total point counts.
    const treeSpent = useMemo(() => {
        const m = {}
        if (data?.trees) {
            data.trees.forEach((tr) => {
                m[tr.id] = tr.talents.reduce((s, t) => s + (points[t.id] || 0), 0)
            })
        }
        return m
    }, [data, points])
    const totalSpent = useMemo(
        () => Object.values(treeSpent).reduce((s, n) => s + n, 0),
        [treeSpent]
    )

    const change = useCallback((talentId, delta) => {
        setPoints((prev) => {
            const next = { ...prev, [talentId]: Math.max(0, (prev[talentId] || 0) + delta) }
            if (next[talentId] === 0) delete next[talentId]
            return next
        })
    }, [])

    const onHover = useCallback((talent, rank, treeName, reqText, e) => {
        setTip({ talent, rank, treeName, reqText, x: e.clientX, y: e.clientY })
    }, [])
    const onLeave = useCallback(() => setTip(null), [])

    const info = selected ? CLASS_INFO[selected] : null

    return (
        <div className="h-full overflow-auto bg-bg-dark">
            {/* class selector */}
            <div className="flex flex-wrap gap-2 px-5 py-3 border-b border-border-dark bg-bg-main sticky top-0 z-20">
                {classes.map((c) => {
                    const ci = CLASS_INFO[c] || { name: c, color: '#ccc' }
                    const active = c === selected
                    return (
                        <button
                            key={c}
                            onClick={() => setSelected(c)}
                            className={`px-3 py-1.5 rounded font-semibold text-sm border transition-colors ${
                                active ? 'bg-bg-active border-border-highlight' : 'bg-bg-panel border-border-dark hover:bg-bg-hover'
                            }`}
                            style={{ color: ci.color }}
                        >
                            {ci.name}
                        </button>
                    )
                })}
            </div>

            {/* totals header */}
            <div className="flex items-center justify-between px-5 py-3">
                <h2 className="text-xl font-semibold" style={{ color: info?.color }}>
                    {info?.name} Talents
                </h2>
                <div className="flex items-center gap-4">
                    <span className="text-zinc-300">
                        Points:{' '}
                        <span className={totalSpent >= MAX_POINTS ? 'text-emerald-400' : 'text-wow-gold'}>
                            {totalSpent}
                        </span>
                        <span className="text-zinc-500"> / {MAX_POINTS}</span>
                    </span>
                    {data?.trees && (
                        <span className="text-zinc-500 text-sm tabular-nums">
                            {data.trees.map((tr) => treeSpent[tr.id] || 0).join(' / ')}
                        </span>
                    )}
                    <button
                        onClick={() => setPoints({})}
                        className="px-3 py-1.5 rounded text-sm bg-bg-panel border border-border-dark hover:bg-bg-hover text-zinc-300"
                    >
                        Reset
                    </button>
                </div>
            </div>

            {/* trees */}
            {loading ? (
                <div className="px-5 py-10 text-zinc-500">Loading talents…</div>
            ) : data?.trees ? (
                <div className="flex gap-6 px-5 pb-8 justify-center flex-wrap">
                    {data.trees
                        .slice()
                        .sort((a, b) => a.order - b.order)
                        .map((tree) => (
                            <TalentTree
                                key={tree.id}
                                tree={tree}
                                points={points}
                                treeSpent={treeSpent[tree.id] || 0}
                                totalSpent={totalSpent}
                                onChange={change}
                                onHover={onHover}
                                onLeave={onLeave}
                            />
                        ))}
                </div>
            ) : (
                <div className="px-5 py-10 text-zinc-500">No talent data.</div>
            )}

            {tip && (
                <TalentTooltip
                    talent={tip.talent}
                    rank={tip.rank}
                    treeName={tip.treeName}
                    reqText={tip.reqText}
                    x={tip.x}
                    y={tip.y}
                />
            )}
        </div>
    )
}

export default TalentsPage
