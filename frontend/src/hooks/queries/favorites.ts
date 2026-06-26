import { useQuery } from '@tanstack/react-query'
import { queryKeys } from './keys'
import { GetAllFavorites, GetFavoriteCategories } from '../../services/api'

export const useFavorites = () =>
    useQuery({ queryKey: queryKeys.favorites, queryFn: GetAllFavorites })

export const useFavoriteCategories = () =>
    useQuery({ queryKey: queryKeys.favoriteCategories, queryFn: GetFavoriteCategories })
