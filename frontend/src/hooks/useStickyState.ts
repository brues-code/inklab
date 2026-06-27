import { useState } from 'react'

/**
 * Like useState, but the value is cached in a module-level store keyed by `key`,
 * so it survives the component unmounting and remounting (e.g. navigating to
 * another route and pressing Back). Scoped to the JS session — a full app
 * reload starts fresh.
 *
 * Use this for view state that should feel persistent across navigation, like a
 * tab's current class/subclass/slot selection and filter strings.
 */
const store = new Map<string, unknown>()

type SetState<T> = (value: T | ((prev: T) => T)) => void

export function useStickyState<T>(key: string, initial: T): [T, SetState<T>] {
    const [state, setState] = useState<T>(() => (store.has(key) ? (store.get(key) as T) : initial))

    const set: SetState<T> = value => {
        setState(prev => {
            const next = typeof value === 'function' ? (value as (prev: T) => T)(prev) : value
            store.set(key, next)
            return next
        })
    }

    return [state, set]
}

export default useStickyState
