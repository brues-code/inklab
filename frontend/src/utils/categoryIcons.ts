/**
 * Mapping of item categories (Classes, SubClasses, Slots) to icon filenames.
 * Filenames should exclude extension (.png/.jpg) assuming png for interface icons or jpg for item icons.
 * Based on standard WoW interface icons.
 */
export const categoryIcons: Record<string, string> = {
    // Classes
    Consumable: 'inv_potion_07',
    Container: 'inv_misc_bag_13',
    Weapon: 'inv_sword_27',
    Gem: 'inv_gizmo_bronzeframework_01', // Fallback
    Armor: 'inv_chest_plate16',
    Reagent: 'inv_gizmo_bronzeframework_01',
    Projectile: 'inv_ammo_bullet_02',
    'Trade Goods': 'inv_gizmo_bronzeframework_01',
    Generic: 'inv_misc_questionmark', // Not present, may fail? Or use misc
    Recipe: 'inv_scroll_04',
    Money: 'inv_misc_coin_01', // Not present
    Quiver: 'inv_misc_quiver_08',
    Quest: 'inv_qiraj_jewelblessed',
    Key: 'inv_misc_key_04',
    Permanent: 'inv_gizmo_bronzeframework_01', // Enchant?
    Junk: 'inv_misc_bone_orcskull_01',
    Miscellaneous: 'inv_misc_bone_orcskull_01',
}

/**
 * Resolve a category (Class/SubClass/Slot) name to its icon NAME (no path or
 * extension). Callers pass this to useIcon(), which loads the icon from the
 * client icon set (data/icons) and falls back to the questionmark placeholder
 * when it's absent — e.g. before a client import. Returns null for unmapped
 * names.
 */
export const getCategoryIcon = (name?: string | null): string | null => {
    if (!name) return null
    // Handle suffixed names like "Mace (2H)".
    return categoryIcons[name] || categoryIcons[name.split(' (')[0]] || null
}
