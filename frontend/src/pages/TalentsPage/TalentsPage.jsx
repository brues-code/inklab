import { useState, useEffect, useMemo, useCallback } from 'react'
import { useIcon, useImage } from '../../services/useImage'

// Direct Wails bindings (mirrors the codebase convention for app methods).
const GetTalentClasses = () =>
    window?.go?.main?.App?.GetTalentClasses ? window.go.main.App.GetTalentClasses() : Promise.resolve([])
const GetTalentTrees = (cls) =>
    window?.go?.main?.App?.GetTalentTrees ? window.go.main.App.GetTalentTrees(cls) : Promise.resolve(null)

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

// ---- talent rules (pure, derived from live `points`) ------------------------
// Kept at module scope so both the tree nodes and the tooltip compute from the
// same source — that's what lets the tooltip refresh live as points change.

const cumPointsBelow = (tree, points, row) =>
    tree.talents.reduce((s, t) => (t.row < row ? s + (points[t.id] || 0) : s), 0)
const prereqMet = (points, t) => !t.reqTalent || (points[t.reqTalent] || 0) >= t.reqRank + 1
const tierOpen = (tree, points, t) => cumPointsBelow(tree, points, t.row) >= 5 * t.row

const reqTextFor = (tree, points, t) => {
    const ok = prereqMet(points, t)
    const open = tierOpen(tree, points, t)
    if (ok && open) return ''
    const parts = []
    if (!open) parts.push(`Requires ${5 * t.row} points in ${tree.name}`)
    if (!ok) {
        const req = tree.talents.find((x) => x.id === t.reqTalent)
        parts.push(`Requires ${req?.name || 'prerequisite'}`)
    }
    return parts.join('\n')
}

// ---- shareable build codes --------------------------------------------------
// Format: "<classId>-<body>", where classId is the WoW class id (Warrior=1 …)
// and body is ONE base62 char per point spent, IN THE ORDER TAKEN. Each char is
// a global talent index into the class's flat talent list (trees by display
// order, talents by row then col). The largest class has 56 talents, which fits
// a single base62 digit (0-61), so one char = one point. Encoding the order —
// not just final ranks like Wowhead — lets a build be replayed level-by-level.
// Example: "11-001a" (Druid; talents 0,0,1,10).
const B62 = '0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz'

// Flat talent list in a stable order: trees by display order, talents by row,col.
const flatTalents = (trees) =>
    [...trees]
        .sort((a, b) => a.order - b.order)
        .flatMap((tr) => [...tr.talents].sort((a, b) => a.row - b.row || a.col - b.col))

function encodeBuild(classId, trees, order) {
    const flat = flatTalents(trees)
    const indexById = {}
    flat.forEach((t, i) => (indexById[t.id] = i))
    let body = ''
    for (const id of order) {
        const i = indexById[id]
        if (i != null && i < B62.length) body += B62[i]
    }
    return `${classId}-${body}`
}

function decodeBuild(trees, body) {
    const flat = flatTalents(trees)
    const order = []
    for (const ch of body) {
        const i = B62.indexOf(ch)
        if (i >= 0 && i < flat.length) order.push(flat[i].id)
    }
    return order
}

// ---- TurtleWoW build import -------------------------------------------------
// turtle's calculator stores each tree as a 28-slot rank array (4 cols x 7 rows,
// slot = row*4 + col), each rank packed as 3 bits, the bytes base64-encoded, and
// the three trees joined by "-" in a ?points= query. This mirrors their decoder.
const turtleBits = (b64) => {
    const bytes = atob(b64)
    let bits = ''
    for (let i = 0; i < bytes.length; i++) bits += bytes.charCodeAt(i).toString(2).padStart(8, '0')
    return (bits.match(/.{1,3}/g) ?? []).map((g) => parseInt(g, 2))
}
function decodeTurtlePoints(points) {
    if (points.endsWith('=')) {
        const all = turtleBits(points)
        return [all.slice(0, 28), all.slice(28, 56), all.slice(56, 84)]
    }
    return points.split('-').slice(0, 3).map((seg) => [...turtleBits(seg.padEnd(14, 'A')), 0])
}

