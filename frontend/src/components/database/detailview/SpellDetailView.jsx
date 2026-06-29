import React, { useState } from 'react'
import { queryClient } from '../../../queryClient'
import { SyncSingleSpell } from '../../../services/api'
import { useSpellDetail } from '../../../hooks/queries/spells'
import { DetailPageLayout, DetailHeader, DetailSection, DetailLoading, DetailError } from '../../ui'
import { useIcon } from '../../../services/useImage'
import {
    getQualityColor,
    getSchoolName,
    getSchoolColor,
    getDispelColor,
    QUESTION_MARK_ICON,
} from '../../../utils/wow'
import { DATABASE_BASE_URL } from '../../../utils/constants'

// Helper component for Spell Icon
const SpellIcon = ({ iconName }) => {
    const icon = useIcon(iconName)

    if (icon.loading) {
        return <div className="h-full w-full animate-pulse bg-white/5" />
    }

    return (
        <img
            src={icon.src || QUESTION_MARK_ICON}
            className="h-full w-full object-cover"
            alt=""
            onError={(e) => {
                e.target.style.display = 'none'
            }}
        />
    )
}

// Helper component for Item Icon in Used By list
const ItemIcon = ({ iconName }) => {
    const icon = useIcon(iconName)

    if (icon.loading) {
        return <div className="h-full w-full animate-pulse bg-white/5" />
    }

    return (
        <img src={icon.src || QUESTION_MARK_ICON} className="h-full w-full object-cover" alt="" />
    )
}

// One label/value row in the Properties grid; renders nothing when value is empty.
const PropRow = ({ label, value, color, capitalize }) =>
    value === 0 || value ? (
        <>
            <span className="text-gray-500">{label}:</span>
            <span
                className={`text-right text-gray-300 ${capitalize ? 'capitalize' : ''}`}
                style={color ? { color, fontWeight: 500 } : undefined}
            >
                {value}
            </span>
        </>
    ) : null

