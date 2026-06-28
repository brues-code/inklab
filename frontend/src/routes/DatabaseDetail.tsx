import { useParams, useNavigate, useRouter, useSearch } from '@tanstack/react-router'
import { EntityDetailView } from '../components/database/detailview/EntityDetailView'
import { useTooltipCtx } from '../hooks/useTooltipContext'

/**
 * Database entity detail route (/database/$tab/$type/$id). Renders in the
 * DatabaseTabs <Outlet>. Sub-navigation keeps the originating $tab so the Back
 * trail unwinds to the tab list; Back uses real browser history.
 *
 * The active relations sub-tab is held in the `rel` search param so each tab
 * switch is a history entry (Back/Forward works) and survives refresh. Switching
 * entities drops `rel`, so a new entity starts on its first tab.
 */
export function DatabaseDetail() {
    const { tab, type, id } = useParams({ strict: false }) as {
        tab: string
        type: string
        id: string
    }
    const { rel } = useSearch({ strict: false }) as { rel?: string }
    const navigate = useNavigate()
    const router = useRouter()
    const tooltipHook = useTooltipCtx()

    const onBack = () => router.history.back()
    const onNavigate = (t: string, entry: number) =>
        navigate({ to: '/database/$tab/$type/$id', params: { tab, type: t, id: String(entry) } })
    const onTabChange = (next: string) =>
        navigate({
            to: '/database/$tab/$type/$id',
            params: { tab, type, id },
            search: { rel: next },
        })

    return (
        <EntityDetailView
            type={type}
            entry={Number(id)}
            onBack={onBack}
            onNavigate={onNavigate}
            tooltipHook={tooltipHook}
            activeTab={rel}
            onTabChange={onTabChange}
        />
    )
}

export default DatabaseDetail