// Encode our build as a TurtleWoW ?points= URL (inverse of the decoder above):
// each tree -> a 28-slot rank array (slot = row*4 + col) -> 3-bit-packed bytes
// -> base64 (drop the final char, trim trailing zero 'A's), trees joined by "-".
function encodeTurtleURL(classKey, trees, points) {
    const sorted = [...trees].sort((a, b) => a.order - b.order)
    const segs = sorted.map((tr) => {
        const arr = new Array(28).fill(0)
        tr.talents.forEach((t) => (arr[t.row * 4 + t.col] = points[t.id] || 0))
        const bits = arr.map((r) => r.toString(2).padStart(3, '0')).join('')
        const bytes = (bits.match(/.{1,8}/g) ?? []).map((b) => parseInt(b, 2))
        return btoa(String.fromCharCode(...bytes)).slice(0, -1).replace(/A+$/, '')
    })
    return `https://talents.turtlecraft.gg/${classKey.toLowerCase()}?points=${segs.join('-')}`
}

// Parse a turtle build (full URL or any string with class + ?points=). Returns
// { classKey, ranks: [tree0[], tree1[], tree2[]] } or null if it isn't one.
function parseTurtle(raw) {
    const pm = raw.match(/points=([^&\s]+)/)
    if (!pm) return null
    const cm = raw.match(/\/([a-zA-Z]+)\s*\?/) || raw.match(/turtlecraft\.gg\/([a-zA-Z]+)/i)
    if (!cm) return null
    try {
        return { classKey: cm[1].toUpperCase(), ranks: decodeTurtlePoints(decodeURIComponent(pm[1])) }
    } catch {
        return null
    }
}

// Synthesize a tier-valid allocation order from a target rank-per-talent map
// (turtle stores only final ranks, not the order). Fills lowest tree/tier first.
function orderFromRanks(trees, treeRanks) {
    const sorted = [...trees].sort((a, b) => a.order - b.order)
    const target = {}
    sorted.forEach((tr, ti) => {
        const arr = treeRanks[ti] || []
        tr.talents.forEach((t) => {
            const r = Math.min(arr[t.row * 4 + t.col] || 0, t.maxRank)
            if (r > 0) target[t.id] = r
        })
    })
    const total = Object.values(target).reduce((s, n) => s + n, 0)
    const cur = {}
    const order = []
    const spent = (tr) => tr.talents.reduce((s, t) => s + (cur[t.id] || 0), 0)
    let guard = 0
    while (order.length < total && guard++ < total * 4 + 20) {
        let pick = null
        for (const tr of sorted) {
            const sp = spent(tr)
            for (const t of [...tr.talents].sort((a, b) => a.row - b.row || a.col - b.col)) {
                if ((cur[t.id] || 0) >= (target[t.id] || 0)) continue
                const tierOpen = sp >= 5 * t.row
                const prereqOk = !t.reqTalent || (cur[t.reqTalent] || 0) >= t.reqRank + 1
                if (tierOpen && prereqOk) { pick = t; break }
            }
            if (pick) break
        }
        if (!pick) break // unreachable target (shouldn't happen for a valid build)
        cur[pick.id] = (cur[pick.id] || 0) + 1
        order.push(pick.id)
    }
    return order
}

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

function TalentTooltip({ talent, rank, reqText, available, x, y }) {
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
            {reqText && <div className="mt-1.5 text-red-400 whitespace-pre-line">{reqText}</div>}
            {!reqText && rank < talent.maxRank && available && (
                <div className="mt-1.5 text-emerald-400/80 text-xs">Click to learn</div>
            )}
        </div>
    )
}

// ---- one talent tree --------------------------------------------------------

// Node border by state: gold = maxed, green = takeable/has points, grey = locked.
const NODE_BORDER = {
    gold: 'border-[#ffd100] shadow-[0_0_8px_rgba(255,209,0,0.55)]',
    green: 'border-emerald-400',
    grey: 'border-zinc-700',
}
const BADGE_TEXT = {
    gold: 'text-[#ffd100]',
    green: 'text-emerald-300',
    grey: 'text-zinc-400',
}

