import React, { useState, useMemo } from 'react'
import { queryClient } from '../../../queryClient'
import { useObjectDetail } from '../../../hooks/queries/objects'
import { queryKeys } from '../../../hooks/queries/keys'
import { DATABASE_BASE_URL } from '../../../utils/constants'
import { useZoneMap, useZoneMinimap } from '../../../services/useImage'
import {
    DetailPageLayout,
    DetailHeader,
    DetailSection,
    DetailLoading,
    DetailError,
    LootItem,
    LootGrid,
} from '../../ui'

const ObjectDetailView = ({ entry, onBack, onNavigate, tooltipHook }) => {
    const [showMapModal, setShowMapModal] = useState(false)
    const [selectedZone, setSelectedZone] = useState(null)
    const [syncing, setSyncing] = useState(false)
    // Map style: 'atlas' = painted WorldMap, 'terrain' = in-game minimap art.
    const [mapStyle, setMapStyle] = useState('atlas')

    const { data: detail, isLoading: loading } = useObjectDetail(entry)

    // Reset the selected zone when the object changes (render-time, no effect).
    const [objKey, setObjKey] = useState(entry)
    if (entry !== objKey) {
        setObjKey(entry)
        setSelectedZone(null)
    }

    const spawns = detail?.spawns || []

    // Spawns span many zones (a mailbox is in every Alliance town). Group by zone
    // so each zone's markers plot on its own map — plotting them all on one map
    // would scatter them at nonsense positions.
    const zones = useMemo(() => {
        const m = new Map()
        for (const s of spawns) {
            const name = s.zoneName || `Map ${s.mapId}`
            if (!m.has(name)) m.set(name, [])
            m.get(name).push(s)
        }
        return [...m.entries()]
            .map(([name, pts]) => ({ name, pts }))
            .sort((a, b) => b.pts.length - a.pts.length)
    }, [spawns])

    const activeZone =
        (selectedZone && zones.find((z) => z.name === selectedZone)) || zones[0] || null
    const atlasMap = useZoneMap(activeZone?.name)
    const terrainMap = useZoneMinimap(activeZone?.name)
    const terrainAvailable = !!terrainMap.src
    const showTerrain = mapStyle === 'terrain' && (terrainMap.src || terrainMap.loading)
    const mapImage = showTerrain ? terrainMap : atlasMap

    // One sync per object: octowow's page drives both spawns and (for chests) loot.
    const handleSync = () => {
        if (syncing) return
        const fn = window?.go?.main?.App?.SyncObject
        if (!fn) return
        setSyncing(true)
        fn(entry)
            .then(() => {
                // A sync changes spawns + loot (this object's Contains and the items'
                // Contained In), so drop the cache to refetch this and the list behind it.
                queryClient.invalidateQueries()
                setSelectedZone(null)
            })
            .finally(() => setSyncing(false))
    }

    if (loading) return <DetailLoading />
    if (!detail) return <DetailError message="Object not found" onBack={onBack} />

    const startsQuests = detail.startsQuests || []
    const endsQuests = detail.endsQuests || []
    const contains = detail.contains || []

    const markers = (activeZone?.pts || []).filter(
        (s) => s.x > 0 && s.x <= 100 && s.y > 0 && s.y <= 100,
    )
    const renderMarkers = (size) =>
        markers.map((s, idx) => (
            <div
                key={idx}
                className="absolute z-10 -ml-2 -mt-2"
                style={{ left: `${s.x}%`, top: `${s.y}%`, width: size, height: size }}
            >
                <div className="absolute inset-0 animate-ping rounded-full bg-red-500/50" />
                <div className="absolute inset-0.5 rounded-full border border-red-400 bg-red-600 shadow-lg" />
            </div>
        ))

    return (
        <>
            <DetailPageLayout onBack={onBack}>
                <DetailHeader
                    icon={
                        <div className="flex h-full w-full items-center justify-center bg-gray-800 text-3xl">
                            🗿
                        </div>
                    }
                    iconBorderColor="text-gray-400"
                    title={detail.name}
                    titleColor="text-white"
                    subtitle={`${detail.typeName || 'Object'} • ID: ${detail.entry}`}
                    action={
                        <div className="flex gap-2">
                            <button
                                onClick={handleSync}
                                disabled={syncing}
                                title="Fetch spawns and loot from octowow.st"
                                className="flex items-center gap-1 rounded bg-blue-600 px-3 py-1.5 text-xs font-bold uppercase text-white transition-colors hover:bg-blue-500 disabled:opacity-50"
                            >
                                <span className="text-sm">↻</span> {syncing ? 'Syncing…' : 'Sync'}
                            </button>
                            <a
                                href={`${DATABASE_BASE_URL}/?object=${detail.entry}`}
                                target="_blank"
                                rel="noreferrer"
                                className="rounded bg-purple-700 px-3 py-1.5 text-xs font-bold uppercase text-white transition-colors hover:bg-purple-600"
                            >
                                🔗 OctoHead
                            </a>
                        </div>
                    }
                />

                <div className="grid grid-cols-1 gap-8 lg:grid-cols-2">
                    {/* Quick Facts */}
                    <DetailSection title="Quick Facts">
                        <table className="infobox-table w-full text-sm">
                            <tbody>
                                <tr>
                                    <th className="py-1 pr-4 text-gray-400">Type:</th>
                                    <td className="text-white">{detail.typeName || detail.type}</td>
                                </tr>
                                <tr>
                                    <th className="py-1 pr-4 text-gray-400">Display ID:</th>
                                    <td className="text-white">{detail.displayId}</td>
                                </tr>
                                {detail.faction > 0 && (
                                    <tr>
                                        <th className="py-1 pr-4 text-gray-400">Faction:</th>
                                        <td className="text-white">{detail.faction}</td>
                                    </tr>
                                )}
                                {detail.size > 0 && detail.size !== 1 && (
                                    <tr>
                                        <th className="py-1 pr-4 text-gray-400">Size:</th>
                                        <td className="text-white">{detail.size.toFixed(2)}</td>
                                    </tr>
                                )}
                                {spawns.length > 0 && (
                                    <tr>
                                        <th className="py-1 pr-4 text-gray-400">Spawns:</th>
                                        <td className="text-white">
                                            {spawns.length} across {zones.length}{' '}
                                            {zones.length === 1 ? 'zone' : 'zones'}
                                        </td>
                                    </tr>
                                )}
                            </tbody>
                        </table>
                    </DetailSection>

                    {/* Location map with spawn markers, per selected zone */}
                    {zones.length > 0 && (
                        <DetailSection title="Location">
                            {/* Zone selector */}
                            {zones.length > 1 && (
                                <div className="mb-2 flex flex-wrap gap-1.5">
                                    {zones.map((z) => (
                                        <button
                                            key={z.name}
                                            onClick={() => setSelectedZone(z.name)}
                                            className={`rounded border px-2 py-1 text-xs transition-colors ${
                                                activeZone?.name === z.name
                                                    ? 'border-wow-gold/60 bg-wow-gold/15 text-wow-gold'
                                                    : 'border-gray-600/40 bg-white/[0.02] text-gray-300 hover:bg-white/5'
                                            }`}
                                        >
                                            {z.name}{' '}
                                            <span className="text-gray-500">({z.pts.length})</span>
                                        </button>
                                    ))}
                                </div>
                            )}

                            {/* Map style toggle — only when a terrain minimap exists for this zone */}
                            {terrainAvailable && (
                                <div className="mb-2 inline-flex overflow-hidden rounded border border-white/10 text-[11px] font-semibold">
                                    {[
                                        ['atlas', 'Atlas'],
                                        ['terrain', 'Terrain'],
                                    ].map(([key, label]) => (
                                        <button
                                            key={key}
                                            onClick={() => setMapStyle(key)}
                                            className={`px-2.5 py-0.5 transition-colors ${
                                                mapStyle === key
                                                    ? 'bg-wow-gold/20 text-wow-gold'
                                                    : 'bg-bg-panel text-gray-400 hover:bg-bg-hover hover:text-gray-200'
                                            }`}
                                        >
                                            {label}
                                        </button>
                                    ))}
                                </div>
                            )}

                            <div
                                className="group relative aspect-[488/325] w-full cursor-pointer overflow-hidden rounded border border-white/20 bg-black bg-cover bg-center shadow-lg"
                                style={{
                                    backgroundImage: mapImage.src ? `url(${mapImage.src})` : 'none',
                                    maxWidth: '488px',
                                }}
                                onClick={() => mapImage.src && setShowMapModal(true)}
                            >
                                {!mapImage.src && !mapImage.loading && (
                                    <div className="flex h-full items-center justify-center text-sm text-gray-500">
                                        No map for {activeZone?.name}
                                    </div>
                                )}
                                {mapImage.loading && (
                                    <div className="flex h-full animate-pulse items-center justify-center text-sm text-gray-500">
                                        Loading Map...
                                    </div>
                                )}
                                {mapImage.src && renderMarkers(16)}
                                {mapImage.src && (
                                    <div className="absolute left-2 top-2 rounded border border-white/10 bg-black/70 px-2 py-0.5 text-xs text-gray-300">
                                        {activeZone?.name} • {markers.length}{' '}
                                        {markers.length === 1 ? 'spawn' : 'spawns'}
                                    </div>
                                )}
                                <div className="absolute right-2 top-2 flex h-6 w-6 items-center justify-center rounded bg-black/50 text-white/80 opacity-0 transition-opacity group-hover:opacity-100">
                                    ⤢
                                </div>
                            </div>
                        </DetailSection>
                    )}

                    {/* Related Quests */}
                    {(startsQuests.length > 0 || endsQuests.length > 0) && (
                        <DetailSection title="Related Quests">
                            {startsQuests.length > 0 && (
                                <div className="mb-4">
                                    <h4 className="mb-2 text-xs uppercase text-gray-500">Starts</h4>
                                    <div className="space-y-1">
                                        {startsQuests.map((q) => (
                                            <div
                                                key={q.entry}
                                                onClick={() => onNavigate('quest', q.entry)}
                                                className="cursor-pointer border-b border-white/5 bg-white/[0.02] p-2 transition-colors hover:bg-white/5"
                                            >
                                                <span className="text-wow-gold hover:text-yellow-300">
                                                    [{q.level}] {q.title}
                                                </span>
                                            </div>
                                        ))}
                                    </div>
                                </div>
                            )}
                            {endsQuests.length > 0 && (
                                <div>
                                    <h4 className="mb-2 text-xs uppercase text-gray-500">Ends</h4>
                                    <div className="space-y-1">
                                        {endsQuests.map((q) => (
                                            <div
                                                key={q.entry}
                                                onClick={() => onNavigate('quest', q.entry)}
                                                className="cursor-pointer border-b border-white/5 bg-white/[0.02] p-2 transition-colors hover:bg-white/5"
                                            >
                                                <span className="text-wow-gold hover:text-yellow-300">
                                                    [{q.level}] {q.title}
                                                </span>
                                            </div>
                                        ))}
                                    </div>
                                </div>
                            )}
                        </DetailSection>
                    )}
                </div>

                {/* Contains (Loot) */}
                {contains.length > 0 && (
                    <DetailSection title={`Contains (${contains.length})`}>
                        <LootGrid>
                            {contains.map((item) => {
                                const handlers = tooltipHook?.getItemHandlers?.(item.itemId) || {}
                                return (
                                    <LootItem
                                        key={item.itemId}
                                        item={{
                                            entry: item.itemId,
                                            name: item.name,
                                            quality: item.quality,
                                            iconPath: item.iconPath,
                                            dropChance: `${item.chance.toFixed(1)}%`,
                                        }}
                                        onClick={() => onNavigate('item', item.itemId)}
                                        showDropChance
                                        {...handlers}
                                    />
                                )
                            })}
                        </LootGrid>
                    </DetailSection>
                )}
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
                                alt={activeZone?.name || 'Location Map'}
                                className="max-h-[85vh] max-w-full rounded-lg object-contain shadow-2xl"
                            />
                            {renderMarkers(20)}
                        </div>
                        {activeZone?.name && (
                            <div className="absolute bottom-4 left-1/2 -translate-x-1/2 rounded-lg bg-black/80 px-4 py-2 font-bold text-white">
                                {activeZone.name}
                            </div>
                        )}
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

export default ObjectDetailView
