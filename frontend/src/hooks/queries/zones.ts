import { useQuery } from '@tanstack/react-query'
import { queryKeys } from './keys'
import { GetZones, GetZoneDetail, GetZoneNames } from '../../utils/databaseApi'

export const useZones = () =>
    useQuery({ queryKey: queryKeys.zones, queryFn: GetZones, staleTime: Infinity })

// Official zone display names keyed by normalized match key, loaded once and
// cached for the session — backs the shared <ZoneName> renderer.
export const useZoneNames = () =>
    useQuery({ queryKey: queryKeys.zoneNames, queryFn: GetZoneNames, staleTime: Infinity })

export const useZoneDetail = (entry: number) =>
    useQuery({
        queryKey: queryKeys.zoneDetail(entry),
        queryFn: () => GetZoneDetail(entry),
        enabled: entry != null,
    })
