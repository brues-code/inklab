import { useEffect, useState } from 'react'

/**
 * Returns `value` after it has stopped changing for `delay` ms. Debouncing is
 * inherently time-based, so the timer lives in an effect here — this is the one
 * place that's the right tool, and keeps it out of the consuming component.
 */
export function useDebouncedValue<T>(value: T, delay = 250): T {
    const [debounced, setDebounced] = useState(value)
    useEffect(() => {
        const t = setTimeout(() => setDebounced(value), delay)
        return () => clearTimeout(t)
    }, [value, delay])
    return debounced
}

export default useDebouncedValue
