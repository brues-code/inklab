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

// Get icon path - default to JPG
export const getIconPath = (icon?: string | null): string => {
    if (!icon) return '/local-icons/inv_misc_questionmark.jpg';
    return `/local-icons/${icon.toLowerCase()}.jpg`;
}

// Get PNG variant of icon path
export const getIconPathPng = (icon?: string | null): string => {
    if (!icon) return '/local-icons/inv_misc_questionmark.jpg';
    return `/local-icons/${icon.toLowerCase()}.png`;
}

export interface Money { g: number; s: number; c: number }

export const formatMoney = (money?: number): Money => {
    if (!money) return { g: 0, s: 0, c: 0 };
    const g = Math.floor(money / 10000);
    const s = Math.floor((money % 10000) / 100);
    const c = money % 100;
    return { g, s, c };
}
