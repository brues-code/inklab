import { useMemo, useState } from 'react'
import { useStickyState } from '../../hooks/useStickyState'
import {
    ContentGrid,
    SidebarPanel,
    ContentPanel,
    ScrollList,
    SectionHeader,
    ListItem,
    ZoneName,
} from '../../components/ui'
import { useZoneMap } from '../../services/useImage'
import { useGatheringProfessions, useGatheringNodes } from '../../hooks/queries/gathering'

// Gathering difficulty at skill S for a node requiring R (the classic
// gathering color ramp: orange until R+25, yellow to R+50, green to R+75,
// then grey; red = can't gather yet).
function difficulty(skill, req) {
    const r = Math.max(req, 1)
    if (skill < r) return 'red'
    if (skill < req + 25) return 'orange'
    if (skill < req + 50) return 'yellow'
    if (skill < req + 75) return 'green'
    return 'grey'
}

const DIFF_COLORS = {
    red: '#e33',
    orange: '#ff8040',
    yellow: '#ffff00',
    green: '#40bf40',
    grey: '#9d9d9d',
}

/**
 * Gathering map (a Maps-page tab): pick a gathering skill (Mining, Herbalism,
 * lockpicking chests, custom lines like Survival) and a skill level; zones
 * list how many nodes are farmable there, and the zone map plots every known
 * spawn, colored by the classic difficulty ramp at your skill (red = too low,
 * orange/yellow/green/grey). Node types in the legend toggle on/off.
 */
