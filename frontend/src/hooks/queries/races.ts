import { useQuery } from '@tanstack/react-query'
import { queryKeys } from './keys'
import { GetRaces } from '../../../wailsjs/go/main/App'

// Playable races (with flavor text, available classes and racial spells), all
// client-derived. Static for a session.
export const useRaces = () =>
    useQuery({ queryKey: queryKeys.races, queryFn: GetRaces, staleTime: Infinity })
