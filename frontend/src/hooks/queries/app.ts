import { useQuery } from '@tanstack/react-query'
import { queryKeys } from './keys'
import { CheckForUpdate } from '../../services/api'

// App-chrome one-shot checks, cached for the session.

const GetDataStatus = () =>
    window?.go?.main?.App?.GetDataStatus
        ? window.go.main.App.GetDataStatus()
        : Promise.resolve(null)

const GetSyncStats = () =>
    window?.go?.main?.App?.GetSyncStats ? window.go.main.App.GetSyncStats() : Promise.resolve(null)

const WhatsNew = async () => {
    const app = window?.go?.main?.App
    if (!app?.WhatsNew) return { error: 'Binding not found (dev build?)' }
    try {
        return await app.WhatsNew()
    } catch (e) {
        return { error: String(e) }
    }
}

export const useUpdateCheck = () =>
    useQuery({ queryKey: queryKeys.updateCheck, queryFn: CheckForUpdate, staleTime: Infinity })

export const useDataStatus = () =>
    useQuery({ queryKey: queryKeys.dataStatus, queryFn: GetDataStatus, staleTime: Infinity })

// Refetched after a sync via explicit invalidation, so it can stay fresh forever.
export const useSyncStats = () =>
    useQuery({ queryKey: queryKeys.syncStats, queryFn: GetSyncStats, staleTime: Infinity })

// Server-derived DB-vs-baseline diff. Kept in the query cache so the result
// survives navigating away from Tools and back. Fetched on demand rather than on
// mount: the hook only subscribes (enabled: false); trigger a (re)check with
// queryClient.fetchQuery(whatsNewQuery) — pass staleTime: 0 to force a refresh.
export const whatsNewQuery = {
    queryKey: queryKeys.whatsNew,
    queryFn: WhatsNew,
    staleTime: Infinity,
    gcTime: Infinity,
}

export const useWhatsNew = () => useQuery({ ...whatsNewQuery, enabled: false })
