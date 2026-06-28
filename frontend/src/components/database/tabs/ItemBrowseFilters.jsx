import { SidebarPanel } from '../../ui'
import { getQualityColor } from '../../../utils/wow.ts'

// Wowhead-style filter sidebar for the Items page. It's purely controlled: it
// reads the current `filter` (a models.SearchFilter shape) and emits partial
// patches via onChange — the parent merges them and resets paging. Reference
// data (item class hierarchy, stat types, player classes) is passed in so this
// stays a dumb presentational component.

const QUALITIES = [
    { id: 0, name: 'Poor' },
    { id: 1, name: 'Common' },
    { id: 2, name: 'Uncommon' },
    { id: 3, name: 'Rare' },
    { id: 4, name: 'Epic' },
    { id: 5, name: 'Legendary' },
]

const SOURCES = [
    { key: 'drop', name: 'Drops' },
    { key: 'vendor', name: 'Vendor' },
    { key: 'quest', name: 'Quest' },
    { key: 'crafted', name: 'Crafted' },
]

// Toggle membership of `val` in an array, returning a new array.
const toggle = (arr, val) =>
    (arr || []).includes(val) ? arr.filter((v) => v !== val) : [...(arr || []), val]

const SELECT_CLS =
    'w-full rounded border border-gray-700 bg-black/40 px-2 py-1 text-xs text-white ' +
    'focus:border-wow-gold focus:outline-none'
const NUM_CLS =
    'w-full rounded border border-gray-700 bg-black/40 px-2 py-1 text-xs text-white ' +
    'focus:border-wow-gold focus:outline-none [appearance:textfield] ' +
    '[&::-webkit-outer-spin-button]:appearance-none [&::-webkit-inner-spin-button]:appearance-none'

// One labelled filter block.
function Field({ label, children }) {
    return (
        <div className="space-y-1">
            <div className="text-[11px] font-semibold uppercase tracking-wide text-gray-500">
                {label}
            </div>
            {children}
        </div>
    )
}

// Parse a number input into a value (0 = unset, the backend ignores 0).
const num = (v) => {
    const n = parseInt(v, 10)
    return isNaN(n) || n < 0 ? 0 : n
}

