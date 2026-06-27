import { useMemo } from 'react'
import { useStickyState } from '../../../hooks/useStickyState'
import {
    SidebarPanel,
    ContentPanel,
    ScrollList,
    SectionHeader,
    ListItem,
    EntityIcon,
} from '../../ui'
import { filterItems } from '../../../utils/databaseApi'
import { useZones } from '../../../hooks/queries/zones'

const ZONE_COLOR = '#4ADE80'

function ZonesTab({ onNavigate }) {
    const [selectedGroup, setSelectedGroup] = useStickyState('zones.selectedGroup', null)

    const [groupFilter, setGroupFilter] = useStickyState('zones.groupFilter', '')
    const [zoneFilter, setZoneFilter] = useStickyState('zones.zoneFilter', '')

    const zonesQuery = useZones()
    const zones = zonesQuery.data || []

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

    // Default to the first group until one is explicitly selected (derived, no effect).
    const effectiveGroup = selectedGroup || groups[0] || null

    const filteredGroups = useMemo(() => filterItems(groups, groupFilter), [groups, groupFilter])
    const zonesInGroup = useMemo(
        () => zones.filter((z) => effectiveGroup && z.groupId === effectiveGroup.id),
        [zones, effectiveGroup],
    )
    const filteredZones = useMemo(
        () => filterItems(zonesInGroup, zoneFilter),
        [zonesInGroup, zoneFilter],
    )

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
                    {zonesQuery.isLoading && (
                        <div className="animate-pulse p-4 text-center italic text-wow-gold">
                            Loading zones...
                        </div>
                    )}
                    {filteredGroups.map((group) => (
                        <ListItem
                            key={group.id}
                            active={effectiveGroup?.id === group.id}
                            onClick={() => {
                                setSelectedGroup(group)
                                setZoneFilter('')
                            }}
                        >
                            <span className="flex w-full justify-between">
                                <span>{group.name}</span>
                                <span className="text-xs text-gray-600">({group.count})</span>
                            </span>
                        </ListItem>
                    ))}
                </ScrollList>
            </SidebarPanel>

            {/* Zone List */}
            <ContentPanel className="col-span-3">
                <SectionHeader
                    title={
                        effectiveGroup
                            ? `${effectiveGroup.name} (${filteredZones.length})`
                            : 'Select a Region'
                    }
                    placeholder="Filter zones..."
                    onFilterChange={setZoneFilter}
                />

                {!zonesQuery.isLoading && filteredZones.length > 0 && (
                    <ScrollList className="space-y-1 p-2">
                        {filteredZones.map((zone) => (
                            <div
                                key={zone.id}
                                className="flex cursor-pointer items-center gap-3 rounded-r border-l-[3px] bg-white/[0.02] p-2 transition-colors hover:bg-white/5"
                                style={{ borderLeftColor: ZONE_COLOR }}
                                onClick={() => onNavigate?.('zone', zone.id)}
                            >
                                <EntityIcon label="MAP" color={ZONE_COLOR} size="md" />

                                <span className="min-w-[50px] font-mono text-[11px] text-gray-600">
                                    [{zone.id}]
                                </span>

                                <span
                                    className="flex-1 truncate font-bold"
                                    style={{ color: ZONE_COLOR }}
                                >
                                    {zone.name}
                                </span>

                                <span className="ml-auto whitespace-nowrap text-xs text-gray-500">
                                    {zone.npcCount} NPCs · {zone.questCount} quests
                                </span>
                            </div>
                        ))}
                    </ScrollList>
                )}

                {!zonesQuery.isLoading && effectiveGroup && filteredZones.length === 0 && (
                    <div className="flex flex-1 items-center justify-center italic text-gray-600">
                        No zones match.
                    </div>
                )}

                {!effectiveGroup && !zonesQuery.isLoading && (
                    <div className="flex flex-1 items-center justify-center italic text-gray-600">
                        Select a region to browse its zones
                    </div>
                )}
            </ContentPanel>
        </>
    )
}

export default ZonesTab
