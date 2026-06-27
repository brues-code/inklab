import { Link, Outlet, useParams, useChildMatches, useNavigate } from '@tanstack/react-router'
import { PageLayout, ContentGrid, TabBar } from '../components/ui'
import { GRID_LAYOUT, SETS_LAYOUT } from '../components/common/layout'
import {
    ItemsTab,
    SetsTab,
    NPCsTab,
    QuestsTab,
    ObjectsTab,
    ZonesTab,
    SpellsTab,
    FactionsTab,
    RacesTab,
} from '../components/database/tabs'
import { useTooltipCtx } from '../hooks/useTooltipContext'

const TABS = ['Items', 'Sets', 'NPCs', 'Quests', 'Objects', 'Zones', 'Spells', 'Factions', 'Races']

const TAB_BASE =
    'px-4 py-2 font-bold text-sm cursor-pointer transition-all duration-200 border ' +
    'bg-transparent border-transparent text-wow-gold uppercase text-[13px] rounded-none hover:bg-bg-hover'
const TAB_ACTIVE = '!bg-bg-active !text-white !border-border-light'

/**
 * Database tabs view (route /database/$tab). The active tab comes from the URL
 * param; clicking an entity navigates to the nested detail route which renders
 * in <Outlet>. While a detail is active the detail renders as an opaque overlay
 * ON TOP of the still-displayed list (rather than display:none-ing the list).
 * This is deliberate: Chromium/WebView2 resets a scroll container's scrollTop
 * when an ancestor toggles display:none, so hiding the list that way wiped your
 * place on Back. Keeping the list displayed (just covered) preserves both its
 * scroll position and filter state for free.
 */
export function DatabaseTabs() {
    const { tab } = useParams({ strict: false }) as { tab?: string }
    const activeTab = (tab || 'items').toLowerCase()
    const navigate = useNavigate()
    const tooltipHook = useTooltipCtx()

    // A detail child route is matched when there are matches below this route.
    const detailActive = useChildMatches().length > 0

    // Start a detail trail under the current tab; Back returns here.
    const onNavigate = (type: string, entry: number | string) =>
        navigate({
            to: '/database/$tab/$type/$id',
            params: { tab: activeTab, type, id: String(entry) },
        })

    return (
        <PageLayout>
            <div className="relative flex-1 flex flex-col overflow-hidden">
            {/* Tab list — always displayed (never display:none) so its scroll
                position and filters survive a detail visit; the detail covers it. */}
            <div className="flex flex-col h-full flex-1 overflow-hidden">
                <TabBar>
                    {TABS.map(label => {
                        const key = label.toLowerCase()
                        return (
                            <Link
                                key={key}
                                to="/database/$tab"
                                params={{ tab: key }}
                                activeOptions={{ exact: false }}
                                className={TAB_BASE}
                                activeProps={{ className: TAB_ACTIVE }}
                            >
                                {label}
                            </Link>
                        )
                    })}
                </TabBar>

                {/* Content area */}
                {activeTab === 'items' ? (
                    <ItemsTab tooltipHook={tooltipHook} onNavigate={onNavigate} />
                ) : (
                    <ContentGrid columns={activeTab === 'sets' ? SETS_LAYOUT : GRID_LAYOUT}>
                        {activeTab === 'sets' && <SetsTab tooltipHook={tooltipHook} onNavigate={onNavigate} />}
                        {activeTab === 'npcs' && <NPCsTab onNavigate={onNavigate} tooltipHook={tooltipHook} />}
                        {activeTab === 'quests' && <QuestsTab onNavigate={onNavigate} />}
                        {activeTab === 'objects' && <ObjectsTab onNavigate={onNavigate} />}
                        {activeTab === 'zones' && <ZonesTab onNavigate={onNavigate} />}
                        {activeTab === 'spells' && <SpellsTab onNavigate={onNavigate} tooltipHook={tooltipHook} />}
                        {activeTab === 'factions' && <FactionsTab onNavigate={onNavigate} />}
                        {activeTab === 'races' && <RacesTab onNavigate={onNavigate} tooltipHook={tooltipHook} />}
                    </ContentGrid>
                )}
            </div>

            {/* Detail overlay: covers the list (opaque) while preserving its
                scroll. Only the overlay is display:none'd when inactive — never
                the list — so the detail's own scroll reset on exit is harmless. */}
            <div className={`absolute inset-0 z-20 flex flex-col bg-bg-dark overflow-hidden ${detailActive ? '' : 'hidden'}`}>
                <Outlet />
            </div>
            </div>
        </PageLayout>
    )
}

export default DatabaseTabs
