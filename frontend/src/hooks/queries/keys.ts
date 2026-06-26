// Centralized TanStack Query keys.
//
// Every queryKey in the app is defined here so cache reads, invalidation, and
// sharing stay consistent and discoverable. Static lists are plain tuples;
// parameterized queries are factory functions. The matching query hooks live in
// the per-domain files in this folder (e.g. ./spells).
export const queryKeys = {
    // Spells
    spellCategories: ['spellCategories'] as const,
    spellClasses: ['spellClasses'] as const,
    spellSkillsByCategory: (categoryId: unknown) => ['spellSkillsByCategory', categoryId] as const,
    spellSkillsByClass: (classId: unknown) => ['spellSkillsByClass', classId] as const,
    spellsBySkill: (skillId: unknown) => ['spellsBySkill', skillId] as const,
    spellDetail: (entry: unknown) => ['spellDetail', entry] as const,
}
