import { useQuery } from '@tanstack/react-query'
import { queryKeys } from './keys'
import { CheckForUpdate } from '../../services/api'

// App-chrome one-shot checks, cached for the session.

const GetDataStatus = () =>
    window?.go?.main?.App?.GetDataStatus ? window.go.main.App.GetDataStatus() : Promise.resolve(null)

export const useUpdateCheck = () =>
    useQuery({ queryKey: queryKeys.updateCheck, queryFn: CheckForUpdate, staleTime: Infinity })

export const useDataStatus = () =>
    useQuery({ queryKey: queryKeys.dataStatus, queryFn: GetDataStatus, staleTime: Infinity })
