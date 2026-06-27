import React from 'react'
import { queryClient } from '../../../queryClient'
import { useQuestDetail } from '../../../hooks/queries/quests'
import { DATABASE_BASE_URL } from '../../../utils/constants'
import { SyncQuestData } from '../../../services/api'
import {
    DetailPageLayout,
    DetailHeader,
    DetailSection,
    DetailSidePanel,
    LootGrid,
    DetailLoading,
    DetailError,
} from '../../ui'
import { LootItem } from '../../ui'

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
                    </DetailSection>

                    {/* Rewards */}
                    <DetailSection title="Rewards">
                        <div className="space-y-4">
                            {detail.rewardMoney > 0 && (
                                <div className="flex w-fit items-center gap-2 rounded border border-wow-gold/10 bg-wow-gold/5 px-3 py-1.5 text-wow-gold">
                                    <span className="text-xs font-bold uppercase text-gray-500">
                                        Money:
                                    </span>
                                    <span>
                                        {Math.floor(detail.rewardMoney / 10000)}g{' '}
                                        {Math.floor((detail.rewardMoney % 10000) / 100)}s
                                    </span>
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
                                        {rep.value} {rep.name}
                                    </span>
                                </div>
                            ))}
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
                            {detail.requiredClasses > 0 && (
                                <div className="flex justify-between border-b border-white/5 pb-1">
                                    <span>Classes:</span>
                                    <span className="font-mono text-white">
                                        {detail.requiredClasses}
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