function GatheringView() {
    const [profId, setProfId] = useStickyState('gathering.profession', null)
    const [skill, setSkill] = useStickyState('gathering.skill', 300)
    const [zone, setZone] = useState(null)
    const [hidden, setHidden] = useState(() => new Set())
    const [onlyGatherable, setOnlyGatherable] = useStickyState('gathering.onlyGatherable', false)

    const { data: professions = [] } = useGatheringProfessions()
    const { data: nodes = [], isLoading } = useGatheringNodes(profId)

    // Highest requirement drives the skill slider range.
    const maxSkill = useMemo(() => Math.max(300, ...nodes.map((n) => n.reqSkill + 75)), [nodes])

    // Node visibility at the current settings (skill gate + legend toggles).
    const activeNodes = useMemo(
        () =>
            nodes.filter(
                (n) =>
                    !hidden.has(n.name) &&
                    (!onlyGatherable || difficulty(skill, n.reqSkill) !== 'red'),
            ),
        [nodes, hidden, onlyGatherable, skill],
    )

    // Zone index: gatherable / total spawn counts per zone, plus the zone's
    // BEST skill-up difficulty at the current skill — that's what colors the
    // count (orange zone = best leveling spot; grey = farmable, no skill-ups).
    const zones = useMemo(() => {
        const rank = { orange: 3, yellow: 2, green: 1 }
        const byZone = new Map()
        for (const n of nodes) {
            const diff = difficulty(skill, n.reqSkill)
            const req = Math.max(n.reqSkill, 1)
            for (const s of n.spawns) {
                let z = byZone.get(s.zone)
                if (!z) {
                    z = {
                        zone: s.zone,
                        total: 0,
                        gatherable: 0,
                        best: null,
                        minReq: req,
                        maxReq: req,
                    }
                    byZone.set(s.zone, z)
                }
                z.total++
                z.minReq = Math.min(z.minReq, req)
                z.maxReq = Math.max(z.maxReq, req)
                if (diff !== 'red') {
                    z.gatherable++
                    if ((rank[diff] || 0) > (rank[z.best] || 0)) z.best = diff
                }
            }
        }
        return [...byZone.values()].sort(
            (a, b) =>
                (rank[b.best] || 0) - (rank[a.best] || 0) ||
                b.gatherable - a.gatherable ||
                b.total - a.total,
        )
    }, [nodes, skill])

    // Spawns to plot on the selected zone, with their difficulty color.
    const pins = useMemo(() => {
        if (!zone) return []
        const out = []
        for (const n of activeNodes) {
            const diff = difficulty(skill, n.reqSkill)
            for (const s of n.spawns) {
                if (s.zone === zone) out.push({ x: s.x, y: s.y, node: n, diff })
            }
        }
        return out
    }, [activeNodes, zone, skill])

    // Legend: node types present in the selected zone (independent of the
    // toggles, so hidden ones can be re-enabled).
    const legend = useMemo(() => {
        if (!zone) return []
        return nodes
            .map((n) => ({ node: n, count: n.spawns.filter((s) => s.zone === zone).length }))
            .filter((e) => e.count > 0)
            .sort((a, b) => a.node.reqSkill - b.node.reqSkill)
    }, [nodes, zone])

    const mapImage = useZoneMap(zone)

    const toggleNode = (name) =>
        setHidden((prev) => {
            const next = new Set(prev)
            if (next.has(name)) next.delete(name)
            else next.add(name)
            return next
        })

    const selectProfession = (id) => {
        setProfId(id)
        setZone(null)
        setHidden(new Set())
    }

    return (
        <ContentGrid columns="260px 260px 1fr">
            {/* Profession + skill */}
            <SidebarPanel>
                <SectionHeader title="Gathering" noSearch />
                <div className="space-y-1 p-2">
                    {professions.map((p) => (
                        <ListItem
                            key={p.id}
                            active={profId === p.id}
                            onClick={() => selectProfession(p.id)}
                        >
                            <span className="flex w-full items-center justify-between">
                                <span>{p.name}</span>
                                <span className="text-[10px] text-gray-500">{p.spawns} spawns</span>
                            </span>
                        </ListItem>
                    ))}
                </div>

                <div className="space-y-2 border-t border-border-dark p-3">
                    <div className="flex items-center justify-between text-xs">
                        <span className="font-bold uppercase text-gray-500">Skill</span>
                        <input
                            type="number"
                            min={1}
                            max={maxSkill}
                            value={skill}
                            onChange={(e) =>
                                setSkill(
                                    Math.max(1, Math.min(maxSkill, Number(e.target.value) || 1)),
                                )
                            }
                            className="w-16 rounded border border-border-light bg-black/50 px-1.5 py-0.5 text-right font-mono text-white outline-none focus:border-wow-gold"
                        />
                    </div>
                    <input
                        type="range"
                        min={1}
                        max={maxSkill}
                        value={skill}
                        onChange={(e) => setSkill(Number(e.target.value))}
                        className="w-full accent-wow-gold"
                    />
                    <label className="flex cursor-pointer select-none items-center gap-1.5 text-[11px] text-gray-400">
                        <input
                            type="checkbox"
                            checked={onlyGatherable}
                            onChange={(e) => setOnlyGatherable(e.target.checked)}
                        />
                        hide nodes above my skill
                    </label>
                    {/* Difficulty ramp legend */}
                    <div className="flex items-center gap-2 pt-1 text-[10px] text-gray-500">
                        {Object.entries(DIFF_COLORS).map(([k, c]) => (
                            <span key={k} className="flex items-center gap-1">
                                <span
                                    className="h-2 w-2 rounded-full"
                                    style={{ backgroundColor: c }}
                                />
                                {k}
                            </span>
                        ))}
                    </div>
                </div>
            </SidebarPanel>

            {/* Zones */}
            <SidebarPanel>
                <SectionHeader title={profId ? `Zones (${zones.length})` : 'Zones'} noSearch />
                <ScrollList>
                    {isLoading && (
                        <div className="animate-pulse p-3 text-xs italic text-gray-500">
                            Loading nodes...
                        </div>
                    )}
                    {zones.map((z) => (
                        <ListItem
                            key={z.zone}
                            active={zone === z.zone}
                            onClick={() => setZone(z.zone)}
                        >
                            <span className="flex w-full items-center justify-between gap-2">
                                <span className="min-w-0 truncate">
                                    <ZoneName name={z.zone} fallback={z.zone} />
                                    {/* Skill range of the nodes spawning here */}
                                    <span className="ml-1.5 font-mono text-[10px] text-gray-500">
                                        {z.minReq === z.maxReq
                                            ? z.minReq
                                            : `${z.minReq}-${z.maxReq}`}
                                    </span>
                                </span>
                                {/* Colored by the zone's best skill-up difficulty:
                                        orange = best leveling; grey = farmable only. */}
                                <span
                                    className="shrink-0 font-mono text-[10px]"
                                    style={{
                                        color:
                                            z.gatherable === 0
                                                ? DIFF_COLORS.red
                                                : DIFF_COLORS[z.best || 'grey'],
                                    }}
                                    title={
                                        z.gatherable === 0
                                            ? 'Nothing gatherable at this skill'
                                            : z.best
                                              ? `Best skill-up here: ${z.best}`
                                              : 'Gatherable, but no skill-ups at this skill'
                                    }
                                >
                                    {z.gatherable}/{z.total}
                                </span>
                            </span>
                        </ListItem>
                    ))}
                    {profId && !isLoading && zones.length === 0 && (
                        <div className="p-3 text-xs italic text-gray-600">
                            No spawn data. Run the object sync to fetch spawns.
                        </div>
                    )}
                </ScrollList>
            </SidebarPanel>

            {/* Map */}
            <ContentPanel>
                <SectionHeader
                    title={
                        zone ? (
                            <>
                                <ZoneName name={zone} fallback={zone} /> — {pins.length} nodes
                            </>
                        ) : (
                            'Select a zone'
                        )
                    }
                    noSearch
                />
                {!profId && (
                    <div className="flex flex-1 items-center justify-center italic text-gray-600">
                        Pick a gathering skill to see its nodes
                    </div>
                )}
                {profId && !zone && !isLoading && (
                    <div className="flex flex-1 items-center justify-center italic text-gray-600">
                        Pick a zone to see its node map
                    </div>
                )}
                {zone && (
                    <div className="flex-1 overflow-auto p-4">
                        {/* Node-type legend for this zone */}
                        <div className="mb-3 flex flex-wrap gap-1.5">
                            {legend.map(({ node, count }) => {
                                const off = hidden.has(node.name)
                                const diff = difficulty(skill, node.reqSkill)
                                return (
                                    <button
                                        key={node.name}
                                        onClick={() => toggleNode(node.name)}
                                        className={`flex items-center gap-1.5 rounded border px-2 py-0.5 text-[11px] transition-all ${
                                            off
                                                ? 'border-gray-800 text-gray-600 opacity-60'
                                                : 'border-gray-600 text-gray-200'
                                        }`}
                                        title={off ? 'Show on map' : 'Hide from map'}
                                    >
                                        <span
                                            className="h-2 w-2 rounded-full"
                                            style={{ backgroundColor: DIFF_COLORS[diff] }}
                                        />
                                        {node.name}
                                        <span className="text-gray-500">
                                            {node.reqSkill > 0 ? `(${node.reqSkill})` : ''} ×{count}
                                        </span>
                                    </button>
                                )
                            })}
                        </div>

                        <div
                            className="relative aspect-[488/325] w-full max-w-[976px] overflow-hidden rounded border border-white/20 bg-black bg-cover bg-center shadow-lg"
                            style={{
                                backgroundImage: mapImage.src ? `url(${mapImage.src})` : 'none',
                            }}
                        >
                            {!mapImage.src && (
                                <div className="flex h-full items-center justify-center text-sm text-gray-500">
                                    {mapImage.loading ? 'Loading map...' : 'No map for this zone'}
                                </div>
                            )}
                            {mapImage.src &&
                                pins.map((p, i) => (
                                    <div
                                        key={i}
                                        className="group/pin absolute z-10 -ml-[5px] -mt-[5px] h-[10px] w-[10px]"
                                        style={{ left: `${p.x}%`, top: `${p.y}%` }}
                                    >
                                        <div
                                            className="h-full w-full rounded-full border border-black/70 shadow"
                                            style={{ backgroundColor: DIFF_COLORS[p.diff] }}
                                        />
                                        <div className="pointer-events-none absolute bottom-full left-1/2 z-20 mb-1 -translate-x-1/2 whitespace-nowrap rounded border border-white/20 bg-black/90 px-2 py-1 text-xs text-white opacity-0 shadow-lg transition-opacity group-hover/pin:opacity-100">
                                            <span className="font-semibold text-wow-gold">
                                                {p.node.name}
                                            </span>
                                            {p.node.reqSkill > 0 && (
                                                <span className="ml-1 text-gray-300">
                                                    ({p.node.reqSkill})
                                                </span>
                                            )}
                                        </div>
                                    </div>
                                ))}
                        </div>
                    </div>
                )}
            </ContentPanel>
        </ContentGrid>
    )
}

export default GatheringView
