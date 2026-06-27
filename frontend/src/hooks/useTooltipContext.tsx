import {
    createContext,
    useContext,
    useEffect,
    useLayoutEffect,
    useRef,
    useState,
    type ReactNode,
} from 'react'
import { useQuery } from '@tanstack/react-query'
import { useRouterState } from '@tanstack/react-router'
import { useItemTooltip, type TooltipHook } from './useItemTooltip'
import { useSpellTooltip, type SpellTooltipHook } from './useSpellTooltip'
import { tooltipQuery } from './queries/tooltip'
import { spellDetailQuery } from './queries/spells'
import { ItemTooltip, SpellTooltip } from '../components/ui'

/**
 * App-wide item tooltip context. One hook instance lives at the router root so
 * every page (Database tabs, entity detail views, AtlasLoot) shares the hover
 * state and the single floating tooltip layer rendered here. The tooltip data
 * itself lives in TanStack Query (keyed ['tooltip', id]).
 *
 * The context value carries `renderTooltip: () => null` so it satisfies the
 * `tooltipHook` prop shape the tab/detail components expect.
 */
type TooltipContextValue = TooltipHook & SpellTooltipHook & { renderTooltip: () => null }

const TooltipContext = createContext<TooltipContextValue | null>(null)

// Floating tooltip for the currently-hovered item. Reads the shared Query cache
// (already warmed by handleItemEnter), so it appears as soon as data resolves.
// Position is the cursor anchor clamped to the viewport using the tooltip's
// MEASURED size (in useLayoutEffect, before paint) — so it can sit anywhere,
// not just the top half.
function HoveredTooltip({
    hoveredItem,
    anchor,
}: {
    hoveredItem: number | null
    anchor: { top: number; left: number }
}) {
    const { data } = useQuery({ ...tooltipQuery(hoveredItem ?? 0), enabled: hoveredItem != null })
    const ref = useRef<HTMLDivElement>(null)
    const [pos, setPos] = useState(anchor)

    useLayoutEffect(() => {
        const el = ref.current
        if (!el) return
        const { width, height } = el.getBoundingClientRect()
        const M = 8
        const top = Math.max(M, Math.min(anchor.top, window.innerHeight - height - M))
        const left = Math.max(M, Math.min(anchor.left, window.innerWidth - width - M))
        setPos({ top, left })
    }, [anchor, data])

    if (hoveredItem == null || !data) return null
    return (
        <div
            ref={ref}
            style={{
                position: 'fixed',
                top: pos.top,
                left: pos.left,
                zIndex: 10000,
                pointerEvents: 'none',
            }}
        >
            <ItemTooltip item={data} tooltip={data} style={{ position: 'static' }} />
        </div>
    )
}

// Floating tooltip for the currently-hovered spell. Mirrors HoveredTooltip: reads
// the shared spell-detail cache (warmed on hover) and clamps to the viewport
// using the tooltip's measured size.
function HoveredSpellTooltip({
    hoveredSpell,
    anchor,
}: {
    hoveredSpell: number | null
    anchor: { top: number; left: number }
}) {
    const { data } = useQuery({
        ...spellDetailQuery(hoveredSpell ?? 0),
        enabled: hoveredSpell != null,
    })
    const ref = useRef<HTMLDivElement>(null)
    const [pos, setPos] = useState(anchor)

    useLayoutEffect(() => {
        const el = ref.current
        if (!el) return
        const { width, height } = el.getBoundingClientRect()
        const M = 8
        const top = Math.max(M, Math.min(anchor.top, window.innerHeight - height - M))
        const left = Math.max(M, Math.min(anchor.left, window.innerWidth - width - M))
        setPos({ top, left })
    }, [anchor, data])

    if (hoveredSpell == null || !data) return null
    return (
        <div
            ref={ref}
            style={{
                position: 'fixed',
                top: pos.top,
                left: pos.left,
                zIndex: 10000,
                pointerEvents: 'none',
            }}
        >
            <SpellTooltip spell={data} style={{ position: 'static' }} />
        </div>
    )
}

export function TooltipProvider({ children }: { children: ReactNode }) {
    const hook = useItemTooltip()
    const spellHook = useSpellTooltip()
    const { setHoveredItem } = hook
    const { setHoveredSpell } = spellHook
    const value: TooltipContextValue = { ...hook, ...spellHook, renderTooltip: () => null }

    // Clear the hovered tooltips on navigation. A mouse/back-button navigation
    // unmounts the hovered element without firing mouseleave, so without this a
    // floating tooltip — rendered here at the root, which survives route changes —
    // would stay stuck until you returned and left the element normally.
    const pathname = useRouterState({ select: (s) => s.location.pathname })
    useEffect(() => {
        setHoveredItem(null)
        setHoveredSpell(null)
    }, [pathname, setHoveredItem, setHoveredSpell])

    return (
        <TooltipContext.Provider value={value}>
            {children}
            <HoveredTooltip hoveredItem={hook.hoveredItem} anchor={hook.tooltipPos} />
            <HoveredSpellTooltip
                hoveredSpell={spellHook.hoveredSpell}
                anchor={spellHook.spellPos}
            />
        </TooltipContext.Provider>
    )
}

/** Access the shared tooltip hook. Must be rendered under <TooltipProvider>. */
export function useTooltipCtx(): TooltipContextValue {
    const ctx = useContext(TooltipContext)
    if (!ctx) throw new Error('useTooltipCtx must be used within a TooltipProvider')
    return ctx
}

export default TooltipProvider