export default function ItemBrowseFilters({
    filter,
    onChange,
    onReset,
    itemClasses = [],
    statTypes = [],
    playerClasses = [],
}) {
    // Single-select class/subclass/slot are stored as 0- or 1-length arrays so
    // they map straight onto the backend's IN(...) filters.
    const selClass = filter.class?.[0] ?? null
    const selSubClass = filter.subClass?.[0] ?? null
    const selSlot = filter.inventoryType?.[0] ?? null

    const classObj = itemClasses.find((c) => c.class === selClass) || null
    const subClasses = classObj?.subClasses || []
    const subObj = subClasses.find((s) => s.subClass === selSubClass) || null
    const slots = subObj?.inventorySlots || []

    return (
        <SidebarPanel>
            <div className="space-y-3 overflow-y-auto p-3">
                <div className="flex items-center justify-between">
                    <span className="text-sm font-bold text-wow-gold">Filters</span>
                    <button
                        onClick={onReset}
                        className="rounded border border-gray-700 px-2 py-0.5 text-[11px] text-gray-400 hover:border-gray-500 hover:text-white"
                    >
                        Reset
                    </button>
                </div>

                {/* Name / ID search */}
                <Field label="Name or ID">
                    <input
                        type="text"
                        value={filter.query || ''}
                        onChange={(e) => onChange({ query: e.target.value })}
                        placeholder="Search…"
                        className={SELECT_CLS}
                    />
                </Field>

                {/* Quality */}
                <Field label="Quality">
                    <div className="flex flex-wrap gap-1">
                        {QUALITIES.map((q) => {
                            const on = (filter.quality || []).includes(q.id)
                            return (
                                <button
                                    key={q.id}
                                    onClick={() => onChange({ quality: toggle(filter.quality, q.id) })}
                                    className={`rounded border px-1.5 py-0.5 text-[11px] transition-colors ${
                                        on ? 'border-current' : 'border-transparent bg-black/30 hover:bg-white/5'
                                    }`}
                                    style={{ color: getQualityColor(q.id) }}
                                >
                                    {q.name}
                                </button>
                            )
                        })}
                    </div>
                </Field>

                {/* Class / subclass / slot */}
                <Field label="Type">
                    <select
                        value={selClass ?? ''}
                        onChange={(e) =>
                            onChange({
                                class: e.target.value === '' ? [] : [Number(e.target.value)],
                                subClass: [],
                                inventoryType: [],
                            })
                        }
                        className={SELECT_CLS}
                    >
                        <option value="">Any type</option>
                        {itemClasses.map((c) => (
                            <option key={c.class} value={c.class}>
                                {c.name}
                            </option>
                        ))}
                    </select>
                    {subClasses.length > 0 && (
                        <select
                            value={selSubClass ?? ''}
                            onChange={(e) =>
                                onChange({
                                    subClass: e.target.value === '' ? [] : [Number(e.target.value)],
                                    inventoryType: [],
                                })
                            }
                            className={SELECT_CLS}
                        >
                            <option value="">Any subtype</option>
                            {subClasses.map((s) => (
                                <option key={s.subClass} value={s.subClass}>
                                    {s.name}
                                </option>
                            ))}
                        </select>
                    )}
                    {slots.length > 0 && (
                        <select
                            value={selSlot ?? ''}
                            onChange={(e) =>
                                onChange({
                                    inventoryType:
                                        e.target.value === '' ? [] : [Number(e.target.value)],
                                })
                            }
                            className={SELECT_CLS}
                        >
                            <option value="">Any slot</option>
                            {slots.map((sl) => (
                                <option key={sl.inventoryType} value={sl.inventoryType}>
                                    {sl.name}
                                </option>
                            ))}
                        </select>
                    )}
                </Field>

                {/* Item level range */}
                <Field label="Item level">
                    <div className="flex items-center gap-1">
                        <input
                            type="number"
                            min="0"
                            value={filter.minLevel || ''}
                            onChange={(e) => onChange({ minLevel: num(e.target.value) })}
                            placeholder="min"
                            className={NUM_CLS}
                        />
                        <span className="text-gray-600">–</span>
                        <input
                            type="number"
                            min="0"
                            value={filter.maxLevel || ''}
                            onChange={(e) => onChange({ maxLevel: num(e.target.value) })}
                            placeholder="max"
                            className={NUM_CLS}
                        />
                    </div>
                </Field>

                {/* Required level range */}
                <Field label="Required level">
                    <div className="flex items-center gap-1">
                        <input
                            type="number"
                            min="0"
                            value={filter.minReqLevel || ''}
                            onChange={(e) => onChange({ minReqLevel: num(e.target.value) })}
                            placeholder="min"
                            className={NUM_CLS}
                        />
                        <span className="text-gray-600">–</span>
                        <input
                            type="number"
                            min="0"
                            value={filter.maxReqLevel || ''}
                            onChange={(e) => onChange({ maxReqLevel: num(e.target.value) })}
                            placeholder="max"
                            className={NUM_CLS}
                        />
                    </div>
                </Field>

                {/* Usable by class */}
                <Field label="Usable by class">
                    <select
                        value={filter.usableByClass || 0}
                        onChange={(e) => onChange({ usableByClass: Number(e.target.value) })}
                        className={SELECT_CLS}
                    >
                        <option value={0}>Any class</option>
                        {playerClasses.map((c) => (
                            <option key={c.classId} value={c.classId}>
                                {c.name || c.class}
                            </option>
                        ))}
                    </select>
                </Field>

                {/* Source */}
                <Field label="Source">
                    <div className="flex flex-wrap gap-1">
                        {SOURCES.map((s) => {
                            const on = (filter.sources || []).includes(s.key)
                            return (
                                <button
                                    key={s.key}
                                    onClick={() => onChange({ sources: toggle(filter.sources, s.key) })}
                                    className={`rounded border px-2 py-0.5 text-[11px] transition-colors ${
                                        on
                                            ? 'border-wow-gold bg-wow-gold/20 text-wow-gold'
                                            : 'border-gray-700 bg-black/30 text-gray-400 hover:text-white'
                                    }`}
                                >
                                    {s.name}
                                </button>
                            )
                        })}
                    </div>
                </Field>

                {/* Stat requirements */}
                <Field label="Stats (min)">
                    <div className="space-y-1">
                        {(filter.stats || []).map((row, i) => (
                            <div key={i} className="flex items-center gap-1">
                                <select
                                    value={row.stat}
                                    onChange={(e) => {
                                        const stats = [...filter.stats]
                                        stats[i] = { ...row, stat: Number(e.target.value) }
                                        onChange({ stats })
                                    }}
                                    className={SELECT_CLS}
                                >
                                    {statTypes.map((st) => (
                                        <option key={st.id} value={st.id}>
                                            {st.name}
                                        </option>
                                    ))}
                                </select>
                                <input
                                    type="number"
                                    min="0"
                                    value={row.min || ''}
                                    onChange={(e) => {
                                        const stats = [...filter.stats]
                                        stats[i] = { ...row, min: num(e.target.value) }
                                        onChange({ stats })
                                    }}
                                    placeholder="min"
                                    className={`${NUM_CLS} w-16 flex-shrink-0`}
                                />
                                <button
                                    onClick={() =>
                                        onChange({ stats: filter.stats.filter((_, j) => j !== i) })
                                    }
                                    className="flex-shrink-0 px-1 text-gray-500 hover:text-red-400"
                                    title="Remove"
                                >
                                    ✕
                                </button>
                            </div>
                        ))}
                        {statTypes.length > 0 && (
                            <button
                                onClick={() =>
                                    onChange({
                                        stats: [
                                            ...(filter.stats || []),
                                            { stat: statTypes[0].id, min: 0 },
                                        ],
                                    })
                                }
                                className="w-full rounded border border-dashed border-gray-700 px-2 py-1 text-[11px] text-gray-400 hover:border-gray-500 hover:text-white"
                            >
                                + Add stat
                            </button>
                        )}
                    </div>
                </Field>
            </div>
        </SidebarPanel>
    )
}
