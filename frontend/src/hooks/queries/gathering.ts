import { useQuery } from '@tanstack/react-query'
import { queryKeys } from './keys'
import { GetGatheringProfessions, GetGatheringNodes } from '../../../wailsjs/go/main/App'

/** Gathering skills (Mining, Herbalism, ...) that have plottable nodes. */
export const useGatheringProfessions = () =>
    useQuery({
        queryKey: queryKeys.gatheringProfessions,
        queryFn: GetGatheringProfessions,
        staleTime: Infinity,
    })

/** Every node of one gathering skill with skill reqs and all spawn points. */
export const useGatheringNodes = (lockType?: number | null) =>
    useQuery({
        queryKey: queryKeys.gatheringNodes(lockType),
        queryFn: () => GetGatheringNodes(lockType as number),
        enabled: !!lockType,
        staleTime: Infinity,
    })
