import React from 'react'
import { queryClient } from '../../../queryClient'
import { useQuestDetail } from '../../../hooks/queries/quests'
import { DATABASE_BASE_URL } from '../../../utils/constants'
import { SyncQuestData } from '../../../services/api'
import { useIcon } from '../../../services/useImage'
import { getQualityColor, QUESTION_MARK_ICON } from '../../../utils/wow'
import {
    DetailPageLayout,
    DetailHeader,
    DetailSection,
    DetailSidePanel,
    LootGrid,
    DetailLoading,
    DetailError,
    Money,
} from '../../ui'
import { LootItem } from '../../ui'

// Small icon resolved through the cached icon service (data/icons + fallback).
const ReqIcon = ({ name }) => {
    const icon = useIcon(name)
    return (
        <img
            src={icon.src || QUESTION_MARK_ICON}
            alt=""
            className="h-7 w-7 shrink-0 rounded border border-black/40 object-cover"
        />
    )
}

const QuestDetailView = ({ entry, onBack, onNavigate, tooltipHook }) => {
    const { data: detail, isLoading: loading, isError, error } = useQuestDetail(entry)

    const handleSync = () => {
        // Drop the cache so the quest list/search behind this overlay refetch too.
        SyncQuestData(entry).then(() => queryClient.invalidateQueries())
    }

    const renderRewardItem = (item) => {
        const handlers = tooltipHook?.getItemHandlers?.(item.entry) || {}
        return (
            <LootItem
                key={item.entry}
                item={item}
                onClick={() => onNavigate('item', item.entry)}
                {...handlers}
            />
        )
    }

    const getQuestType = (type) => {
        const types = { 1: 'Group', 41: 'PVP', 62: 'Raid', 81: 'Dungeon' }
        return types[type] || 'Normal'
    }

    if (loading) return <DetailLoading />
    if (isError) return <DetailError message={String(error)} onBack={onBack} />
    if (!detail) return <DetailError message="Quest data is empty or invalid." onBack={onBack} />
    if (!detail) return <DetailError message="Quest not found" onBack={onBack} />

    return (
        <DetailPageLayout onBack={onBack}>
            <DetailHeader
                title={`${detail.title} [${detail.entry}]`}
                titleColor="#FFD100"
                subtitle={
                    <div className="flex items-center">
                        <span>
                            Level {detail.questLevel} (Min {detail.minLevel}) -{' '}
                            {getQuestType(detail.type)}
                        </span>
                        {detail.side && detail.side !== 'Both' && (
                            <span
                                className={`ml-3 inline-flex items-center gap-1.5 rounded border border-white/5 bg-black/20 px-2 py-0.5 ${detail.side === 'Horde' ? 'text-red-400' : 'text-blue-400'}`}
                            >
                                <img
                                    src={
                                        detail.side === 'Horde'
                                            ? '/Horde_15.webp'
                                            : '/Alliance_15.webp'
                                    }
                                    className="h-4 w-4 object-contain"
                                    alt={detail.side}
                                />
                                <span className="text-[10px] font-bold uppercase tracking-wider">
                                    {detail.side}
                                </span>
                            </span>
                        )}
                    </div>
                }
                action={
                    <div className="flex gap-2">
                        <button
                            onClick={handleSync}
                            className="flex items-center gap-1 rounded border border-blue-700 bg-blue-600 px-3 py-1.5 text-xs font-bold uppercase text-white transition-colors hover:bg-blue-500"
                            title="Re-download data from external sources"
                        >
                            <span>↻</span> Sync
                        </button>
                        <a
                            href={`${DATABASE_BASE_URL}/?quest=${entry}`}
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
                            {detail.details || 'No description available.'}
                        </p>
                    </DetailSection>

                    <DetailSection title="Objectives">
                        <p className="whitespace-pre-wrap leading-relaxed text-gray-300">
                            {detail.objectives || 'No objectives listed.'}
                        </p>

                        {(detail.requiredItems?.length > 0 ||
                            detail.requiredObjectives?.length > 0) && (
                            <div className="mt-4 space-y-1">
                                <h4 className="mb-2 text-xs font-semibold uppercase tracking-wider text-gray-400">
                                    Required
                                </h4>
                                {detail.requiredItems?.map((it) => (
                                    <div
                                        key={`i${it.entry}`}
                                        onClick={() => onNavigate('item', it.entry)}
                                        className="flex cursor-pointer items-center gap-2 rounded border border-white/5 bg-white/[0.02] p-2 transition-colors hover:bg-white/5"
                                        {...(tooltipHook?.getItemHandlers?.(it.entry) || {})}
                                    >
                                        <ReqIcon name={it.iconPath} />
                                        <span
                                            className="font-semibold"
                                            style={{ color: getQualityColor(it.quality) }}
                                        >
                                            {it.name || `Item #${it.entry}`}
                                        </span>
                                        {it.count > 1 && (
                                            <span className="ml-auto font-mono text-xs text-gray-400">
                                                ×{it.count}
                                            </span>
                                        )}
                                    </div>
                                ))}
                                {detail.requiredObjectives?.map((o) => (
                                    <div
                                        key={`o${o.kind}${o.entry}`}
                                        onClick={() => onNavigate(o.kind, o.entry)}
                                        className="flex cursor-pointer items-center gap-2 rounded border border-white/5 bg-white/[0.02] p-2 transition-colors hover:bg-white/5"
                                    >
                                        <span className="flex h-7 w-7 shrink-0 items-center justify-center rounded border border-wow-rare/30 bg-wow-rare/10 text-[9px] font-bold text-wow-rare">
                                            {o.kind === 'object' ? 'OBJ' : 'NPC'}
                                        </span>
                                        <span className="text-gray-200">
                                            {o.name || `#${o.entry}`}
                                        </span>
                                        {o.count > 1 && (
                                            <span className="ml-auto font-mono text-xs text-gray-400">
                                                ×{o.count}
                                            </span>
                                        )}
                                    </div>
                                ))}
                            </div>
                        )}
                    </DetailSection>

                    {/* Rewards */}
                    <DetailSection title="Rewards">
                        <div className="space-y-4">
                            {detail.rewardMoney > 0 && (
                                <div className="flex w-fit items-center gap-2 rounded border border-wow-gold/10 bg-wow-gold/5 px-3 py-1.5">
                                    <span className="text-xs font-bold uppercase text-gray-500">
                                        Money:
                                    </span>
                                    <Money copper={detail.rewardMoney} />
                                </div>
                            )}
                            {detail.rewardXp > 0 && (
                                <div className="flex w-fit items-center gap-2 rounded border border-wow-rare/10 bg-wow-rare/5 px-3 py-1.5 text-wow-rare">
                                    <span className="text-xs font-bold uppercase text-gray-500">
                                        Experience:
                                    </span>
                                    <span>{detail.rewardXp} XP</span>
                                </div>
                            )}
                            {detail.reputation?.map((rep, idx) => (
                                <div
                                    key={idx}
                                    className={`flex w-fit items-center gap-2 rounded border px-3 py-1.5 ${rep.value >= 0 ? 'border-green-400/10 bg-green-400/5 text-green-400' : 'border-red-400/10 bg-red-400/5 text-red-400'}`}
                                >
                                    <span className="text-xs font-bold uppercase text-gray-500">
                                        Reputation:
                                    </span>
                                    <span>
                                        {rep.value >= 0 ? '+' : ''}
                                        {rep.value}{' '}
                                        {rep.factionId ? (
                                            <span
                                                className="cursor-pointer hover:underline"
                                                onClick={() => onNavigate?.('faction', rep.factionId)}
                                            >
                                                {rep.name}
                                            </span>
                                        ) : (
                                            rep.name
                                        )}
                                    </span>
                                </div>
                            ))}
                            {detail.rewardSpellInfo && (
                                <div
                                    className="flex w-fit cursor-pointer items-center gap-2 rounded border border-wow-rare/10 bg-wow-rare/5 px-3 py-1.5 transition-colors hover:bg-wow-rare/10"
                                    onClick={() =>
                                        onNavigate?.('spell', detail.rewardSpellInfo.spellId)
                                    }
                                >
                                    <span className="text-xs font-bold uppercase text-gray-500">
                                        Learn:
                                    </span>
                                    <ReqIcon name={detail.rewardSpellInfo.iconName} />
                                    <span className="font-medium text-wow-rare">
                                        {detail.rewardSpellInfo.name ||
                                            `Spell #${detail.rewardSpellInfo.spellId}`}
                                    </span>
                                </div>
                            )}
                        </div>

                        {detail.rewardItems?.length > 0 && (
                            <div className="mt-6">
                                <h4 className="mb-3 text-sm font-semibold uppercase tracking-wider text-gray-400">
                                    You will receive:
                                </h4>
                                <LootGrid>
                                    {detail.rewardItems.map((i) => renderRewardItem(i))}
                                </LootGrid>
                            </div>
                        )}

                        {detail.choiceItems?.length > 0 && (
                            <div className="mt-6">
                                <h4 className="mb-3 text-sm font-semibold uppercase tracking-wider text-gray-400">
                                    Choose one of:
                                </h4>
                                <LootGrid>
                                    {detail.choiceItems.map((i) => renderRewardItem(i))}
                                </LootGrid>
                            </div>
                        )}
                    </DetailSection>
                </div>

                {/* Side Panel */}
                <DetailSidePanel className="space-y-6">
                    {/* Quest Chain */}
                    <div>
                        <h3 className="mb-3 border-b border-wow-gold/20 pb-1 font-bold text-wow-gold">
                            Quest Chain
                        </h3>
                        {detail.series?.length > 0 ? (
                            <div className="space-y-1">
                                {detail.series.map((s, index) => {
                                    const isChild = s.depth > 0
                                    const indent = isChild ? s.depth * 16 : 0

                                    return (
                                        <div
                                            key={s.entry}
                                            className="flex gap-2 text-[13px]"
                                            style={{ paddingLeft: `${indent}px` }}
                                        >
                                            <span className="w-6 flex-shrink-0 text-right font-mono text-gray-600">
                                                {isChild ? '└─' : `${index + 1}.`}
                                            </span>
                                            {s.entry === detail.entry ? (
                                                <b className="rounded bg-white/5 px-1 text-white">
                                                    {s.title}
                                                </b>
                                            ) : (
                                                <a
                                                    className="cursor-pointer text-wow-rare hover:underline"
                                                    onClick={() => onNavigate('quest', s.entry)}
                                                >
                                                    {s.title}
                                                </a>
                                            )}
                                        </div>
                                    )
                                })}
                            </div>
                        ) : (
                            <div className="text-sm italic text-gray-600">Standalone quest.</div>
                        )}
                    </div>

                    {/* Prerequisites */}
                    {detail.prevQuests?.length > 0 && (
                        <div>
                            <h3 className="mb-3 border-b border-wow-gold/20 pb-1 font-bold text-wow-gold">
                                Prerequisites
                            </h3>
                            <div className="space-y-2">
                                {detail.prevQuests.map((q) => (
                                    <div key={q.entry} className="text-[13px]">
                                        <a
                                            className="cursor-pointer text-wow-rare hover:underline"
                                            onClick={() => onNavigate('quest', q.entry)}
                                        >
                                            • {q.title}
                                        </a>
                                    </div>
                                ))}
                            </div>
                        </div>
                    )}

                    {/* Requirements */}
                    <div>
                        <h3 className="mb-3 border-b border-wow-gold/20 pb-1 font-bold text-wow-gold">
                            Requirements
                        </h3>
                        <div className="space-y-2 text-[13px] text-gray-300">
                            {(detail.raceNames || detail.requiredRaces > 0) && (
                                <div className="flex justify-between border-b border-white/5 pb-1">
                                    <span>Races:</span>
                                    <span className="max-w-[200px] pl-4 text-right font-mono text-xs leading-tight text-white">
                                        {detail.raceNames || detail.requiredRaces}
                                    </span>
                                </div>
                            )}
                            {(detail.classes?.length > 0 || detail.requiredClasses > 0) && (
                                <div className="flex justify-between border-b border-white/5 pb-1">
                                    <span>Classes:</span>
                                    <span className="max-w-[200px] pl-4 text-right font-mono text-xs leading-tight text-white">
                                        {detail.classes?.length > 0
                                            ? detail.classes.map((c, i) => (
                                                  <React.Fragment key={c.name}>
                                                      {i > 0 && ', '}
                                                      <span style={c.color ? { color: c.color } : undefined}>
                                                          {c.name}
                                                      </span>
                                                  </React.Fragment>
                                              ))
                                            : detail.requiredClasses}
                                    </span>
                                </div>
                            )}
                            {detail.srcItemId > 0 && (
                                <div className="flex items-center gap-2">
                                    <span>Starts from:</span>
                                    <a
                                        className="cursor-pointer rounded border border-wow-gold/20 bg-wow-gold/5 px-2 py-0.5 text-wow-gold hover:underline"
                                        onClick={() => onNavigate('item', detail.srcItemId)}
                                    >
                                        [Item {detail.srcItemId}]
                                    </a>
                                </div>
                            )}
                            {!detail.requiredRaces &&
                                !detail.requiredClasses &&
                                !detail.srcItemId && (
                                    <div className="italic text-gray-600">None</div>
                                )}
                        </div>
                    </div>

                    {/* Relations */}
                    <div>
                        <h3 className="mb-3 border-b border-wow-gold/20 pb-1 font-bold text-wow-gold">
                            Relations
                        </h3>
                        <div className="space-y-4">
                            {detail.starters?.length > 0 && (
                                <div>
                                    <h4 className="mb-2 text-xs font-bold uppercase tracking-tighter text-gray-500">
                                        Starts with:
                                    </h4>
                                    <div className="space-y-1">
                                        {detail.starters.map((s) => (
                                            <div
                                                key={s.entry}
                                                onClick={() =>
                                                    s.type === 'npc' && onNavigate('npc', s.entry)
                                                }
                                                className={`rounded border px-2 py-1 text-xs ${
                                                    s.type === 'npc'
                                                        ? 'cursor-pointer border-wow-rare/20 bg-wow-rare/5 text-wow-rare hover:bg-wow-rare/10'
                                                        : 'border-white/10 bg-white/5 text-gray-400'
                                                }`}
                                            >
                                                {s.name}{' '}
                                                <span className="opacity-50">({s.type})</span>
                                            </div>
                                        ))}
                                    </div>
                                </div>
                            )}
                            {detail.enders?.length > 0 && (
                                <div>
                                    <h4 className="mb-2 text-xs font-bold uppercase tracking-tighter text-gray-500">
                                        Ends with:
                                    </h4>
                                    <div className="space-y-1">
                                        {detail.enders.map((s) => (
                                            <div
                                                key={s.entry}
                                                onClick={() =>
                                                    s.type === 'npc' && onNavigate('npc', s.entry)
                                                }
                                                className={`rounded border px-2 py-1 text-xs ${
                                                    s.type === 'npc'
                                                        ? 'cursor-pointer border-wow-rare/20 bg-wow-rare/5 text-wow-rare hover:bg-wow-rare/10'
                                                        : 'border-white/10 bg-white/5 text-gray-400'
                                                }`}
                                            >
                                                {s.name}{' '}
                                                <span className="opacity-50">({s.type})</span>
                                            </div>
                                        ))}
                                    </div>
                                </div>
                            )}
                        </div>
                    </div>
                </DetailSidePanel>
            </div>
        </DetailPageLayout>
    )
}

export default QuestDetailView
