import { useQuery } from '@tanstack/react-query'
import { queryKeys } from './keys'
import { GetTalentClasses } from '../../utils/databaseApi'

const GetTalentTrees = (cls: string) =>
    window?.go?.main?.App?.GetTalentTrees ? window.go.main.App.GetTalentTrees(cls) : Promise.resolve(null)

// Class list is shared with the Sets-tab class filter; both static for a session.

export const useTalentClasses = () =>
    useQuery({ queryKey: queryKeys.talentClasses, queryFn: GetTalentClasses, staleTime: Infinity })

// Options factory for a class's trees, reused by useTalentTrees and by the
// import flow's queryClient.ensureQueryData (fetch a target class on demand).
export const talentTreesQuery = (cls: string | null) => ({
    queryKey: queryKeys.talentTrees(cls),
    queryFn: () => GetTalentTrees(cls!),
    staleTime: Infinity,
})

export const useTalentTrees = (cls: string | null, enabled: boolean) =>
    useQuery({ ...talentTreesQuery(cls), enabled })
