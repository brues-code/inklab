import { useState, useEffect } from 'react'
import { useItemTooltip } from '../../hooks/useItemTooltip'
import { 
    PageLayout, 
    ContentGrid, 
    TabButton, 
    TabBar,
    ItemTooltip 
} from '../../components/ui'
import { NPCDetailView, QuestDetailView, ItemDetailView, SpellDetailView, ObjectDetailView, ZoneDetailView, FactionDetailView } from '../../components/database/detailview'
import { GRID_LAYOUT, ITEMS_LAYOUT, SETS_LAYOUT } from '../../components/common/layout'

// Import tab components
import { ItemsTab, SetsTab, NPCsTab, QuestsTab, ObjectsTab, ZonesTab, SpellsTab, FactionsTab } from '../../components/database/tabs'

const TABS = ['Items', 'Sets', 'NPCs', 'Quests', 'Objects', 'Zones', 'Spells', 'Factions']

function DatabasePage({ pendingNavigation, onNavigationHandled }) {
    const [activeTab, setActiveTab] = useState('items')
    
    // Navigation State for Detail Views
    const [detailStack, setDetailStack] = useState([]) // Stack of views: { type, entry }
    
    // Use shared tooltip hook
    const tooltipHook = useItemTooltip()
    const {
        hoveredItem,
        tooltipCache,
        getTooltipStyle,
    } = tooltipHook

    // Handle pending navigation from other pages (e.g., SearchPage)
    useEffect(() => {
        if (pendingNavigation) {
            console.log(`[DatabasePage] Received pending navigation: ${pendingNavigation.type} #${pendingNavigation.entry}`)
            navigateTo(pendingNavigation.type, pendingNavigation.entry)
            onNavigationHandled?.()
        }
    }, [pendingNavigation, onNavigationHandled])

    // Detail View Logic
    const navigateTo = (type, entry) => {
        console.log(`[DatabasePage] Navigating to ${type} with entry: ${entry}`);
        // Clear tooltip before navigation to prevent it from persisting
        tooltipHook.setHoveredItem(null)
        // Ignore a push that just repeats the current top of the stack. This
        // keeps StrictMode's double-fired mount effect (and stray double-clicks)
        // from stacking the same detail twice, which would otherwise need two
        // Back presses to leave.
        setDetailStack(prev => {
            const top = prev[prev.length - 1]
            if (top && top.type === type && top.entry === entry) return prev
            return [...prev, { type, entry }]
        })
    }
    const goBack = () => {
        console.log(`[DatabasePage] Going back. Previous stack size: ${detailStack.length}`);
        setDetailStack(prev => prev.slice(0, -1))
    }

    const currentDetail = detailStack.length > 0 ? detailStack[detailStack.length - 1] : null
    
    // Enhanced tooltip hook to pass to tabs
    const enhancedTooltipHook = {
        ...tooltipHook,
        renderTooltip: () => null,
    }

    return (
        <PageLayout>
            {/* Tabs View - Hidden when detail active, but kept mounted to preserve state */}
            <div className={`flex flex-col h-full flex-1 overflow-hidden ${currentDetail ? 'hidden' : ''}`}>
                {/* Tab Bar */}
                <TabBar>
                    {TABS.map(tab => (
                        <TabButton
                            key={tab}
                            active={activeTab === tab.toLowerCase()}
                            onClick={() => setActiveTab(tab.toLowerCase())}
                        >
                            {tab}
                        </TabButton>
                    ))}
                </TabBar>

                {/* Content Area */}
                {activeTab === 'items' ? (
                    /* ItemsTab manages its own ContentGrid for dynamic layout */
                    <ItemsTab 
                        tooltipHook={enhancedTooltipHook} 
                        onNavigate={navigateTo}
                    />
                ) : (
                    <ContentGrid columns={activeTab === 'sets' ? SETS_LAYOUT : GRID_LAYOUT}>
                        {activeTab === 'sets' && (
                            <SetsTab tooltipHook={enhancedTooltipHook} />
                        )}
                        {activeTab === 'npcs' && (
                            <NPCsTab 
                                onNavigate={navigateTo}
                                tooltipHook={enhancedTooltipHook}
                            />
                        )}
                        {activeTab === 'quests' && (
                            <QuestsTab onNavigate={navigateTo} />
                        )}
                        {activeTab === 'objects' && (
                            <ObjectsTab onNavigate={navigateTo} />
                        )}
                        {activeTab === 'zones' && (
                            <ZonesTab onNavigate={navigateTo} />
                        )}
                        {activeTab === 'spells' && (
                            <SpellsTab onNavigate={navigateTo} />
                        )}
                        {activeTab === 'factions' && (
                            <FactionsTab onNavigate={navigateTo} />
                        )}
                    </ContentGrid>
                )}
            </div>

            {/* Detail View Overlay */}
            {currentDetail && (
                <div className="flex flex-col h-full flex-1 overflow-hidden">
                    {/* Detail Header with breadcrumb */}
                    <div className="bg-bg-hover px-4 py-2 border-b border-border-dark flex items-center gap-4">
                        <button 
                            onClick={goBack}
                            className="bg-bg-panel border border-border-light text-gray-400 px-4 py-1.5 rounded hover:bg-bg-active hover:text-white transition-colors text-sm"
                        >
                            ← Back
                        </button>
                        <span className="text-gray-500 text-sm">
                            Viewing: <b className="text-gray-300 uppercase">{currentDetail.type}</b> 
                            <span className="ml-2 font-mono bg-black/20 px-1.5 py-0.5 rounded">#{currentDetail.entry}</span>
                        </span>
                    </div>
                    
                    {/* Detail Content */}
                    <div className="flex-1 overflow-auto">
                        {currentDetail.type === 'npc' && (
                            <NPCDetailView 
                                entry={currentDetail.entry} 
                                onNavigate={navigateTo}
                                onBack={goBack}
                                tooltipHook={enhancedTooltipHook}
                            />
                        )}
                        {currentDetail.type === 'quest' && (
                            <QuestDetailView 
                                entry={currentDetail.entry} 
                                onNavigate={navigateTo}
                                onBack={goBack}
                                tooltipHook={enhancedTooltipHook}
                            />
                        )}
                        {currentDetail.type === 'item' && (
                            <ItemDetailView 
                                entry={currentDetail.entry} 
                                onNavigate={navigateTo}
                                onBack={goBack}
                                tooltipHook={enhancedTooltipHook}
                            />
                        )}
                        {currentDetail.type === 'spell' && (
                            <SpellDetailView 
                                entry={currentDetail.entry} 
                                onNavigate={navigateTo}
                                onBack={goBack}
                                tooltipHook={enhancedTooltipHook}
                            />
                        )}
                        {currentDetail.type === 'object' && (
                            <ObjectDetailView
                                entry={currentDetail.entry}
                                onNavigate={navigateTo}
                                onBack={goBack}
                                tooltipHook={enhancedTooltipHook}
                            />
                        )}
                        {currentDetail.type === 'zone' && (
                            <ZoneDetailView
                                entry={currentDetail.entry}
                                onNavigate={navigateTo}
                                onBack={goBack}
                            />
                        )}
                        {currentDetail.type === 'faction' && (
                            <FactionDetailView 
                                id={currentDetail.entry} 
                                onNavigate={navigateTo}
                                onBack={goBack}
                            />
                        )}
                    </div>
                </div>
            )}

            {/* Global Tooltip Layer */}
            {hoveredItem && tooltipCache[hoveredItem] && (
                <ItemTooltip
                    item={tooltipCache[hoveredItem]}
                    tooltip={tooltipCache[hoveredItem]}
                    style={getTooltipStyle()}
                />
            )}
        </PageLayout>
    )
}

export default DatabasePage
