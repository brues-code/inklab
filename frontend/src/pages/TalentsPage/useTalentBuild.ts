import { useState, useCallback } from 'react'

// Session memory: the working build per class (class token -> allocation order,
// one talentId per point spent, in the order taken). Kept in module scope so it
// survives navigating away from and back to Talents within a session; cleared
// on full app reload — a future "save builds" feature can persist these.
const buildsByClass = new Map<string, number[]>()
let lastClassToken: string | null = null

/** The last class viewed this session, for the /talents landing redirect. */
export const getLastTalentClass = () => lastClassToken

/**
 * Working talent build for `selected`, persisted per class in session memory.
 *
 * Switching class restores that class's stored build — done during render via
 * the "adjust state during render" pattern, so the page needs no effect.
 * `applyOrder` updates the current class's build; `setBuildFor` writes any
 * class's build (used by cross-class import before navigating to it).
 */
export function useTalentBuild(selected: string | null) {
    const [order, setOrder] = useState<number[]>([])
    const [orderClass, setOrderClass] = useState<string | null>(null)

    if (selected !== orderClass) {
        setOrderClass(selected)
        setOrder(selected ? buildsByClass.get(selected) || [] : [])
        if (selected) lastClassToken = selected
    }

    const applyOrder = useCallback(
        (next: number[]) => {
            setOrder(next)
            if (selected) buildsByClass.set(selected, next)
        },
        [selected],
    )

    const setBuildFor = useCallback(
        (classKey: string, next: number[]) => {
            buildsByClass.set(classKey, next)
            if (classKey === selected) setOrder(next)
        },
        [selected],
    )

    return { order, applyOrder, setBuildFor }
}
