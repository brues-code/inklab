// Spell school index (mangos spell_template.school) -> display name.
export const SPELL_SCHOOLS: Record<number, string> = {
    0: 'Physical',
    1: 'Holy',
    2: 'Fire',
    3: 'Nature',
    4: 'Frost',
    5: 'Shadow',
    6: 'Arcane',
}

// School text colors, from the client's damageTypeFontColors (CreateSpellColor
// RGB floats, converted to hex). Physical (school 0) has no client color, so it
// falls back to the default text color.
export const SPELL_SCHOOL_COLORS: Record<number, string> = {
    1: '#f6f99e', // Holy
    2: '#ee7d80', // Fire
    3: '#a1db65', // Nature
    4: '#76caed', // Frost
    5: '#5b4a98', // Shadow
    6: '#f4a8e4', // Arcane
}

export const getSchoolName = (school: number): string => SPELL_SCHOOLS[school] || 'Unknown'
export const getSchoolColor = (school: number): string | undefined => SPELL_SCHOOL_COLORS[school]

// Debuff dispel-type colors — the client's DebuffTypeColor (FrameXML), used for
// debuff borders. Keyed by the dispel-type name (from SpellDispelType.dbc).
export const DISPEL_COLORS: Record<string, string> = {
    Magic: '#3399ff',
    Curse: '#9900ff',
    Disease: '#996600',
    Poison: '#009900',
}
export const getDispelColor = (name?: string | null): string | undefined =>
    name ? DISPEL_COLORS[name] : undefined

export const getQualityColor = (quality: number): string => {
    const colors: Record<number, string> = {
        0: '#9d9d9d', // Poor
        1: '#ffffff', // Common
        2: '#1eff00', // Uncommon
        3: '#0070dd', // Rare
        4: '#a335ee', // Epic
        5: '#ff8000', // Legendary
        6: '#e6cc80'  // Artifact
    }
    return colors[quality] || '#ffffff'
}

export const QUESTION_MARK_ICON = '/local-icons/inv_misc_questionmark.jpg'

export interface Money { g: number; s: number; c: number }

export const formatMoney = (money?: number): Money => {
    if (!money) return { g: 0, s: 0, c: 0 };
    const g = Math.floor(money / 10000);
    const s = Math.floor((money % 10000) / 100);
    const c = money % 100;
    return { g, s, c };
}
