import { useState, useEffect, useMemo, useCallback, useRef } from 'react'
import { useImage, useIcon } from '../../services/useImage'
import { useFlightContinents, useFlightMap, useFlightZone } from '../../hooks/queries/maps'

// Faction colors.
const C_ALLIANCE = '#3b82f6'
const C_HORDE = '#e0294a'
const C_NEUTRAL = '#ffd100'
const C_BOAT = '#38bdf8'
const C_ZEP = '#c084fc'

// Continent images keep ~1002:668; render each panel at native-ish size and let
// pan/zoom scale. Left-to-right world order (Kalimdor west, Eastern Kingdoms east).
const PANEL_W = 1002
const PANEL_H = 668
const WORLD_ORDER = { Kalimdor: 0, Azeroth: 1 }
// The map-0 WorldMapArea name is "Azeroth", but the continent is Eastern
// Kingdoms. Relabel for display only — "Azeroth" stays the map-image key.
const CONTINENT_LABEL = { Azeroth: 'Eastern Kingdoms' }
const continentLabel = (name) => CONTINENT_LABEL[name] || name

const nodeColor = (n) => (n.alliance && n.horde ? C_NEUTRAL : n.alliance ? C_ALLIANCE : n.horde ? C_HORDE : C_NEUTRAL)
const visibleForFaction = (n, f) => (f === 'alliance' ? n.alliance : f === 'horde' ? n.horde : true)
const transportColor = (t) => (t.type === 'zeppelin' ? C_ZEP : C_BOAT)
const TRANSPORT_ICON = { boat: 'inv_garrison_cargoship', zeppelin: 'ability_mount_gyrocoptor' }
const transportLabel = (t) => (t.type === 'zeppelin' ? 'Zeppelin' : 'Boat')

// ---- pan/zoom viewport -----------------------------------------------------

function PanZoom({ canvasW, canvasH, fitKey, children }) {
    const wrapRef = useRef(null)
    const [t, setT] = useState({ s: 1, x: 0, y: 0 })
    const drag = useRef(null)

    // Fit the canvas into the viewport whenever the canvas (fitKey) changes.
    useEffect(() => {
        const el = wrapRef.current
        if (!el) return
        const vw = el.clientWidth || 1
        const vh = el.clientHeight || 1
        const s = Math.min(vw / canvasW, vh / canvasH) * 0.98
        setT({ s, x: (vw - canvasW * s) / 2, y: (vh - canvasH * s) / 2 })
    }, [fitKey, canvasW, canvasH])

    const onWheel = (e) => {
        e.preventDefault()
        const rect = wrapRef.current.getBoundingClientRect()
        const mx = e.clientX - rect.left
        const my = e.clientY - rect.top
        const ds = e.deltaY < 0 ? 1.15 : 1 / 1.15
        setT((p) => {
            const ns = Math.max(0.2, Math.min(8, p.s * ds))
            return { s: ns, x: mx - (mx - p.x) * (ns / p.s), y: my - (my - p.y) * (ns / p.s) }
        })
    }
    const onDown = (e) => {
        drag.current = { sx: e.clientX, sy: e.clientY, ox: t.x, oy: t.y }
    }
    const onMove = (e) => {
        if (!drag.current) return
        const d = drag.current
        setT((p) => ({ ...p, x: d.ox + (e.clientX - d.sx), y: d.oy + (e.clientY - d.sy) }))
    }
    const onUp = () => (drag.current = null)

    return (
        <div
            ref={wrapRef}
            className="relative w-full overflow-hidden rounded-md border border-border-dark bg-black/40 select-none"
            style={{ height: '78vh', cursor: drag.current ? 'grabbing' : 'grab' }}
            onWheel={onWheel}
            onMouseDown={onDown}
            onMouseMove={onMove}
            onMouseUp={onUp}
            onMouseLeave={onUp}
        >
            <div style={{ transform: `translate(${t.x}px, ${t.y}px) scale(${t.s})`, transformOrigin: '0 0', width: canvasW, height: canvasH }}>
                {children}
            </div>
        </div>
    )
}

// ---- shared marker pieces --------------------------------------------------

function PanelImage({ name, x }) {
    const { src } = useImage('zone_map', name)
    return (
        <img
            src={src || undefined}
            alt={name}
            draggable={false}
            className="absolute top-0"
            style={{ left: x, width: PANEL_W, height: PANEL_H, objectFit: 'fill', background: '#16130c' }}
        />
    )
}

