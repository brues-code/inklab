import { useQuery } from '@tanstack/react-query'
import { queryKeys } from './keys'
import { GetProfessions, GetProfessionRecipes } from '../../../wailsjs/go/main/App'

/** Professions (primary + crafting secondaries) with recipe counts. */
export const useProfessions = () =>
    useQuery({ queryKey: queryKeys.professions, queryFn: GetProfessions, staleTime: Infinity })

/** Every recipe of one profession: thresholds, reagents, crafted item, sources. */
export const useProfessionRecipes = (skillId?: number | null) =>
    useQuery({
        queryKey: queryKeys.professionRecipes(skillId),
        queryFn: () => GetProfessionRecipes(skillId as number),
        enabled: !!skillId,
        staleTime: Infinity,
    })
