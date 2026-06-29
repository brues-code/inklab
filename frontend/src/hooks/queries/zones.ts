import { useQuery } from '@tanstack/react-query'
import { queryKeys } from './keys'
import { GetZones, GetZoneDetail, GetZoneNames, GetZoneLoot } from '../../utils/databaseApi'

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

// Zone loot is the heaviest zone query, so it's loaded lazily — only when the
// caller enables it (i.e. the Loot tab is opened).
export const useZoneLoot = (entry: number, enabled: boolean) =>
    useQuery({
        queryKey: queryKeys.zoneLoot(entry),
        queryFn: () => GetZoneLoot(entry),
        enabled: enabled && entry != null,
        staleTime: Infinity,
    })
