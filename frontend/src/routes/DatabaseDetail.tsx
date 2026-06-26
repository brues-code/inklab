import { useParams, useNavigate, useRouter } from '@tanstack/react-router'
import { EntityDetailView } from '../components/database/detailview/EntityDetailView'
import { useTooltipCtx } from '../hooks/useTooltipContext'

/**
 * Database entity detail route (/database/$tab/$type/$id). Renders in the
 * DatabaseTabs <Outlet>. Sub-navigation keeps the originating $tab so the Back
 * trail unwinds to the tab list; Back uses real browser history.
 */
export function DatabaseDetail() {
    const { tab, type, id } = useParams({ strict: false }) as { tab: string; type: string; id: string }
    const navigate = useNavigate()
    const router = useRouter()
    const tooltipHook = useTooltipCtx()

    const onBack = () => router.history.back()
    const onNavigate = (t: string, entry: number) =>
        navigate({ to: '/database/$tab/$type/$id', params: { tab, type: t, id: String(entry) } })

    return (
        <EntityDetailView
            type={type}
            entry={Number(id)}
            onBack={onBack}
            onNavigate={onNavigate}
            tooltipHook={tooltipHook}
        />
    )
}

export default DatabaseDetail
