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
    DetailError
} from '../../ui'
import { LootItem } from '../../ui'

const QuestDetailView = ({ entry, onBack, onNavigate, tooltipHook }) => {
    const { data: detail, isLoading: loading, isError, error } = useQuestDetail(entry)

    const handleSync = () => {
        // Drop the cache so the quest list/search behind this overlay refetch too.
        SyncQuestData(entry).then(() => queryClient.invalidateQueries());
    };

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
                        <span>Level {detail.questLevel} (Min {detail.minLevel}) - {getQuestType(detail.type)}</span>
                        {detail.side && detail.side !== "Both" && (
                            <span className={`inline-flex items-center gap-1.5 ml-3 px-2 py-0.5 rounded bg-black/20 border border-white/5 ${detail.side === "Horde" ? "text-red-400" : "text-blue-400"}`}>
                                <img 
                                    src={detail.side === "Horde" ? "/Horde_15.webp" : "/Alliance_15.webp"} 
                                    className="w-4 h-4 object-contain" 
                                    alt={detail.side}
                                />
                                <span className="font-bold text-[10px] uppercase tracking-wider">{detail.side}</span>
                            </span>
                        )}
                    </div>
                }
                action={
                    <div className="flex gap-2">
                        <button
                            onClick={handleSync}
                            className="px-3 py-1.5 bg-blue-600 hover:bg-blue-500 text-white text-xs font-bold uppercase rounded border border-blue-700 transition-colors flex items-center gap-1"
                            title="Re-download data from external sources"
                        >
                            <span>↻</span> Sync
                        </button>
                        <a
                            href={`${DATABASE_BASE_URL}/?quest=${entry}`}
                            target="_blank"
                            rel="noreferrer"
                            className="px-3 py-1.5 text-xs font-bold uppercase rounded transition-colors bg-purple-700 hover:bg-purple-600 text-white"
                            title="View on Turtle WoW Database"
                        >
                            🔗 OctoHead
                        </a>
                    </div>
                }
            />
            
            <div className="grid grid-cols-1 lg:grid-cols-[2fr_1fr] gap-10">
                {/* Main Content */}
                <div className="space-y-8">
                    <DetailSection title="Description">
                        <p className="text-gray-300 leading-relaxed whitespace-pre-wrap">
                            {detail.details || 'No description available.'}
                        </p>
                    </DetailSection>
                    
                    <DetailSection title="Objectives">
                        <p className="text-gray-300 leading-relaxed whitespace-pre-wrap">
                            {detail.objectives || 'No objectives listed.'}
                        </p>
                    </DetailSection>

                    {/* Rewards */}
                    <DetailSection title="Rewards">
                        <div className="space-y-4">
                            {detail.rewardMoney > 0 && (
                                <div className="flex items-center gap-2 text-wow-gold bg-wow-gold/5 px-3 py-1.5 rounded border border-wow-gold/10 w-fit">
                                    <span className="text-xs uppercase font-bold text-gray-500">Money:</span>
                                    <span>{Math.floor(detail.rewardMoney/10000)}g {Math.floor((detail.rewardMoney%10000)/100)}s</span>
                                </div>
                            )}
                            {detail.rewardXp > 0 && (
                                <div className="flex items-center gap-2 text-wow-rare bg-wow-rare/5 px-3 py-1.5 rounded border border-wow-rare/10 w-fit">
                                    <span className="text-xs uppercase font-bold text-gray-500">Experience:</span>
                                    <span>{detail.rewardXp} XP</span>
                                </div>
                            )}
                            {detail.reputation?.map((rep, idx) => (
                                <div
                                    key={idx}
                                    className={`flex items-center gap-2 px-3 py-1.5 rounded border w-fit ${rep.value >= 0 ? 'text-green-400 bg-green-400/5 border-green-400/10' : 'text-red-400 bg-red-400/5 border-red-400/10'}`}
                                >
                                    <span className="text-xs uppercase font-bold text-gray-500">Reputation:</span>
                                    <span>{rep.value >= 0 ? '+' : ''}{rep.value} {rep.name}</span>
                                </div>
                            ))}
                        </div>
                        
                        {detail.rewardItems?.length > 0 && (
                            <div className="mt-6">
                                <h4 className="text-gray-400 text-sm font-semibold mb-3 uppercase tracking-wider">
                                    You will receive:
                                </h4>
                                <LootGrid>
                                    {detail.rewardItems.map(i => renderRewardItem(i))}
                                </LootGrid>
                            </div>
                        )}
                        
                        {detail.choiceItems?.length > 0 && (
                            <div className="mt-6">
                                <h4 className="text-gray-400 text-sm font-semibold mb-3 uppercase tracking-wider">
                                    Choose one of:
                                </h4>
                                <LootGrid>
                                    {detail.choiceItems.map(i => renderRewardItem(i))}
                                </LootGrid>
                            </div>
                        )}
                    </DetailSection>
                </div>
                
                {/* Side Panel */}
                <DetailSidePanel className="space-y-6">
                    {/* Quest Chain */}
                    <div>
                        <h3 className="text-wow-gold font-bold mb-3 border-b border-wow-gold/20 pb-1">
                            Quest Chain
                        </h3>
                        {detail.series?.length > 0 ? (
                            <div className="space-y-1">
                                {detail.series.map((s, index) => {
                                    const isChild = s.depth > 0;
                                    const indent = isChild ? s.depth * 16 : 0;
                                    
                                    return (
                                        <div key={s.entry} className="flex gap-2 text-[13px]" style={{ paddingLeft: `${indent}px` }}>
                                            <span className="text-gray-600 w-6 flex-shrink-0 text-right font-mono">
                                                {isChild ? '└─' : `${index + 1}.`}
                                            </span>
                                            {s.entry === detail.entry ? (
                                                <b className="text-white bg-white/5 px-1 rounded">{s.title}</b>
                                            ) : (
                                                <a 
                                                    className="text-wow-rare hover:underline cursor-pointer"
                                                    onClick={() => onNavigate('quest', s.entry)}
                                                >
                                                    {s.title}
                                                </a>
                                            )}
                                        </div>
                                    );
                                })}
                            </div>
                        ) : (
                            <div className="text-gray-600 text-sm italic">Standalone quest.</div>
                        )}
                    </div>

                    {/* Prerequisites */}
                    {detail.prevQuests?.length > 0 && (
                        <div>
                            <h3 className="text-wow-gold font-bold mb-3 border-b border-wow-gold/20 pb-1">
                                Prerequisites
                            </h3>
                            <div className="space-y-2">
                                {detail.prevQuests.map(q => (
                                    <div key={q.entry} className="text-[13px]">
                                        <a 
                                            className="text-wow-rare hover:underline cursor-pointer"
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
                        <h3 className="text-wow-gold font-bold mb-3 border-b border-wow-gold/20 pb-1">
                            Requirements
                        </h3>
                        <div className="text-[13px] space-y-2 text-gray-300">
                            {(detail.raceNames || detail.requiredRaces > 0) && (
                                <div className="flex justify-between border-b border-white/5 pb-1">
                                    <span>Races:</span>
                                    <span className="text-white font-mono text-right text-xs leading-tight pl-4 max-w-[200px]">
                                        {detail.raceNames || detail.requiredRaces}
                                    </span>
                                </div>
                            )}
                            {detail.requiredClasses > 0 && (
                                <div className="flex justify-between border-b border-white/5 pb-1">
                                    <span>Classes:</span>
                                    <span className="text-white font-mono">{detail.requiredClasses}</span>
                                </div>
                            )}
                            {detail.srcItemId > 0 && (
                                <div className="flex items-center gap-2">
                                    <span>Starts from:</span>
                                    <a 
                                        className="text-wow-gold hover:underline cursor-pointer bg-wow-gold/5 px-2 py-0.5 rounded border border-wow-gold/20"
                                        onClick={() => onNavigate('item', detail.srcItemId)}
                                    >
                                        [Item {detail.srcItemId}]
                                    </a>
                                </div>
                            )}
                            {!detail.requiredRaces && !detail.requiredClasses && !detail.srcItemId && (
                                <div className="text-gray-600 italic">None</div>
                            )}
                        </div>
                    </div>

                    {/* Relations */}
                    <div>
                        <h3 className="text-wow-gold font-bold mb-3 border-b border-wow-gold/20 pb-1">
                            Relations
                        </h3>
                        <div className="space-y-4">
                            {detail.starters?.length > 0 && (
                                <div>
                                    <h4 className="text-gray-500 text-xs font-bold uppercase mb-2 tracking-tighter">
                                        Starts with:
                                    </h4>
                                    <div className="space-y-1">
                                        {detail.starters.map(s => (
                                            <div 
                                                key={s.entry}
                                                onClick={() => s.type === 'npc' && onNavigate('npc', s.entry)}
                                                className={`text-xs px-2 py-1 rounded border ${
                                                    s.type === 'npc'
                                                        ? 'bg-wow-rare/5 border-wow-rare/20 text-wow-rare cursor-pointer hover:bg-wow-rare/10'
                                                        : 'bg-white/5 border-white/10 text-gray-400'
                                                }`}
                                            >
                                                {s.name} <span className="opacity-50">({s.type})</span>
                                            </div>
                                        ))}
                                    </div>
                                </div>
                            )}
                            {detail.enders?.length > 0 && (
                                <div>
                                    <h4 className="text-gray-500 text-xs font-bold uppercase mb-2 tracking-tighter">
                                        Ends with:
                                    </h4>
                                    <div className="space-y-1">
                                        {detail.enders.map(s => (
                                            <div 
                                                key={s.entry}
                                                onClick={() => s.type === 'npc' && onNavigate('npc', s.entry)}
                                                className={`text-xs px-2 py-1 rounded border ${
                                                    s.type === 'npc'
                                                        ? 'bg-wow-rare/5 border-wow-rare/20 text-wow-rare cursor-pointer hover:bg-wow-rare/10'
                                                        : 'bg-white/5 border-white/10 text-gray-400'
                                                }`}
                                            >
                                                {s.name} <span className="opacity-50">({s.type})</span>
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
