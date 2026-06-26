import { useState, useEffect, useMemo, useCallback } from 'react'
import { useImage, useIcon } from '../../services/useImage'

// Direct Wails bindings (codebase convention for app methods).
const GetFlightContinents = () =>
    window?.go?.main?.App?.GetFlightContinents ? window.go.main.App.GetFlightContinents() : Promise.resolve([])
const GetFlightData = (mapId) =>
    window?.go?.main?.App?.GetFlightData ? window.go.main.App.GetFlightData(mapId) : Promise.resolve(null)

// Faction colors.
const C_ALLIANCE = '#3b82f6'
const C_HORDE = '#e0294a'
const C_NEUTRAL = '#ffd100'
// Transport colors.
const C_BOAT = '#38bdf8'
const C_ZEP = '#c084fc'

const nodeColor = (n) => {
    if (n.alliance && n.horde) return C_NEUTRAL
    if (n.alliance) return C_ALLIANCE
    if (n.horde) return C_HORDE
    return C_NEUTRAL
}
const visibleForFaction = (n, faction) => {
    if (faction === 'alliance') return n.alliance
    if (faction === 'horde') return n.horde
    return true
}
const transportColor = (t) => (t.type === 'zeppelin' ? C_ZEP : C_BOAT)
// Real game art (served via the local icon set): a galleon for boats, a goblin
// flying machine for zeppelins (no literal zeppelin icon exists in the set).
const TRANSPORT_ICON = { boat: 'inv_garrison_cargoship', zeppelin: 'ability_mount_gyrocoptor' }
const transportLabel = (t) => (t.type === 'zeppelin' ? 'Zeppelin' : 'Boat')

// TransportMarker renders a transport's embarkation point as the framed game
// icon, ringed in the route's color.
function TransportMarker({ t, onEnter, onMove, onLeave }) {
    const { src } = useIcon(TRANSPORT_ICON[t.type] || TRANSPORT_ICON.boat)
    return (
        <div
            className="absolute -translate-x-1/2 -translate-y-1/2 cursor-pointer rounded-sm overflow-hidden shadow"
            style={{
                left: `${t.marker[0]}%`,
                top: `${t.marker[1]}%`,
                width: 22,
                height: 22,
                border: `2px solid ${transportColor(t)}`,
                background: '#000',
            }}
            onMouseEnter={(e) => onEnter(t, e)}
            onMouseMove={(e) => onMove(t, e)}
            onMouseLeave={onLeave}
        >
            {src && <img src={src} alt="" className="w-full h-full object-cover" draggable={false} />}
        </div>
    )
}

