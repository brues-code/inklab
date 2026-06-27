import React, { useState } from 'react'
import { useZoneDetail } from '../../../hooks/queries/zones'
import { useZoneMap } from '../../../services/useImage'
import { DetailPageLayout, DetailLoading, DetailError } from '../../ui'

const ZONE_COLOR = '#4ADE80'

// Plotting every spawn point gets heavy in dense zones; cap the markers we draw.
const MAX_MARKERS = 800

// Map "services" filters, à la Wowhead. NPC services are creature npc_flags
// bits; books/mailboxes are game-object types. Selecting one narrows the map
// markers and the matching list to just that service.
const SERVICES = [
    { id: 'questgiver', label: 'Quest Givers', kind: 'npc', bit: 0x2 },
    { id: 'vendor', label: 'Vendors', kind: 'npc', bit: 0x80 },
    { id: 'trainer', label: 'Trainers', kind: 'npc', bit: 0x10 | 0x20 | 0x40 },
    { id: 'repair', label: 'Repairers', kind: 'npc', bit: 0x1000 },
    { id: 'flightmaster', label: 'Flight Masters', kind: 'npc', bit: 0x2000 },
    { id: 'spirithealer', label: 'Spirit Healers', kind: 'npc', bit: 0x4000 },
    { id: 'innkeeper', label: 'Innkeepers', kind: 'npc', bit: 0x10000 },
    { id: 'banker', label: 'Bankers', kind: 'npc', bit: 0x20000 },
    { id: 'battlemaster', label: 'Battlemasters', kind: 'npc', bit: 0x100000 },
    { id: 'auctioneer', label: 'Auctioneers', kind: 'npc', bit: 0x200000 },
    { id: 'stablemaster', label: 'Stable Masters', kind: 'npc', bit: 0x400000 },
    { id: 'books', label: 'Books', kind: 'obj', type: 9 },
    { id: 'mailbox', label: 'Mailboxes', kind: 'obj', type: 19 },
]

const npcMatchesService = (n, svc) => svc.kind === 'npc' && (n.npcFlags & svc.bit) !== 0
const objMatchesService = (o, svc) => svc.kind === 'obj' && o.type === svc.type

