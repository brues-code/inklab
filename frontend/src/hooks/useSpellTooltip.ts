import { useState, useCallback, type MouseEvent } from 'react'
import { queryClient } from '../queryClient'
import { spellDetailQuery } from './queries/spells'

/**
 * Spell tooltip behavior — the spell analogue of useItemTooltip. Owns only the
 * transient hover/position state; the spell data itself lives in TanStack Query
 * (keyed via spellDetailQuery, shared with the spell detail view).
 */
export function useSpellTooltip() {
    const [hoveredSpell, setHoveredSpell] = useState<number | null>(null)
    const [spellPos, setSpellPos] = useState({ top: 0, left: 0 })

    // Anchor below-right of the cursor, never over the hovered row; the renderer
    // clamps to the viewport using the tooltip's measured size.
    const handleSpellMove = useCallback((e: MouseEvent<HTMLElement>, spellId: number) => {
        const rect = e.currentTarget.getBoundingClientRect()
        setSpellPos({ left: e.clientX + 15, top: Math.max(e.clientY + 15, rect.bottom + 5) })
        setHoveredSpell(spellId)
    }, [])

    // Preload the spell into the cache on enter (idempotent; dedupes).
    const handleSpellEnter = useCallback((spellId: number) => {
        queryClient.ensureQueryData(spellDetailQuery(spellId)).catch(() => {})
    }, [])

    const handleSpellLeave = useCallback(() => setHoveredSpell(null), [])

    // Event handlers for a hoverable spell element.
    const getSpellHandlers = useCallback(
        (spellId: number) => ({
            onMouseEnter: () => handleSpellEnter(spellId),
            onMouseMove: (e: MouseEvent<HTMLElement>) => handleSpellMove(e, spellId),
            onMouseLeave: handleSpellLeave,
        }),
        [handleSpellEnter, handleSpellMove, handleSpellLeave],
    )

    return { hoveredSpell, setHoveredSpell, spellPos, getSpellHandlers }
}

export type SpellTooltipHook = ReturnType<typeof useSpellTooltip>

export default useSpellTooltip
