import { useQuery } from '@tanstack/react-query'
import { queryKeys } from './keys'

// Wails bindings (codebase convention for app methods).
const GetFlightContinents = () =>
    window?.go?.main?.App?.GetFlightContinents ? window.go.main.App.GetFlightContinents() : Promise.resolve([])
const GetFlightData = (mapId: any) =>
    window?.go?.main?.App?.GetFlightData ? window.go.main.App.GetFlightData(mapId) : Promise.resolve(null)
const GetWorldData = () =>
    window?.go?.main?.App?.GetWorldData ? window.go.main.App.GetWorldData() : Promise.resolve(null)
const GetZoneData = (mapId: any, zone: any) =>
    window?.go?.main?.App?.GetZoneData ? window.go.main.App.GetZoneData(mapId, zone) : Promise.resolve(null)

export const useFlightContinents = () =>
    useQuery({ queryKey: queryKeys.flightContinents, queryFn: GetFlightContinents, staleTime: Infinity })

// One query per view: the world overview ('world') or a continent's flight data.
// data is `any` — the two queryFns return different shapes (WorldData vs
// FlightData), normalized by the caller.
export const useFlightMap = (view: any) =>
    useQuery<any>({
        queryKey: queryKeys.flightMap(view),
        queryFn: () => (view === 'world' ? GetWorldData() : GetFlightData(view)),
        staleTime: Infinity,
    })

export const useFlightZone = (mapId: any, zoneKey: any) =>
    useQuery({ queryKey: queryKeys.flightZone(mapId, zoneKey), queryFn: () => GetZoneData(mapId, zoneKey) })