function TransportMarker({ t, cx, cy, onEnter, onMove, onLeave }) {
    const { src } = useIcon(TRANSPORT_ICON[t.type] || TRANSPORT_ICON.boat)
    return (
        <div
            className="absolute -translate-x-1/2 -translate-y-1/2 cursor-pointer rounded-sm overflow-hidden shadow"
            style={{ left: cx, top: cy, width: 22, height: 22, border: `2px solid ${transportColor(t)}`, background: '#000' }}
            onMouseEnter={(e) => onEnter(t, e)}
            onMouseMove={(e) => onMove(t, e)}
            onMouseLeave={onLeave}
            onMouseDown={(e) => e.stopPropagation()}
        >
            {src && <img src={src} alt="" className="w-full h-full object-cover" draggable={false} />}
        </div>
    )
}

// ---- page ------------------------------------------------------------------

function MapsPage() {
    const [view, setView] = useState('world') // 'world' | mapId
    const [faction, setFaction] = useState('all')
    const [layers, setLayers] = useState({ flights: true, transports: true })
    const [hover, setHover] = useState(null) // flight node hover
    const [tHover, setTHover] = useState(null) // transport hover
    const [zone, setZone] = useState(null) // {mapId, key} drill-down, or null

    const isWorld = view === 'world'

    const continentsQuery = useFlightContinents()
    const continents = continentsQuery.data || []

    // One query per view (world overview or a continent's flight data).
    const mapDataQuery = useFlightMap(view)
    const worldData = isWorld ? mapDataQuery.data : null
    const contData = isWorld ? null : mapDataQuery.data
    const loading = mapDataQuery.isLoading

    // World order + per-map x offset for the combined canvas.
    const worldPanels = useMemo(() => {
        const cs = (worldData?.continents || continents).slice()
        cs.sort((a, b) => (WORLD_ORDER[a.name] ?? 9 + a.mapId) - (WORLD_ORDER[b.name] ?? 9 + b.mapId))
        return cs.map((c, i) => ({ ...c, x: i * PANEL_W }))
    }, [worldData, continents])
    const offsetByMap = useMemo(() => {
        const m = {}
        worldPanels.forEach((p) => (m[p.mapId] = p.x))
        return m
    }, [worldPanels])

    const canvasW = isWorld ? Math.max(PANEL_W, worldPanels.length * PANEL_W) : PANEL_W
    const canvasH = PANEL_H

    // Canvas-space coordinate for a node/point.
    const cx = (mapId, px) => (isWorld ? (offsetByMap[mapId] || 0) : 0) + (px / 100) * PANEL_W
    const cy = (py) => (py / 100) * PANEL_H

    // Active data set normalized to {nodes, connections, transports}.
    const model = useMemo(() => {
        if (isWorld) {
            const d = worldData || {}
            const byId = {}
            ;(d.nodes || []).forEach((n) => (byId[n.id] = n))
            return {
                nodes: d.nodes || [],
                nodesById: byId,
                connections: (d.connections || []).map((c) => ({ a: byId[c.from], b: byId[c.to] })).filter((c) => c.a && c.b),
                transports: (d.transports || []).map((t, i) => ({
                    id: `w${i}`,
                    type: t.type,
                    x1: cx(t.aMap, t.aPx),
                    y1: cy(t.aPy),
                    x2: cx(t.bMap, t.bPx),
                    y2: cy(t.bPy),
                    here: t.aName,
                    dest: t.bName,
                    sameContinent: t.aMap === t.bMap,
                })),
            }
        }
        const d = contData || {}
        const byId = {}
        ;(d.nodes || []).forEach((n) => (byId[n.id] = n))
        return { nodes: d.nodes || [], nodesById: byId, connections: null, raw: d }
    }, [isWorld, worldData, contData, offsetByMap])

    const visibleNodes = useMemo(() => model.nodes.filter((n) => visibleForFaction(n, faction)), [model, faction])
    const visibleIds = useMemo(() => new Set(visibleNodes.map((n) => n.id)), [visibleNodes])

    // Connections as canvas line segments.
    const lines = useMemo(() => {
        if (isWorld) {
            return model.connections
                .filter((c) => visibleIds.has(c.a.id) && visibleIds.has(c.b.id))
                .map((c) => ({ x1: cx(c.a.mapId, c.a.px), y1: cy(c.a.py), x2: cx(c.b.mapId, c.b.px), y2: cy(c.b.py), from: c.a.id, to: c.b.id }))
        }
        const d = model.raw || {}
        const seen = new Set()
        const out = []
        for (const c of d.connections || []) {
            if (!visibleIds.has(c.from) || !visibleIds.has(c.to)) continue
            const key = c.from < c.to ? `${c.from}-${c.to}` : `${c.to}-${c.from}`
            if (seen.has(key)) continue
            seen.add(key)
            const a = model.nodesById[c.from]
            const b = model.nodesById[c.to]
            if (a && b) out.push({ x1: cx(0, a.px), y1: cy(a.py), x2: cx(0, b.px), y2: cy(b.py), from: c.from, to: c.to })
        }
        return out
    }, [isWorld, model, visibleIds])

    const hoverDests = useMemo(() => {
        if (!hover) return new Set()
        const s = new Set()
        const conns = isWorld ? (worldData?.connections || []) : (contData?.connections || [])
        conns.forEach((c) => {
            if ((c.from === hover.node.id || c.to === hover.node.id)) {
                const other = c.from === hover.node.id ? c.to : c.from
                if (visibleIds.has(other)) s.add(other)
            }
        })
        return s
    }, [hover, isWorld, worldData, contData, visibleIds])

    // Continent-mode transports (on-continent waypoint legs + embarkation marker).
    const contTransports = useMemo(() => {
        if (isWorld) return []
        const nodes = model.nodes
        return (model.raw?.transports || [])
            .filter((t) => t.waypoints?.length)
            .map((t, i) => {
                let best = t.waypoints[0]
                let bestD = Infinity
                for (const w of t.waypoints) {
                    for (const n of nodes) {
                        const d = (n.px - w[0]) ** 2 + (n.py - w[1]) ** 2
                        if (d < bestD) { bestD = d; best = w }
                    }
                }
                return { ...t, id: `c${i}`, marker: best }
            })
    }, [isWorld, model])

    const onEnter = useCallback((node, e) => setHover({ node, x: e.clientX, y: e.clientY }), [])
    const onMove = useCallback((node, e) => setHover((h) => ({ ...(h || {}), node, x: e.clientX, y: e.clientY })), [])
    const onLeave = useCallback(() => setHover(null), [])
    const tEnter = useCallback((tr, e) => setTHover({ ...tr, x: e.clientX, y: e.clientY }), [])
    const tMove = useCallback((tr, e) => setTHover((h) => ({ ...(h || tr), x: e.clientX, y: e.clientY })), [])
    const tLeave = useCallback(() => setTHover(null), [])

    const tabBtn = (id, label, active, color) => (
        <button
            key={id}
            onClick={() => { setZone(null); setView(id) }}
            className={`px-3 py-1.5 rounded font-semibold text-sm border transition-colors ${
                active ? 'bg-bg-active border-border-highlight text-wow-gold' : 'bg-bg-panel border-border-dark hover:bg-bg-hover text-zinc-300'
            }`}
            style={color ? { color } : undefined}
        >
            {label}
        </button>
    )
    const factionBtn = (id, label, color) => (
        <button onClick={() => setFaction(id)} className={`px-3 py-1.5 rounded text-sm font-semibold border transition-colors ${faction === id ? 'bg-bg-active border-border-highlight' : 'bg-bg-panel border-border-dark hover:bg-bg-hover'}`} style={{ color }}>
            {label}
        </button>
    )
    const layerChip = (id, label, color) => (
        <button onClick={() => setLayers((l) => ({ ...l, [id]: !l[id] }))} className={`px-2.5 py-1 rounded text-xs font-semibold border flex items-center gap-1.5 ${layers[id] ? 'bg-bg-active border-border-highlight' : 'bg-bg-panel border-border-dark opacity-50'}`}>
            <span className="inline-block w-2.5 h-2.5 rounded-full" style={{ background: color }} />
            {label}
        </button>
    )

    const transports = isWorld ? model.transports : contTransports

    return (
        <div className="h-full overflow-hidden flex flex-col bg-bg-dark">
            {/* controls */}
            <div className="flex flex-wrap items-center gap-4 px-5 py-3 border-b border-border-dark bg-bg-main">
                <div className="flex gap-2">
                    {tabBtn('world', 'World', isWorld)}
                    {continents.map((c) => tabBtn(c.mapId, continentLabel(c.name), view === c.mapId))}
                </div>
                <div className="flex gap-2">
                    {factionBtn('all', 'All', '#e5e7eb')}
                    {factionBtn('alliance', 'Alliance', C_ALLIANCE)}
                    {factionBtn('horde', 'Horde', C_HORDE)}
                </div>
                <div className="flex gap-1.5">
                    {layerChip('flights', 'Flights', C_NEUTRAL)}
                    {layerChip('transports', 'Transports', C_BOAT)}
                </div>
                <span className="ml-auto text-xs text-zinc-500">scroll to zoom · drag to pan</span>
            </div>

            {/* map */}
            <div className="flex-1 p-3">
                {zone ? (
                    <ZoneView
                        mapId={zone.mapId}
                        zoneKey={zone.key}
                        backLabel={continentLabel(continents.find((c) => c.mapId === zone.mapId)?.name) || 'map'}
                        onBack={() => setZone(null)}
                    />
                ) : loading ? (
                    <div className="py-10 text-zinc-500 text-center">Loading map…</div>
                ) : continents.length === 0 ? (
                    <div className="py-10 text-zinc-500 text-center">
                        No map data found.
                        <div className="text-xs mt-1">If you just updated InkLab, restart the app; otherwise run a Client Data import.</div>
                    </div>
                ) : (
                    <PanZoom canvasW={canvasW} canvasH={canvasH} fitKey={view}>
                        {/* continent images */}
                        {isWorld
                            ? worldPanels.map((p) => <PanelImage key={p.mapId} name={p.name} x={p.x} />)
                            : <PanelImage key={view} name={continents.find((c) => c.mapId === view)?.name} x={0} />}
                        <div className="absolute inset-0 bg-black/25" style={{ width: canvasW, height: canvasH }} />

                        {/* lines */}
                        <svg className="absolute inset-0" width={canvasW} height={canvasH}>
                            {layers.flights &&
                                lines.map((l, i) => {
                                    const hot = hover && (l.from === hover.node.id || l.to === hover.node.id)
                                    return (
                                        <line
                                            key={i}
                                            x1={l.x1}
                                            y1={l.y1}
                                            x2={l.x2}
                                            y2={l.y2}
                                            stroke={hot ? C_NEUTRAL : '#d4d4d4'}
                                            strokeWidth={hot ? 2.5 : 1}
                                            opacity={hover ? (hot ? 0.95 : 0.1) : 0.45}
                                        />
                                    )
                                })}
                            {layers.transports &&
                                transports.map((t) => {
                                    const hot = tHover && tHover.id === t.id
                                    if (isWorld) {
                                        return <line key={t.id} x1={t.x1} y1={t.y1} x2={t.x2} y2={t.y2} stroke={transportColor(t)} strokeWidth={hot ? 3 : 1.75} strokeDasharray="6 4" opacity={hot ? 1 : 0.85} />
                                    }
                                    return (
                                        <polyline
                                            key={t.id}
                                            points={t.waypoints.map((w) => `${cx(0, w[0])},${cy(w[1])}`).join(' ')}
                                            fill="none"
                                            stroke={transportColor(t)}
                                            strokeWidth={hot ? 3 : 2}
                                            strokeDasharray="6 4"
                                            opacity={hot ? 1 : 0.85}
                                        />
                                    )
                                })}
                        </svg>

                        {/* flight nodes */}
                        {layers.flights &&
                            visibleNodes.map((n) => {
                                const active = hover && (n.id === hover.node.id || hoverDests.has(n.id))
                                return (
                                    <div
                                        key={n.id}
                                        className="absolute -translate-x-1/2 -translate-y-1/2 cursor-pointer rounded-full border-2 border-black/80"
                                        style={{ left: cx(n.mapId || 0, n.px), top: cy(n.py), width: active ? 15 : 10, height: active ? 15 : 10, background: nodeColor(n), boxShadow: active ? `0 0 8px ${nodeColor(n)}` : undefined }}
                                        onMouseEnter={(e) => onEnter(n, e)}
                                        onMouseMove={(e) => onMove(n, e)}
                                        onMouseLeave={onLeave}
                                        onMouseDown={(e) => e.stopPropagation()}
                                        onClick={(e) => { e.stopPropagation(); if (n.zone) { setHover(null); setZone({ mapId: isWorld ? (n.mapId || 0) : view, key: n.zone }) } }}
                                    />
                                )
                            })}

                        {/* transport markers */}
                        {layers.transports &&
                            transports.map((t) => {
                                const mx = isWorld ? (t.x1 + t.x2) / 2 : cx(0, t.marker[0])
                                const my = isWorld ? (t.y1 + t.y2) / 2 : cy(t.marker[1])
                                return <TransportMarker key={`m${t.id}`} t={t} cx={mx} cy={my} onEnter={tEnter} onMove={tMove} onLeave={tLeave} />
                            })}
                    </PanZoom>
                )}
            </div>

            {hover && <FlightTooltip node={hover.node} dests={[...hoverDests].map((id) => model.nodesById[id]?.name).filter(Boolean)} x={hover.x} y={hover.y} />}
            {tHover && <TransportTooltip t={tHover} x={tHover.x} y={tHover.y} />}
        </div>
    )
}

