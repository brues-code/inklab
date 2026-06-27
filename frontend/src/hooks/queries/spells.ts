import { useQuery } from '@tanstack/react-query'
import { queryKeys } from './keys'
import {
    GetSpellSkillCategories,
    GetSpellSkillsByCategory,
    GetSpellsBySkill,
    GetSpellClasses,
    GetSpellSkillsByClass,
} from '../../utils/databaseApi'
import { GetSpellDetail } from '../../services/api'

// Spell browse + detail queries. Categories and the Class-Skills class list are
// static for a session (staleTime: Infinity); the rest key by selection and are
// enabled-gated by the caller so cascading panes only fetch when their parent
// selection exists.

export const useSpellCategories = () =>
    useQuery({
        queryKey: queryKeys.spellCategories,
        queryFn: GetSpellSkillCategories,
        staleTime: Infinity,
    })

export const useSpellClasses = (enabled: boolean) =>
    useQuery({
        queryKey: queryKeys.spellClasses,
        queryFn: GetSpellClasses,
        enabled,
        staleTime: Infinity,
    })

export const useSpellSkillsByCategory = (categoryId: number | undefined, enabled: boolean) =>
    useQuery({
        queryKey: queryKeys.spellSkillsByCategory(categoryId),
        queryFn: () => GetSpellSkillsByCategory(categoryId!),
        enabled,
    })

export const useSpellSkillsByClass = (classId: number | undefined, enabled: boolean) =>
    useQuery({
        queryKey: queryKeys.spellSkillsByClass(classId),
        queryFn: () => GetSpellSkillsByClass(classId!),
        enabled,
    })

export const useSpellsBySkill = (skillId: number | undefined, enabled: boolean) =>
    useQuery({
        queryKey: queryKeys.spellsBySkill(skillId),
        queryFn: () => GetSpellsBySkill(skillId!, ''),
        enabled,
    })

// Shared Query options for a spell's full detail. Spread into useQuery so every
// reader — the detail view and the global spell tooltip layer — keys the same
// cache entry and reuses an already-fetched spell.
export const spellDetailQuery = (entry: number) => ({
    queryKey: queryKeys.spellDetail(entry),
    queryFn: () => GetSpellDetail(Number(entry)),
})

export const useSpellDetail = (entry: number) =>
    useQuery({ ...spellDetailQuery(entry), enabled: entry != null })
