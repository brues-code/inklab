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
 * in <Outlet>. While a detail is active the tab list is hidden (but kept
 * mounted) so Back returns to the same scroll/filter state — mirroring the old
 * detailStack overlay behavior, now backed by browser history.
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
            {/* Tab list — hidden (kept mounted) while a detail is open */}
            <div className={`flex flex-col h-full flex-1 overflow-hidden ${detailActive ? 'hidden' : ''}`}>
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
                        {activeTab === 'sets' && <SetsTab tooltipHook={tooltipHook} />}
                        {activeTab === 'npcs' && <NPCsTab onNavigate={onNavigate} tooltipHook={tooltipHook} />}
                        {activeTab === 'quests' && <QuestsTab onNavigate={onNavigate} />}
                        {activeTab === 'objects' && <ObjectsTab onNavigate={onNavigate} />}
                        {activeTab === 'zones' && <ZonesTab onNavigate={onNavigate} />}
                        {activeTab === 'spells' && <SpellsTab onNavigate={onNavigate} />}
                        {activeTab === 'factions' && <FactionsTab onNavigate={onNavigate} />}
                        {activeTab === 'races' && <RacesTab onNavigate={onNavigate} />}
                    </ContentGrid>
                )}
            </div>

            {/* Detail overlay */}
            <Outlet />
        </PageLayout>
    )
}

export default DatabaseTabs
