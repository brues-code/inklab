import { useMemo } from 'react'
import { useStickyState } from '../../../hooks/useStickyState'
import { SidebarPanel, ContentPanel, ScrollList, SectionHeader, ListItem, LootItem } from '../../ui'
import { filterItems } from '../../../utils/databaseApi'
import { useItemSets, useItemSetDetail } from '../../../hooks/queries/sets'
import { useTalentClasses } from '../../../hooks/queries/talents'

function SetsTab({ tooltipHook, onNavigate }) {
    const [selectedSet, setSelectedSet] = useStickyState('sets.selectedSet', null)
    // First column: playable classes, each with its allowable_class bit
    // (1<<(id-1)). bit 0 = "All Classes".
    const [classBit, setClassBit] = useStickyState('sets.classBit', 0)

    const [classFilter, setClassFilter] = useStickyState('sets.classFilter', '')
    const [setFilter, setSetFilter] = useStickyState('sets.setFilter', '')
    const [itemFilter, setItemFilter] = useStickyState('sets.itemFilter', '')

    const { setHoveredItem, handleItemEnter, handleMouseMove } = tooltipHook

    // Item sets + playable classes are static for a session; set detail is keyed
    // by the shown set. No effects — everything derives from the queries.
    const itemSetsQuery = useItemSets()
    const classesQuery = useTalentClasses()

    const itemSets = itemSetsQuery.data || []

    // Class filter: a set matches when its derived classMask has the bit set.
    // Unrestricted sets (classMask 0) only show under "All".
    const classFilteredSets = useMemo(
        () => (classBit === 0 ? itemSets : itemSets.filter((s) => (s.classMask & classBit) !== 0)),
        [itemSets, classBit],
    )

    // The set to show detail for: the explicit selection, else the first set of
    // the current class filter, so the detail panel always shows something.
    const effectiveSet = selectedSet || classFilteredSets[0] || null

    const setDetailQuery = useItemSetDetail(effectiveSet?.itemsetId, !!effectiveSet)
    const setDetail = setDetailQuery.data || null

    // Class options from game data (ChrClasses.dbc); bit = 1 << (classId - 1) to
    // match items' allowable_class.
    const classOptions = useMemo(
        () =>
            (classesQuery.data || [])
                .map((c) => ({
                    name: c.name || c.class,
                    bit: 1 << (c.classId - 1),
                    color: c.color,
                }))
                .sort((a, b) => a.name.localeCompare(b.name)),
        [classesQuery.data],
    )

    const filteredItemSets = useMemo(
        () => filterItems(classFilteredSets, setFilter),
        [classFilteredSets, setFilter],
    )
    const filteredSetItems = useMemo(() => {
        if (!setDetail?.items) return []
        return filterItems(setDetail.items, itemFilter)
    }, [setDetail, itemFilter])

    // Per-class set counts for the first column.
    const countForBit = (bit) =>
        bit === 0 ? itemSets.length : itemSets.filter((s) => (s.classMask & bit) !== 0).length
    const filteredClasses = useMemo(
        () => filterItems(classOptions, classFilter),
        [classOptions, classFilter],
    )

    // Selecting a class resets filters and clears the explicit set, so the detail
    // defaults to that class's first set (via effectiveSet).
    const selectClass = (bit) => {
        setClassBit(bit)
        setSetFilter('')
        setItemFilter('')
        setSelectedSet(null)
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
                        <span className="flex w-full justify-between">
                            <span>All Classes</span>
                            <span className="text-xs text-gray-600">({countForBit(0)})</span>
                        </span>
                    </ListItem>
                    {filteredClasses.map((c) => (
                        <ListItem
                            key={c.bit}
                            active={classBit === c.bit}
                            onClick={() => selectClass(c.bit)}
                        >
                            <span className="flex w-full justify-between">
                                <span style={{ color: c.color || undefined }}>{c.name}</span>
                                <span className="text-xs text-gray-600">
                                    ({countForBit(c.bit)})
                                </span>
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
                    {itemSetsQuery.isLoading && (
                        <div className="animate-pulse p-4 text-center italic text-wow-gold">
                            Loading sets...
                        </div>
                    )}
                    {filteredItemSets.map((set) => (
                        <ListItem
                            key={set.itemsetId}
                            active={effectiveSet?.itemsetId === set.itemsetId}
                            onClick={() => {
                                setSelectedSet(set)
                                setItemFilter('')
                            }}
                        >
                            <span className="flex w-full items-start justify-between gap-2">
                                <span className="whitespace-normal break-words text-left">
                                    {set.name}
                                </span>
                                <span className="mt-0.5 shrink-0 text-xs text-gray-600">
                                    ({set.itemCount})
                                </span>
                            </span>
                        </ListItem>
                    ))}
                </ScrollList>
            </SidebarPanel>

            {/* Set Details */}
            <ContentPanel>
                <SectionHeader
                    title={
                        effectiveSet
                            ? `${effectiveSet.name} (${filteredSetItems.length})`
                            : 'Select a Set'
                    }
                    placeholder="Filter items..."
                    onFilterChange={setItemFilter}
                />

                {setDetailQuery.isLoading && (
                    <div className="flex flex-1 animate-pulse items-center justify-center italic text-wow-gold">
                        Loading set details...
                    </div>
                )}

                {setDetail && !setDetailQuery.isLoading && (
                    <ScrollList className="space-y-2 p-2">
                        {/* Set Items */}
                        <div className="grid grid-cols-1 gap-1 xl:grid-cols-2">
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
                                        onClick={() => onNavigate?.('item', item.entry)}
                                        {...handlers}
                                    />
                                )
                            })}
                        </div>

                        {/* Set Bonuses */}
                        {setDetail.bonuses?.length > 0 && (
                            <div className="mt-4 rounded-lg border border-border-dark bg-bg-main p-4">
                                <h3 className="mb-3 text-sm font-bold uppercase tracking-wider text-wow-gold">
                                    Set Bonuses
                                </h3>
                                <div className="space-y-2">
                                    {setDetail.bonuses.map((bonus, idx) => (
                                        <div
                                            key={idx}
                                            className="flex items-center gap-2 text-sm text-wow-uncommon"
                                        >
                                            <span className="rounded bg-wow-uncommon/10 px-2 py-0.5 font-mono text-xs text-wow-uncommon">
                                                {bonus.threshold}pc
                                            </span>
                                            <span>
                                                {bonus.description || `Spell ID: ${bonus.spellId}`}
                                            </span>
                                        </div>
                                    ))}
                                </div>
                            </div>
                        )}
                    </ScrollList>
                )}

                {!effectiveSet && !itemSetsQuery.isLoading && (
                    <div className="flex flex-1 items-center justify-center italic text-gray-600">
                        Select an item set to view its items
                    </div>
                )}
            </ContentPanel>
        </>
    )
}

export default SetsTab
