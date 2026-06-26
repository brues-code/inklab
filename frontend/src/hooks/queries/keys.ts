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

    // Item tooltip (shared app-wide)
    tooltip: (id: unknown) => ['tooltip', id] as const,

    // Talents (talentClasses also drives the Sets-tab class filter)
    talentClasses: ['talentClasses'] as const,
    talentTrees: (cls: unknown) => ['talentTrees', cls] as const,

    // Items
    itemClasses: ['itemClasses'] as const,
    items: (cls: unknown, subClass: unknown, slot: unknown) => ['items', cls, subClass, slot] as const,
    itemDetail: (entry: unknown) => ['itemDetail', entry] as const,
    itemFavorite: (entry: unknown) => ['itemFavorite', entry] as const,
    itemSets: ['itemSets'] as const,
    itemSetDetail: (id: unknown) => ['itemSetDetail', id] as const,

    // NPCs
    creatureTypes: ['creatureTypes'] as const,
    beastFamilies: ['beastFamilies'] as const,
    creatures: (selection: unknown) => ['creatures', selection] as const,
    npcDetail: (entry: unknown) => ['npcDetail', entry] as const,

    // Quests
    questGroups: ['questGroups'] as const,
    questCategories: (groupId: unknown) => ['questCategories', groupId] as const,
    questsByCategory: (categoryId: unknown) => ['questsByCategory', categoryId] as const,
    questDetail: (entry: unknown) => ['questDetail', entry] as const,

    // Objects
    objectTypes: ['objectTypes'] as const,
    objectsByType: (typeId: unknown) => ['objectsByType', typeId] as const,
    objectDetail: (entry: unknown) => ['objectDetail', entry] as const,

    // Zones
    zones: ['zones'] as const,
    zoneDetail: (entry: unknown) => ['zoneDetail', entry] as const,

    // Factions
    factions: ['factions'] as const,
    factionDetail: (id: unknown) => ['factionDetail', id] as const,

    // AtlasLoot
    atlasCategories: ['atlasCategories'] as const,
    atlasModules: (category: unknown) => ['atlasModules', category] as const,
    atlasTables: (category: unknown, module: unknown) => ['atlasTables', category, module] as const,
    atlasLoot: (category: unknown, module: unknown, table: unknown) => ['atlasLoot', category, module, table] as const,

    // Maps (flight network)
    flightContinents: ['flightContinents'] as const,
    flightMap: (view: unknown) => ['flightMap', view] as const,
    flightZone: (mapId: unknown, zoneKey: unknown) => ['flightZone', mapId, zoneKey] as const,

    // Favorites
    favorites: ['favorites'] as const,
    favoriteCategories: ['favoriteCategories'] as const,

    // Global search
    search: (query: unknown) => ['search', query] as const,

    // App chrome
    updateCheck: ['updateCheck'] as const,
    dataStatus: ['dataStatus'] as const,

    // Images (data URLs from the local image service). reloadKey lets a forced
    // refresh produce a fresh key after the underlying file is regenerated.
    image: (type: unknown, name: unknown) => ['image', type, name] as const,
    icon: (name: unknown) => ['icon', name] as const,
    npcModel: (displayId: unknown, creatureEntry: unknown, reloadKey: unknown) =>
        ['npcModel', displayId, creatureEntry, reloadKey] as const,
    npcPortrait: (displayId: unknown, creatureEntry: unknown, generate: unknown, reloadKey: unknown) =>
        ['npcPortrait', displayId, creatureEntry, generate, reloadKey] as const,
    zoneMap: (zoneName: unknown, reloadKey: unknown) => ['zoneMap', zoneName, reloadKey] as const,
}
