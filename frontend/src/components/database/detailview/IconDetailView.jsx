import { useState } from 'react'
import { QUESTION_MARK_ICON } from '../../../utils/wow'
import { useIcon } from '../../../services/useImage'
import { useIconUsage } from '../../../hooks/queries/icons'
import { LootItem, ScrollList } from '../../ui'

/**
 * Icon detail page: the icon itself plus every entity that renders it — items
 * (via item_display_info) and spells (via spell_icons) — as tabs, so the popup's
 * usage counts can deep-link to a specific one (?rel=items / ?rel=spells). When
 * the route supplies activeTab/onTabChange the tab lives in the URL (Back /
 * Forward / refresh work); otherwise it falls back to local state.
 */
function IconDetailView({ name, onNavigate, tooltipHook, activeTab, onTabChange }) {
    const icon = useIcon(name)
    const { data: usage, isLoading } = useIconUsage(name)
    const [localTab, setLocalTab] = useState(null)
    const rawTab = onTabChange ? activeTab : localTab
    const setTab = onTabChange || setLocalTab

    const items = usage?.items || []
    const spells = usage?.spells || []

    const tabs = [
        items.length > 0 && { id: 'items', label: `Items (${items.length})` },
        spells.length > 0 && { id: 'spells', label: `Spells (${spells.length})` },
    ].filter(Boolean)
    const currentTab = tabs.some((t) => t.id === rawTab) ? rawTab : tabs[0]?.id

    return (
        <div className="flex h-full flex-col overflow-hidden">
            {/* Header */}
            <div className="flex items-center gap-4 border-b border-border-dark bg-bg-hover p-4">
                <div className="flex h-14 w-14 shrink-0 items-center justify-center overflow-hidden rounded border border-gray-600 bg-black/40">
                    <img
                        src={icon.src || QUESTION_MARK_ICON}
                        alt=""
                        className="h-full w-full object-cover"
                    />
                </div>
                <div>
                    <h2 className="font-mono text-xl font-bold text-white">{name}</h2>
                    <div className="text-xs text-gray-500">
                        {isLoading
                            ? 'Loading usage...'
                            : `Used by ${items.length} item${items.length !== 1 ? 's' : ''} and ${spells.length} spell${spells.length !== 1 ? 's' : ''}`}
                    </div>
                </div>
            </div>

            <ScrollList className="p-4">
                {!isLoading && tabs.length === 0 && (
                    <div className="py-8 text-center text-sm italic text-gray-600">
                        Nothing in the database uses this icon.
                    </div>
                )}

                {tabs.length > 0 && (
                    <>
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
                            {currentTab === 'items' && (
                                <div className="grid grid-cols-1 gap-1 md:grid-cols-2 lg:grid-cols-3">
                                    {items.map((item) => {
                                        const handlers =
                                            tooltipHook?.getItemHandlers?.(item.entry) || {}
                                        return (
                                            <LootItem
                                                key={item.entry}
                                                item={{
                                                    entry: item.entry,
                                                    name: item.name,
                                                    quality: item.quality,
                                                    iconPath: name,
                                                }}
                                                onClick={() => onNavigate?.('item', item.entry)}
                                                {...handlers}
                                            />
                                        )
                                    })}
                                </div>
                            )}

                            {currentTab === 'spells' && (
                                <div className="grid grid-cols-1 gap-1 md:grid-cols-2 lg:grid-cols-3">
                                    {spells.map((spell) => (
                                        <div
                                            key={spell.entry}
                                            className="flex cursor-pointer items-center gap-2 rounded border border-white/5 bg-white/[0.02] p-1.5 transition-all hover:bg-white/5"
                                            onClick={() => onNavigate?.('spell', spell.entry)}
                                            {...(tooltipHook?.getSpellHandlers?.(spell.entry) ||
                                                {})}
                                        >
                                            <div className="flex h-8 w-8 shrink-0 items-center justify-center overflow-hidden rounded border border-gray-700 bg-black/40">
                                                <img
                                                    src={icon.src || QUESTION_MARK_ICON}
                                                    alt=""
                                                    className="h-full w-full object-cover"
                                                />
                                            </div>
                                            <span className="min-w-[40px] font-mono text-[11px] text-gray-600">
                                                [{spell.entry}]
                                            </span>
                                            <span className="truncate text-[13px] font-bold text-gray-200">
                                                {spell.name}
                                            </span>
                                            {spell.rank && (
                                                <span className="ml-auto shrink-0 text-[11px] text-gray-500">
                                                    {spell.rank}
                                                </span>
                                            )}
                                        </div>
                                    ))}
                                </div>
                            )}
                        </div>
                    </>
                )}
            </ScrollList>
        </div>
    )
}

export default IconDetailView
