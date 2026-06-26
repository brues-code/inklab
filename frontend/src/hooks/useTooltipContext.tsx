import { createContext, useContext, type ReactNode } from 'react'
import { useItemTooltip, type TooltipHook } from './useItemTooltip'
import { ItemTooltip } from '../components/ui'

/**
 * App-wide item tooltip context. One hook instance lives at the router root so
 * every page (Database tabs, entity detail views, AtlasLoot) shares a single
 * tooltip cache and the single floating tooltip layer rendered here. This
 * replaces the per-page `useItemTooltip()` + duplicated `<ItemTooltip>` blocks
 * that previously lived in DatabasePage and AtlasLootPage.
 *
 * The context value carries `renderTooltip: () => null` so it satisfies the
 * `tooltipHook` prop shape the tab/detail components expect.
 */
type TooltipContextValue = TooltipHook & { renderTooltip: () => null }

const TooltipContext = createContext<TooltipContextValue | null>(null)

export function TooltipProvider({ children }: { children: ReactNode }) {
    const hook = useItemTooltip()
    const value: TooltipContextValue = { ...hook, renderTooltip: () => null }
    const { hoveredItem, tooltipCache, getTooltipStyle } = hook

    return (
        <TooltipContext.Provider value={value}>
            {children}
            {/* Single global tooltip layer for the whole app */}
            {hoveredItem != null && tooltipCache[hoveredItem] && (
                <ItemTooltip
                    item={tooltipCache[hoveredItem]}
                    tooltip={tooltipCache[hoveredItem]}
                    style={getTooltipStyle()}
                />
            )}
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
