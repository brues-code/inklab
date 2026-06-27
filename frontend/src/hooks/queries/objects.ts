import { useQuery } from '@tanstack/react-query'
import { queryKeys } from './keys'
import { GetObjectTypes, GetObjectsByType } from '../../utils/databaseApi'
import { GetObjectDetail } from '../../../wailsjs/go/main/App'

export const useObjectTypes = () =>
    useQuery({ queryKey: queryKeys.objectTypes, queryFn: GetObjectTypes, staleTime: Infinity })

export const useObjectsByType = (typeId: number | undefined, enabled: boolean) =>
    useQuery({
        queryKey: queryKeys.objectsByType(typeId),
        queryFn: () => GetObjectsByType(typeId!, ''),
        enabled,
    })

export const useObjectDetail = (entry: number) =>
    useQuery({
        queryKey: queryKeys.objectDetail(entry),
        queryFn: () => GetObjectDetail(entry),
        enabled: entry != null,
    })
