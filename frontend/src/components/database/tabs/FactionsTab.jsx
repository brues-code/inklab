import { useMemo } from 'react'
import { useStickyState } from '../../../hooks/useStickyState'
import { SidebarPanel, ContentPanel, ScrollList, SectionHeader, ListItem, EntityIcon } from '../../ui'
import { filterItems } from '../../../utils/databaseApi'
import { useFactions } from '../../../hooks/queries/factions'

// Faction side colors
const getSideInfo = (side) => {
    const sides = {
        1: { label: 'A', color: '#0070DE', name: 'Alliance' },
        2: { label: 'H', color: '#C41F3B', name: 'Horde' },
    }
    return sides[side] || { label: 'N', color: '#FFD100', name: 'Neutral' }
}

function FactionsTab({ onNavigate }) {
    const [selectedGroup, setSelectedGroup] = useStickyState('factions.selectedGroup', null)

    const [groupFilter, setGroupFilter] = useStickyState('factions.groupFilter', '')
    const [factionFilter, setFactionFilter] = useStickyState('factions.factionFilter', '')

    // All factions load once (static for a session); groups + filtering derive.
    const { data: factions = [], isLoading } = useFactions()

    // Derive Groups from data
    const groups = useMemo(() => {
        if (factions.length === 0) return []
        
        const map = new Map(factions.map(f => [f.id, f]))
        const parentIds = new Set(factions.map(f => f.categoryId).filter(id => id !== 0))
        
        const g = Array.from(parentIds).map(id => {
             const parent = map.get(id)
             return {
                 id: id,
                 name: parent ? parent.name : `Group ${id}`
             }
        }).sort((a,b) => a.name.localeCompare(b.name))
        
        const hasOrphans = factions.some(f => f.categoryId === 0 && !parentIds.has(f.id))
        if (hasOrphans) {
            g.push({ id: 0, name: 'Others' })
        }

        return g
    }, [factions])

    const filteredGroups = useMemo(() => filterItems(groups, groupFilter), [groups, groupFilter])

    const filteredFactions = useMemo(() => {
        if (!selectedGroup) return []
        
        let subset = []
        if (selectedGroup.id === 0) {
            const parentIds = new Set(factions.map(f => f.categoryId))
            subset = factions.filter(f => f.categoryId === 0 && !parentIds.has(f.id))
        } else {
            subset = factions.filter(f => f.categoryId === selectedGroup.id)
        }
        
        return filterItems(subset, factionFilter)
    }, [factions, selectedGroup, factionFilter])

    return (
        <>
            {/* Groups */}
            <SidebarPanel className="col-span-1">
                <SectionHeader 
                    title={`Faction Groups (${filteredGroups.length})`}
                    placeholder="Filter groups..."
                    onFilterChange={setGroupFilter}
                />
                <ScrollList>
                    {filteredGroups.map(group => (
                        <ListItem
                            key={group.id}
                            active={selectedGroup?.id === group.id}
                            onClick={() => {
                                setSelectedGroup(group)
                                setFactionFilter('')
                            }}
                        >
                            {group.name}
                        </ListItem>
                    ))}
                </ScrollList>
            </SidebarPanel>

            {/* Factions List */}
            <ContentPanel className="col-span-3">
                <SectionHeader 
                    title={selectedGroup ? `${selectedGroup.name} (${filteredFactions.length})` : 'Select a Group'}
                    placeholder="Filter factions..."
                    onFilterChange={setFactionFilter}
                    titleColor="#FFD100"
                />
                
                {isLoading && selectedGroup && (
                    <div className="flex-1 flex items-center justify-center text-wow-gold italic animate-pulse">
                        Loading factions...
                    </div>
                )}

                {selectedGroup && !isLoading && (
                    <ScrollList className="p-2 space-y-1">
                        {filteredFactions.map(faction => {
                            const sideInfo = getSideInfo(faction.side)
                            
                            return (
                                <div 
                                    key={faction.id}
                                    className="flex items-center gap-3 p-2 bg-white/[0.02] hover:bg-white/5 border-l-[3px] cursor-pointer transition-colors rounded-r"
                                    style={{ borderLeftColor: sideInfo.color }}
                                    onClick={() => onNavigate?.('faction', faction.id)}
                                >
                                    {/* Icon */}
                                    <div className="w-8 h-8 flex items-center justify-center bg-gray-900 border border-gray-700/50 p-1 shrink-0">
                                        {faction.side === 1 && (
                                           <img src="/Alliance_15.webp" alt="Alliance" className="w-full h-full object-contain" />
                                        )}
                                        {faction.side === 2 && (
                                           <img src="/Horde_15.webp" alt="Horde" className="w-full h-full object-contain" />
                                        )}
                                        {faction.side !== 1 && faction.side !== 2 && (
                                            <img src="/Neutral_15.webp" alt="Neutral" className="w-full h-full object-contain" />
                                        )}
                                    </div>
                                    
                                    <span className="text-gray-600 text-[11px] font-mono min-w-[50px]">
                                        [{faction.id}]
                                    </span>
                                    
                                    <span 
                                        className="font-bold flex-1 truncate"
                                        style={{ color: sideInfo.color }}
                                    >
                                        {faction.name}
                                    </span>
                                    
                                    <span className="text-gray-500 text-xs ml-auto">
                                        {faction.sideName || sideInfo.name}
                                    </span>
                                </div>
                            )
                        })}
                    </ScrollList>
                )}
                
                {!selectedGroup && !isLoading && (
                    <div className="flex-1 flex items-center justify-center text-gray-600 italic">
                        Select a faction group to view reputations
                    </div>
                )}
            </ContentPanel>
        </>
    )
}

export default FactionsTab