// ---- zone drill-down view --------------------------------------------------

function ZoneView({ mapId, zoneKey, backLabel, onBack }) {
    const [hover, setHover] = useState(null)
    const { src } = useImage('zone_map', zoneKey)

    const { data, isLoading: loading } = useFlightZone(mapId, zoneKey)

    const nodes = data?.nodes || []
    return (
        <div className="h-full flex flex-col">
            <div className="flex items-center gap-2 mb-2 text-sm">
                <button onClick={onBack} className="px-2.5 py-1 rounded bg-bg-panel border border-border-dark hover:bg-bg-hover text-zinc-300">
                    ← {backLabel}
                </button>
                <span className="text-wow-gold font-semibold">{zoneKey}</span>
            </div>
            {loading ? (
                <div className="py-10 text-zinc-500 text-center">Loading zone…</div>
            ) : !src ? (
                <div className="py-10 text-zinc-500 text-center">No map image for “{zoneKey}”.</div>
            ) : (
                <div className="flex-1">
                    <PanZoom canvasW={PANEL_W} canvasH={PANEL_H} fitKey={zoneKey}>
                        <img src={src} alt={zoneKey} draggable={false} className="absolute top-0 left-0" style={{ width: PANEL_W, height: PANEL_H, objectFit: 'fill' }} />
                        {nodes.map((n) => (
                            <div
                                key={n.id}
                                className="absolute -translate-x-1/2 -translate-y-1/2 cursor-pointer rounded-full border-2 border-black/80"
                                style={{ left: (n.px / 100) * PANEL_W, top: (n.py / 100) * PANEL_H, width: 14, height: 14, background: nodeColor(n), boxShadow: `0 0 8px ${nodeColor(n)}` }}
                                onMouseEnter={(e) => setHover({ node: n, x: e.clientX, y: e.clientY })}
                                onMouseMove={(e) => setHover((h) => ({ ...(h || {}), node: n, x: e.clientX, y: e.clientY }))}
                                onMouseLeave={() => setHover(null)}
                                onMouseDown={(e) => e.stopPropagation()}
                            />
                        ))}
                    </PanZoom>
                </div>
            )}
            {hover && <FlightTooltip node={hover.node} dests={hover.node.dests || []} x={hover.x} y={hover.y} />}
        </div>
    )
}

