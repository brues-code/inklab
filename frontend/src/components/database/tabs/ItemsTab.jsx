import { useMemo } from 'react'
import { useStickyState } from '../../../hooks/useStickyState'
import { ContentGrid, ContentPanel } from '../../ui'
import { useItemClasses, useItemStatTypes, useBrowseItems } from '../../../hooks/queries/items'
import { useTalentClasses } from '../../../hooks/queries/talents'
import { useIcon } from '../../../services/useImage'
import { getQualityColor, QUESTION_MARK_ICON } from '../../../utils/wow.ts'
import ItemBrowseFilters from './ItemBrowseFilters'

const PAGE_SIZE = 50

// The Items page is a Wowhead-style browser: a filter sidebar + a sortable,
// paginated table over the WHOLE item table (no mandatory class drill-down).
// All filtering/sorting/paging happens server-side via BrowseItems; the filter
// (including sort + offset) is the query key, so paging is cached and snappy.
const DEFAULT_FILTER = {
    query: '',
    quality: [],
    class: [],
    subClass: [],
    inventoryType: [],
    minLevel: 0,
    maxLevel: 0,
    minReqLevel: 0,
    maxReqLevel: 0,
    stats: [],
    usableByClass: 0,
    sources: [],
    sort: 'quality',
    sortDir: 'desc',
    limit: PAGE_SIZE,
    offset: 0,
}

// One table row. Split out so each can call useIcon for its own icon.
function ItemRow({ item, slotName, typeName, onClick, handlers }) {
    const icon = useIcon(item.iconPath || item.iconName)
    const color = getQualityColor(item.quality || 0)
    return (
        <tr
            onClick={onClick}
            {...handlers}
            className="cursor-pointer border-b border-white/5 hover:bg-white/5"
        >
            <td className="py-1 pl-2 pr-3">
                <div className="flex items-center gap-2">
                    <div
                        className="h-7 w-7 flex-shrink-0 overflow-hidden rounded border bg-black/40"
                        style={{ borderColor: color }}
                    >
                        <img
                            src={icon.src || QUESTION_MARK_ICON}
                            alt=""
                            className="h-full w-full object-cover"
                        />
                    </div>
                    <span className="truncate font-medium" style={{ color }}>
                        {item.name}
                    </span>
                </div>
            </td>
            <td className="px-3 text-right tabular-nums text-gray-300">{item.itemLevel || '—'}</td>
            <td className="px-3 text-right tabular-nums text-gray-400">
                {item.requiredLevel || '—'}
            </td>
            <td className="px-3 text-gray-400">{slotName || '—'}</td>
            <td className="px-3 text-gray-400">{typeName || '—'}</td>
        </tr>
    )
}

// A sortable column header. Clicking the active column flips direction;
// clicking a new one selects it (numeric columns default desc, text asc).
function SortHeader({ label, field, filter, onSort, align = 'left', defaultDir = 'asc' }) {
    const active = filter.sort === field
    const arrow = active ? (filter.sortDir === 'desc' ? ' ▼' : ' ▲') : ''
    return (
        <th
            onClick={() =>
                onSort(field, active ? (filter.sortDir === 'desc' ? 'asc' : 'desc') : defaultDir)
            }
            className={`cursor-pointer select-none px-3 py-2 font-semibold uppercase tracking-wide hover:text-white ${
                align === 'right' ? 'text-right' : 'text-left'
            } ${active ? 'text-wow-gold' : 'text-gray-500'}`}
        >
            {label}
            {arrow}
        </th>
    )
}

