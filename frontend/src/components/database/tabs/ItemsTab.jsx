import { useEffect, useMemo } from 'react'
import { useStickyState } from '../../../hooks/useStickyState'
import { SidebarPanel, ContentPanel, ScrollList, SectionHeader, ListItem, LootItem, ContentGrid } from '../../ui'
import { filterItems } from '../../../utils/databaseApi'
import { useItemClasses, useItems } from '../../../hooks/queries/items'
import { getCategoryIcon } from '../../../utils/categoryIcons'
import { 
    GRID_LAYOUT, ITEMS_LAYOUT, 
    GRID_LAYOUT_NO_FILTER, ITEMS_LAYOUT_NO_FILTER 
} from '../../common/layout'
import ItemFilters from './ItemFilters'

// Resistance Fields
const RESISTANCE_FIELDS = {
    'fire_res': 'fireRes',
    'frost_res': 'frostRes',
    'nature_res': 'natureRes',
    'shadow_res': 'shadowRes',
    'arcane_res': 'arcaneRes',
    'holy_res': 'holyRes', // rare but possible
}

function ItemsTab({ tooltipHook, onNavigate }) {
    // Selection + filter state is sticky (module-scoped) so it survives leaving
    // the Database route and coming Back — e.g. Armor → Mail → Waist stays put.
    const [selectedClass, setSelectedClass] = useStickyState('items.selectedClass', null)
    const [selectedSubClass, setSelectedSubClass] = useStickyState('items.selectedSubClass', null)
    const [selectedSlot, setSelectedSlot] = useStickyState('items.selectedSlot', null)

    // Independent filter states for each column
    const [classFilter, setClassFilter] = useStickyState('items.classFilter', '')
    const [subClassFilter, setSubClassFilter] = useStickyState('items.subClassFilter', '')
    const [slotFilter, setSlotFilter] = useStickyState('items.slotFilter', '')
    const [itemFilter, setItemFilter] = useStickyState('items.itemFilter', '')

    // Advanced Filters
    const [advancedFilters, setAdvancedFilters] = useStickyState('items.advancedFilters', {})
    
    // Track filter changes with detailed logging
    useEffect(() => {
        console.group('🔍 [ItemsTab] Filter Conditions Changed')
        
        // Item Level
        if (advancedFilters.minIlvl || advancedFilters.maxIlvl) {
            const min = advancedFilters.minIlvl || '-'
            const max = advancedFilters.maxIlvl || '-'
            console.log(`📊 Item Level: ${min} - ${max}`)
        }
        
        // Required Level
        if (advancedFilters.minRl || advancedFilters.maxRl) {
            const min = advancedFilters.minRl || '-'
            const max = advancedFilters.maxRl || '-'
            console.log(`⚔️ Required Level: ${min} - ${max}`)
        }
        
        // Quality
        if (advancedFilters.quality && Array.isArray(advancedFilters.quality) && advancedFilters.quality.length > 0) {
            const qualityNames = ['Poor', 'Common', 'Uncommon', 'Rare', 'Epic', 'Legendary']
            const selectedNames = advancedFilters.quality.map(q => qualityNames[q] || 'Unknown').join(', ')
            console.log(`💎 Quality: ${selectedNames}`)
        }
        
        // Stats
        if (advancedFilters.stats && advancedFilters.stats.length > 0) {
            console.log('📈 Stats:')
            advancedFilters.stats.forEach((f, i) => {
                if (f.stat && (f.minVal || f.maxVal)) {
                    const min = f.minVal || '-'
                    const max = f.maxVal || '-'
                    console.log(`   ${i + 1}. ${f.stat}: ${min} - ${max}`)
                }
            })
        }
        
        // Other Stats
        if (advancedFilters.otherStats && advancedFilters.otherStats.length > 0) {
            console.log('🎯 Other Stats:')
            advancedFilters.otherStats.forEach((f, i) => {
                if (f.stat && (f.minVal || f.maxVal)) {
                    const min = f.minVal || '-'
                    const max = f.maxVal || '-'
                    console.log(`   ${i + 1}. ${f.stat}: ${min} - ${max}`)
                }
            })
        }
        
        // Show raw data
        console.log('📋 Raw Filter Data:', advancedFilters)
        console.groupEnd()
    }, [advancedFilters])

    const { setHoveredItem, handleItemEnter, handleMouseMove } = tooltipHook

    // Item classes load once (static for a session).
    const classesQuery = useItemClasses()
    const itemClasses = classesQuery.data || []

    // Check if the selected class needs slot filtering (Armor = 4, Weapon = 2)
    const needsSlotFilter = selectedClass?.class === 4 || selectedClass?.class === 2

    // Browse items for the current class/subclass/(slot). Gated until the needed
    // selections exist (the slot-aware vs class+subclass choice lives in the hook).
    const itemsQuery = useItems(
        selectedClass,
        selectedSubClass,
        selectedSlot,
        selectedClass !== null && selectedSubClass !== null && (!needsSlotFilter || selectedSlot !== null)
    )
    const items = itemsQuery.data || []
    const loading = itemsQuery.isLoading

    // Filtered lists
    const filteredClasses = useMemo(() => filterItems(itemClasses, classFilter), [itemClasses, classFilter])
    const filteredSubClasses = useMemo(() => {
        if (!selectedClass?.subClasses) return []
        return filterItems(selectedClass.subClasses, subClassFilter)
    }, [selectedClass, subClassFilter])
    const filteredSlots = useMemo(() => {
        if (!selectedSubClass?.inventorySlots) return []
        return filterItems(selectedSubClass.inventorySlots, slotFilter)
    }, [selectedSubClass, slotFilter])
    
    // Advanced Item Filtering
    const filteredItems = useMemo(() => {
        let result = filterItems(items, itemFilter)

        // Min Item Level
        if (advancedFilters.minIlvl) {
            const min = parseInt(advancedFilters.minIlvl)
            if (!isNaN(min)) result = result.filter(i => (i.itemLevel || 0) >= min)
        }
        // Max Item Level
        if (advancedFilters.maxIlvl) {
            const max = parseInt(advancedFilters.maxIlvl)
            if (!isNaN(max)) result = result.filter(i => (i.itemLevel || 0) <= max)
        }
        
        // Min Req Level
        if (advancedFilters.minRl) {
            const min = parseInt(advancedFilters.minRl)
            if (!isNaN(min)) result = result.filter(i => (i.requiredLevel || 0) >= min)
        }
        // Max Req Level
        if (advancedFilters.maxRl) {
            const max = parseInt(advancedFilters.maxRl)
            if (!isNaN(max)) result = result.filter(i => (i.requiredLevel || 0) <= max)
        }

        // Quality
        if (advancedFilters.quality && Array.isArray(advancedFilters.quality) && advancedFilters.quality.length > 0) {
            result = result.filter(i => advancedFilters.quality.includes(Number(i.quality)))
        }

        // Helper to check stats with range support
        const checkStat = (item, statName, minValStr, maxValStr) => {
            const minVal = parseFloat(minValStr)
            const maxVal = parseFloat(maxValStr)
            const hasMin = !isNaN(minVal) && minValStr !== ''
            const hasMax = !isNaN(maxVal) && maxValStr !== ''
            
            // If both are empty, ignore this stat filter
            if (!hasMin && !hasMax) return true

            // Check if it's armor
            if (statName === 'armor') {
                const armorVal = item.armor || 0
                if (hasMin && armorVal < minVal) return false
                if (hasMax && armorVal > maxVal) return false
                return true
            }

            // Check if it's a resistance field
            if (RESISTANCE_FIELDS[statName]) {
                const resVal = item[RESISTANCE_FIELDS[statName]] || 0
                if (hasMin && resVal < minVal) return false
                if (hasMax && resVal > maxVal) return false
                return true
            }

            // Otherwise statName is a stat_type id (from the dynamic dropdown).
            const targetStatId = parseInt(statName, 10)
            if (isNaN(targetStatId)) return true // Unknown stat, ignore

            // Item stats are in statType1/statValue1 ... statType10/statValue10
            // We need to check all 10 slots
            let totalVal = 0
            for (let i = 1; i <= 10; i++) {
                if (item[`statType${i}`] === targetStatId) {
                    totalVal += (item[`statValue${i}`] || 0)
                }
            }
            
            // Check range
            if (hasMin && totalVal < minVal) return false
            if (hasMax && totalVal > maxVal) return false
            return true
        }

        // Apply Stats filters
        // Apply Stats filters
        if (advancedFilters.stats) {
            advancedFilters.stats.forEach(f => {
                if (f.stat && (f.minVal || f.maxVal)) {
                    result = result.filter(i => checkStat(i, f.stat, f.minVal, f.maxVal))
                }
            })
        }

        // Apply Other Stats filters (same logic, just another list)
        if (advancedFilters.otherStats) {
            advancedFilters.otherStats.forEach(f => {
                if (f.stat && (f.minVal || f.maxVal)) {
                    result = result.filter(i => checkStat(i, f.stat, f.minVal, f.maxVal))
                }
            })
        }

        return result
    }, [items, itemFilter, advancedFilters])
    
    // Debug: Log filtering results
    useEffect(() => {
        if (items.length > 0 && (advancedFilters.stats?.length > 0 || advancedFilters.otherStats?.length > 0)) {
            console.group('🔬 [ItemsTab] Filtering Debug')
            console.log(`📦 Total items before filter: ${items.length}`)
            console.log(`✅ Items after filter: ${filteredItems.length}`)
            
            // Show first item's stat structure
            if (items.length > 0) {
                console.log('📋 First item stat data check:')
                const item = items[0]
                console.log(`  Name: ${item.name}`)
                
                let foundStats = false
                // Check statTypeX/statValueX
                for (let j = 1; j <= 10; j++) {
                    const typeKey = `statType${j}`
                    const valueKey = `statValue${j}`
                    if (item[typeKey] !== undefined && item[typeKey] !== null) {
                        console.log(`  stats.${typeKey}: ${item[typeKey]} (Value: ${item[valueKey]})`)
                        if (item[typeKey] > 0) foundStats = true
                    }
                }
                
                // Check Armor and Res
                if (item.armor > 0) console.log(`  Armor: ${item.armor}`)
                if (item.holyRes > 0) console.log(`  Holy Res: ${item.holyRes}`)
                
                if (!foundStats && !item.armor) console.log('  ⚠️ No stats found on first item')
            }
            console.groupEnd()
        }
    }, [items, filteredItems, advancedFilters])
    
    // Count active filters and build summary
    const filterSummary = useMemo(() => {
        const parts = []
        
        if (advancedFilters.minIlvl || advancedFilters.maxIlvl) {
            const min = advancedFilters.minIlvl || '0'
            const max = advancedFilters.maxIlvl || '∞'
            parts.push(`iLvl ${min}-${max}`)
        }
        
        if (advancedFilters.minRl || advancedFilters.maxRl) {
            const min = advancedFilters.minRl || '0'
            const max = advancedFilters.maxRl || '∞'
            parts.push(`Req ${min}-${max}`)
        }
        
        if (advancedFilters.quality && Array.isArray(advancedFilters.quality) && advancedFilters.quality.length > 0) {
            const qualities = ['Poor', 'Common', 'Uncommon', 'Rare', 'Epic', 'Legendary']
            if (advancedFilters.quality.length === 1) {
                parts.push(qualities[advancedFilters.quality[0]])
            } else {
                parts.push(`${advancedFilters.quality.length} Qualities`)
            }
        }
        
        // Count stats filters
        let statsCount = 0
        if (advancedFilters.stats) {
            advancedFilters.stats.forEach(f => {
                if (f.stat && (f.minVal || f.maxVal)) statsCount++
            })
        }
        if (advancedFilters.otherStats) {
            advancedFilters.otherStats.forEach(f => {
                if (f.stat && (f.minVal || f.maxVal)) statsCount++
            })
        }
        if (statsCount > 0) {
            parts.push(`${statsCount} stat${statsCount > 1 ? 's' : ''}`)
        }
        
        return parts.length > 0 ? parts.join(' • ') : null
    }, [advancedFilters])

    // Layout and Filter visibility
    const [showFilters, setShowFilters] = useStickyState('items.showFilters', false)
    
    // Determine current layout based on Armor class and Filter visibility
    const currentLayout = useMemo(() => {
        if (showFilters) {
            return needsSlotFilter ? ITEMS_LAYOUT : GRID_LAYOUT
        } else {
            return needsSlotFilter ? ITEMS_LAYOUT_NO_FILTER : GRID_LAYOUT_NO_FILTER
        }
    }, [needsSlotFilter, showFilters])

    // Filter toggle button
    const filterToggleButton = (
        <button 
            onClick={() => {
                if (showFilters) {
                    setAdvancedFilters({})
                }
                setShowFilters(!showFilters)
            }}
            className={`
                px-2 py-1 text-xs rounded border transition-colors flex items-center gap-1
                ${showFilters 
                    ? 'bg-wow-gold text-black border-wow-gold hover:bg-yellow-500' 
                    : 'bg-black/30 text-gray-400 border-gray-700 hover:text-white hover:border-gray-500'
                }
            `}
            title={showFilters ? "Hide Filters" : "Show Advanced Filters"}
        >
            <svg width="14" height="14" viewBox="0 0 24 24" fill="currentColor">
                <path d="M10 18h4v-2h-4v2zM3 6v2h18V6H3zm3 7h12v-2H6v2z"/>
            </svg>
            {showFilters ? 'Hide' : 'Filters'}
        </button>
    )



    return (
        <ContentGrid columns={currentLayout}>
            {/* 1. Classes */}
            <SidebarPanel>
                <SectionHeader 
                    title={`Item Class (${filteredClasses.length})`}
                    placeholder="Filter classes..."
                    onFilterChange={setClassFilter}
                />
                <ScrollList>
                    {filteredClasses.map(cls => {
                        const icon = getCategoryIcon(cls.name)
                        return (
                            <ListItem
                                key={cls.class}
                                active={selectedClass?.class === cls.class}
                                onClick={() => {
                                    setSelectedClass(cls)
                                    setSelectedSubClass(null)
                                    setSelectedSlot(null)
                                    setSubClassFilter('')
                                    setSlotFilter('')
                                    setItemFilter('')
                                    // Keep advanced filters? Usually yes.
                                }}
                            >
                                <div className="flex items-center gap-2">
                                    {icon && <img src={icon} alt="" className="w-5 h-5 object-contain opacity-80" />}
                                    <span>{cls.name}</span>
                                </div>
                            </ListItem>
                        )
                    })}
                </ScrollList>
            </SidebarPanel>

            {/* 2. SubClasses */}
            <SidebarPanel>
                <SectionHeader 
                    title={selectedClass ? `${selectedClass.name} (${filteredSubClasses.length})` : 'Select Class'}
                    placeholder="Filter types..."
                    onFilterChange={setSubClassFilter}
                />
                <ScrollList>
                    {filteredSubClasses.map(sc => (
                        <ListItem
                            key={sc.subClass}
                            active={selectedSubClass?.subClass === sc.subClass}
                            onClick={() => {
                                setSelectedSubClass(sc)
                                setSelectedSlot(null)
                                setSlotFilter('')
                                setItemFilter('')
                            }}
                        >
                            {sc.name}
                        </ListItem>
                    ))}
                </ScrollList>
            </SidebarPanel>

            {/* 3. Inventory Slots - Show for Armor and Weapon classes */}
            {needsSlotFilter && (
                <SidebarPanel>
                    <SectionHeader 
                        title={selectedSubClass ? `Slot (${filteredSlots.length})` : 'Select Type'}
                        placeholder="Filter slots..."
                        onFilterChange={setSlotFilter}
                    />
                    <ScrollList>
                        {filteredSlots.map(slot => (
                            <ListItem
                                key={slot.inventoryType}
                                active={selectedSlot?.inventoryType === slot.inventoryType}
                                onClick={() => {
                                    setSelectedSlot(slot)
                                    setItemFilter('')
                                }}
                            >
                                {slot.name}
                            </ListItem>
                        ))}

                        {selectedSubClass?.inventorySlots?.length > 1 && (
                            <ListItem
                                active={selectedSlot?.inventoryType === -1}
                                onClick={() => {
                                    setSelectedSlot({ inventoryType: -1, name: 'All Slots' })
                                    setItemFilter('')
                                }}
                                className="italic text-gray-500"
                            >
                                All Slots
                            </ListItem>
                        )}
                    </ScrollList>
                </SidebarPanel>
            )}

            {/* 4. Advanced Filters */}
            {showFilters && (
                <ItemFilters 
                    filters={advancedFilters} 
                    onChange={setAdvancedFilters}
                    onSearch={() => {/* Filters are auto-applied via useMemo */}}
                    onReset={() => setAdvancedFilters({})}
                />
            )}

            {/* 5. Items List */}
            <ContentPanel>
                <SectionHeader 
                    title={
                        selectedSubClass 
                            ? `${selectedSlot ? selectedSlot.name : selectedSubClass.name} (${filteredItems.length})${filterSummary ? ` • ${filterSummary}` : ''}` 
                            : 'Select SubClass'
                    }
                    placeholder="Filter by name..."
                    onFilterChange={setItemFilter}
                    actions={filterToggleButton}
                />
                
                {loading && (
                    <div className="flex-1 flex items-center justify-center text-wow-gold italic animate-pulse">
                        Loading items...
                    </div>
                )}
                
                {!loading && items.length > 0 && (
                    <ScrollList className="grid grid-cols-1 xl:grid-cols-2 gap-1 p-2 auto-rows-min">
                        {filteredItems.map((item, idx) => {
                            const itemId = item.entry || item.id || item.itemId
                            const handlers = tooltipHook.getItemHandlers?.(itemId) || {
                                onMouseEnter: () => handleItemEnter(itemId),
                                onMouseMove: (e) => handleMouseMove(e, itemId),
                                onMouseLeave: () => setHoveredItem(null),
                            }
                            
                            return (
                                <LootItem 
                                    key={itemId || idx}
                                    item={item}
                                    onClick={() => onNavigate && onNavigate('item', itemId)}
                                    {...handlers}
                                />
                            )
                        })}
                    </ScrollList>
                )}
                
                {!loading && items.length === 0 && selectedSubClass && (
                    <div className="flex-1 flex items-center justify-center text-gray-600 italic">
                        No items found
                    </div>
                )}
            </ContentPanel>
        </ContentGrid>
    )
}

export default ItemsTab
