import React, { useState } from 'react'
import { queryClient } from '../../../queryClient'
import { SyncNpcData, RefreshNpcImages } from '../../../services/api'
import { useNpcDetail } from '../../../hooks/queries/npcs'
import { queryKeys } from '../../../hooks/queries/keys'
import {
    useNpcModel,
    useNpcPortrait,
    useZoneMap,
    useZoneMinimap,
    useIcon,
} from '../../../services/useImage'
import { evictImage } from '../../../services/imageService'
import { getQualityColor, QUESTION_MARK_ICON } from '../../../utils/wow'
import { DATABASE_BASE_URL } from '../../../utils/constants'
import {
    DetailPageLayout,
    DetailHeader,
    DetailSection,
    DetailGrid,
    LootGrid,
    StatBadge,
    DetailLoading,
    DetailError,
    LootItem,
    Money,
} from '../../ui'

// AbilityIcon resolves a spell's icon through the local icon service (local
// data/icons first, then CDN, then the questionmark placeholder) — same as the
// rest of the app. The previous remote-only <img> 404'd on octo-custom icons
// and hid itself, which looked like the icon flashing then disappearing.
const AbilityIcon = ({ iconName }) => {
    const icon = useIcon(iconName)
    if (!iconName) return null
    return (
        <img
            src={icon.src || QUESTION_MARK_ICON}
            alt=""
            className="h-10 w-10 rounded border border-gray-600 bg-black/40"
        />
    )
}

