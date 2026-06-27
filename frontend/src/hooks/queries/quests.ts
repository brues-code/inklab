import { useQuery } from '@tanstack/react-query'
import { queryKeys } from './keys'
import {
    GetQuestCategoryGroups,
    GetQuestCategoriesByGroup,
    GetQuestsByEnhancedCategory,
} from '../../utils/databaseApi'
import { GetQuestDetail } from '../../services/api'

export const useQuestGroups = () =>
    useQuery({
        queryKey: queryKeys.questGroups,
        queryFn: GetQuestCategoryGroups,
        staleTime: Infinity,
    })

export const useQuestCategories = (groupId: number | undefined, enabled: boolean) =>
    useQuery({
        queryKey: queryKeys.questCategories(groupId),
        queryFn: () => GetQuestCategoriesByGroup(groupId!),
        enabled,
    })

export const useQuestsByCategory = (categoryId: number | undefined, enabled: boolean) =>
    useQuery({
        queryKey: queryKeys.questsByCategory(categoryId),
        queryFn: () => GetQuestsByEnhancedCategory(categoryId!, ''),
        enabled,
    })

export const useQuestDetail = (entry: number) =>
    useQuery({
        queryKey: queryKeys.questDetail(entry),
        queryFn: () => GetQuestDetail(entry),
        enabled: entry != null,
    })
