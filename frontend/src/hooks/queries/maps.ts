import { useQuery } from '@tanstack/react-query'
import { queryKeys } from './keys'
import { main } from '../../../wailsjs/go/models'

// Wails bindings (codebase convention for app methods).
const GetFlightContinents = () =>
    window?.go?.main?.App?.GetFlightContinents
        ? window.go.main.App.GetFlightContinents()
        : Promise.resolve([])
const GetFlightData = (mapId: number): Promise<main.FlightData | null> =>
    window?.go?.main?.App?.GetFlightData
        ? window.go.main.App.GetFlightData(mapId)
        : Promise.resolve(null)
const GetWorldData = (): Promise<main.WorldData | null> =>
    window?.go?.main?.App?.GetWorldData ? window.go.main.App.GetWorldData() : Promise.resolve(null)
const GetZoneData = (mapId: number, zone: string) =>
    window?.go?.main?.App?.GetZoneData
        ? window.go.main.App.GetZoneData(mapId, zone)
        : Promise.resolve(null)

export const useFlightContinents = () =>
    useQuery({
        queryKey: queryKeys.flightContinents,
        queryFn: GetFlightContinents,
        staleTime: Infinity,
    })

// One query per view: the world overview ('world') or a continent's flight data
// (different shapes, normalized by the caller).
export const useFlightMap = (view: 'world' | number) =>
    useQuery<main.WorldData | main.FlightData | null>({
        queryKey: queryKeys.flightMap(view),
        queryFn: () => (view === 'world' ? GetWorldData() : GetFlightData(view)),
        staleTime: Infinity,
    })

export const useFlightZone = (mapId: number, zoneKey: string) =>
    useQuery({
        queryKey: queryKeys.flightZone(mapId, zoneKey),
        queryFn: () => GetZoneData(mapId, zoneKey),
    })