function FlightTooltip({ node, dests, x, y }) {
    const faction = node.alliance && node.horde ? 'Neutral' : node.alliance ? 'Alliance' : node.horde ? 'Horde' : 'Neutral'
    const color = node.alliance && node.horde ? C_NEUTRAL : node.alliance ? C_ALLIANCE : C_HORDE
    const style = { left: Math.min(x + 16, (typeof window !== 'undefined' ? window.innerWidth : 1200) - 280), top: Math.max(8, y - 10) }
    const sorted = [...new Set(dests)].sort()
    return (
        <div className="fixed z-50 w-[260px] rounded border border-zinc-600 bg-black/95 p-2.5 text-sm shadow-xl pointer-events-none" style={style}>
            <div className="text-wow-gold font-semibold leading-tight">{node.name}</div>
            <div className="text-xs mb-1" style={{ color }}>{faction}</div>
            {sorted.length ? (
                <>
                    <div className="text-zinc-400 text-xs">Connects to ({sorted.length}):</div>
                    <ul className="text-zinc-300 text-xs leading-snug mt-0.5 max-h-48 overflow-hidden">
                        {sorted.map((d) => <li key={d}>• {d}</li>)}
                    </ul>
                </>
            ) : (
                <div className="text-zinc-500 text-xs">No outgoing routes for this faction.</div>
            )}
        </div>
    )
}

function TransportTooltip({ t, x, y }) {
    const color = transportColor(t)
    const style = { left: Math.min(x + 16, (typeof window !== 'undefined' ? window.innerWidth : 1200) - 260), top: Math.max(8, y - 10) }
    return (
        <div className="fixed z-50 w-[240px] rounded border border-zinc-600 bg-black/95 p-2.5 text-sm shadow-xl pointer-events-none" style={style}>
            <div className="font-semibold leading-tight" style={{ color }}>{transportLabel(t)}</div>
            <div className="text-zinc-300 text-xs mt-1">{t.here} <span className="text-zinc-500">↔</span> <span className="text-wow-gold">{t.dest}</span></div>
            {!isFiniteSame(t) && <div className="text-zinc-500 text-xs mt-0.5">cross-continent</div>}
        </div>
    )
}
function isFiniteSame(t) {
    return t.sameContinent
}

export default MapsPage