const SpellDetailView = ({ entry, onBack, onNavigate, tooltipHook }) => {
    const [syncing, setSyncing] = useState(false)

    const { data: detail, isLoading: loading, isError, error } = useSpellDetail(entry)

    const handleSync = async () => {
        setSyncing(true)
        try {
            const result = await SyncSingleSpell(parseInt(entry))
            if (result?.success) {
                // Drop the cache so lists/search/tooltips referencing this spell refetch.
                await queryClient.invalidateQueries()
            } else {
                console.error('Sync failed:', result?.error)
            }
        } catch (err) {
            console.error('Sync error:', err)
        }
        setSyncing(false)
    }

    if (loading) return <DetailLoading />
    if (isError) return <DetailError message={String(error)} onBack={onBack} />
    if (!detail) return <DetailError message="Spell not found" onBack={onBack} />

    // Localized school name from the client (spell_schools), English fallback.
    // Color is a client UI constant keyed by school index (Physical has none).
    const schoolName = detail.schoolName || getSchoolName(detail.school)
    const schoolColor = getSchoolColor(detail.school)

    // Format power type
    const powerTypes = {
        0: 'Mana',
        1: 'Rage',
        2: 'Focus',
        3: 'Energy',
        4: 'Happiness',
    }
    const powerType = powerTypes[detail.powerType] || 'Power'

    return (
        <DetailPageLayout onBack={onBack}>
            <DetailHeader
                title={`${detail.name} [${detail.entry}]`}
                icon={<SpellIcon iconName={detail.icon} />}
                titleColor="#FFD100"
                subtitle={
                    <>
                        {[detail.nameSubtext, `Level ${detail.spellLevel}`]
                            .filter(Boolean)
                            .join(' • ')}
                        {' • '}
                        <span style={{ color: schoolColor }}>{schoolName}</span>
                    </>
                }
                action={
                    <div className="flex gap-2">
                        <button
                            onClick={handleSync}
                            disabled={syncing}
                            className={`rounded px-3 py-1.5 text-xs font-bold uppercase transition-colors ${
                                syncing
                                    ? 'cursor-not-allowed bg-gray-600 text-gray-400'
                                    : 'bg-green-700 text-white hover:bg-green-600'
                            }`}
                            title="Resolve this spell's description from local DBC data"
                        >
                            {syncing ? '⏳ Resolving...' : '🔄 Resolve'}
                        </button>
                        <a
                            href={`${DATABASE_BASE_URL}/?spell=${entry}`}
                            target="_blank"
                            rel="noreferrer"
                            className="rounded bg-purple-700 px-3 py-1.5 text-xs font-bold uppercase text-white transition-colors hover:bg-purple-600"
                            title="View on Turtle WoW Database"
                        >
                            🔗 OctoHead
                        </a>
                    </div>
                }
            />

            <div className="grid grid-cols-1 gap-10 lg:grid-cols-[2fr_1fr]">
                {/* Main Content */}
                <div className="space-y-8">
                    <DetailSection title="Description">
                        <p className="whitespace-pre-wrap leading-relaxed text-gray-300">
                            {detail.description || 'No description available.'}
                        </p>
                    </DetailSection>

                    {detail.toolTip && detail.toolTip !== detail.description && (
                        <DetailSection title="Tooltip">
                            <p className="whitespace-pre-wrap leading-relaxed text-gray-300">
                                {detail.toolTip}
                            </p>
                        </DetailSection>
                    )}

                    {/* Spell effects */}
                    {detail.effects?.length > 0 && (
                        <DetailSection title="Effects">
                            <div className="space-y-2">
                                {detail.effects.map((e) => (
                                    <div
                                        key={e.index}
                                        className="rounded border border-border-dark/50 bg-white/[0.02] p-2.5"
                                    >
                                        <div className="text-sm font-semibold text-gray-200">
                                            Effect #{e.index}: {e.effect}
                                            {e.auraName ? `: ${e.auraName}` : ''}
                                        </div>
                                        {e.value && (
                                            <div className="mt-0.5 text-xs text-gray-400">
                                                Value: {e.value}
                                            </div>
                                        )}
                                        {(e.radius || e.mechanic || e.triggerSpell > 0) && (
                                            <div className="mt-0.5 flex flex-wrap gap-x-4 gap-y-0.5 text-[11px] text-gray-500">
                                                {e.radius && <span>Radius: {e.radius}</span>}
                                                {e.mechanic && (
                                                    <span className="capitalize">
                                                        Mechanic: {e.mechanic}
                                                    </span>
                                                )}
                                                {e.triggerSpell > 0 && (
                                                    <span
                                                        className="cursor-pointer text-wow-rare/80 hover:text-wow-rare"
                                                        onClick={() =>
                                                            onNavigate?.('spell', e.triggerSpell)
                                                        }
                                                        {...(tooltipHook?.getSpellHandlers?.(
                                                            e.triggerSpell,
                                                        ) || {})}
                                                    >
                                                        Triggers spell #{e.triggerSpell}
                                                    </span>
                                                )}
                                            </div>
                                        )}
                                        {e.createdItem && (
                                            <div
                                                className="mt-1.5 flex cursor-pointer items-center gap-2 rounded p-1 transition-colors hover:bg-white/5"
                                                onClick={() =>
                                                    onNavigate?.('item', e.createdItem.entry)
                                                }
                                                onMouseEnter={() =>
                                                    tooltipHook?.onHover?.(e.createdItem.entry)
                                                }
                                                onMouseLeave={() => tooltipHook?.onLeave?.()}
                                            >
                                                <span className="text-[11px] text-gray-500">
                                                    Creates
                                                </span>
                                                <div className="h-6 w-6 flex-shrink-0 overflow-hidden rounded bg-black">
                                                    <ItemIcon iconName={e.createdItem.iconPath} />
                                                </div>
                                                <span
                                                    className="truncate text-sm font-medium"
                                                    style={{
                                                        color: getQualityColor(e.createdItem.quality),
                                                    }}
                                                >
                                                    {e.createdItem.name}
                                                </span>
                                            </div>
                                        )}
                                    </div>
                                ))}
                            </div>
                        </DetailSection>
                    )}

                    {/* Used By Items */}
                    {detail.usedByItems && detail.usedByItems.length > 0 && (
                        <DetailSection title={`Used By (${detail.usedByItems.length})`}>
                            <div className="space-y-1">
                                {detail.usedByItems.map((item) => (
                                    <div
                                        key={item.entry}
                                        className="flex cursor-pointer items-center gap-2 rounded p-1.5 transition-colors hover:bg-white/5"
                                        onClick={() => onNavigate?.('item', item.entry)}
                                        onMouseEnter={() => tooltipHook?.onHover?.(item.entry)}
                                        onMouseLeave={() => tooltipHook?.onLeave?.()}
                                    >
                                        <div className="h-6 w-6 flex-shrink-0 overflow-hidden rounded bg-black">
                                            <ItemIcon iconName={item.iconPath} />
                                        </div>
                                        <span
                                            className="truncate text-sm font-medium"
                                            style={{ color: getQualityColor(item.quality) }}
                                        >
                                            {item.name}
                                        </span>
                                        <span className="ml-auto text-xs text-gray-500">
                                            {item.triggerType === 0
                                                ? 'Use'
                                                : item.triggerType === 1
                                                  ? 'Equip'
                                                  : item.triggerType === 2
                                                    ? 'Chance on Hit'
                                                    : item.triggerType === 5
                                                      ? 'Learn'
                                                      : ''}
                                        </span>
                                    </div>
                                ))}
                            </div>
                        </DetailSection>
                    )}

                    {/* Taught By — trainers (scraped) + recipe items + quests */}
                    {((detail.taughtByNpcs && detail.taughtByNpcs.length > 0) ||
                        (detail.taughtByItems && detail.taughtByItems.length > 0) ||
                        (detail.taughtByQuests && detail.taughtByQuests.length > 0)) && (
                        <DetailSection title="Taught By">
                            <div className="space-y-1">
                                {detail.taughtByItems?.map((item) => (
                                    <div
                                        key={`item-${item.entry}`}
                                        className="flex cursor-pointer items-center gap-2 rounded p-1.5 transition-colors hover:bg-white/5"
                                        onClick={() => onNavigate?.('item', item.entry)}
                                    >
                                        <div className="h-6 w-6 flex-shrink-0 overflow-hidden rounded bg-black">
                                            <ItemIcon iconName={item.iconPath} />
                                        </div>
                                        <span
                                            className="truncate text-sm font-medium"
                                            style={{ color: getQualityColor(item.quality) }}
                                        >
                                            {item.name}
                                        </span>
                                        <span className="ml-auto text-xs text-gray-500">Item</span>
                                    </div>
                                ))}
                                {detail.taughtByNpcs?.map((npc) => (
                                    <div
                                        key={`npc-${npc.entry}`}
                                        className="flex cursor-pointer items-center gap-2 rounded p-1.5 transition-colors hover:bg-white/5"
                                        onClick={() => onNavigate?.('npc', npc.entry)}
                                    >
                                        <span className="flex h-6 w-6 flex-shrink-0 items-center justify-center rounded border border-wow-gold/40 bg-wow-gold/20 text-[8px] font-bold text-wow-gold">
                                            NPC
                                        </span>
                                        <span className="truncate text-sm font-medium text-wow-gold">
                                            {npc.name || `NPC #${npc.entry}`}
                                        </span>
                                        {npc.levelMax > 0 && (
                                            <span className="ml-auto text-xs text-gray-500">
                                                Lvl {npc.levelMin}
                                                {npc.levelMax > npc.levelMin
                                                    ? `-${npc.levelMax}`
                                                    : ''}
                                            </span>
                                        )}
                                    </div>
                                ))}
                                {detail.taughtByQuests?.map((q) => (
                                    <div
                                        key={`quest-${q.entry}`}
                                        className="flex cursor-pointer items-center gap-2 rounded p-1.5 transition-colors hover:bg-white/5"
                                        onClick={() => onNavigate?.('quest', q.entry)}
                                    >
                                        {q.side === 'Horde' || q.side === 'Alliance' ? (
                                            <img
                                                src={
                                                    q.side === 'Horde'
                                                        ? '/Horde_15.webp'
                                                        : '/Alliance_15.webp'
                                                }
                                                className="h-6 w-6 flex-shrink-0 object-contain"
                                                alt={q.side}
                                                title={q.side}
                                            />
                                        ) : (
                                            <span className="flex h-6 w-6 flex-shrink-0 items-center justify-center rounded border border-emerald-400/40 bg-emerald-400/20 text-[8px] font-bold text-emerald-300">
                                                Q
                                            </span>
                                        )}
                                        <span className="truncate text-sm font-medium text-wow-gold">
                                            {q.title || `Quest #${q.entry}`}
                                        </span>
                                        {q.level > 0 && (
                                            <span className="ml-auto text-xs text-gray-500">
                                                Lvl {q.level}
                                            </span>
                                        )}
                                    </div>
                                ))}
                            </div>
                        </DetailSection>
                    )}
                </div>

                {/* Side Panel */}
                <div className="space-y-6">
                    <DetailSection title="Properties">
                        <div className="grid grid-cols-2 gap-y-2 text-sm">
                            <PropRow
                                label="Cost"
                                value={
                                    detail.manaCost > 0 ? `${detail.manaCost} ${powerType}` : 'None'
                                }
                            />
                            <PropRow label="Cast Time" value={detail.castTime} />
                            <PropRow label="Cooldown" value={detail.cooldown} />
                            <PropRow label="GCD" value={detail.gcd} />
                            <PropRow label="Range" value={detail.range} />
                            <PropRow label="Duration" value={detail.duration} />
                            <PropRow label="School" value={schoolName} color={schoolColor} />
                            <PropRow label="Mechanic" value={detail.mechanicName} capitalize />
                            <PropRow
                                label="Dispel type"
                                value={detail.dispelType}
                                color={getDispelColor(detail.dispelType)}
                            />
                            {/* Real proc rate (PPM / %) from the world DB proc tables; "" when none. */}
                            <PropRow label="Proc" value={detail.proc} />
                            <PropRow
                                label="Max targets"
                                value={
                                    detail.maxAffectedTargets > 0 ? detail.maxAffectedTargets : ''
                                }
                            />
                            <PropRow label="Level" value={detail.spellLevel} />
                        </div>
                    </DetailSection>

                    {detail.flags?.length > 0 && (
                        <DetailSection title="Flags">
                            <ul className="list-inside list-disc space-y-1 text-xs text-amber-300/80">
                                {detail.flags.map((f, i) => (
                                    <li key={i}>{f}</li>
                                ))}
                            </ul>
                        </DetailSection>
                    )}
                </div>
            </div>
        </DetailPageLayout>
    )
}

export default SpellDetailView
