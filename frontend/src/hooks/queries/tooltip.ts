import { queryKeys } from './keys'

/** Fetch one item's tooltip payload from the backend. */
const fetchTooltip = (itemId: number) =>
    window?.go?.main?.App?.GetTooltipData ? window.go.main.App.GetTooltipData(itemId) : Promise.resolve(null)

/**
 * Shared Query options for an item tooltip. Spread into a useQuery call so every
 * reader — the global tooltip layer, the item detail block, favorite cards —
 * keys the same cache entry and reuses an already-fetched tooltip.
 */
export const tooltipQuery = (itemId: number) => ({
    queryKey: queryKeys.tooltip(itemId),
    queryFn: () => fetchTooltip(itemId),
})