function TalentTree({ tree, points, treeSpent, totalSpent, onChange, onHover, onLeave }) {
    const { src: bg } = useImage('talent_bg', tree.background)

    const byId = useMemo(() => {
        const m = {}
        tree.talents.forEach((t) => (m[t.id] = t))
        return m
    }, [tree])

    const nodeState = (t) => {
        const rank = points[t.id] || 0
        const maxed = rank >= t.maxRank
        const reachable = tierOpen(tree, points, t) && prereqMet(points, t)
        const available = totalSpent < MAX_POINTS
        const canInc = !maxed && available && reachable

        // gold: maxed · green: has points, or empty-but-takeable · grey: rest
        let color
        if (maxed) color = 'gold'
        else if (rank > 0 || (available && reachable)) color = 'green'
        else color = 'grey'

        // Show "x/max" only when the talent has points or you still have points
        // left to spend; an empty talent shows nothing once your pool is spent.
        const showNumbers = rank > 0 || available
        return { rank, canInc, color, showNumbers, dim: color === 'grey' }
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
                            onMouseEnter={(e) => onHover(t, tree, e)}
                            onMouseMove={(e) => onHover(t, tree, e)}
                            onMouseLeave={onLeave}
                            onClick={() => st.canInc && onChange(t.id, 1)}
                            onContextMenu={(e) => {
                                e.preventDefault()
                                if (st.rank > 0) onChange(t.id, -1)
                            }}
                        >
                            <div
                                className={`w-full h-full rounded-sm border-2 ${NODE_BORDER[st.color]} overflow-hidden cursor-pointer transition-shadow ${
                                    st.dim ? 'opacity-50 grayscale' : ''
                                }`}
                            >
                                <TalentIcon icon={t.icon} />
                            </div>
                            {st.showNumbers && (
                                <div
                                    className={`absolute -bottom-1 -right-1 px-1 rounded-sm text-[11px] font-bold leading-tight border border-black/70 tabular-nums bg-black ${BADGE_TEXT[st.color]}`}
                                >
                                    {st.rank}/{t.maxRank}
                                </div>
                            )}
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
    // Allocation order is the source of truth (one talentId per point spent, in
    // the order taken). Ranks are derived from it, and it's what the shareable
    // build code encodes so a build can be replayed level-by-level.
    const [order, setOrder] = useState([])
    const [loading, setLoading] = useState(false)
    const [tip, setTip] = useState(null)
    const [importText, setImportText] = useState('')
    const [importErr, setImportErr] = useState('')
    const [pendingImport, setPendingImport] = useState(null) // { classKey, makeOrder } awaiting trees
    const [copied, setCopied] = useState('') // '' | 'mine' | 'turtle'
    // class token <-> WoW class id, sourced from the backend (not hardcoded)
    const [classMap, setClassMap] = useState({ byId: {}, byClass: {} })

    const points = useMemo(() => {
        const m = {}
        for (const id of order) m[id] = (m[id] || 0) + 1
        return m
    }, [order])

    useEffect(() => {
        GetTalentClasses().then((list) => {
            const arr = list || []
            const byId = {}
            const byClass = {}
            arr.forEach((x) => {
                if (x && x.class) {
                    byId[x.classId] = x.class
                    byClass[x.class] = x.classId
                }
            })
            setClassMap({ byId, byClass })
            // {class, classId, name, color}, sorted alphabetically by name.
            const sorted = [...arr].sort((a, b) => (a.name || a.class).localeCompare(b.name || b.class))
            setClasses(sorted)
            if (sorted.length && !selected) setSelected(sorted[0].class)
        })
    }, [])

    useEffect(() => {
        if (!selected) return
        setLoading(true)
        GetTalentTrees(selected)
            .then((d) => {
                setData(d)
                setOrder([])
            })
            .finally(() => setLoading(false))
    }, [selected])

    // Apply a pending imported build once its class's trees have loaded.
    useEffect(() => {
        if (pendingImport && data?.trees && selected === pendingImport.classKey) {
            setOrder(pendingImport.makeOrder(data.trees))
            setPendingImport(null)
        }
    }, [pendingImport, data, selected])

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

    // Flat talent lookup, and the leveling order: one row per point spent, with
    // the level it was taken (first point is level 10) and the rank reached.
    const talentById = useMemo(() => {
        const m = {}
        data?.trees?.forEach((tr) => tr.talents.forEach((t) => (m[t.id] = t)))
        return m
    }, [data])
    const leveling = useMemo(() => {
        const seen = {}
        return order.map((id, i) => {
            seen[id] = (seen[id] || 0) + 1
            const t = talentById[id]
            return {
                level: i + 10,
                name: t?.name || `Talent ${id}`,
                icon: t?.icon,
                rank: seen[id],
                maxRank: t?.maxRank || 0,
            }
        })
    }, [order, talentById])

    const change = useCallback((talentId, delta) => {
        setOrder((prev) => {
            if (delta > 0) return [...prev, talentId]
            // Remove the most recent point spent on this talent.
            const i = prev.lastIndexOf(talentId)
            if (i === -1) return prev
            const next = prev.slice()
            next.splice(i, 1)
            return next
        })
    }, [])

    // Live shareable build code (class id + ordered allocations).
    const code = useMemo(() => {
        const classId = classMap.byClass[selected]
        return selected && data?.trees && classId != null
            ? encodeBuild(classId, data.trees, order)
            : ''
    }, [selected, data, order, classMap])

    // Equivalent build as a TurtleWoW link (final ranks; order isn't expressible).
    const turtleUrl = useMemo(
        () => (selected && data?.trees ? encodeTurtleURL(selected, data.trees, points) : ''),
        [selected, data, points]
    )

    const copy = useCallback(async (text, tag) => {
        if (!text) return
        try {
            await navigator.clipboard.writeText(text)
        } catch {
            /* clipboard blocked — the code field is selectable as a fallback */
        }
        setCopied(tag)
        setTimeout(() => setCopied(''), 1200)
    }, [])

    const doImport = useCallback(() => {
        const raw = importText.trim()
        if (!raw) return
        // Apply an order builder for a class, loading its trees first if needed.
        const apply = (classKey, makeOrder) => {
            setImportErr('')
            setImportText('')
            if (classKey === selected && data?.trees) setOrder(makeOrder(data.trees))
            else {
                setPendingImport({ classKey, makeOrder })
                setSelected(classKey)
            }
        }

        // TurtleWoW build URL/code (?points=…) — final ranks, order synthesized.
        const turtle = parseTurtle(raw)
        if (turtle) {
            if (!classMap.byClass[turtle.classKey]) {
                setImportErr('Unrecognized build code')
                return
            }
            apply(turtle.classKey, (trees) => orderFromRanks(trees, turtle.ranks))
            return
        }

        // Our own "<classId>-<body>" order-preserving code.
        const dash = raw.indexOf('-')
        if (dash === -1) {
            setImportErr('Unrecognized build code')
            return
        }
        const classKey = classMap.byId[parseInt(raw.slice(0, dash).trim(), 10)]
        const body = raw.slice(dash + 1).trim()
        if (!classKey) {
            setImportErr('Unrecognized build code')
            return
        }
        apply(classKey, (trees) => decodeBuild(trees, body))
    }, [importText, selected, data, classMap])

    // Store only the hovered talent + its tree; rank and requirement text are
    // computed from live `points` at render time so the tooltip refreshes as you
    // add/remove points without needing to move the mouse.
    const onHover = useCallback((talent, tree, e) => {
        setTip({ talent, tree, x: e.clientX, y: e.clientY })
    }, [])
    const onLeave = useCallback(() => setTip(null), [])

    const info = selected ? classes.find((c) => c.class === selected) : null

    return (
        <div className="h-full overflow-auto bg-bg-dark">
            {/* class selector */}
            <div className="flex flex-wrap gap-2 px-5 py-3 border-b border-border-dark bg-bg-main sticky top-0 z-20">
                {classes.map((c) => {
                    const active = c.class === selected
                    return (
                        <button
                            key={c.class}
                            onClick={() => setSelected(c.class)}
                            className={`px-3 py-1.5 rounded font-semibold text-sm border transition-colors ${
                                active ? 'bg-bg-active border-border-highlight' : 'bg-bg-panel border-border-dark hover:bg-bg-hover'
                            }`}
                            style={{ color: c.color || '#ccc' }}
                        >
                            {c.name || c.class}
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
                        onClick={() => setOrder([])}
                        className="px-3 py-1.5 rounded text-sm bg-bg-panel border border-border-dark hover:bg-bg-hover text-zinc-300"
                    >
                        Reset
                    </button>
                </div>
            </div>

            {/* share / import build code */}
            {data?.trees && (
                <div className="flex items-center justify-center gap-2 px-5 pb-3 flex-wrap">
                    <span className="text-xs text-zinc-500">Build:</span>
                    <input
                        readOnly
                        value={code}
                        onFocus={(e) => e.target.select()}
                        className="font-mono text-xs bg-bg-panel border border-border-dark rounded px-2 py-1 text-zinc-300 w-[240px]"
                    />
                    <button
                        onClick={() => copy(code, 'mine')}
                        className="px-2.5 py-1 rounded text-xs bg-bg-panel border border-border-dark hover:bg-bg-hover text-zinc-300"
                    >
                        {copied === 'mine' ? 'Copied!' : 'Copy'}
                    </button>
                    <button
                        onClick={() => copy(turtleUrl, 'turtle')}
                        title="Copy as a TurtleWoW calculator link"
                        className="px-2.5 py-1 rounded text-xs bg-bg-panel border border-border-dark hover:bg-bg-hover text-zinc-300"
                    >
                        {copied === 'turtle' ? 'Copied!' : 'Turtle link'}
                    </button>
                    <span className="mx-1 text-zinc-700">|</span>
                    <input
                        value={importText}
                        onChange={(e) => {
                            setImportText(e.target.value)
                            if (importErr) setImportErr('')
                        }}
                        onKeyDown={(e) => e.key === 'Enter' && doImport()}
                        placeholder="Paste a build code or TurtleWoW link…"
                        className="font-mono text-xs bg-bg-main border border-border-dark rounded px-2 py-1 text-white w-[240px] outline-none focus:border-wow-rare"
                    />
                    <button
                        onClick={doImport}
                        className="px-2.5 py-1 rounded text-xs bg-wow-rare/80 hover:bg-wow-rare text-white font-semibold"
                    >
                        Import
                    </button>
                    {importErr && <span className="text-xs text-red-400">{importErr}</span>}
                </div>
            )}

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

            {/* leveling order: the sequence points were spent in */}
            {leveling.length > 0 && (
                <div className="px-5 pb-12 flex justify-center">
                    <div className="w-full max-w-md">
                        <h3 className="text-wow-gold font-semibold uppercase text-sm mb-2 text-center">
                            Leveling Order
                        </h3>
                        <table className="w-full text-sm border-collapse">
                            <thead>
                                <tr className="text-zinc-500 text-[11px] uppercase tracking-wide">
                                    <th className="text-left font-medium px-2 py-1 w-12">Lvl</th>
                                    <th className="text-left font-medium px-2 py-1" colSpan={2}>Talent</th>
                                </tr>
                            </thead>
                            <tbody>
                                {leveling.map((row, i) => (
                                    <tr key={i} className="border-t border-border-dark/40">
                                        <td className="px-2 py-1 font-mono text-zinc-400 tabular-nums">{row.level}</td>
                                        <td className="px-2 py-1 w-7">
                                            <div className="w-6 h-6 rounded-sm overflow-hidden border border-black/50">
                                                <TalentIcon icon={row.icon} />
                                            </div>
                                        </td>
                                        <td className="px-2 py-1">
                                            <span className="text-zinc-200">{row.name}</span>
                                            <span className="ml-2 text-zinc-500 text-xs tabular-nums">
                                                {row.rank}/{row.maxRank}
                                            </span>
                                        </td>
                                    </tr>
                                ))}
                            </tbody>
                        </table>
                    </div>
                </div>
            )}

            {tip && (
                <TalentTooltip
                    talent={tip.talent}
                    rank={points[tip.talent.id] || 0}
                    reqText={reqTextFor(tip.tree, points, tip.talent)}
                    available={totalSpent < MAX_POINTS}
                    x={tip.x}
                    y={tip.y}
                />
            )}
        </div>
    )
}

export default TalentsPage
