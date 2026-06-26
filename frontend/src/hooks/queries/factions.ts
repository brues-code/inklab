import { useQuery } from '@tanstack/react-query'
import { queryKeys } from './keys'
import { GetFactions } from '../../utils/databaseApi'
import { GetFactionDetail } from '../../../wailsjs/go/main/App'

export const useFactions = () =>
    useQuery({ queryKey: queryKeys.factions, queryFn: GetFactions, staleTime: Infinity })

export const useFactionDetail = (id: number) =>
    useQuery({ queryKey: queryKeys.factionDetail(id), queryFn: () => GetFactionDetail(id), enabled: id != null })