const ZoneDetailView = ({ entry, onBack, onNavigate }) => {
    const [activeTab, setActiveTab] = useState('npcs')
    const [showMapModal, setShowMapModal] = useState(false)
    const [service, setService] = useState(null) // active service filter id

    const { data: detail, isLoading: loading } = useZoneDetail(entry)

    // Reset the service filter when the zone changes (render-time, no effect).
    const [zoneKey, setZoneKey] = useState(entry)
    if (entry !== zoneKey) {
        setZoneKey(entry)
        setService(null)
    }

    const mapImage = useZoneMap(detail?.mapName)

    if (loading) return <DetailLoading />
    if (!detail) return <DetailError message="Zone not found" onBack={onBack} />

    const allNpcs = detail.npcs || []
    const quests = detail.quests || []
    const allObjects = detail.objects || []

    const activeSvc = SERVICES.find((s) => s.id === service) || null

    // Lists are narrowed to the active service (if any).
    const npcs =
        activeSvc?.kind === 'npc' ? allNpcs.filter((n) => npcMatchesService(n, activeSvc)) : allNpcs
    const objects =
        activeSvc?.kind === 'obj'
            ? allObjects.filter((o) => objMatchesService(o, activeSvc))
            : allObjects

    // Only offer service chips that actually match something in this zone.
    const services = SERVICES.map((s) => ({
        ...s,
        count:
            s.kind === 'npc'
                ? allNpcs.filter((n) => npcMatchesService(n, s)).length
                : allObjects.filter((o) => objMatchesService(o, s)).length,
    })).filter((s) => s.count > 0)

    const levelLabel =
        detail.maxLevel > 0
            ? detail.minLevel && detail.minLevel !== detail.maxLevel
                ? `${detail.minLevel} – ${detail.maxLevel}`
                : `${detail.maxLevel}`
            : '—'

    const tabs = [
        { id: 'npcs', label: `NPCs (${npcs.length})` },
        { id: 'quests', label: `Quests (${quests.length})` },
        { id: 'objects', label: `Objects (${objects.length})` },
    ]

    // Map markers: when a service is active, show only its matching spawns;
    // otherwise follow the active tab. Object markers are cyan, creatures emerald.
    let showingObjects
    let markerSource
    if (activeSvc) {
        showingObjects = activeSvc.kind === 'obj'
        const ok = new Set((showingObjects ? objects : npcs).map((e) => e.entry))
        markerSource = (showingObjects ? detail.objectSpawns : detail.spawns)?.filter((s) =>
            ok.has(s.entry),
        )
    } else {
        showingObjects = activeTab === 'objects'
        markerSource = showingObjects ? detail.objectSpawns : detail.spawns
    }
    const spawns = (markerSource || []).slice(0, MAX_MARKERS)
    const markerClass = showingObjects
        ? 'bg-cyan-400/80 border-cyan-200'
        : 'bg-emerald-500/80 border-emerald-300'

    const selectService = (s) => {
        if (service === s.id) {
            setService(null)
            return
        }
        setService(s.id)
        setActiveTab(s.kind === 'obj' ? 'objects' : 'npcs')
    }

    const renderMarkers = (size) =>
        spawns.map((s, idx) => (
            <div
                key={idx}
                className={`absolute rounded-full border shadow ${markerClass}`}
                style={{
                    width: size,
                    height: size,
                    left: `${s.x}%`,
                    top: `${s.y}%`,
                    marginLeft: -size / 2,
                    marginTop: -size / 2,
                }}
            />
        ))

    return (
        <>
            <DetailPageLayout onBack={onBack}>
                {/* Header */}
                <div className="mb-6">
                    <h1 className="text-2xl font-bold" style={{ color: ZONE_COLOR }}>
                        {detail.name}
                    </h1>
                    {detail.groupName && (
                        <div className="mt-1 text-sm text-gray-400">{detail.groupName}</div>
                    )}
                </div>

                {/* Services filter — narrows the map markers and the matching list */}
                {services.length > 0 && (
                    <div className="mb-5 flex flex-wrap gap-1.5">
                        <button
                            onClick={() => setService(null)}
                            className={`rounded border px-2.5 py-1 text-xs transition-colors ${
                                !service
                                    ? 'border-wow-gold/60 bg-wow-gold/15 text-wow-gold'
                                    : 'border-gray-600/40 bg-white/[0.02] text-gray-300 hover:bg-white/5'
                            }`}
                        >
                            All
                        </button>
                        {services.map((s) => (
                            <button
                                key={s.id}
                                onClick={() => selectService(s)}
                                className={`rounded border px-2.5 py-1 text-xs transition-colors ${
                                    service === s.id
                                        ? 'border-wow-gold/60 bg-wow-gold/15 text-wow-gold'
                                        : 'border-gray-600/40 bg-white/[0.02] text-gray-300 hover:bg-white/5'
                                }`}
                            >
                                {s.label} <span className="text-gray-500">({s.count})</span>
                            </button>
                        ))}
                    </div>
                )}

                <div className="flex flex-col gap-8 lg:flex-row">
                    {/* Left: Map */}
                    <div className="w-full flex-shrink-0 lg:w-[488px]">
                        <div className="mb-2 flex items-baseline justify-between border-b border-white/10 pb-1">
                            <h3 className="text-sm font-bold uppercase text-wow-gold">Map</h3>
                            {spawns.length > 0 && (
                                <span className="font-mono text-xs text-gray-400">
                                    {spawns.length}
                                    {(markerSource?.length || 0) > spawns.length ? '+' : ''}{' '}
                                    {showingObjects ? 'object' : 'spawn'} points
                                </span>
                            )}
                        </div>

                        <div
                            className="group relative aspect-[488/325] w-full cursor-pointer overflow-hidden rounded border border-white/20 bg-black bg-cover bg-center shadow-lg"
                            style={{
                                backgroundImage: mapImage.src ? `url(${mapImage.src})` : 'none',
                            }}
                            onClick={() => mapImage.src && setShowMapModal(true)}
                        >
                            {!mapImage.src && !mapImage.loading && (
                                <div className="flex h-full items-center justify-center text-sm text-gray-500">
                                    No Map Data
                                </div>
                            )}
                            {mapImage.loading && (
                                <div className="flex h-full animate-pulse items-center justify-center text-sm text-gray-500">
                                    Loading Map...
                                </div>
                            )}

                            {mapImage.src && renderMarkers(8)}

                            <div className="absolute right-2 top-2 flex h-6 w-6 items-center justify-center rounded bg-black/50 text-white/80 opacity-0 transition-opacity group-hover:opacity-100">
                                ⤢
                            </div>
                        </div>
                    </div>

                    {/* Right: Quick Facts */}
                    <div className="min-w-0 flex-1">
                        <table className="infobox-table w-full text-sm">
                            <thead>
                                <tr>
                                    <th
                                        colSpan="2"
                                        className="mb-2 border-b border-white/10 pb-1 text-left text-sm font-bold uppercase text-wow-gold"
                                    >
                                        Quick Facts
                                    </th>
                                </tr>
                            </thead>
                            <tbody>
                                <tr>
                                    <th>Region:</th>
                                    <td>{detail.groupName || '—'}</td>
                                </tr>
                                <tr>
                                    <th>Creature Levels:</th>
                                    <td>{levelLabel}</td>
                                </tr>
                                <tr>
                                    <th>NPCs:</th>
                                    <td>{allNpcs.length}</td>
                                </tr>
                                <tr>
                                    <th>Quests:</th>
                                    <td>{quests.length}</td>
                                </tr>
                                <tr>
                                    <th>Objects:</th>
                                    <td>{allObjects.length}</td>
                                </tr>
                            </tbody>
                        </table>
                    </div>
                </div>

                {/* Tabs */}
                <div className="mb-4 mt-8 flex gap-1 border-b border-white/20">
                    {tabs.map((tab) => (
                        <button
                            key={tab.id}
                            onClick={() => setActiveTab(tab.id)}
                            className={`relative top-[1px] px-4 py-2 text-sm font-bold transition-all ${
                                activeTab === tab.id
                                    ? 'tab-btn-active border-b-2 border-wow-gold text-white'
                                    : 'tab-btn-inactive text-gray-400 hover:text-gray-200'
                            }`}
                        >
                            {tab.label}
                        </button>
                    ))}
                </div>

                <div className="animate-fade-in min-h-[200px]">
                    {activeTab === 'npcs' && (
                        <>
                            {npcs.length > 0 ? (
                                <div className="bg-bg-sub rounded border border-border-light">
                                    {npcs.map((n, i) => (
                                        <div
                                            key={n.entry}
                                            onClick={() => onNavigate('npc', n.entry)}
                                            className={`flex cursor-pointer items-center gap-3 p-3 transition-colors hover:bg-white/5 ${
                                                i !== npcs.length - 1
                                                    ? 'border-b border-border-light/50'
                                                    : ''
                                            }`}
                                        >
                                            <span className="min-w-[50px] font-mono text-[11px] text-gray-600">
                                                [{n.entry}]
                                            </span>
                                            <span className="hover:text-wow-gold-light flex-1 truncate text-sm font-medium text-wow-gold">
                                                {n.name}
                                                {n.subname && (
                                                    <span className="ml-1 text-gray-500">
                                                        &lt;{n.subname}&gt;
                                                    </span>
                                                )}
                                            </span>
                                            <span className="whitespace-nowrap text-xs text-gray-500">
                                                {n.levelMin === n.levelMax
                                                    ? `Lvl ${n.levelMax}`
                                                    : `Lvl ${n.levelMin}-${n.levelMax}`}
                                                {n.rank > 0 && n.rankName ? ` · ${n.rankName}` : ''}
                                            </span>
                                        </div>
                                    ))}
                                </div>
                            ) : (
                                <div className="italic text-gray-500">
                                    No NPCs recorded in this zone.
                                </div>
                            )}
                        </>
                    )}

                    {activeTab === 'quests' && (
                        <>
                            {quests.length > 0 ? (
                                <div className="bg-bg-sub rounded border border-border-light">
                                    {quests.map((q, i) => (
                                        <div
                                            key={q.entry}
                                            onClick={() => onNavigate('quest', q.entry)}
                                            className={`flex cursor-pointer items-center gap-3 p-3 transition-colors hover:bg-white/5 ${
                                                i !== quests.length - 1
                                                    ? 'border-b border-border-light/50'
                                                    : ''
                                            }`}
                                        >
                                            <span className="min-w-[50px] font-mono text-[11px] text-gray-600">
                                                [{q.entry}]
                                            </span>
                                            <span className="hover:text-wow-gold-light flex-1 truncate text-sm font-medium text-wow-gold">
                                                {q.title}
                                            </span>
                                            {q.questLevel > 0 && (
                                                <span className="whitespace-nowrap text-xs text-gray-500">
                                                    Lvl {q.questLevel}
                                                </span>
                                            )}
                                        </div>
                                    ))}
                                </div>
                            ) : (
                                <div className="italic text-gray-500">No quests in this zone.</div>
                            )}
                        </>
                    )}

                    {activeTab === 'objects' && (
                        <>
                            {objects.length > 0 ? (
                                <div className="bg-bg-sub rounded border border-border-light">
                                    {objects.map((o, i) => (
                                        <div
                                            key={o.entry}
                                            onClick={() => onNavigate('object', o.entry)}
                                            className={`flex cursor-pointer items-center gap-3 p-3 transition-colors hover:bg-white/5 ${
                                                i !== objects.length - 1
                                                    ? 'border-b border-border-light/50'
                                                    : ''
                                            }`}
                                        >
                                            <span className="min-w-[50px] font-mono text-[11px] text-gray-600">
                                                [{o.entry}]
                                            </span>
                                            <span
                                                className="flex-1 truncate text-sm font-medium"
                                                style={{ color: '#4ADE80' }}
                                            >
                                                {o.name}
                                            </span>
                                            {o.typeName && (
                                                <span className="whitespace-nowrap text-xs text-gray-500">
                                                    {o.typeName}
                                                </span>
                                            )}
                                        </div>
                                    ))}
                                </div>
                            ) : (
                                <div className="italic text-gray-500">
                                    No objects recorded in this zone.
                                </div>
                            )}
                        </>
                    )}
                </div>
            </DetailPageLayout>

            {/* Map Zoom Modal */}
            {showMapModal && mapImage.src && (
                <div
                    className="animate-fade-in fixed inset-0 z-50 flex cursor-pointer items-center justify-center bg-black/90 p-4"
                    onClick={() => setShowMapModal(false)}
                >
                    <div
                        className="relative max-h-[90vh] max-w-[90vw]"
                        onClick={(e) => e.stopPropagation()}
                    >
                        <div className="relative inline-block">
                            <img
                                src={mapImage.src}
                                alt={detail.name}
                                className="max-h-[85vh] max-w-full rounded-lg object-contain shadow-2xl"
                            />
                            {renderMarkers(10)}
                        </div>
                        <div className="absolute bottom-4 left-1/2 -translate-x-1/2 rounded-lg bg-black/80 px-4 py-2 font-bold text-white">
                            {detail.name}
                        </div>
                        <button
                            className="absolute right-2 top-2 flex h-8 w-8 items-center justify-center rounded-full bg-red-600 text-lg font-bold text-white transition-colors hover:bg-red-500"
                            onClick={(e) => {
                                e.stopPropagation()
                                setShowMapModal(false)
                            }}
                        >
                            ×
                        </button>
                    </div>
                </div>
            )}
        </>
    )
}

export default ZoneDetailView
