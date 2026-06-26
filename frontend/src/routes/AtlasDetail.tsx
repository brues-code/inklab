import { useParams, useNavigate, useRouter } from '@tanstack/react-router'
import { EntityDetailView } from '../components/database/detailview/EntityDetailView'
import { useTooltipCtx } from '../hooks/useTooltipContext'

/**
 * AtlasLoot entity detail route (/atlas/$type/$id). Renders in the AtlasLoot
 * <Outlet>; sub-navigation stays under /atlas so Back returns to the loot table
 * (whose selection state is preserved because the AtlasLoot route stays mounted).
 */
export function AtlasDetail() {
    const { type, id } = useParams({ strict: false }) as { type: string; id: string }
    const navigate = useNavigate()
    const router = useRouter()
    const tooltipHook = useTooltipCtx()

    const onBack = () => router.history.back()
    const onNavigate = (t: string, entry: number) =>
        navigate({ to: '/atlas/$type/$id', params: { type: t, id: String(entry) } })

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

export default AtlasDetail
