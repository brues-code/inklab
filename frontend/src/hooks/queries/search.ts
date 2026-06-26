import { useQuery } from '@tanstack/react-query'
import { queryKeys } from './keys'
import { AdvancedSearch } from '../../../wailsjs/go/main/App'

// Global header search. Keyed by the (debounced) query string; Query dedupes and
// drops out-of-order responses for superseded keys, so the caller needs no
// manual request-id race guard. Gated to 2+ chars.
export const useGlobalSearch = (query: string) =>
    useQuery({
        queryKey: queryKeys.search(query),
        queryFn: () => AdvancedSearch({ query, minLevel: 0, maxLevel: 0, quality: [], limit: 50, offset: 0 }),
        enabled: query.length >= 2,
    })
