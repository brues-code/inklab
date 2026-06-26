import { useQuery } from '@tanstack/react-query'
import { queryKeys } from './keys'
import { GetCategories, GetInstances, GetTables } from '../../../wailsjs/go/main/App'

const GetLoot = (category: string, instance: string, boss: string) =>
    window?.go?.main?.App?.GetLoot
        ? window.go.main.App.GetLoot(category, instance, boss)
        : Promise.resolve({ bossName: boss, items: [] })

// AtlasLoot cascade: categories → instances → tables → loot. The first three are
// static for a session; loot keys by the full category/module/table path.

export const useAtlasCategories = () =>
    useQuery({ queryKey: queryKeys.atlasCategories, queryFn: GetCategories, staleTime: Infinity })

export const useAtlasModules = (category: string, enabled: boolean) =>
    useQuery({ queryKey: queryKeys.atlasModules(category), queryFn: () => GetInstances(category), enabled, staleTime: Infinity })

export const useAtlasTables = (category: string, module: string, enabled: boolean) =>
    useQuery({ queryKey: queryKeys.atlasTables(category, module), queryFn: () => GetTables(category, module), enabled, staleTime: Infinity })

export const useAtlasLoot = (category: string, module: string, table: string, enabled: boolean) =>
    useQuery<any>({ queryKey: queryKeys.atlasLoot(category, module, table), queryFn: () => GetLoot(category, module, table), enabled })
