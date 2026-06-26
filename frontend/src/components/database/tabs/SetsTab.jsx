import { useState, useEffect, useMemo } from 'react'
import { SidebarPanel, ContentPanel, ScrollList, SectionHeader, ListItem, LootItem } from '../../ui'
import { GetItemSets, GetItemSetDetail, GetTalentClasses, filterItems } from '../../../utils/databaseApi'

function SetsTab({ tooltipHook }) {
    const [itemSets, setItemSets] = useState([])
    const [selectedSet, setSelectedSet] = useState(null)
    const [setDetail, setSetDetail] = useState(null)
    const [loading, setLoading] = useState(false)

    const [classFilter, setClassFilter] = useState('')
    const [setFilter, setSetFilter] = useState('')
    const [itemFilter, setItemFilter] = useState('')

    // First column: playable classes, each with its allowable_class bit
    // (1<<(id-1)). bit 0 = "All Classes".
    const [classOptions, setClassOptions] = useState([])
    const [classBit, setClassBit] = useState(0)

    const { setHoveredItem, loadTooltipData, handleItemEnter, handleMouseMove } = tooltipHook

    // Load item sets on mount
    useEffect(() => {
        setLoading(true)
        GetItemSets()
            .then(sets => {
                setItemSets(sets || [])
                setSelectedSet((sets || [])[0] || null) // default to the first set
                setLoading(false)
            })
            .catch(err => {
                console.error("Failed to load item sets:", err)
                setLoading(false)
            })
    }, [])

    // Load classes for the filter; name comes from game data (ChrClasses.dbc),
    // bit = 1 << (classId - 1) to match items' allowable_class.
    useEffect(() => {
        GetTalentClasses().then(list => {
            setClassOptions(
                (list || [])
                    .map(c => ({ name: c.name || c.class, bit: 1 << (c.classId - 1), color: c.color }))
                    .sort((a, b) => a.name.localeCompare(b.name))
            )
        })
    }, [])

    // Load set detail when a set is selected
    useEffect(() => {
        if (selectedSet) {
            setLoading(true)
            GetItemSetDetail(selectedSet.itemsetId)
                .then(detail => {
                    setSetDetail(detail)
                    setLoading(false)
                    // Preload tooltips for set items (idempotent — cached/deduped)
                    if (detail?.items) {
                        detail.items.forEach(item => {
                            if (item.entry) loadTooltipData(item.entry)
                        })
                    }
                })
                .catch(err => {
                    console.error("Failed to load set detail:", err)
                    setLoading(false)
                })
        }
    }, [selectedSet])

    // Class filter: a set matches when its derived classMask has the bit set.
    // Unrestricted sets (classMask 0) only show under "All".
    const classFilteredSets = useMemo(
        () => classBit === 0 ? itemSets : itemSets.filter(s => (s.classMask & classBit) !== 0),
        [itemSets, classBit]
    )
    const filteredItemSets = useMemo(() => filterItems(classFilteredSets, setFilter), [classFilteredSets, setFilter])
    const filteredSetItems = useMemo(() => {
        if (!setDetail?.items) return []
        return filterItems(setDetail.items, itemFilter)
    }, [setDetail, itemFilter])

    // Per-class set counts for the first column.
    const countForBit = (bit) =>
        bit === 0 ? itemSets.length : itemSets.filter(s => (s.classMask & bit) !== 0).length
    const filteredClasses = useMemo(() => filterItems(classOptions, classFilter), [classOptions, classFilter])

    // First set of a class's list (bit 0 = all), used to default the selection.
    const firstSetForClass = (bit) => {
        const list = bit === 0 ? itemSets : itemSets.filter(s => (s.classMask & bit) !== 0)
        return list[0] || null
    }

    // Selecting a class resets the set filter and defaults to its first set, so
    // the detail panel always shows something.
    const selectClass = (bit) => {
        setClassBit(bit)
        setSetFilter('')
        setItemFilter('')
        setSelectedSet(firstSetForClass(bit))
    }

    return (
        <>
            {/* Classes (1st column) */}
            <SidebarPanel>
                <SectionHeader
                    title="Classes"
                    placeholder="Filter classes..."
                    onFilterChange={setClassFilter}
                />
                <ScrollList>
                    <ListItem active={classBit === 0} onClick={() => selectClass(0)}>
                        <span className="flex justify-between w-full">
                            <span>All Classes</span>
                            <span className="text-gray-600 text-xs">({countForBit(0)})</span>
                        </span>
                    </ListItem>
                    {filteredClasses.map(c => (
                        <ListItem key={c.bit} active={classBit === c.bit} onClick={() => selectClass(c.bit)}>
                            <span className="flex justify-between w-full">
                                <span style={{ color: c.color || undefined }}>{c.name}</span>
                                <span className="text-gray-600 text-xs">({countForBit(c.bit)})</span>
                            </span>
                        </ListItem>
                    ))}
                </ScrollList>
            </SidebarPanel>

            {/* Item Sets (2nd column) */}
            <SidebarPanel>
                <SectionHeader
                    title={`Item Sets (${filteredItemSets.length})`}
                    placeholder="Filter sets..."
                    onFilterChange={setSetFilter}
                />
                <ScrollList>
                    {loading && itemSets.length === 0 && (
                        <div className="p-4 text-center text-wow-gold italic animate-pulse">Loading sets...</div>
                    )}
                    {filteredItemSets.map(set => (
                        <ListItem
                            key={set.itemsetId}
                            active={selectedSet?.itemsetId === set.itemsetId}
                            onClick={() => {
                                setSelectedSet(set)
                                setItemFilter('')
                            }}
                        >
                            <span className="flex justify-between w-full items-start gap-2">
                                <span className="whitespace-normal break-words text-left">{set.name}</span>
                                <span className="text-gray-600 text-xs shrink-0 mt-0.5">({set.itemCount})</span>
                            </span>
                        </ListItem>
                    ))}
                </ScrollList>
            </SidebarPanel>

            {/* Set Details */}
            <ContentPanel>
                <SectionHeader 
                    title={selectedSet ? `${selectedSet.name} (${filteredSetItems.length})` : 'Select a Set'}
                    placeholder="Filter items..."
                    onFilterChange={setItemFilter}
                />
                
                {loading && selectedSet && (
                    <div className="flex-1 flex items-center justify-center text-wow-gold italic animate-pulse">
                        Loading set details...
                    </div>
                )}
                
                {setDetail && !loading && (
                    <ScrollList className="p-2 space-y-2">
                        {/* Set Items */}
                        <div className="grid grid-cols-1 xl:grid-cols-2 gap-1">
                            {filteredSetItems.map((item, idx) => {
                                const handlers = tooltipHook.getItemHandlers?.(item.entry) || {
                                    onMouseEnter: () => handleItemEnter(item.entry),
                                    onMouseMove: (e) => handleMouseMove(e, item.entry),
                                    onMouseLeave: () => setHoveredItem(null),
                                }
                                
                                return (
                                    <LootItem 
                                        key={item.entry || idx}
                                        item={item}
                                        {...handlers}
                                    />
                                )
                            })}
                        </div>
                        
                        {/* Set Bonuses */}
                        {setDetail.bonuses?.length > 0 && (
                            <div className="mt-4 p-4 bg-bg-main rounded-lg border border-border-dark">
                                <h3 className="text-wow-gold font-bold mb-3 text-sm uppercase tracking-wider">
                                    Set Bonuses
                                </h3>
                                <div className="space-y-2">
                                    {setDetail.bonuses.map((bonus, idx) => (
                                        <div 
                                            key={idx} 
                                            className="text-wow-uncommon text-sm flex items-center gap-2"
                                        >
                                            <span className="bg-wow-uncommon/10 text-wow-uncommon px-2 py-0.5 rounded text-xs font-mono">
                                                {bonus.threshold}pc
                                            </span>
                                            <span>{bonus.description || `Spell ID: ${bonus.spellId}`}</span>
                                        </div>
                                    ))}
                                </div>
                            </div>
                        )}
                    </ScrollList>
                )}
                
                {!selectedSet && !loading && (
                    <div className="flex-1 flex items-center justify-center text-gray-600 italic">
                        Select an item set to view its items
                    </div>
                )}
            </ContentPanel>
        </>
    )
}

export default SetsTab