function MapsPage() {
    const [continents, setContinents] = useState([])
    const [mapId, setMapId] = useState(null)
    const [data, setData] = useState(null)
    const [loading, setLoading] = useState(false)
    const [faction, setFaction] = useState('all') // all | alliance | horde
    const [routeMode, setRouteMode] = useState('straight') // straight | route
    const [layers, setLayers] = useState({ flights: true, transports: true })
    const [hover, setHover] = useState(null) // flight node hover
    const [tHover, setTHover] = useState(null) // transport hover

    const continent = continents.find((c) => c.mapId === mapId)
    const { src: mapSrc } = useImage('zone_map', continent?.name)

    useEffect(() => {
        GetFlightContinents().then((list) => {
            setContinents(list || [])
            if (list?.length) setMapId(list[0].mapId)
        })
    }, [])

    useEffect(() => {
        if (mapId == null) return
        setLoading(true)
        GetFlightData(mapId)
            .then(setData)
            .finally(() => setLoading(false))
    }, [mapId])

    const nodesById = useMemo(() => {
        const m = {}
        ;(data?.nodes || []).forEach((n) => (m[n.id] = n))
        return m
    }, [data])

    const visibleNodes = useMemo(
        () => (data?.nodes || []).filter((n) => visibleForFaction(n, faction)),
        [data, faction]
    )
    const visibleIds = useMemo(() => new Set(visibleNodes.map((n) => n.id)), [visibleNodes])

    const connections = useMemo(() => {
        const all = (data?.connections || []).filter((c) => visibleIds.has(c.from) && visibleIds.has(c.to))
        if (routeMode === 'route') return all
        const seen = new Set()
        const out = []
        for (const c of all) {
            const key = c.from < c.to ? `${c.from}-${c.to}` : `${c.to}-${c.from}`
            if (seen.has(key)) continue
            seen.add(key)
            out.push(c)
        }
        return out
    }, [data, visibleIds, routeMode])

    const hoverDests = useMemo(() => {
        if (!hover) return new Set()
        const s = new Set()
        ;(data?.connections || []).forEach((c) => {
            if (c.from === hover.node.id && visibleIds.has(c.to)) s.add(c.to)
        })
        return s
    }, [hover, data, visibleIds])

    const isHot = (c) => hover && (c.from === hover.node.id || c.to === hover.node.id)

    // Embarkation marker for each transport = the on-continent waypoint nearest
    // any flight node (the dock by the city).
    const transports = useMemo(() => {
        const nodes = data?.nodes || []
        return (data?.transports || [])
            .filter((t) => t.waypoints?.length)
            .map((t) => {
                let best = t.waypoints[0]
                let bestD = Infinity
                for (const w of t.waypoints) {
                    for (const n of nodes) {
                        const d = (n.px - w[0]) ** 2 + (n.py - w[1]) ** 2
                        if (d < bestD) {
                            bestD = d
                            best = w
                        }
                    }
                }
                return { ...t, marker: best }
            })
    }, [data])

    const onEnter = useCallback((node, e) => setHover({ node, x: e.clientX, y: e.clientY }), [])
    const onMove = useCallback((node, e) => setHover((h) => ({ ...(h || {}), node, x: e.clientX, y: e.clientY })), [])
    const onLeave = useCallback(() => setHover(null), [])

    const factionBtn = (id, label, color) => (
        <button
            onClick={() => setFaction(id)}
            className={`px-3 py-1.5 rounded text-sm font-semibold border transition-colors ${
                faction === id ? 'bg-bg-active border-border-highlight' : 'bg-bg-panel border-border-dark hover:bg-bg-hover'
            }`}
            style={{ color }}
        >
            {label}
        </button>
    )
    const layerChip = (id, label, color) => (
        <button
            onClick={() => setLayers((l) => ({ ...l, [id]: !l[id] }))}
            className={`px-2.5 py-1 rounded text-xs font-semibold border flex items-center gap-1.5 transition-opacity ${
                layers[id] ? 'bg-bg-active border-border-highlight' : 'bg-bg-panel border-border-dark opacity-50'
            }`}
        >
            <span className="inline-block w-2.5 h-2.5 rounded-full" style={{ background: color }} />
            {label}
        </button>
    )

    return (
        <div className="h-full overflow-auto bg-bg-dark">
            {/* controls */}
            <div className="flex flex-wrap items-center gap-4 px-5 py-3 border-b border-border-dark bg-bg-main sticky top-0 z-20">
                <div className="flex gap-2">
                    {continents.map((c) => (
                        <button
                            key={c.mapId}
                            onClick={() => setMapId(c.mapId)}
                            className={`px-3 py-1.5 rounded font-semibold text-sm border transition-colors ${
                                c.mapId === mapId
                                    ? 'bg-bg-active border-border-highlight text-wow-gold'
                                    : 'bg-bg-panel border-border-dark hover:bg-bg-hover text-zinc-300'
                            }`}
                        >
                            {c.name}
                        </button>
                    ))}
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
                {layers.flights && (
                    <div className="flex gap-1 ml-auto bg-bg-panel border border-border-dark rounded p-0.5">
                        <button
                            onClick={() => setRouteMode('straight')}
                            className={`px-2.5 py-1 rounded text-xs ${routeMode === 'straight' ? 'bg-bg-active text-white' : 'text-zinc-400'}`}
                        >
                            Direct
                        </button>
                        <button
                            onClick={() => setRouteMode('route')}
                            className={`px-2.5 py-1 rounded text-xs ${routeMode === 'route' ? 'bg-bg-active text-white' : 'text-zinc-400'}`}
                        >
                            Flight routes
                        </button>
                    </div>
                )}
            </div>

            {/* map */}
            <div className="p-5 flex justify-center">
                {loading ? (
                    <div className="py-10 text-zinc-500">Loading map…</div>
                ) : continents.length === 0 ? (
                    <div className="py-10 text-zinc-500 text-center">
                        No map data found.
                        <div className="text-xs mt-1">If you just updated InkLab, restart the app; otherwise run a Client Data import.</div>
                    </div>
                ) : !mapSrc ? (
                    <div className="py-10 text-zinc-500 text-center">
                        No continent map image for “{continent?.name}”.
                        <div className="text-xs mt-1">Run a Client Data import to extract zone maps.</div>
                    </div>
                ) : (
                    <div className="relative inline-block max-w-full">
                        <img src={mapSrc} alt={continent?.name} className="block max-h-[78vh] w-auto rounded-md border border-border-dark" />

                        {/* lines: flight connections + transport routes */}
                        <svg className="absolute inset-0 w-full h-full pointer-events-none" viewBox="0 0 100 100" preserveAspectRatio="none">
                            {layers.flights &&
                                connections.map((c) => {
                                    const hot = isHot(c)
                                    const stroke = hot ? C_NEUTRAL : '#cfcfcf'
                                    const opacity = hover ? (hot ? 0.95 : 0.12) : 0.5
                                    if (routeMode === 'route' && c.waypoints?.length > 1) {
                                        return (
                                            <polyline
                                                key={`f${c.pathId}`}
                                                points={c.waypoints.map((w) => `${w[0]},${w[1]}`).join(' ')}
                                                fill="none"
                                                stroke={stroke}
                                                strokeWidth={hot ? 2 : 1}
                                                strokeLinejoin="round"
                                                opacity={opacity}
                                                vectorEffect="non-scaling-stroke"
                                            />
                                        )
                                    }
                                    const a = nodesById[c.from]
                                    const b = nodesById[c.to]
                                    if (!a || !b) return null
                                    return (
                                        <line
                                            key={`f${c.pathId}`}
                                            x1={a.px}
                                            y1={a.py}
                                            x2={b.px}
                                            y2={b.py}
                                            stroke={stroke}
                                            strokeWidth={hot ? 2 : 1}
                                            opacity={opacity}
                                            vectorEffect="non-scaling-stroke"
                                        />
                                    )
                                })}

                            {layers.transports &&
                                transports.map((t) => {
                                    const hot = tHover && tHover.id === t.id
                                    return (
                                        <polyline
                                            key={`t${t.id}`}
                                            points={t.waypoints.map((w) => `${w[0]},${w[1]}`).join(' ')}
                                            fill="none"
                                            stroke={transportColor(t)}
                                            strokeWidth={hot ? 2.5 : 1.5}
                                            strokeDasharray="3 2"
                                            strokeLinejoin="round"
                                            opacity={hot ? 1 : 0.85}
                                            vectorEffect="non-scaling-stroke"
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
                                        className="absolute -translate-x-1/2 -translate-y-1/2 cursor-pointer"
                                        style={{ left: `${n.px}%`, top: `${n.py}%` }}
                                        onMouseEnter={(e) => onEnter(n, e)}
                                        onMouseMove={(e) => onMove(n, e)}
                                        onMouseLeave={onLeave}
                                    >
                                        <div
                                            className="rounded-full border-2 border-black/80 shadow transition-transform"
                                            style={{
                                                width: active ? 16 : 11,
                                                height: active ? 16 : 11,
                                                background: nodeColor(n),
                                                boxShadow: active ? `0 0 8px ${nodeColor(n)}` : undefined,
                                            }}
                                        />
                                    </div>
                                )
                            })}

                        {/* transport embarkation markers */}
                        {layers.transports &&
                            transports.map((t) => (
                                <TransportMarker
                                    key={`tm${t.id}`}
                                    t={t}
                                    onEnter={(tr, e) => setTHover({ ...tr, x: e.clientX, y: e.clientY })}
                                    onMove={(tr, e) => setTHover((h) => ({ ...(h || tr), x: e.clientX, y: e.clientY }))}
                                    onLeave={() => setTHover(null)}
                                />
                            ))}
                    </div>
                )}
            </div>

            {/* tooltips */}
            {hover && (
                <FlightTooltip
                    node={hover.node}
                    dests={[...hoverDests].map((id) => nodesById[id]?.name).filter(Boolean)}
                    x={hover.x}
                    y={hover.y}
                />
            )}
            {tHover && <TransportTooltip t={tHover} x={tHover.x} y={tHover.y} />}
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
            <div className="text-xs mb-1" style={{ color }}>
                {faction}
            </div>
            {sorted.length ? (
                <>
                    <div className="text-zinc-400 text-xs">Connects to ({sorted.length}):</div>
                    <ul className="text-zinc-300 text-xs leading-snug mt-0.5 max-h-48 overflow-hidden">
                        {sorted.map((d) => (
                            <li key={d}>• {d}</li>
                        ))}
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
            <div className="font-semibold leading-tight" style={{ color }}>
                {transportLabel(t)}
            </div>
            <div className="text-zinc-300 text-xs mt-1">
                {t.here} → <span className="text-wow-gold">{t.dest}</span>
            </div>
            {!t.sameContinent && t.destContinent && (
                <div className="text-zinc-500 text-xs mt-0.5">to {t.destContinent}</div>
            )}
        </div>
    )
}

export default MapsPage
