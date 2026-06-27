import { useQuery } from '@tanstack/react-query'
import { queryKeys } from './keys'
import { GetZones, GetZoneDetail } from '../../utils/databaseApi'

export const useZones = () =>
    useQuery({ queryKey: queryKeys.zones, queryFn: GetZones, staleTime: Infinity })

export const useZoneDetail = (entry: number) =>
    useQuery({
        queryKey: queryKeys.zoneDetail(entry),
        queryFn: () => GetZoneDetail(entry),
        enabled: entry != null,
    })