const NPCDetailView = ({ entry, onBack, onNavigate, tooltipHook, activeTab, onTabChange }) => {
    // Active tab. When the route supplies it (activeTab/onTabChange) it lives in
    // the URL so Back/Forward and refresh work; otherwise fall back to local
    // state. The effective tab is `currentTab` (validated against `tabs` below).
    const [localTab, setLocalTab] = useState('overview')
    const rawTab = onTabChange ? activeTab : localTab
    const setTab = onTabChange || setLocalTab
    const [showMapModal, setShowMapModal] = useState(false)
    // Map style: 'atlas' = the painted WorldMap image, 'terrain' = the in-game
    // minimap art. Falls back to atlas automatically when a zone has no minimap.
    const [mapStyle, setMapStyle] = useState('atlas')
    const [imgReload, setImgReload] = useState(0)
    const [refreshingImages, setRefreshingImages] = useState(false)
    // When an NPC spawns across multiple zones, this overrides which zone's map is
    // shown; null = use the resolved primary zone (detail.zoneName).
    const [selectedZone, setSelectedZone] = useState(null)

    const { data: detail, isLoading: loading } = useNpcDetail(entry)

    // Reset the zone toggle when viewing a different NPC (render-time, no effect).
    const [npcKey, setNpcKey] = useState(entry)
    if (entry !== npcKey) {
        setNpcKey(entry)
        setSelectedZone(null)
    }

    // Model renders are produced locally from the client MPQs (per-creature when
    // the NPC has weapons, else the shared display render) — no network fetching.
    // The map is a locally-generated zone map keyed by the NPC's resolved zone name.
    const displayId = detail?.displayId1
    // Pass the creature entry so a per-creature render (with held weapons) is
    // preferred over the shared display render when one exists locally.
    const modelImage = useNpcModel(displayId, imgReload, entry)
    // Portrait (head shot) for the header avatar — rendered from the model's
    // embedded portrait camera; written alongside the full-body model.
    const portraitImage = useNpcPortrait(displayId, imgReload, entry)
    // Distinct zones this NPC spawns in (primary first, by first appearance), each
    // with its spawn count — drives the zone toggle when there's more than one.
    const allSpawns = detail?.spawns || []
    const zoneList = (() => {
        const counts = new Map()
        for (const s of allSpawns) {
            if (!s.zoneName) continue
            counts.set(s.zoneName, (counts.get(s.zoneName) || 0) + 1)
        }
        return [...counts.entries()].map(([name, count]) => ({ name, count }))
    })()

    // The active zone: an explicit toggle selection, else the resolved primary.
    const activeZone = selectedZone || detail?.zoneName || zoneList[0]?.name || null
    const atlasMap = useZoneMap(activeZone, imgReload)
    // Terrain source: the zone's minimap when it has one, else a raw whole-map
    // minimap keyed by map id — covers maps with no WorldMapArea zone (e.g. the
    // development map, where spawns fall back to the "Azeroth" continent label).
    const terrainByZone = useZoneMinimap(activeZone, imgReload)
    const primaryMapId = detail?.spawns?.[0]?.mapId
    const terrainByMap = useZoneMinimap(
        primaryMapId != null ? `map_${primaryMapId}` : null,
        imgReload,
    )
    const terrainMap = terrainByZone.src ? terrainByZone : terrainByMap
    // Terrain is only offered when a minimap actually exists for this NPC.
    const terrainAvailable = !!terrainMap.src
    // Show terrain when selected and present (or still loading); otherwise the atlas.
    const showTerrain = mapStyle === 'terrain' && (terrainMap.src || terrainMap.loading)
    const mapImage = showTerrain ? terrainMap : atlasMap

    // Only plot markers for spawns in the active zone — a spawn from another zone
    // carries coordinates relative to ITS own zone map, so drawing it on this map
    // would land it in the wrong place. Fall back to all spawns if none carry a
    // zone name to match against.
    const zoneSpawns = activeZone ? allSpawns.filter((s) => s.zoneName === activeZone) : allSpawns
    const mapSpawns = zoneSpawns.length > 0 ? zoneSpawns : allSpawns

    const handleSync = () => {
        // A synced NPC can change wherever it appears (the list behind this overlay,
        // search, "dropped by" sources), so drop the cache; active queries refetch.
        SyncNpcData(entry).then(() => queryClient.invalidateQueries())
    }

    // Re-fetch just the model/map images (does NOT replace creature data).
    const handleRefreshImages = () => {
        if (refreshingImages) return
        setRefreshingImages(true)
        RefreshNpcImages(entry)
            .then((res) => {
                if (res) queryClient.setQueryData(queryKeys.npcDetail(entry), res)
                if (displayId) {
                    evictImage('npc_model', `model_${displayId}`)
                    evictImage('npc_model', `model_portrait_${displayId}`)
                }
                evictImage('npc_model', `model_creature_${entry}`)
                setImgReload((n) => n + 1)
            })
            .finally(() => setRefreshingImages(false))
    }

    const renderLootItem = (item) => {
        const handlers = tooltipHook?.getItemHandlers?.(item.itemId) || {}
        return (
            <LootItem
                key={item.itemId}
                item={{
                    entry: item.itemId,
                    name: item.name,
                    quality: item.quality,
                    iconPath: '', // Icon paths might be missing in scraping-only mode, but DB should have them if joined
                    dropChance: `${item.chance.toFixed(1)}%`,
                }}
                onClick={() => onNavigate('item', item.itemId)}
                showDropChance
                {...handlers}
            />
        )
    }

    if (loading) return <DetailLoading />
    if (!detail) return <DetailError message="NPC not found" onBack={onBack} />

    const startsQuests = detail.quests?.filter((q) => q.type === 'starts') || []
    const endsQuests = detail.quests?.filter((q) => q.type === 'ends') || []
    const loot = detail.loot || []
    const abilities = detail.abilities || []
    const sells = detail.sells || []
    const trains = detail.trains || []

    const tabs = [
        { id: 'overview', label: 'Overview' },
        { id: 'loot', label: `Loot (${loot.length})` },
        {
            id: 'quests',
            label: `Quests (${startsQuests.length + endsQuests.length})`,
        },
        { id: 'abilities', label: `Abilities (${abilities.length})` },
        ...(sells.length > 0 ? [{ id: 'sells', label: `Sells (${sells.length})` }] : []),
        ...(trains.length > 0 ? [{ id: 'trains', label: `Trains (${trains.length})` }] : []),
    ]

    // Effective tab: the URL/local value if it's a real tab, else the first.
    const currentTab = tabs.some((t) => t.id === rawTab) ? rawTab : tabs[0]?.id

    return (
        <>
            <DetailPageLayout onBack={onBack}>
                {/* --- Header Section --- */}
                <div className="mb-6 flex items-start justify-between">
                    <div className="flex items-center gap-3">
                        {/* Portrait avatar (head shot) — falls back to nothing when the
                model can't be rendered, so the title just sits flush. */}
                        {portraitImage.src && (
                            <img
                                src={portraitImage.src}
                                alt={detail.name}
                                className="h-14 w-14 flex-shrink-0 rounded-full border-2 border-white/20 bg-black object-cover shadow-lg"
                            />
                        )}
                        <div>
                            <h1
                                className={`text-2xl font-bold ${getQualityColor(
                                    detail.rank >= 1 ? 5 : 1,
                                )}`}
                            >
                                {detail.name}
                            </h1>
                            {detail.subname && (
                                <div className="mt-1 text-sm text-yellow-200">
                                    &lt;{detail.subname}&gt;
                                </div>
                            )}
                        </div>
                    </div>
                    <div className="flex gap-2">
                        <button
                            onClick={handleSync}
                            className="flex items-center gap-1 rounded border border-blue-700 bg-blue-600 px-3 py-1 text-xs font-bold text-white transition-colors hover:bg-blue-500"
                            title="Re-download data from external sources"
                        >
                            <span className="text-sm">↻</span> Sync
                        </button>
                        <a
                            href={`${DATABASE_BASE_URL}/?npc=${detail.entry}`}
                            target="_blank"
                            rel="noreferrer"
                            className="rounded border border-purple-800 bg-purple-700 px-3 py-1 text-xs font-bold text-white transition-colors hover:bg-purple-600"
                            title="View on Turtle WoW Database"
                        >
                            🔗 OctoHead
                        </a>
                        <a
                            href={`https://www.wowhead.com/classic/npc=${detail.entry}`}
                            target="_blank"
                            rel="noreferrer"
                            className="rounded border border-red-900 bg-red-800 px-3 py-1 text-xs font-bold text-white transition-colors hover:bg-red-700"
                        >
                            Wowhead
                        </a>
                    </div>
                </div>

                <div className="flex flex-col gap-8 lg:flex-row">
                    {/* --- Left Column: Visuals (Model Only) --- */}
                    <div className="w-full flex-shrink-0 space-y-4 lg:w-64">
                        {/* Model Image (if available) - Centered or Top aligned */}
                        {modelImage.loading ? (
                            <div className="flex aspect-[3/4] animate-pulse items-center justify-center rounded border border-white/10 bg-black/40 text-xs text-gray-500">
                                Loading...
                            </div>
                        ) : modelImage.src ? (
                            <div className="mb-4 overflow-hidden rounded border border-white/20 bg-black shadow-lg">
                                <img
                                    src={modelImage.src}
                                    alt={detail.name}
                                    className="h-auto w-full object-cover"
                                />
                            </div>
                        ) : (
                            <div
                                onClick={handleRefreshImages}
                                title="Click to render model from client"
                                className="flex aspect-[3/4] cursor-pointer flex-col items-center justify-center rounded border border-white/10 bg-black/40 text-xs text-gray-500 transition-colors hover:bg-black/60 hover:text-gray-300"
                            >
                                {refreshingImages ? (
                                    <span className="animate-pulse">Rendering…</span>
                                ) : (
                                    <>
                                        <span>No Model</span>
                                        <span className="mt-1 text-[10px] text-gray-600">
                                            click to render
                                        </span>
                                    </>
                                )}
                            </div>
                        )}
                    </div>

                    {/* --- Right Column: Data & Tabs --- */}
                    <div className="min-w-0 flex-1">
                        {/* Top Section: Location & Quick Facts Side-by-Side */}
                        <div className="mb-8 grid grid-cols-1 gap-6 md:grid-cols-2">
                            {/* Location Box (Updated Style) */}
                            <div className="h-fit">
                                <div className="mb-2 flex items-baseline justify-between border-b border-white/10 pb-1">
                                    <h3 className="text-sm font-bold uppercase text-wow-gold">
                                        Location
                                    </h3>
                                    {mapSpawns.length > 0 && (
                                        <span className="font-mono text-xs text-gray-400">
                                            {mapSpawns[0].zoneName || `Map ${mapSpawns[0].mapId}`}
                                            {(mapSpawns[0].x > 0 || mapSpawns[0].y > 0) && (
                                                <span className="ml-1">
                                                    ({mapSpawns[0].x.toFixed(1)},{' '}
                                                    {mapSpawns[0].y.toFixed(1)})
                                                </span>
                                            )}
                                        </span>
                                    )}
                                </div>

                                {/* Zone toggle — only when the NPC spawns in more than one zone */}
                                {zoneList.length > 1 && (
                                    <div className="mb-2 flex flex-wrap gap-1">
                                        {zoneList.map((z) => (
                                            <button
                                                key={z.name}
                                                onClick={() => setSelectedZone(z.name)}
                                                className={`rounded border px-2 py-0.5 text-[11px] font-semibold transition-colors ${
                                                    z.name === activeZone
                                                        ? 'border-wow-gold/60 bg-wow-gold/20 text-wow-gold'
                                                        : 'border-white/10 bg-bg-panel text-gray-400 hover:bg-bg-hover hover:text-gray-200'
                                                }`}
                                            >
                                                {z.name}{' '}
                                                <span className="opacity-60">{z.count}</span>
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

                                {/* User Recommended Map Style with Spawn Markers */}
                                <div
                                    className="mapper-map group relative aspect-[488/325] w-full cursor-pointer overflow-hidden rounded border border-white/20 bg-black bg-cover bg-center shadow-lg"
                                    style={{
                                        backgroundImage: mapImage.src
                                            ? `url(${mapImage.src})`
                                            : 'none',
                                        maxWidth: '488px',
                                        maxHeight: '325px',
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

                                    {/* Spawn Point Markers (only this zone's spawns) */}
                                    {mapImage.src &&
                                        mapSpawns.map((spawn, idx) => {
                                            // Only show markers for coordinates in valid 0-100 range
                                            const hasValidCoords =
                                                spawn.x > 0 &&
                                                spawn.x <= 100 &&
                                                spawn.y > 0 &&
                                                spawn.y <= 100
                                            if (!hasValidCoords) return null

                                            return (
                                                <div
                                                    key={idx}
                                                    className="group/marker absolute z-10 -ml-2 -mt-2 h-4 w-4 cursor-pointer"
                                                    style={{
                                                        left: `${spawn.x}%`,
                                                        top: `${spawn.y}%`,
                                                    }}
                                                    onClick={(e) => e.stopPropagation()}
                                                >
                                                    {/* Outer pulsing ring */}
                                                    <div className="absolute inset-0 animate-ping rounded-full bg-red-500/50" />
                                                    {/* Inner solid dot */}
                                                    <div className="absolute inset-0.5 rounded-full border border-red-400 bg-red-600 shadow-lg" />

                                                    {/* Tooltip on hover */}
                                                    <div className="pointer-events-none absolute bottom-full left-1/2 mb-2 -translate-x-1/2 whitespace-nowrap rounded border border-white/20 bg-black/90 px-2 py-1 text-xs text-white opacity-0 shadow-lg transition-opacity group-hover/marker:opacity-100">
                                                        <div className="font-semibold text-wow-gold">
                                                            {spawn.zoneName || 'Spawn Point'}
                                                        </div>
                                                        <div className="font-mono text-gray-300">
                                                            ({spawn.x.toFixed(1)},{' '}
                                                            {spawn.y.toFixed(1)})
                                                        </div>
                                                        {/* Arrow */}
                                                        <div className="absolute left-1/2 top-full -translate-x-1/2 border-4 border-transparent border-t-black/90" />
                                                    </div>
                                                </div>
                                            )
                                        })}

                                    {/* Overlay / Expander */}
                                    <div className="pointer-events-none absolute inset-0 bg-transparent transition-colors group-hover:bg-white/5"></div>
                                    <div className="absolute right-2 top-2 flex h-6 w-6 items-center justify-center rounded bg-black/50 text-white/80 opacity-0 transition-opacity group-hover:opacity-100">
                                        ⤢
                                    </div>

                                    {/* Spawn Count Badge — count markers shown on THIS zone map */}
                                    {mapSpawns.length > 1 && (
                                        <div className="absolute left-2 top-2 rounded border border-white/10 bg-black/70 px-2 py-0.5 text-xs text-gray-300">
                                            {mapSpawns.length} spawns
                                        </div>
                                    )}

                                    {/* Zoom Tip */}
                                    <div className="absolute bottom-0 left-0 right-0 bg-black/80 py-1 text-center text-xs text-gray-300 opacity-0 transition-opacity group-hover:opacity-100">
                                        Tip: Click map to zoom
                                    </div>
                                </div>
                            </div>

                            {/* Quick Facts / Stats Block */}
                            <div>
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
                                            <th>Level:</th>
                                            <td>
                                                {detail.levelMin !== detail.levelMax
                                                    ? `${detail.levelMin} - ${detail.levelMax}`
                                                    : detail.levelMax}
                                            </td>
                                        </tr>
                                        <tr>
                                            <th>Classification:</th>
                                            <td>{detail.rankName || detail.rank}</td>
                                        </tr>
                                        <tr>
                                            <th>React:</th>
                                            <td>
                                                <span
                                                    className={
                                                        detail.faction === 35
                                                            ? 'text-wow-quality-2'
                                                            : 'text-wow-quality-7'
                                                    }
                                                >
                                                    A
                                                </span>{' '}
                                                <span
                                                    className={
                                                        detail.faction === 35
                                                            ? 'text-wow-quality-2'
                                                            : 'text-wow-quality-7'
                                                    }
                                                >
                                                    H
                                                </span>
                                            </td>
                                        </tr>
                                        <tr>
                                            <th>Faction:</th>
                                            <td>
                                                {detail.factionName ? (
                                                    <>
                                                        <span
                                                            onClick={() =>
                                                                onNavigate(
                                                                    'faction',
                                                                    detail.factionId,
                                                                )
                                                            }
                                                            className="cursor-pointer text-wow-gold hover:text-yellow-300"
                                                        >
                                                            {detail.factionName}
                                                        </span>
                                                        <span className="ml-1 text-gray-500">
                                                            ({detail.faction})
                                                        </span>
                                                    </>
                                                ) : (
                                                    detail.faction
                                                )}
                                            </td>
                                        </tr>
                                        <tr>
                                            <th>Health:</th>
                                            <td>
                                                {detail.healthMin !== detail.healthMax
                                                    ? `${detail.healthMin} - ${detail.healthMax}`
                                                    : detail.healthMax}
                                            </td>
                                        </tr>
                                        {(detail.manaMin > 0 || detail.manaMax > 0) && (
                                            <tr>
                                                <th>Mana:</th>
                                                <td>
                                                    {detail.manaMin !== detail.manaMax
                                                        ? `${detail.manaMin} - ${detail.manaMax}`
                                                        : detail.manaMax}
                                                </td>
                                            </tr>
                                        )}
                                        {(detail.goldMin > 0 || detail.goldMax > 0) && (
                                            <tr>
                                                <th>Wealth:</th>
                                                <td>
                                                    {detail.goldMin > 0 &&
                                                    detail.goldMin !== detail.goldMax ? (
                                                        <span className="inline-flex items-center gap-1">
                                                            <Money copper={detail.goldMin} />
                                                            <span className="text-gray-500">-</span>
                                                            <Money copper={detail.goldMax} />
                                                        </span>
                                                    ) : (
                                                        <Money
                                                            copper={
                                                                detail.goldMax || detail.goldMin
                                                            }
                                                        />
                                                    )}
                                                </td>
                                            </tr>
                                        )}
                                        {(detail.minDmg > 0 || detail.maxDmg > 0) && (
                                            <tr>
                                                <th>Damage:</th>
                                                <td>
                                                    {Math.floor(detail.minDmg)} -{' '}
                                                    {Math.floor(detail.maxDmg)}
                                                </td>
                                            </tr>
                                        )}
                                        {detail.armor > 0 && (
                                            <tr>
                                                <th>Armor:</th>
                                                <td>{detail.armor}</td>
                                            </tr>
                                        )}
                                        <tr>
                                            <th>Display ID:</th>
                                            <td>{detail.displayId1}</td>
                                        </tr>
                                    </tbody>
                                </table>
                            </div>
                        </div>

                        {/* Tabs Navigation */}
                        <div className="mb-4 flex gap-1 border-b border-white/20">
                            {tabs.map((tab) => (
                                <button
                                    key={tab.id}
                                    onClick={() => setTab(tab.id)}
                                    className={`relative top-[1px] px-4 py-2 text-sm font-bold transition-all ${
                                        currentTab === tab.id
                                            ? 'tab-btn-active border-b-2 border-wow-gold text-white'
                                            : 'tab-btn-inactive text-gray-400 hover:text-gray-200'
                                    }`}
                                >
                                    {tab.label}
                                </button>
                            ))}
                        </div>

                        {/* Tab Content */}
                        <div className="animate-fade-in min-h-[200px]">
                            {currentTab === 'overview' && (
                                <div className="text-sm text-gray-400">
                                    <h4 className="mb-2 font-bold text-white">Abilities Summary</h4>
                                    {detail.abilities?.length > 0 ? (
                                        <ul className="list-disc space-y-1 pl-5">
                                            {detail.abilities.slice(0, 5).map((spell, i) => (
                                                <li key={spell.spellId || i}>
                                                    <span
                                                        onClick={() =>
                                                            spell.spellId &&
                                                            onNavigate('spell', spell.spellId)
                                                        }
                                                        className={`text-wow-quality-1 ${
                                                            spell.spellId
                                                                ? 'cursor-pointer hover:text-wow-gold'
                                                                : ''
                                                        }`}
                                                    >
                                                        {spell.name}
                                                    </span>
                                                </li>
                                            ))}
                                        </ul>
                                    ) : (
                                        'No abilities found.'
                                    )}
                                </div>
                            )}

                            {currentTab === 'loot' && (
                                <div className="animate-fade-in">
                                    {loot.length > 0 ? (
                                        <LootGrid>
                                            {loot
                                                .sort((a, b) => b.chance - a.chance)
                                                .map((item) => {
                                                    // Support both entry and itemId for compatibility
                                                    const itemId = item.entry || item.itemId
                                                    const handlers =
                                                        tooltipHook?.getItemHandlers?.(itemId) || {}
                                                    return (
                                                        <LootItem
                                                            key={itemId}
                                                            item={{
                                                                entry: itemId,
                                                                name: item.name,
                                                                quality: item.quality,
                                                                iconPath:
                                                                    item.iconPath ||
                                                                    item.icon ||
                                                                    '',
                                                                dropChance: `${item.chance.toFixed(1)}%`,
                                                            }}
                                                            onClick={() =>
                                                                onNavigate('item', itemId)
                                                            }
                                                            showDropChance
                                                            {...handlers}
                                                        />
                                                    )
                                                })}
                                        </LootGrid>
                                    ) : (
                                        <div className="p-8 text-center italic text-gray-500">
                                            No loot table available.
                                        </div>
                                    )}
                                </div>
                            )}

                            {currentTab === 'sells' && (
                                <div className="animate-fade-in">
                                    <LootGrid>
                                        {sells.map((item) => {
                                            const handlers =
                                                tooltipHook?.getItemHandlers?.(item.itemId) || {}
                                            return (
                                                <LootItem
                                                    key={item.itemId}
                                                    item={{
                                                        entry: item.itemId,
                                                        name: item.name || `Item ${item.itemId}`,
                                                        quality: item.quality,
                                                        iconPath: item.iconPath || '',
                                                        dropChance:
                                                            item.cost > 0 ? (
                                                                <Money copper={item.cost} />
                                                            ) : null,
                                                    }}
                                                    onClick={() => onNavigate('item', item.itemId)}
                                                    showDropChance={item.cost > 0}
                                                    {...handlers}
                                                />
                                            )
                                        })}
                                    </LootGrid>
                                </div>
                            )}

                            {currentTab === 'trains' && (
                                <div className="animate-fade-in bg-bg-sub rounded border border-border-light">
                                    {trains.map((sp, i) => (
                                        <div
                                            key={sp.spellId}
                                            onClick={() => onNavigate('spell', sp.spellId)}
                                            className={`flex cursor-pointer items-center gap-3 p-2.5 transition-colors hover:bg-white/5 ${
                                                i !== trains.length - 1
                                                    ? 'border-b border-border-light/50'
                                                    : ''
                                            }`}
                                        >
                                            <AbilityIcon iconName={sp.iconName} />
                                            <span className="flex-1 truncate text-sm font-medium text-wow-rare">
                                                {sp.name || `Spell ${sp.spellId}`}
                                                {sp.subtext && (
                                                    <span className="ml-1 text-gray-500">
                                                        ({sp.subtext})
                                                    </span>
                                                )}
                                            </span>
                                            {sp.level > 0 && (
                                                <span className="whitespace-nowrap text-xs text-gray-500">
                                                    Req Lvl {sp.level}
                                                </span>
                                            )}
                                        </div>
                                    ))}
                                </div>
                            )}

                            {currentTab === 'quests' && (
                                <div className="animate-fade-in grid grid-cols-1 gap-6 md:grid-cols-2">
                                    <DetailSection title={`Starts Quests (${startsQuests.length})`}>
                                        {startsQuests.length > 0 ? (
                                            <div className="bg-bg-sub rounded border border-border-light">
                                                {startsQuests.map((q, i) => (
                                                    <div
                                                        key={q.entry || q.questId}
                                                        onClick={() =>
                                                            onNavigate(
                                                                'quest',
                                                                q.entry || q.questId,
                                                            )
                                                        }
                                                        className={`flex cursor-pointer items-center justify-between p-3 transition-colors hover:bg-white/5 ${
                                                            i !== startsQuests.length - 1
                                                                ? 'border-b border-border-light/50'
                                                                : ''
                                                        }`}
                                                    >
                                                        <span className="hover:text-wow-gold-light truncate font-medium text-wow-gold md:text-sm">
                                                            {q.name || q.title}
                                                        </span>
                                                    </div>
                                                ))}
                                            </div>
                                        ) : (
                                            <div className="italic text-gray-500">None</div>
                                        )}
                                    </DetailSection>

                                    <DetailSection title={`Ends Quests (${endsQuests.length})`}>
                                        {endsQuests.length > 0 ? (
                                            <div className="bg-bg-sub rounded border border-border-light">
                                                {endsQuests.map((q, i) => (
                                                    <div
                                                        key={q.entry || q.questId}
                                                        onClick={() =>
                                                            onNavigate(
                                                                'quest',
                                                                q.entry || q.questId,
                                                            )
                                                        }
                                                        className={`flex cursor-pointer items-center justify-between p-3 transition-colors hover:bg-white/5 ${
                                                            i !== endsQuests.length - 1
                                                                ? 'border-b border-border-light/50'
                                                                : ''
                                                        }`}
                                                    >
                                                        <span className="hover:text-wow-gold-light truncate font-medium text-wow-gold md:text-sm">
                                                            {q.name || q.title}
                                                        </span>
                                                    </div>
                                                ))}
                                            </div>
                                        ) : (
                                            <div className="italic text-gray-500">None</div>
                                        )}
                                    </DetailSection>
                                </div>
                            )}

                            {currentTab === 'abilities' && (
                                <div className="animate-fade-in">
                                    {abilities.length > 0 ? (
                                        <div className="grid grid-cols-1 gap-4">
                                            {abilities.map((spell, idx) => (
                                                <div
                                                    key={spell.spellId || idx}
                                                    onClick={() =>
                                                        spell.spellId &&
                                                        onNavigate('spell', spell.spellId)
                                                    }
                                                    className={`bg-bg-sub hover:border-border-hover rounded border border-border-light p-4 transition-colors ${
                                                        spell.spellId ? 'cursor-pointer' : ''
                                                    }`}
                                                >
                                                    <div className="mb-2 flex items-start justify-between">
                                                        <div className="flex items-center gap-3">
                                                            <AbilityIcon iconName={spell.icon} />
                                                            <h4 className="text-wow-quality-1 text-lg font-bold hover:text-wow-gold">
                                                                {spell.name}
                                                            </h4>
                                                        </div>
                                                    </div>
                                                    <p className="pl-[3.25rem] text-sm leading-relaxed text-gray-300">
                                                        {spell.description &&
                                                        spell.description.length > 2 ? (
                                                            spell.description
                                                        ) : (
                                                            <span className="italic text-gray-600">
                                                                No description available.
                                                            </span>
                                                        )}
                                                    </p>
                                                </div>
                                            ))}
                                        </div>
                                    ) : (
                                        <div className="p-8 text-center italic text-gray-500">
                                            No abilities data found.
                                        </div>
                                    )}
                                </div>
                            )}
                        </div>
                    </div>
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
                        {/* Map Image Container with Markers */}
                        <div className="relative">
                            <img
                                src={mapImage.src}
                                alt={activeZone || 'Location Map'}
                                className="max-h-[85vh] max-w-full rounded-lg object-contain shadow-2xl"
                            />

                            {/* Spawn Point Markers on Modal Map (only this zone's spawns) */}
                            {mapSpawns.map((spawn, idx) => {
                                const hasValidCoords =
                                    spawn.x > 0 && spawn.x <= 100 && spawn.y > 0 && spawn.y <= 100
                                if (!hasValidCoords) return null

                                return (
                                    <div
                                        key={idx}
                                        className="pointer-events-none absolute -ml-2.5 -mt-2.5 h-5 w-5"
                                        style={{
                                            left: `${spawn.x}%`,
                                            top: `${spawn.y}%`,
                                        }}
                                        title={`${spawn.zoneName || 'Spawn'} (${spawn.x.toFixed(1)}, ${spawn.y.toFixed(1)})`}
                                    >
                                        {/* Outer pulsing ring */}
                                        <div className="absolute inset-0 animate-ping rounded-full bg-red-500/50" />
                                        {/* Inner solid dot */}
                                        <div className="absolute inset-1 rounded-full border-2 border-red-400 bg-red-600 shadow-lg" />
                                    </div>
                                )
                            })}
                        </div>

                        {/* Zone Name Label */}
                        {(activeZone || mapSpawns.length > 0) && (
                            <div className="absolute bottom-4 left-1/2 -translate-x-1/2 rounded-lg bg-black/80 px-4 py-2 font-bold text-white">
                                {activeZone || mapSpawns[0]?.zoneName || 'Unknown Zone'}
                                {mapSpawns[0]?.x > 0 &&
                                    ` (${mapSpawns[0].x.toFixed(1)}, ${mapSpawns[0].y.toFixed(1)})`}
                                {mapSpawns.length > 1 && (
                                    <span className="ml-2 text-sm text-gray-400">
                                        +{mapSpawns.length - 1} more
                                    </span>
                                )}
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

export default NPCDetailView
