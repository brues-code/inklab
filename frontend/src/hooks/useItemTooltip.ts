import { useState, useCallback, type MouseEvent } from 'react'
import { queryClient } from '../queryClient'
import { queryKeys } from './queries/keys'
import { tooltipQuery } from './queries/tooltip'

/**
 * Item tooltip behavior: mouse-follow positioning + which item is hovered, plus
 * thin wrappers over the Query cache for the tooltip data itself. The data lives
 * in TanStack Query (keyed via queryKeys.tooltip); this hook owns only the
 * transient hover/position UI state.
 */
export function useItemTooltip() {
    const [hoveredItem, setHoveredItem] = useState<number | null>(null)
    const [tooltipPos, setTooltipPos] = useState({ top: 0, left: 0 })

    // Preload (and optionally force-refresh) an item's tooltip into the cache.
    // Idempotent: repeated calls dedupe / serve cached data. Returns the data.
    const loadTooltipData = useCallback(async (itemId: number, forceReload = false) => {
        if (forceReload) await queryClient.invalidateQueries({ queryKey: queryKeys.tooltip(itemId) })
        try {
            return await queryClient.ensureQueryData(tooltipQuery(itemId))
        } catch (err) {
            console.error('Failed to load tooltip:', err)
            return null
        }
    }, [])

    // Invalidate a cached tooltip; any active reader refetches automatically.
    const invalidateTooltip = useCallback((itemId: number) => {
        queryClient.invalidateQueries({ queryKey: queryKeys.tooltip(itemId) })
    }, [])

    // Anchor the tooltip below-right of the cursor (and never over the hovered
    // row). Keeping-on-screen is left to the renderer, which measures the
    // tooltip's REAL size — so a short tooltip near the page bottom isn't pinned
    // into the top by a guessed height.
    const handleMouseMove = useCallback((e: MouseEvent<HTMLElement>, itemId: number) => {
        const itemRect = e.currentTarget.getBoundingClientRect()
        setTooltipPos({
            left: e.clientX + 15,
            top: Math.max(e.clientY + 15, itemRect.bottom + 5),
        })
        setHoveredItem(itemId)
    }, [])

    // Handle item enter - preload tooltip data
    const handleItemEnter = useCallback((itemId: number) => {
        loadTooltipData(itemId)
    }, [loadTooltipData])

    // Handle item leave - hide tooltip
    const handleItemLeave = useCallback(() => {
        setHoveredItem(null)
    }, [])

    // Get event handlers for an item element
    const getItemHandlers = useCallback((itemId: number) => ({
        onMouseEnter: () => handleItemEnter(itemId),
        onMouseMove: (e: MouseEvent<HTMLElement>) => handleMouseMove(e, itemId),
        onMouseLeave: handleItemLeave,
    }), [handleItemEnter, handleMouseMove, handleItemLeave])

    return {
        hoveredItem,
        setHoveredItem,
        tooltipPos,
        loadTooltipData,
        invalidateTooltip,
        handleMouseMove,
        handleItemEnter,
        handleItemLeave,
        getItemHandlers,
    }
}

/** Return shape of useItemTooltip — pass as the `tooltipHook` prop. */
export type TooltipHook = ReturnType<typeof useItemTooltip>

export default useItemTooltip
