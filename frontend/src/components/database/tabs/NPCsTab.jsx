import { useMemo, useRef, useCallback } from 'react'
import { useStickyState } from '../../../hooks/useStickyState'
import { SidebarPanel, ContentPanel, ScrollList, SectionHeader, ListItem, EntityIcon } from '../../ui'
import { filterItems } from '../../../utils/databaseApi'
import { useCreatureTypes, useBeastFamilies, useCreatures } from '../../../hooks/queries/npcs'

const BEAST_TYPE = 1
import { useNpcPortrait } from '../../../services/useImage'

// NPC rank colors
const getRankColor = (rank) => {
    if (rank >= 3) return '#a335ee' // Boss - Epic purple
    if (rank >= 1) return '#ff8000' // Elite - Legendary orange
    return '#1eff00' // Normal - Uncommon green
}

// NpcPortraitThumb shows a creature's cached head-shot portrait in list rows.
// It loads cached-only (generate=false) so scrolling never triggers a render
// storm; rows without a cached portrait simply show nothing here.
const NpcPortraitThumb = ({ displayId, rankColor }) => {
    const { src } = useNpcPortrait(displayId, 0, 0, false)
    if (!src) return null
    return (
        <img
            src={src}
            alt=""
            className="w-8 h-8 rounded-full object-cover border bg-black flex-shrink-0"
            style={{ borderColor: `${rankColor}66` }}
        />
    )
}

