import { useState, useEffect, useMemo } from 'react'
import { SidebarPanel, ContentPanel, ScrollList, SectionHeader, ListItem, EntityIcon } from '../../ui'
import { GetZones, filterItems } from '../../../utils/databaseApi'

const ZONE_COLOR = '#4ADE80'

function ZonesTab({ onNavigate }) {
    const [zones, setZones] = useState([])
    const [selectedGroup, setSelectedGroup] = useState(null)
    const [loading, setLoading] = useState(false)

    const [groupFilter, setGroupFilter] = useState('')
    const [zoneFilter, setZoneFilter] = useState('')

    useEffect(() => {
        setLoading(true)
        GetZones()
            .then(res => {
                setZones(res || [])
                setLoading(false)
            })
            .catch(err => {
                console.error('Failed to load zones:', err)
                setLoading(false)
            })
    }, [])

    // Derive the continent/type groups from the flat zone list, preserving the
    // backend's group ordering.
    const groups = useMemo(() => {
        const byId = new Map()
        for (const z of zones) {
            if (!byId.has(z.groupId)) {
                byId.set(z.groupId, { id: z.groupId, name: z.groupName || 'Other', count: 0 })
            }
            byId.get(z.groupId).count++
        }
        return Array.from(byId.values())
    }, [zones])

    // Default to the first group once zones load.
    useEffect(() => {
        if (selectedGroup === null && groups.length > 0) {
            setSelectedGroup(groups[0])
        }
    }, [groups, selectedGroup])

    const filteredGroups = useMemo(() => filterItems(groups, groupFilter), [groups, groupFilter])
    const zonesInGroup = useMemo(
        () => zones.filter(z => selectedGroup && z.groupId === selectedGroup.id),
        [zones, selectedGroup]
    )
    const filteredZones = useMemo(() => filterItems(zonesInGroup, zoneFilter), [zonesInGroup, zoneFilter])

    return (
        <>
            {/* Continents / Types */}
            <SidebarPanel className="col-span-1">
                <SectionHeader
                    title={`Regions (${filteredGroups.length})`}
                    placeholder="Filter regions..."
                    onFilterChange={setGroupFilter}
                />
                <ScrollList>
                    {loading && zones.length === 0 && (
                        <div className="p-4 text-center text-wow-gold italic animate-pulse">Loading zones...</div>
                    )}
                    {filteredGroups.map(group => (
                        <ListItem
                            key={group.id}
                            active={selectedGroup?.id === group.id}
                            onClick={() => {
                                setSelectedGroup(group)
                                setZoneFilter('')
                            }}
                        >
                            <span className="flex justify-between w-full">
                                <span>{group.name}</span>
                                <span className="text-gray-600 text-xs">({group.count})</span>
                            </span>
                        </ListItem>
                    ))}
                </ScrollList>
            </SidebarPanel>

            {/* Zone List */}
            <ContentPanel className="col-span-3">
                <SectionHeader
                    title={selectedGroup ? `${selectedGroup.name} (${filteredZones.length})` : 'Select a Region'}
                    placeholder="Filter zones..."
                    onFilterChange={setZoneFilter}
                />

                {!loading && filteredZones.length > 0 && (
                    <ScrollList className="p-2 space-y-1">
                        {filteredZones.map(zone => (
                            <div
                                key={zone.id}
                                className="flex items-center gap-3 p-2 bg-white/[0.02] hover:bg-white/5 border-l-[3px] cursor-pointer transition-colors rounded-r"
                                style={{ borderLeftColor: ZONE_COLOR }}
                                onClick={() => onNavigate?.('zone', zone.id)}
                            >
                                <EntityIcon label="MAP" color={ZONE_COLOR} size="md" />

                                <span className="text-gray-600 text-[11px] font-mono min-w-[50px]">
                                    [{zone.id}]
                                </span>

                                <span className="font-bold flex-1 truncate" style={{ color: ZONE_COLOR }}>
                                    {zone.name}
                                </span>

                                <span className="text-gray-500 text-xs ml-auto whitespace-nowrap">
                                    {zone.npcCount} NPCs · {zone.questCount} quests
                                </span>
                            </div>
                        ))}
                    </ScrollList>
                )}

                {!loading && selectedGroup && filteredZones.length === 0 && (
                    <div className="flex-1 flex items-center justify-center text-gray-600 italic">
                        No zones match.
                    </div>
                )}

                {!selectedGroup && !loading && (
                    <div className="flex-1 flex items-center justify-center text-gray-600 italic">
                        Select a region to browse its zones
                    </div>
                )}
            </ContentPanel>
        </>
    )
}

export default ZonesTab
