import { QueryClient } from '@tanstack/react-query'

/**
 * App-wide TanStack Query client. Defaults tuned for a Wails desktop app over
 * local data: no refetch-on-focus churn, modest retry, and a default staleTime
 * so repeated reads (lists, tooltips, talent trees) serve from cache. Queries
 * over data that's static for a session can override with staleTime: Infinity.
 */
export const queryClient = new QueryClient({
    defaultOptions: {
        queries: {
            staleTime: 5 * 60 * 1000,
            refetchOnWindowFocus: false,
            retry: 1,
        },
    },
})

export default queryClient
