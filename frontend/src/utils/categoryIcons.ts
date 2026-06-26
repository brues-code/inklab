/**
 * Mapping of item categories (Classes, SubClasses, Slots) to icon filenames.
 * Filenames should exclude extension (.png/.jpg) assuming png for interface icons or jpg for item icons.
 * Based on standard WoW interface icons.
 */
export const categoryIcons: Record<string, string> = {
    // Classes
    "Consumable": "inv_potion_07",
    "Container": "inv_misc_bag_13",
    "Weapon": "inv_sword_27",
    "Gem": "inv_gizmo_bronzeframework_01", // Fallback
    "Armor": "inv_chest_plate16",
    "Reagent": "inv_gizmo_bronzeframework_01",
    "Projectile": "inv_ammo_bullet_02",
    "Trade Goods": "inv_gizmo_bronzeframework_01",
    "Generic": "inv_misc_questionmark", // Not present, may fail? Or use misc
    "Recipe": "inv_scroll_04",
    "Money": "inv_misc_coin_01", // Not present
    "Quiver": "inv_misc_quiver_08", 
    "Quest": "inv_qiraj_jewelblessed",
    "Key": "inv_misc_key_04",
    "Permanent": "inv_gizmo_bronzeframework_01", // Enchant?
    "Junk": "inv_misc_bone_orcskull_01", 
    "Miscellaneous": "inv_misc_bone_orcskull_01",
}

export const getCategoryIcon = (name?: string | null): string | null => {
    if (!name) return null;
    const icon = categoryIcons[name] || categoryIcons[name.split(' (')[0]] // Handle "Mace (2H)" if needed
    // Assuming UI icons are in /items/ (not items/icons/) and are PNGs. 
    // Adjust path based on actual file location.
    if (icon) {
        return `/items/${icon}.png`
    }
    return null
}
