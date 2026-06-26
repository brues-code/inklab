import { useQuery } from '@tanstack/react-query'
import { queryKeys } from './keys'
import { GetItemSets, GetItemSetDetail } from '../../utils/databaseApi'

export const useItemSets = () =>
    useQuery({ queryKey: queryKeys.itemSets, queryFn: GetItemSets, staleTime: Infinity })

export const useItemSetDetail = (itemsetId: number | undefined, enabled: boolean) =>
    useQuery({
        queryKey: queryKeys.itemSetDetail(itemsetId),
        queryFn: () => GetItemSetDetail(itemsetId!),
        enabled,
    })
