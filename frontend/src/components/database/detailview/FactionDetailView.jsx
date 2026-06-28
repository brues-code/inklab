import React, { useState } from 'react'
import { DATABASE_BASE_URL } from '../../../utils/constants'
import { useFactionDetail } from '../../../hooks/queries/factions'
import { DetailPageLayout, DetailHeader, DetailSection, DetailLoading, DetailError } from '../../ui'

const FactionDetailView = ({ id, onBack, onNavigate, activeTab, onTabChange }) => {
    // Active tab. When the route supplies it (activeTab/onTabChange) it lives in
    // the URL so Back/Forward and refresh work; otherwise fall back to local
    // state. null = first tab with data.
    const [localTab, setLocalTab] = useState(null)
    const rawTab = onTabChange ? activeTab : localTab
    const setTab = onTabChange || setLocalTab
    const { data: detail, isLoading: loading } = useFactionDetail(id)

    if (loading) return <DetailLoading />
    if (!detail) return <DetailError message="Faction not found" onBack={onBack} />

    const quests = detail.quests || []
    const questGivers = detail.questGivers || []
    const members = detail.members || []

    // Tabs for the relationship tables (only those with data).
    const tabs = [
        quests.length > 0 && { id: 'quests', label: `Reputation Quests (${quests.length})` },
        questGivers.length > 0 && { id: 'givers', label: `Quest Givers (${questGivers.length})` },
        members.length > 0 && { id: 'members', label: `Faction Members (${members.length})` },
    ].filter(Boolean)
    const currentTab = tabs.some((t) => t.id === rawTab) ? rawTab : tabs[0]?.id

    const npcGrid = (npcs) => (
        <div className="grid grid-cols-1 gap-2 md:grid-cols-2 lg:grid-cols-3">
            {npcs.map((n) => (
                <div
                    key={n.entry}
                    onClick={() => onNavigate('npc', n.entry)}
                    className="flex cursor-pointer items-center justify-between gap-2 rounded border border-white/5 bg-white/[0.02] p-3 transition-colors hover:bg-white/5"
                >
                    <span className="min-w-0 truncate">
                        <span className="font-medium text-wow-gold hover:text-yellow-300">
                            {n.name}
                        </span>
                        {n.subname && (
                            <span className="ml-1 text-xs text-gray-500">&lt;{n.subname}&gt;</span>
                        )}
                    </span>
                    <span className="whitespace-nowrap text-xs text-gray-500">
                        Lvl {n.levelMin}
                        {n.levelMax > n.levelMin ? `-${n.levelMax}` : ''}
                    </span>
                </div>
            ))}
        </div>
    )

    // Side color and icon
    const getSideStyle = () => {
        switch (detail.side) {
            case 1:
                return {
                    color: 'text-blue-400',
                    icon: '🔵',
                    img: '/Alliance_15.webp',
                    name: 'Alliance',
                }
            case 2:
                return { color: 'text-red-400', icon: '🔴', img: '/Horde_15.webp', name: 'Horde' }
            default:
                return {
                    color: 'text-yellow-400',
                    icon: '🟡',
                    img: '/Neutral_15.webp',
                    name: 'Neutral',
                }
        }
    }

    const sideStyle = getSideStyle()

    return (
        <DetailPageLayout onBack={onBack}>
            <DetailHeader
                icon={
                    <div className="flex h-full w-full items-center justify-center border border-gray-700 bg-gray-900 p-1">
                        <img
                            src={sideStyle.img}
                            alt={sideStyle.name}
                            className="h-full w-full object-contain"
                        />
                    </div>
                }
                iconBorderColor={sideStyle.color}
                title={detail.name}
                titleColor={sideStyle.color}
                subtitle={sideStyle.name}
                action={
                    <a
                        href={`${DATABASE_BASE_URL}/?faction=${detail.id}`}
                        target="_blank"
                        rel="noreferrer"
                        className="rounded bg-purple-700 px-3 py-1.5 text-xs font-bold uppercase text-white transition-colors hover:bg-purple-600"
                    >
                        🔗 OctoHead
                    </a>
                }
            />

            <div className="grid grid-cols-1 gap-8 lg:grid-cols-2">
                {/* Description */}
                {detail.description && (
                    <DetailSection title="Description">
                        <p className="text-sm leading-relaxed text-gray-300">
                            {detail.description}
                        </p>
                    </DetailSection>
                )}

                {/* Quick Facts */}
                <DetailSection title="Quick Facts">
                    <table className="infobox-table w-full text-sm">
                        <tbody>
                            <tr>
                                <th className="py-1 pr-4 text-gray-400">Faction ID:</th>
                                <td className="text-white">{detail.id}</td>
                            </tr>
                            <tr>
                                <th className="py-1 pr-4 text-gray-400">Side:</th>
                                <td className={sideStyle.color}>{sideStyle.name}</td>
                            </tr>
                            <tr>
                                <th className="py-1 pr-4 text-gray-400">Related Quests:</th>
                                <td className="text-white">{quests.length}</td>
                            </tr>
                        </tbody>
                    </table>
                </DetailSection>
            </div>

            {/* Relationship tables — tabbed */}
            {tabs.length > 0 && (
                <div className="mt-8">
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

                    <div className="animate-fade-in">
                        {currentTab === 'quests' && (
                            <div className="grid grid-cols-1 gap-2 md:grid-cols-2 lg:grid-cols-3">
                                {quests.map((q) => (
                                    <div
                                        key={q.entry}
                                        onClick={() => onNavigate('quest', q.entry)}
                                        className="flex cursor-pointer items-center gap-2 rounded border border-white/5 bg-white/[0.02] p-3 transition-colors hover:bg-white/5"
                                    >
                                        {(q.side === 'Horde' || q.side === 'Alliance') && (
                                            <img
                                                src={
                                                    q.side === 'Horde'
                                                        ? '/Horde_15.webp'
                                                        : '/Alliance_15.webp'
                                                }
                                                className="h-4 w-4 flex-shrink-0 object-contain"
                                                alt={q.side}
                                                title={q.side}
                                            />
                                        )}
                                        <span className="min-w-0 truncate font-medium text-wow-gold hover:text-yellow-300">
                                            [{q.level}] {q.title}
                                        </span>
                                    </div>
                                ))}
                            </div>
                        )}
                        {currentTab === 'givers' && npcGrid(questGivers)}
                        {currentTab === 'members' && npcGrid(members)}
                    </div>
                </div>
            )}
        </DetailPageLayout>
    )
}

export default FactionDetailView
