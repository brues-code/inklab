import { useMemo } from 'react'
import { useStickyState } from '../../../hooks/useStickyState'
import { SidebarPanel, ContentPanel, ScrollList, SectionHeader, ListItem, EntityIcon } from '../../ui'
import { filterItems } from '../../../utils/databaseApi'
import { useQuestGroups, useQuestCategories, useQuestsByCategory } from '../../../hooks/queries/quests'

// Quest type badge colors
const getQuestTypeInfo = (type) => {
    const types = {
        1: { label: 'Group', color: '#1eff00' },
        41: { label: 'PvP', color: '#ff8000' },
        62: { label: 'Raid', color: '#a335ee' },
        81: { label: 'Dungeon', color: '#a335ee' },
    }
    return types[type] || null
}

function QuestsTab({ onNavigate }) {
    const [selectedGroup, setSelectedGroup] = useStickyState('quests.selectedGroup', null)
    const [selectedCategory, setSelectedCategory] = useStickyState('quests.selectedCategory', null)

    const [groupFilter, setGroupFilter] = useStickyState('quests.groupFilter', '')
    const [categoryFilter, setCategoryFilter] = useStickyState('quests.categoryFilter', '')
    const [questFilter, setQuestFilter] = useStickyState('quests.questFilter', '')

    // Cascading queries keyed by selection; resets are handler-driven (no effects).
    const groupsQuery = useQuestGroups()
    const categoriesQuery = useQuestCategories(selectedGroup?.id, !!selectedGroup)
    const questsQuery = useQuestsByCategory(selectedCategory?.id, !!selectedCategory)

    const groups = groupsQuery.data || []
    const categories = categoriesQuery.data || []
    const quests = questsQuery.data || []

    const pickGroup = (group) => {
        setSelectedGroup(group)
        setSelectedCategory(null)
        setCategoryFilter('')
        setQuestFilter('')
    }
    const pickCategory = (cat) => {
        setSelectedCategory(cat)
        setQuestFilter('')
    }

    const filteredGroups = useMemo(() => filterItems(groups, groupFilter), [groups, groupFilter])
    const filteredCategories = useMemo(() => filterItems(categories, categoryFilter), [categories, categoryFilter])
    const filteredQuests = useMemo(() => filterItems(quests, questFilter), [quests, questFilter])

    return (
        <>
            {/* 1. Groups */}
            <SidebarPanel>
                <SectionHeader 
                    title={`Quest Types (${filteredGroups.length})`}
                    placeholder="Filter groups..."
                    onFilterChange={setGroupFilter}
                />
                <ScrollList>
                    {filteredGroups.map(group => (
                        <ListItem
                            key={group.id}
                            active={selectedGroup?.id === group.id}
                            onClick={() => pickGroup(group)}
                        >
                            {group.name}
                        </ListItem>
                    ))}
                </ScrollList>
            </SidebarPanel>

            {/* 2. Categories */}
            <SidebarPanel>
                <SectionHeader 
                    title={selectedGroup ? `${selectedGroup.name} (${filteredCategories.length})` : 'Select Type'}
                    placeholder="Filter zones..."
                    onFilterChange={setCategoryFilter}
                />
                <ScrollList>
                    {filteredCategories.map(cat => (
                        <ListItem
                            key={cat.id}
                            active={selectedCategory?.id === cat.id}
                            onClick={() => pickCategory(cat)}
                        >
                            <span className="flex justify-between w-full">
                                <span>{cat.name}</span>
                                <span className="text-gray-600 text-xs">({cat.questCount})</span>
                            </span>
                        </ListItem>
                    ))}
                </ScrollList>
            </SidebarPanel>

            {/* 3. Quests List (spans 2 columns) */}
            <ContentPanel className="col-span-2">
                <SectionHeader 
                    title={selectedCategory ? `${selectedCategory.name} (${filteredQuests.length})` : 'Select Category'}
                    placeholder="Filter quests..."
                    onFilterChange={setQuestFilter}
                    titleColor="#FFD100"
                />

                {questsQuery.isLoading && (
                    <div className="flex-1 flex items-center justify-center text-wow-gold italic animate-pulse">
                        Loading quests...
                    </div>
                )}

                {!selectedCategory && (
                    <div className="flex-1 flex items-center justify-center text-gray-600 italic">
                        Select a category to browse quests.
                    </div>
                )}

                {!questsQuery.isLoading && quests.length > 0 && (
                    <ScrollList className="p-2 space-y-1">
                        {filteredQuests.map(quest => {
                            const typeInfo = getQuestTypeInfo(quest.type)
                            
                            return (
                                <div 
                                    key={quest.entry}
                                    onClick={() => onNavigate('quest', quest.entry)}
                                    className="flex items-center gap-3 p-2 bg-white/[0.02] hover:bg-white/5 border-l-[3px] border-wow-gold cursor-pointer transition-colors rounded-r group"
                                >
                                    {/* Level Badge */}
                                    <EntityIcon 
                                        label={quest.questLevel > 0 ? quest.questLevel : '-'}
                                        color="#FFD100"
                                        size="md"
                                    />
                                    
                                    {/* Entry ID */}
                                    <span className="text-gray-600 text-[11px] font-mono min-w-[50px]">
                                        [{quest.entry}]
                                    </span>
                                    
                                    {/* Title */}
                                    <span className="text-wow-gold font-bold flex-1 group-hover:brightness-110 transition-all truncate">
                                        {quest.title}
                                    </span>
                                    
                                    {/* Min Level */}
                                    {quest.minLevel > 0 && (
                                        <span className="text-gray-500 text-xs">
                                            Req Lvl {quest.minLevel}
                                        </span>
                                    )}
                                    
                                    {/* Type Badge */}
                                    {typeInfo && (
                                        <span 
                                            className="px-1.5 py-0.5 rounded text-[10px] uppercase border"
                                            style={{ 
                                                color: typeInfo.color, 
                                                borderColor: `${typeInfo.color}40` 
                                            }}
                                        >
                                            {typeInfo.label}
                                        </span>
                                    )}
                                    
                                    {/* XP */}
                                    <span className="text-gray-500 text-xs font-mono">
                                        XP: <b className="text-gray-400">{quest.rewardXp > 0 ? quest.rewardXp : '-'}</b>
                                    </span>
                                </div>
                            )
                        })}
                    </ScrollList>
                )}
            </ContentPanel>
        </>
    )
}

export default QuestsTab
