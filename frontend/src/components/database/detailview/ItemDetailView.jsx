import React, { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { tooltipQuery } from '../../../hooks/queries/tooltip'
import { useItemDetail, useItemFavorite } from '../../../hooks/queries/items'
import { queryClient } from '../../../queryClient'
import { ToggleFavorite } from '../../../services/api'
import { FixSingleItemIcon, SyncSingleItem } from '../../../../wailsjs/go/main/App'
import { getQualityColor, QUESTION_MARK_ICON } from '../../../utils/wow'
import { DATABASE_BASE_URL } from '../../../utils/constants'
import { useIcon, useNpcModel } from '../../../services/useImage'
import {
    DetailPageLayout,
    DetailHeader,
    DetailLoading,
    DetailError,
    IconPopupAnchor,
    ItemTooltip,
    Money,
    ZoneName,
} from '../../ui'

// reactionColor maps a faction reaction ("friendly"/"hostile"/"neutral") to a
// text color: friendly = green, hostile = red, neutral = gray.
const reactionColor = (reaction) => {
    switch (reaction) {
        case 'friendly':
            return 'text-green-400'
        case 'hostile':
            return 'text-red-400'
        default:
            return 'text-gray-500'
    }
}

// RelTable / RelRow: the shared table shell for the item's relation tabs, so
// Sold By, Dropped By, Created By, etc. all render as consistent tables. columns
// is [{ label, align }]; the first cell is left-padded-right, the last is
// left-padded, the rest get horizontal padding.
const cellPad = (i, n) => (i === 0 ? 'pr-2' : i === n - 1 ? 'pl-2' : 'px-2')
const alignClass = (a) =>
    a === 'right' ? 'text-right' : a === 'center' ? 'text-center' : 'text-left'

const RelTable = ({ columns, children }) => (
    <table className="w-full text-sm">
        <thead>
            <tr className="border-b border-white/10 text-[11px] uppercase tracking-wider text-gray-500">
                {columns.map((c, i) => (
                    <th
                        key={i}
                        className={`py-1.5 font-semibold ${alignClass(c.align)} ${cellPad(i, columns.length)}`}
                    >
                        {c.label}
                    </th>
                ))}
            </tr>
        </thead>
        <tbody>{children}</tbody>
    </table>
)

const RelRow = ({ onClick, children }) => (
    <tr
        className="cursor-pointer border-b border-white/5 transition-colors hover:bg-white/5"
        onClick={onClick}
    >
        {children}
    </tr>
)

// ReagentIcon: a compact icon-only reagent (with count badge) for the Reagents
// column of the Created By table. Clicking opens the reagent's item page without
// triggering the row's own navigation.
const ReagentIcon = ({ reagent, onNavigate, tooltipHook }) => {
    const handlers = tooltipHook?.getItemHandlers?.(reagent.entry) || {}
    return (
        <div
            className="relative h-7 w-7 shrink-0 cursor-pointer"
            title={reagent.name || `Item #${reagent.entry}`}
            onClick={(e) => {
                e.stopPropagation()
                onNavigate?.('item', reagent.entry)
            }}
            {...handlers}
        >
            <IconImg
                name={reagent.iconPath}
                className="h-full w-full rounded border border-black/40 object-cover"
            />
            {reagent.count > 1 && (
                <span className="absolute -bottom-0.5 -right-0.5 rounded bg-black/80 px-0.5 text-[10px] font-bold leading-none text-white">
                    {reagent.count}
                </span>
            )}
        </div>
    )
}

// Helper component for Icon Header. Hovering the icon pops up its name
// (copyable) with a link to the icon page listing everything that uses it.
const ItemIconHeader = ({
    iconName,
    iconPath,
    imgError,
    fixing,
    handleFixIcon,
    qualityColor,
    onNavigate,
}) => {
    // Determine icon name to use
    const name = iconPath || iconName
    const icon = useIcon(name)

    // If explicit error state (from parent) or missing icon name
    const showFixButton = !name || imgError

    if (showFixButton) {
        return (
            <button
                onClick={handleFixIcon}
                disabled={fixing}
                className="flex h-full w-full flex-col items-center justify-center gap-1 bg-red-900/30 text-red-400 transition-colors hover:bg-red-800/50"
                title={
                    !name ? 'No icon data - Click to fetch' : 'Icon failed to load - Click to fix'
                }
            >
                <span className="text-2xl">{fixing ? '⏳' : '🔧'}</span>
                <span className="text-[10px]">{fixing ? 'Fixing...' : 'Fix Icon'}</span>
            </button>
        )
    }

    return (
        <IconPopupAnchor name={name} onNavigate={onNavigate} className="h-full w-full">
            {icon.loading ? (
                <div className="h-full w-full animate-pulse bg-white/5" />
            ) : (
                <img
                    src={icon.src || QUESTION_MARK_ICON}
                    className="h-full w-full object-cover"
                    alt=""
                />
            )}
        </IconPopupAnchor>
    )
}

// Small icon image resolved through the cached icon service.
const IconImg = ({ name, className }) => {
    const icon = useIcon(name)
    return <img src={icon.src || QUESTION_MARK_ICON} className={className} alt="" />
}

const ItemDetailView = ({ entry, onBack, onNavigate, tooltipHook, activeTab, onTabChange }) => {
    // The item's tooltip payload, from the shared Query cache (warmed by hover
    // elsewhere). The blanket invalidate in reloadData refetches this too.
    const { data: tooltip } = useQuery({ ...tooltipQuery(entry), enabled: !!entry })
    const { data: detail, isLoading: loading } = useItemDetail(entry)
    // Mount model: for a mount item, detail.mountDisplayId is a creature display
    // id we render the same way NPC pages do (entry 0 = shared display render).
    // displayId 0 (non-mount) is a no-op in the hook.
    const mountModel = useNpcModel(detail?.mountDisplayId || 0, 0, 0)
    const [imgError, setImgError] = useState(false)
    const [fixing, setFixing] = useState(false)
    const [syncing, setSyncing] = useState(false)
    // Active relations tab. When the route supplies it (activeTab/onTabChange) it
    // lives in the URL so Back/Forward and refresh work; otherwise fall back to
    // local state. null = first tab with data.
    const [localRelTab, setLocalRelTab] = useState(null)
    const activeRelTab = onTabChange ? activeTab : localRelTab
    const selectRelTab = onTabChange || setLocalRelTab

    // Favorite status, cached per item and derived from the query (not state);
    // toggled optimistically in handleFavoriteToggle.
    const { data: isFavorite = false } = useItemFavorite(entry)

    // Reset the icon-error flag when the item changes (render-time, no effect).
    const [imgErrKey, setImgErrKey] = useState(entry)
    if (entry !== imgErrKey) {
        setImgErrKey(entry)
        setImgError(false)
        setLocalRelTab(null)
    }

    // A sync or icon-fix can change this item anywhere it appears (the grid behind
    // this overlay, search, sets, loot tables, tooltips), so drop the whole cache;
    // active queries — this detail, its tooltip, the list — refetch immediately.
    const reloadData = async () => {
        await queryClient.invalidateQueries()
    }

    const handleFixIcon = async () => {
        setFixing(true)
        try {
            const result = await FixSingleItemIcon(entry)
            if (result.fixed > 0) {
                setImgError(false)
                await reloadData()
            } else {
                alert(
                    `Auto-fetch failed: ${result.message}\n\n` +
                        `This item's icon data could not be automatically retrieved.\n` +
                        `Visit ${DATABASE_BASE_URL}/?item=${entry} to check if the item exists.`,
                )
            }
        } catch (error) {
            alert(`Error: ${error}`)
        } finally {
            setFixing(false)
        }
    }

    const handleFavoriteToggle = async () => {
        let category = ''
        if (!isFavorite) {
            // If adding, ask for category (optional)
            const userInput = window.prompt(
                'Enter a category for this favorite (optional):',
                'General',
            )
            if (userInput === null) return // Cancelled
            category = userInput
        }

        try {
            const result = await ToggleFavorite(entry, category)
            if (result.success) {
                queryClient.setQueryData(['itemFavorite', entry], !isFavorite)
            } else {
                alert('Failed to toggle favorite: ' + result.message)
            }
        } catch (err) {
            console.error('Favorite error:', err)
        }
    }

    // Sync full item data from turtlecraft.gg
    const handleSync = async () => {
        setSyncing(true)
        try {
            const result = await SyncSingleItem(entry)
            if (result && result.success) {
                setImgError(false)
                await reloadData()
            } else {
                alert(`Sync failed: ${result?.error || 'Unknown error'}`)
            }
        } catch (error) {
            alert(`Sync error: ${error}`)
        } finally {
            setSyncing(false)
        }
    }

    const renderTooltipBlock = () => {
        if (!detail) return null
        const dummyItem = {
            entry: detail.entry,
            quality: detail.quality,
            name: detail.name,
        }

        return (
            <div className="inline-block min-w-[300px] align-top">
                <ItemTooltip
                    item={dummyItem}
                    tooltip={tooltip}
                    style={{ position: 'static' }}
                    interactive={true}
                    onSpellClick={(spellId) => onNavigate?.('spell', spellId)}
                    onItemClick={(id) => onNavigate?.('item', id)}
                    tooltipHook={tooltipHook}
                />
            </div>
        )
    }

    if (loading) return <DetailLoading />

    if (!detail) {
        return (
            <DetailPageLayout onBack={onBack}>
                <div className="flex flex-col items-center justify-center gap-6 p-20 text-gray-400">
                    <div className="text-xl">
                        Item <span className="font-mono text-white">{entry}</span> not found in
                        local database.
                    </div>
                    <p className="max-w-md text-center text-sm text-gray-500">
                        This item exists in the remote database reference but hasn't been synced to
                        your local database yet.
                    </p>
                    <button
                        onClick={handleSync}
                        disabled={syncing}
                        className={`rounded bg-wow-gold px-6 py-3 font-bold uppercase tracking-wider text-black shadow-[0_0_10px_rgba(255,209,0,0.2)] transition-all hover:bg-yellow-400 hover:shadow-[0_0_15px_rgba(255,209,0,0.4)] disabled:cursor-not-allowed disabled:opacity-50`}
                    >
                        {syncing ? (
                            <span className="flex items-center gap-2">
                                <span className="animate-spin">↻</span> Syncing...
                            </span>
                        ) : (
                            'Sync from Remote'
                        )}
                    </button>
                </div>
            </DetailPageLayout>
        )
    }

    const qualityColor = getQualityColor(detail.quality)

    // Relation sections become tabs (Contained In / Dropped By etc. can get
    // long). Only tabs with data are shown, each with a count; the active tab
    // defaults to the first available.
    const relationTabs = [
        detail.createdBy?.length && {
            id: 'createdBy',
            label: 'Created By',
            count: detail.createdBy.length,
            content: (
                <RelTable
                    columns={[
                        { label: 'Name' },
                        { label: 'Skill' },
                        { label: 'Creates', align: 'right' },
                        { label: 'Reagents' },
                    ]}
                >
                    {detail.createdBy.map((source) => (
                        <RelRow
                            key={source.spellId}
                            onClick={() => onNavigate?.('spell', source.spellId)}
                        >
                            <td className="py-1.5 pr-2">
                                <div className="flex min-w-0 items-center gap-2">
                                    <IconImg
                                        name={source.spellIcon}
                                        className="h-7 w-7 shrink-0 rounded border border-black/40 object-cover"
                                    />
                                    <span
                                        className="truncate font-bold text-wow-rare hover:underline"
                                        {...(tooltipHook?.getSpellHandlers?.(source.spellId) || {})}
                                    >
                                        {source.spellName}
                                    </span>
                                </div>
                            </td>
                            <td className="px-2 py-1.5 text-gray-400">
                                {source.skillName
                                    ? `${source.skillName}${source.reqSkill > 0 ? ` (${source.reqSkill})` : ''}`
                                    : '—'}
                            </td>
                            <td className="px-2 py-1.5 text-right font-mono text-gray-300">
                                {source.producedCount > 1 ? source.producedCount : '1'}
                            </td>
                            <td className="py-1.5 pl-2">
                                {source.reagents?.length > 0 ? (
                                    <div className="flex flex-wrap gap-1">
                                        {source.reagents.map((rg) => (
                                            <ReagentIcon
                                                key={rg.entry}
                                                reagent={rg}
                                                onNavigate={onNavigate}
                                                tooltipHook={tooltipHook}
                                            />
                                        ))}
                                    </div>
                                ) : (
                                    <span className="text-gray-600">—</span>
                                )}
                            </td>
                        </RelRow>
                    ))}
                </RelTable>
            ),
        },
        detail.reagentFor?.length && {
            id: 'reagentFor',
            label: 'Reagent For',
            count: detail.reagentFor.length,
            content: (
                <RelTable
                    columns={[
                        { label: 'Name' },
                        { label: 'Skill' },
                        { label: 'Creates', align: 'right' },
                        { label: 'Uses', align: 'right' },
                    ]}
                >
                    {detail.reagentFor.map((use) => {
                        // Prefer the produced item as the row (clickable to it);
                        // fall back to the spell when the recipe makes no item.
                        const toItem = use.producedItem > 0
                        return (
                            <RelRow
                                key={use.spellId}
                                onClick={() =>
                                    toItem
                                        ? onNavigate('item', use.producedItem)
                                        : onNavigate('spell', use.spellId)
                                }
                            >
                                <td className="py-1.5 pr-2">
                                    <div
                                        className="flex min-w-0 items-center gap-2"
                                        {...(toItem
                                            ? {}
                                            : tooltipHook?.getSpellHandlers?.(use.spellId) || {})}
                                    >
                                        <IconImg
                                            name={toItem ? use.producedIcon : use.spellIcon}
                                            className="h-7 w-7 shrink-0 rounded border border-black/40 object-cover"
                                        />
                                        <span
                                            className="truncate font-bold hover:underline"
                                            style={{
                                                color: toItem
                                                    ? getQualityColor(use.producedQuality)
                                                    : '#a335ee',
                                            }}
                                        >
                                            {toItem ? use.producedName : use.spellName}
                                        </span>
                                    </div>
                                </td>
                                <td className="px-2 py-1.5 text-gray-400">
                                    {use.skillName
                                        ? `${use.skillName}${use.reqSkill > 0 ? ` (${use.reqSkill})` : ''}`
                                        : '—'}
                                </td>
                                <td className="px-2 py-1.5 text-right font-mono text-gray-300">
                                    {use.producedCount > 1 ? use.producedCount : '1'}
                                </td>
                                <td className="py-1.5 pl-2 text-right font-mono text-gray-300">
                                    {use.reagentCount > 1 ? use.reagentCount : '1'}
                                </td>
                            </RelRow>
                        )
                    })}
                </RelTable>
            ),
        },
        detail.gatheredFrom?.length && {
            id: 'gatheredFrom',
            label: 'Gathered From',
            count: detail.gatheredFrom.length,
            content: (
                <RelTable
                    columns={[
                        { label: 'Name' },
                        { label: 'Skill' },
                        { label: 'Chance', align: 'right' },
                    ]}
                >
                    {detail.gatheredFrom.map((c) => (
                        <RelRow key={c.entry} onClick={() => onNavigate('object', c.entry)}>
                            <td className="py-1.5 pr-2">
                                <div className="flex min-w-0 items-center gap-2">
                                    <span className="flex h-7 w-7 shrink-0 items-center justify-center rounded border border-[#00B4FF]/40 bg-[#00B4FF]/20 text-[9px] font-bold text-[#00B4FF]">
                                        OBJ
                                    </span>
                                    <span className="truncate font-bold text-[#00B4FF] hover:underline">
                                        {c.name}
                                    </span>
                                </div>
                            </td>
                            <td className="px-2 py-1.5 text-gray-400">
                                {c.skill
                                    ? `${c.skill}${c.skillReq > 0 ? ` (${c.skillReq})` : ''}`
                                    : '—'}
                            </td>
                            <td className="py-1.5 pl-2 text-right font-mono text-wow-gold">
                                {c.chance > 0 ? `${c.chance.toFixed(1)}%` : '—'}
                            </td>
                        </RelRow>
                    ))}
                </RelTable>
            ),
        },
        detail.containedIn?.length && {
            id: 'containedIn',
            label: 'Contained In',
            count: detail.containedIn.length,
            content: (
                <RelTable
                    columns={[
                        { label: 'Name' },
                        { label: 'Skill' },
                        { label: 'Chance', align: 'right' },
                    ]}
                >
                    {detail.containedIn.map((c) => (
                        <RelRow key={c.entry} onClick={() => onNavigate('object', c.entry)}>
                            <td className="py-1.5 pr-2">
                                <div className="flex min-w-0 items-center gap-2">
                                    <span className="flex h-7 w-7 shrink-0 items-center justify-center rounded border border-[#00B4FF]/40 bg-[#00B4FF]/20 text-[9px] font-bold text-[#00B4FF]">
                                        OBJ
                                    </span>
                                    <span className="truncate font-bold text-[#00B4FF] hover:underline">
                                        {c.name}
                                    </span>
                                </div>
                            </td>
                            <td className="px-2 py-1.5 text-gray-400">
                                {c.skill
                                    ? `${c.skill}${c.skillReq > 0 ? ` (${c.skillReq})` : ''}`
                                    : '—'}
                            </td>
                            <td className="py-1.5 pl-2 text-right font-mono text-wow-gold">
                                {c.chance > 0 ? `${c.chance.toFixed(1)}%` : '—'}
                            </td>
                        </RelRow>
                    ))}
                </RelTable>
            ),
        },
        detail.containedInItem?.length && {
            id: 'containedInItem',
            label: 'Contained In Item',
            count: detail.containedInItem.length,
            content: (
                <RelTable columns={[{ label: 'Name' }, { label: 'Chance', align: 'right' }]}>
                    {detail.containedInItem.map((c) => (
                        <RelRow key={c.entry} onClick={() => onNavigate('item', c.entry)}>
                            <td className="py-1.5 pr-2">
                                <div className="flex min-w-0 items-center gap-2">
                                    <IconImg
                                        name={c.iconPath}
                                        className="h-7 w-7 shrink-0 rounded border border-black/40 object-cover"
                                    />
                                    <span
                                        className="truncate font-bold hover:underline"
                                        style={{ color: getQualityColor(c.quality) }}
                                    >
                                        {c.name}
                                    </span>
                                </div>
                            </td>
                            <td className="py-1.5 pl-2 text-right font-mono text-wow-gold">
                                {c.chance > 0 ? `${c.chance.toFixed(1)}%` : '—'}
                            </td>
                        </RelRow>
                    ))}
                </RelTable>
            ),
        },
        detail.droppedBy?.length && {
            id: 'droppedBy',
            label: 'Dropped By',
            count: detail.droppedBy.length,
            content: (
                <RelTable
                    columns={[
                        { label: 'Name' },
                        { label: 'Level', align: 'right' },
                        { label: 'Chance', align: 'right' },
                    ]}
                >
                    {detail.droppedBy.map((npc) => (
                        <RelRow key={npc.entry} onClick={() => onNavigate('npc', npc.entry)}>
                            <td className="py-1.5 pr-2 font-bold text-white">{npc.name}</td>
                            <td className="px-2 py-1.5 text-right font-mono text-gray-300">
                                {npc.levelMin}
                                {npc.levelMax > npc.levelMin ? `-${npc.levelMax}` : ''}
                            </td>
                            <td className="py-1.5 pl-2 text-right font-mono text-wow-gold">
                                {npc.chance.toFixed(1)}%
                            </td>
                        </RelRow>
                    ))}
                </RelTable>
            ),
        },
        detail.soldBy?.length && {
            id: 'soldBy',
            label: 'Sold By',
            count: detail.soldBy.length,
            content: (
                <table className="w-full text-sm">
                    <thead>
                        <tr className="border-b border-white/10 text-left text-[11px] uppercase tracking-wider text-gray-500">
                            <th className="py-1.5 pr-2 font-semibold">Name</th>
                            <th className="px-2 py-1.5 font-semibold">Location</th>
                            <th className="px-2 py-1.5 text-center font-semibold">React</th>
                            <th className="px-2 py-1.5 text-right font-semibold">Stock</th>
                            <th className="px-2 py-1.5 text-right font-semibold">Stack</th>
                            <th className="py-1.5 pl-2 text-right font-semibold">Cost</th>
                        </tr>
                    </thead>
                    <tbody>
                        {detail.soldBy.map((npc) => {
                            return (
                                <tr
                                    key={npc.entry}
                                    className="cursor-pointer border-b border-white/5 transition-colors hover:bg-white/5"
                                    onClick={() => onNavigate('npc', npc.entry)}
                                >
                                    <td className="py-1.5 pr-2">
                                        <div className="font-bold text-white">{npc.name}</div>
                                        <div className="text-xs text-gray-500">
                                            Level {npc.levelMin}
                                            {npc.levelMax > npc.levelMin ? `-${npc.levelMax}` : ''}
                                        </div>
                                    </td>
                                    <td className="px-2 py-1.5 text-wow-rare">
                                        <ZoneName
                                            name={npc.location}
                                            onNavigate={onNavigate}
                                            fallback="—"
                                        />
                                    </td>
                                    <td className="px-2 py-1.5 text-center">
                                        {npc.reactionA || npc.reactionH ? (
                                            <span className="inline-flex gap-0.5 font-mono text-[11px] font-bold">
                                                <span
                                                    className={reactionColor(npc.reactionA)}
                                                    title={`Alliance: ${npc.reactionA}`}
                                                >
                                                    A
                                                </span>
                                                <span
                                                    className={reactionColor(npc.reactionH)}
                                                    title={`Horde: ${npc.reactionH}`}
                                                >
                                                    H
                                                </span>
                                            </span>
                                        ) : (
                                            <span className="text-gray-600">—</span>
                                        )}
                                    </td>
                                    <td className="px-2 py-1.5 text-right font-mono text-gray-300">
                                        {npc.stock > 0 ? npc.stock : '∞'}
                                    </td>
                                    <td className="px-2 py-1.5 text-right font-mono text-gray-300">
                                        {detail.buyCount || 1}
                                    </td>
                                    <td className="py-1.5 pl-2 text-right">
                                        {npc.cost > 0 ? (
                                            <Money copper={npc.cost} className="justify-end" />
                                        ) : (
                                            <span className="text-gray-600">—</span>
                                        )}
                                    </td>
                                </tr>
                            )
                        })}
                    </tbody>
                </table>
            ),
        },
        detail.rewardFrom?.length && {
            id: 'rewardFrom',
            label: 'Reward From',
            count: detail.rewardFrom.length,
            content: (
                <RelTable
                    columns={[
                        { label: 'Quest' },
                        { label: 'Level', align: 'right' },
                        { label: 'Type', align: 'right' },
                    ]}
                >
                    {detail.rewardFrom.map((q) => (
                        <RelRow key={q.entry} onClick={() => onNavigate('quest', q.entry)}>
                            <td className="py-1.5 pr-2 font-bold text-wow-gold">{q.title}</td>
                            <td className="px-2 py-1.5 text-right font-mono text-gray-300">
                                {q.level}
                            </td>
                            <td className="py-1.5 pl-2 text-right text-xs text-gray-400">
                                {q.isChoice ? 'Choice' : 'Reward'}
                            </td>
                        </RelRow>
                    ))}
                </RelTable>
            ),
        },
        detail.startsQuest && {
            id: 'startsQuest',
            label: 'Starts Quest',
            count: 1,
            content: (
                <RelTable columns={[{ label: 'Quest' }, { label: 'Level', align: 'right' }]}>
                    <RelRow onClick={() => onNavigate('quest', detail.startsQuest.entry)}>
                        <td className="py-1.5 pr-2 font-bold text-wow-gold">
                            {detail.startsQuest.title}
                        </td>
                        <td className="py-1.5 pl-2 text-right font-mono text-gray-300">
                            {detail.startsQuest.level}
                        </td>
                    </RelRow>
                </RelTable>
            ),
        },
        detail.objectiveOf?.length && {
            id: 'objectiveOf',
            label: 'Objective Of',
            count: detail.objectiveOf.length,
            content: (
                <RelTable columns={[{ label: 'Quest' }, { label: 'Level', align: 'right' }]}>
                    {detail.objectiveOf.map((q) => (
                        <RelRow key={q.entry} onClick={() => onNavigate('quest', q.entry)}>
                            <td className="py-1.5 pr-2 font-bold text-wow-gold">{q.title}</td>
                            <td className="py-1.5 pl-2 text-right font-mono text-gray-300">
                                {q.level}
                            </td>
                        </RelRow>
                    ))}
                </RelTable>
            ),
        },
        detail.contains?.length && {
            id: 'contains',
            label: 'Contains',
            count: detail.contains.length,
            content: (
                <RelTable
                    columns={[
                        { label: 'Name' },
                        { label: 'Qty', align: 'right' },
                        { label: 'Chance', align: 'right' },
                    ]}
                >
                    {detail.contains.map((item) => (
                        <RelRow key={item.entry} onClick={() => onNavigate('item', item.entry)}>
                            <td className="py-1.5 pr-2">
                                <div className="flex min-w-0 items-center gap-2">
                                    <IconImg
                                        name={item.iconPath}
                                        className="h-7 w-7 shrink-0 rounded border border-black/40 object-cover"
                                    />
                                    <span
                                        className="truncate font-bold hover:underline"
                                        style={{ color: getQualityColor(item.quality) }}
                                    >
                                        {item.name}
                                    </span>
                                </div>
                            </td>
                            <td className="px-2 py-1.5 text-right font-mono text-gray-300">
                                {item.maxCount > item.minCount
                                    ? `${item.minCount}-${item.maxCount}`
                                    : item.minCount || 1}
                            </td>
                            <td className="py-1.5 pl-2 text-right font-mono text-wow-gold">
                                {item.chance > 0 ? `${item.chance.toFixed(1)}%` : '—'}
                            </td>
                        </RelRow>
                    ))}
                </RelTable>
            ),
        },
    ].filter(Boolean)

    const currentTab = relationTabs.find((t) => t.id === activeRelTab) || relationTabs[0]

    return (
        <DetailPageLayout onBack={onBack}>
            <DetailHeader
                icon={
                    <ItemIconHeader
                        iconPath={detail.iconPath}
                        imgError={imgError}
                        fixing={fixing}
                        handleFixIcon={handleFixIcon}
                        qualityColor={qualityColor}
                        onNavigate={onNavigate}
                    />
                }
                iconBorderColor={qualityColor}
                title={detail.name}
                titleColor={qualityColor}
                subtitle={`Item Level ${detail.itemLevel}`}
                action={
                    <div className="flex gap-2">
                        <a
                            href={`${DATABASE_BASE_URL}/?item=${entry}`}
                            target="_blank"
                            rel="noreferrer"
                            className="rounded bg-purple-700 px-3 py-1.5 text-xs font-bold uppercase text-white transition-colors hover:bg-purple-600"
                            title="View on Turtle WoW Database"
                        >
                            🔗 OctoHead
                        </a>
                        <button
                            onClick={() => {
                                // Quality color codes (WoW format)
                                const qualityColors = {
                                    0: 'ff9d9d9d', // Poor (grey)
                                    1: 'ffffffff', // Common (white)
                                    2: 'ff1eff00', // Uncommon (green)
                                    3: 'ff0070dd', // Rare (blue)
                                    4: 'ffa335ee', // Epic (purple)
                                    5: 'ffff8000', // Legendary (orange)
                                    6: 'ffe6cc80', // Artifact (gold)
                                }
                                const colorCode = qualityColors[detail.quality] || 'ffffffff'
                                // Format: |cCOLOR|Hitem:ID:0:0:0:0:0:0:0:0|h[NAME]|h|r
                                // \124 is the escape for | in Lua
                                // Escape quotes in name for Lua string
                                const escapedName = detail.name.replace(/"/g, '\\"')
                                const itemLink = `/script DEFAULT_CHAT_FRAME:AddMessage("\\124c${colorCode}\\124Hitem:${detail.entry}:0:0:0:0:0:0:0:0\\124h[${escapedName}]\\124h\\124r");`
                                navigator.clipboard
                                    .writeText(itemLink)
                                    .then(() =>
                                        alert(
                                            'In-game link copied to clipboard!\n\nPaste this in WoW chat to see the item link.',
                                        ),
                                    )
                                    .catch((err) => alert('Failed to copy: ' + err))
                            }}
                            className="rounded bg-green-700 px-3 py-1.5 text-xs font-bold uppercase text-white transition-colors hover:bg-green-600"
                            title="Copy in-game item link command to clipboard"
                        >
                            🔗 In-Game Link
                        </button>
                        <button
                            onClick={handleFavoriteToggle}
                            className={`flex items-center gap-1 rounded px-3 py-1.5 text-xs font-bold uppercase transition-colors ${
                                isFavorite
                                    ? 'bg-red-600 text-white hover:bg-red-500'
                                    : 'bg-gray-700 text-gray-300 hover:bg-gray-600'
                            }`}
                            title={isFavorite ? 'Remove from Favorites' : 'Add to Favorites'}
                        >
                            {isFavorite ? '❤️ Favorited' : '🤍 Favorite'}
                        </button>
                        <button
                            onClick={handleSync}
                            disabled={syncing}
                            className={`rounded px-3 py-1.5 text-xs font-bold uppercase transition-colors ${
                                syncing
                                    ? 'cursor-not-allowed bg-gray-600 text-gray-400'
                                    : 'bg-blue-600 text-white hover:bg-blue-500'
                            }`}
                            title="Refresh item data from Turtle WoW Database"
                        >
                            {syncing ? '⏳ Syncing...' : '🔄 Sync'}
                        </button>
                    </div>
                }
            />

            <div className="flex flex-wrap gap-10">
                {/* Tooltip Block */}
                {renderTooltipBlock()}

                {/* Mount model (for mount items: collection_mount -> mount spell
                    -> creature display id), rendered like NPC model pages. */}
                {detail.mountDisplayId > 0 && (mountModel.loading || mountModel.src) && (
                    <div className="w-56 flex-shrink-0">
                        <div className="mb-2 text-xs font-semibold uppercase tracking-wide text-gray-500">
                            Mount
                        </div>
                        {mountModel.loading ? (
                            <div className="flex aspect-[3/4] animate-pulse items-center justify-center rounded border border-white/10 bg-black/40 text-xs text-gray-500">
                                Loading…
                            </div>
                        ) : (
                            <div className="overflow-hidden rounded border border-white/20 bg-black shadow-lg">
                                <img
                                    src={mountModel.src}
                                    alt={detail.name}
                                    className="h-auto w-full object-cover"
                                />
                            </div>
                        )}
                    </div>
                )}

                {/* Relations — tabbed (Contained In / Dropped By can get long) */}
                {relationTabs.length > 0 && currentTab && (
                    <div className="min-w-[300px] flex-1">
                        <div className="mb-4 flex flex-wrap gap-1 border-b border-white/20">
                            {relationTabs.map((tab) => (
                                <button
                                    key={tab.id}
                                    onClick={() => selectRelTab(tab.id)}
                                    className={`relative top-[1px] px-3 py-2 text-sm font-bold transition-all ${
                                        currentTab.id === tab.id
                                            ? 'tab-btn-active border-b-2 border-wow-gold text-white'
                                            : 'tab-btn-inactive text-gray-400 hover:text-gray-200'
                                    }`}
                                >
                                    {tab.label}{' '}
                                    <span className="ml-0.5 text-xs opacity-60">{tab.count}</span>
                                </button>
                            ))}
                        </div>
                        <div className="animate-fade-in">{currentTab.content}</div>
                    </div>
                )}
            </div>
        </DetailPageLayout>
    )
}

export default ItemDetailView
