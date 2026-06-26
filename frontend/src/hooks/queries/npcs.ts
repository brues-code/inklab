import { useQuery, useInfiniteQuery } from '@tanstack/react-query'
import { queryKeys } from './keys'
import {
    GetCreatureTypes,
    GetBeastFamilies,
    BrowseCreaturesByTypePaged,
    BrowseCreaturesByFamilyPaged,
} from '../../utils/databaseApi'
import { GetNpcFullDetails } from '../../services/api'

export const NPC_PAGE_SIZE = 100

export const useCreatureTypes = () =>
    useQuery({ queryKey: queryKeys.creatureTypes, queryFn: GetCreatureTypes, staleTime: Infinity })

export const useBeastFamilies = (enabled: boolean) =>
    useQuery({ queryKey: queryKeys.beastFamilies, queryFn: GetBeastFamilies, enabled, staleTime: Infinity })

// Paginated creature browse for the active selection (a beast family when one is
// picked, else the type), keyed by that selection.
export const useCreatures = (selectedType: any, selectedFamily: any, isBeast: boolean) =>
    useInfiniteQuery({
        queryKey: queryKeys.creatures(
            isBeast && selectedFamily ? `family:${selectedFamily.family}` : `type:${selectedType?.type}`
        ),
        queryFn: ({ pageParam }) =>
            isBeast && selectedFamily
                ? BrowseCreaturesByFamilyPaged(selectedFamily.family, '', NPC_PAGE_SIZE, pageParam)
                : BrowseCreaturesByTypePaged(selectedType.type, '', NPC_PAGE_SIZE, pageParam),
        enabled: selectedType != null,
        initialPageParam: 0,
        getNextPageParam: (lastPage: any, allPages: any[]) =>
            lastPage?.hasMore ? allPages.length * NPC_PAGE_SIZE : undefined,
    })

export const useNpcDetail = (entry: number) =>
    useQuery({ queryKey: queryKeys.npcDetail(entry), queryFn: () => GetNpcFullDetails(entry), enabled: entry != null })