function NPCsTab({ onNavigate, tooltipHook }) {
    const [selectedCreatureType, setSelectedCreatureType] = useStickyState('npcs.selectedCreatureType', null)
    const [selectedFamily, setSelectedFamily] = useStickyState('npcs.selectedFamily', null)

    const [typeFilter, setTypeFilter] = useStickyState('npcs.typeFilter', '')
    const [creatureFilter, setCreatureFilter] = useStickyState('npcs.creatureFilter', '')
    const [familyFilter, setFamilyFilter] = useStickyState('npcs.familyFilter', '')

    const isBeast = selectedCreatureType?.type === BEAST_TYPE

    const scrollRef = useRef(null)

    const typesQuery = useCreatureTypes()
    const creatureTypes = typesQuery.data || []

    // Beast families load only for the Beast type (the dynamic 3rd column).
    const familiesQuery = useBeastFamilies(isBeast)
    const families = familiesQuery.data || []

    // Paginated creature browse for the active selection (beast family or type).
    const creaturesQuery = useCreatures(selectedCreatureType, selectedFamily, isBeast)

    const creatures = useMemo(
        () => creaturesQuery.data?.pages.flatMap(p => p.creatures || []) || [],
        [creaturesQuery.data]
    )
    const total = creaturesQuery.data?.pages?.[0]?.total || 0

    // Infinite scroll: fetch the next page near the bottom.
    const handleScroll = useCallback((e) => {
        const { scrollTop, scrollHeight, clientHeight } = e.target
        if (scrollHeight - scrollTop - clientHeight < 200 && creaturesQuery.hasNextPage && !creaturesQuery.isFetchingNextPage) {
            creaturesQuery.fetchNextPage()
        }
    }, [creaturesQuery])

    // Selecting a type clears the family + filters; families load via the query.
    const pickType = (type) => {
        setSelectedCreatureType(type)
        setSelectedFamily(null)
        setCreatureFilter('')
        setFamilyFilter('')
    }

    const filteredTypes = useMemo(() => filterItems(creatureTypes, typeFilter), [creatureTypes, typeFilter])
    const filteredFamilies = useMemo(() => filterItems(families, familyFilter), [families, familyFilter])
    const filteredCreatures = useMemo(() => filterItems(creatures, creatureFilter), [creatures, creatureFilter])

    return (
        <>
            {/* Creature Types (spans 1 column) */}
            <SidebarPanel className="col-span-1">
                <SectionHeader 
                    title={`Creature Types (${filteredTypes.length})`}
                    placeholder="Filter types..."
                    onFilterChange={setTypeFilter}
                />
                <ScrollList>
                    {typesQuery.isLoading && (
                        <div className="p-4 text-center text-wow-gold italic animate-pulse">Loading types...</div>
                    )}
                    {filteredTypes.map(type => (
                        <ListItem
                            key={type.type}
                            active={selectedCreatureType?.type === type.type}
                            onClick={() => pickType(type)}
                        >
                            <span className="flex justify-between w-full">
                                <span>{type.name}</span>
                                <span className="text-gray-600 text-xs">({type.count})</span>
                            </span>
                        </ListItem>
                    ))}
                </ScrollList>
            </SidebarPanel>

            {/* Beast families (dynamic 3rd column, only for the Beast type) */}
            {isBeast && families.length > 0 && (
                <SidebarPanel className="col-span-1">
                    <SectionHeader
                        title={`Families (${filteredFamilies.length})`}
                        placeholder="Filter families..."
                        onFilterChange={setFamilyFilter}
                    />
                    <ScrollList>
                        <ListItem active={!selectedFamily} onClick={() => setSelectedFamily(null)}>
                            <span className="flex justify-between w-full">
                                <span>All {selectedCreatureType.name}</span>
                                <span className="text-gray-600 text-xs">({selectedCreatureType.count})</span>
                            </span>
                        </ListItem>
                        {filteredFamilies.map(f => (
                            <ListItem
                                key={f.family}
                                active={selectedFamily?.family === f.family}
                                onClick={() => setSelectedFamily(f)}
                            >
                                <span className="flex justify-between w-full">
                                    <span>{f.name}</span>
                                    <span className="text-gray-600 text-xs">({f.count})</span>
                                </span>
                            </ListItem>
                        ))}
                    </ScrollList>
                </SidebarPanel>
            )}

            {/* Creatures List (spans remaining columns) */}
            <ContentPanel className={isBeast && families.length > 0 ? 'col-span-2' : 'col-span-3'}>
                <SectionHeader
                    title={selectedCreatureType
                        ? `${selectedFamily ? selectedFamily.name : selectedCreatureType.name} (${filteredCreatures.length}${total > creatures.length ? ` of ${total}` : ''})`
                        : 'Select a Type'
                    }
                    placeholder="Filter NPCs..."
                    onFilterChange={setCreatureFilter}
                />


                {creaturesQuery.isLoading && (
                    <div className="flex-1 flex items-center justify-center text-wow-gold italic animate-pulse">
                        Loading creatures...
                    </div>
                )}

                {!creaturesQuery.isLoading && creatures.length > 0 && (
                    <ScrollList 
                        ref={scrollRef}
                        className="p-2 space-y-1"
                        onScroll={handleScroll}
                    >
                        {filteredCreatures.map(creature => {
                            const rankColor = getRankColor(creature.rank)
                            const levelText = creature.levelMin === creature.levelMax 
                                ? `${creature.levelMin}` 
                                : `${creature.levelMin}-${creature.levelMax}`
                            
                            return (
                                <div 
                                    key={creature.entry}
                                    onClick={() => onNavigate('npc', creature.entry)}
                                    className="flex items-center gap-3 p-2 bg-white/[0.02] hover:bg-white/5 border-l-[3px] cursor-pointer transition-colors rounded-r group"
                                    style={{ borderLeftColor: rankColor }}
                                >
                                    {/* Portrait (cached only) + Level Badge */}
                                    <NpcPortraitThumb displayId={creature.displayId1} rankColor={rankColor} />
                                    <EntityIcon
                                        label={levelText}
                                        color={rankColor}
                                        size="md"
                                    />
                                    
                                    {/* Entry ID */}
                                    <span className="text-gray-600 text-[11px] font-mono min-w-[50px]">
                                        [{creature.entry}]
                                    </span>
                                    
                                    {/* Name & Subname */}
                                    <div className="flex-1 min-w-0">
                                        <span 
                                            className="font-bold group-hover:brightness-110 transition-all"
                                            style={{ color: rankColor }}
                                        >
                                            {creature.name}
                                        </span>
                                        {creature.subname && (
                                            <span className="text-gray-500 ml-2 text-sm">
                                                &lt;{creature.subname}&gt;
                                            </span>
                                        )}
                                    </div>
                                    
                                    {/* Stats */}
                                    <div className="flex items-center gap-3 text-gray-500 text-xs ml-auto">
                                        {creature.rankName !== 'Normal' && (
                                            <span 
                                                className="px-1.5 py-0.5 rounded border"
                                                style={{ color: rankColor, borderColor: `${rankColor}40` }}
                                            >
                                                {creature.rankName}
                                            </span>
                                        )}
                                        <span className="font-mono">
                                            HP: <b className="text-gray-400">{creature.healthMax.toLocaleString()}</b>
                                        </span>
                                    </div>
                                </div>
                            )
                        })}
                        
                        {/* Loading more indicator */}
                        {creaturesQuery.isFetchingNextPage && (
                            <div className="p-4 text-center text-wow-gold italic animate-pulse">
                                Loading more...
                            </div>
                        )}

                        {/* Has more indicator */}
                        {creaturesQuery.hasNextPage && !creaturesQuery.isFetchingNextPage && (
                            <div className="p-2 text-center text-gray-600 text-sm">
                                Scroll for more ({creatures.length} of {total})
                            </div>
                        )}
                    </ScrollList>
                )}

                {!selectedCreatureType && (
                    <div className="flex-1 flex items-center justify-center text-gray-600 italic">
                        Select a creature type to browse NPCs
                    </div>
                )}
            </ContentPanel>
        </>
    )
}

export default NPCsTab