function ItemsTab({ tooltipHook, onNavigate }) {
    // Whole filter is sticky so the view survives leaving Database and Back.
    const [filter, setFilter] = useStickyState('items.browseFilter', DEFAULT_FILTER)

    const classesQuery = useItemClasses()
    const itemClasses = classesQuery.data || []
    const statTypes = useItemStatTypes().data || []
    const playerClasses = useTalentClasses().data || []

    const { data, isFetching } = useBrowseItems(filter)
    const items = data?.items || []
    const total = data?.totalCount || 0

    // Name lookups for the Slot / Type columns, built from the class hierarchy.
    const { slotNames, subClassNames } = useMemo(() => {
        const slotNames = {}
        const subClassNames = {}
        for (const c of itemClasses) {
            for (const sc of c.subClasses || []) {
                subClassNames[`${c.class}:${sc.subClass}`] = sc.name
                for (const sl of sc.inventorySlots || []) {
                    if (sl.inventoryType > 0 && !slotNames[sl.inventoryType]) {
                        slotNames[sl.inventoryType] = sl.name
                    }
                }
            }
        }
        return { slotNames, subClassNames }
    }, [itemClasses])

    // Merge a partial filter patch; any change but paging resets to page 1.
    const update = (patch) =>
        setFilter((prev) => ({
            ...prev,
            ...patch,
            offset: 'offset' in patch ? patch.offset : 0,
        }))

    const onSort = (sort, sortDir) => update({ sort, sortDir })

    const pageSize = filter.limit || PAGE_SIZE
    const page = Math.floor(filter.offset / pageSize) + 1
    const totalPages = Math.max(1, Math.ceil(total / pageSize))
    const goTo = (p) => update({ offset: (Math.max(1, Math.min(p, totalPages)) - 1) * pageSize })

    const rangeStart = total === 0 ? 0 : filter.offset + 1
    const rangeEnd = Math.min(filter.offset + items.length, total)

    // Page-size options; capped at the backend's max (200).
    const PAGE_SIZES = [25, 50, 100, 200]

    return (
        <ContentGrid columns="300px 1fr">
            <ItemBrowseFilters
                filter={filter}
                onChange={update}
                onReset={() => setFilter(DEFAULT_FILTER)}
                itemClasses={itemClasses}
                statTypes={statTypes}
                playerClasses={playerClasses}
            />

            <ContentPanel>
                {/* Result summary */}
                <div className="flex items-center justify-between border-b border-white/10 px-3 py-2 text-xs text-gray-400">
                    <span>
                        {total.toLocaleString()} item{total === 1 ? '' : 's'}
                        {isFetching && <span className="ml-2 animate-pulse text-wow-gold">…</span>}
                    </span>
                    <span>
                        {rangeStart.toLocaleString()}–{rangeEnd.toLocaleString()}
                    </span>
                </div>

                {/* Table */}
                <div className="min-h-0 flex-1 overflow-y-auto">
                    <table className="w-full table-fixed border-collapse text-sm">
                        <colgroup>
                            <col />
                            <col className="w-20" />
                            <col className="w-20" />
                            <col className="w-32" />
                            <col className="w-32" />
                        </colgroup>
                        <thead className="sticky top-0 z-10 bg-bg-dark text-[11px]">
                            <tr className="border-b border-white/10">
                                <SortHeader
                                    label="Name"
                                    field="name"
                                    filter={filter}
                                    onSort={onSort}
                                    defaultDir="asc"
                                />
                                <SortHeader
                                    label="iLvl"
                                    field="itemLevel"
                                    filter={filter}
                                    onSort={onSort}
                                    align="right"
                                    defaultDir="desc"
                                />
                                <SortHeader
                                    label="Req"
                                    field="requiredLevel"
                                    filter={filter}
                                    onSort={onSort}
                                    align="right"
                                    defaultDir="desc"
                                />
                                <th className="px-3 py-2 text-left font-semibold uppercase tracking-wide text-gray-500">
                                    Slot
                                </th>
                                <th className="px-3 py-2 text-left font-semibold uppercase tracking-wide text-gray-500">
                                    Type
                                </th>
                            </tr>
                        </thead>
                        <tbody>
                            {items.map((item) => {
                                const id = item.entry || item.id
                                return (
                                    <ItemRow
                                        key={id}
                                        item={item}
                                        slotName={slotNames[item.inventoryType]}
                                        typeName={subClassNames[`${item.class}:${item.subClass}`]}
                                        onClick={() => onNavigate?.('item', id)}
                                        handlers={tooltipHook?.getItemHandlers?.(id)}
                                    />
                                )
                            })}
                        </tbody>
                    </table>

                    {items.length === 0 && !isFetching && (
                        <div className="flex h-40 items-center justify-center italic text-gray-600">
                            No items match these filters
                        </div>
                    )}
                </div>

                {/* Pagination */}
                <div className="relative flex items-center justify-center gap-3 border-t border-white/10 px-3 py-2 text-xs">
                    <button
                        onClick={() => goTo(page - 1)}
                        disabled={page <= 1}
                        className="rounded border border-gray-700 px-3 py-1 text-gray-300 hover:border-gray-500 disabled:cursor-not-allowed disabled:opacity-40"
                    >
                        ◀ Prev
                    </button>
                    <span className="text-gray-400">
                        Page {page.toLocaleString()} / {totalPages.toLocaleString()}
                    </span>
                    <button
                        onClick={() => goTo(page + 1)}
                        disabled={page >= totalPages}
                        className="rounded border border-gray-700 px-3 py-1 text-gray-300 hover:border-gray-500 disabled:cursor-not-allowed disabled:opacity-40"
                    >
                        Next ▶
                    </button>

                    {/* Page size — absolute so it doesn't shift the centered controls */}
                    <label className="absolute right-3 flex items-center gap-1 text-gray-500">
                        Per page
                        <select
                            value={pageSize}
                            onChange={(e) => update({ limit: Number(e.target.value) })}
                            className="rounded border border-gray-700 bg-black/40 px-1 py-0.5 text-gray-300 focus:border-wow-gold focus:outline-none"
                        >
                            {PAGE_SIZES.map((n) => (
                                <option key={n} value={n}>
                                    {n}
                                </option>
                            ))}
                        </select>
                    </label>
                </div>
            </ContentPanel>
        </ContentGrid>
    )
}

export default ItemsTab
