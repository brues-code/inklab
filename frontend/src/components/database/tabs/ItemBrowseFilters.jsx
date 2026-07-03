import { SidebarPanel } from '../../ui'
import { getQualityColor } from '../../../utils/wow.ts'

// Wowhead-style filter sidebar for the Items page. Purely controlled: it reads
// the current `filter` (a models.SearchFilter shape) and emits partial patches
// via onChange — the parent merges them and resets paging. Reference data (item
// class hierarchy, stat types, player classes) is passed in.

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
    { key: 'object', name: 'Object' },
    { key: 'container', name: 'Container' },
    { key: 'vendor', name: 'Vendor' },
    { key: 'quest', name: 'Quest' },
    { key: 'crafted', name: 'Crafted' },
    { key: 'disenchant', name: 'Disenchant' },
]

const BONDINGS = [
    { id: 1, name: 'BoP' },
    { id: 2, name: 'BoE' },
    { id: 3, name: 'BoU' },
    { id: 4, name: 'Quest' },
]

// Damage schools (dmg_type1). 0 = any (Physical, the common default).
const DAMAGE_SCHOOLS = [
    { id: 0, name: 'Any' },
    { id: 1, name: 'Holy' },
    { id: 2, name: 'Fire' },
    { id: 3, name: 'Nature' },
    { id: 4, name: 'Frost' },
    { id: 5, name: 'Shadow' },
    { id: 6, name: 'Arcane' },
]

const RESIST_SCHOOLS = [
    { id: 1, name: 'Holy' },
    { id: 2, name: 'Fire' },
    { id: 3, name: 'Nature' },
    { id: 4, name: 'Frost' },
    { id: 5, name: 'Shadow' },
    { id: 6, name: 'Arcane' },
]

const toggle = (arr, val) =>
    (arr || []).includes(val) ? arr.filter((v) => v !== val) : [...(arr || []), val]

const SELECT_CLS =
    'w-full rounded border border-gray-700 bg-black/40 px-2 py-1 text-xs text-white ' +
    'focus:border-wow-gold focus:outline-none'
const NUM_CLS =
    'w-full rounded border border-gray-700 bg-black/40 px-2 py-1 text-xs text-white ' +
    'focus:border-wow-gold focus:outline-none [appearance:textfield] ' +
    '[&::-webkit-outer-spin-button]:appearance-none [&::-webkit-inner-spin-button]:appearance-none'

const num = (v) => {
    const n = parseInt(v, 10)
    return isNaN(n) || n < 0 ? 0 : n
}
const fnum = (v) => {
    const n = parseFloat(v)
    return isNaN(n) || n < 0 ? 0 : n
}

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

// A bold section divider.
function Group({ title, children }) {
    return (
        <div className="space-y-3 border-t border-white/10 pt-3">
            <div className="text-xs font-bold uppercase tracking-wide text-wow-gold/80">
                {title}
            </div>
            {children}
        </div>
    )
}

// A min–max pair of number inputs.
function Range({ label, min, max, onMin, onMax, step }) {
    return (
        <Field label={label}>
            <div className="flex items-center gap-1">
                <input
                    type="number"
                    min="0"
                    step={step}
                    value={min || ''}
                    onChange={(e) => onMin(e.target.value)}
                    placeholder="min"
                    className={NUM_CLS}
                />
                <span className="text-gray-600">–</span>
                <input
                    type="number"
                    min="0"
                    step={step}
                    value={max || ''}
                    onChange={(e) => onMax(e.target.value)}
                    placeholder="max"
                    className={NUM_CLS}
                />
            </div>
        </Field>
    )
}

// A checkbox toggle row.
function Toggle({ label, checked, onChange }) {
    return (
        <label className="flex cursor-pointer items-center gap-2 text-xs text-gray-300">
            <input
                type="checkbox"
                checked={!!checked}
                onChange={(e) => onChange(e.target.checked)}
                className="accent-wow-gold"
            />
            {label}
        </label>
    )
}

export default function ItemBrowseFilters({
    filter,
    onChange,
    onReset,
    itemClasses = [],
    statTypes = [],
    playerClasses = [],
}) {
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
                                    onClick={() =>
                                        onChange({ quality: toggle(filter.quality, q.id) })
                                    }
                                    className={`rounded border px-1.5 py-0.5 text-[11px] transition-colors ${
                                        on
                                            ? 'border-current'
                                            : 'border-transparent bg-black/30 hover:bg-white/5'
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

                <Range
                    label="Item level"
                    min={filter.minLevel}
                    max={filter.maxLevel}
                    onMin={(v) => onChange({ minLevel: num(v) })}
                    onMax={(v) => onChange({ maxLevel: num(v) })}
                />
                <Range
                    label="Required level"
                    min={filter.minReqLevel}
                    max={filter.maxReqLevel}
                    onMin={(v) => onChange({ minReqLevel: num(v) })}
                    onMax={(v) => onChange({ maxReqLevel: num(v) })}
                />

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

                {/* ---- Item properties ---- */}
                <Group title="Properties">
                    <Field label="Binding">
                        <div className="flex flex-wrap gap-1">
                            {BONDINGS.map((b) => {
                                const on = (filter.bonding || []).includes(b.id)
                                return (
                                    <button
                                        key={b.id}
                                        onClick={() =>
                                            onChange({ bonding: toggle(filter.bonding, b.id) })
                                        }
                                        className={`rounded border px-2 py-0.5 text-[11px] transition-colors ${
                                            on
                                                ? 'border-wow-gold bg-wow-gold/20 text-wow-gold'
                                                : 'border-gray-700 bg-black/30 text-gray-400 hover:text-white'
                                        }`}
                                    >
                                        {b.name}
                                    </button>
                                )
                            })}
                        </div>
                    </Field>
                    <Toggle
                        label="Unique"
                        checked={filter.onlyUnique}
                        onChange={(v) => onChange({ onlyUnique: v })}
                    />
                    <Toggle
                        label="Class-specific"
                        checked={filter.classSpecific}
                        onChange={(v) => onChange({ classSpecific: v })}
                    />
                    <Toggle
                        label="Race-specific"
                        checked={filter.raceSpecific}
                        onChange={(v) => onChange({ raceSpecific: v })}
                    />
                    <Toggle
                        label="Starts a quest"
                        checked={filter.startsQuest}
                        onChange={(v) => onChange({ startsQuest: v })}
                    />
                    <Toggle
                        label="Has on-use / equip effect"
                        checked={filter.hasEffect}
                        onChange={(v) => onChange({ hasEffect: v })}
                    />
                    <Toggle
                        label="Has random suffix"
                        checked={filter.hasRandomSuffix}
                        onChange={(v) => onChange({ hasRandomSuffix: v })}
                    />
                </Group>

                {/* ---- Requirements & economy ---- */}
                <Group title="Requirements & economy">
                    <Toggle
                        label="Requires a profession"
                        checked={filter.requiresProf}
                        onChange={(v) => onChange({ requiresProf: v })}
                    />
                    <Range
                        label="Required skill rank"
                        min={filter.minSkillRank}
                        max={filter.maxSkillRank}
                        onMin={(v) => onChange({ minSkillRank: num(v) })}
                        onMax={(v) => onChange({ maxSkillRank: num(v) })}
                    />
                    <Toggle
                        label="Requires reputation"
                        checked={filter.requiresRep}
                        onChange={(v) => onChange({ requiresRep: v })}
                    />
                    <Range
                        label="Buy price (copper)"
                        min={filter.minBuyPrice}
                        max={filter.maxBuyPrice}
                        onMin={(v) => onChange({ minBuyPrice: num(v) })}
                        onMax={(v) => onChange({ maxBuyPrice: num(v) })}
                    />
                    <Range
                        label="Sell price (copper)"
                        min={filter.minSellPrice}
                        max={filter.maxSellPrice}
                        onMin={(v) => onChange({ minSellPrice: num(v) })}
                        onMax={(v) => onChange({ maxSellPrice: num(v) })}
                    />
                    <Range
                        label="Durability"
                        min={filter.minDurability}
                        max={filter.maxDurability}
                        onMin={(v) => onChange({ minDurability: num(v) })}
                        onMax={(v) => onChange({ maxDurability: num(v) })}
                    />
                </Group>

                {/* ---- Weapon & armor stats ---- */}
                <Group title="Weapon & armor">
                    <Range
                        label="Weapon DPS"
                        step="0.1"
                        min={filter.minDps}
                        max={filter.maxDps}
                        onMin={(v) => onChange({ minDps: fnum(v) })}
                        onMax={(v) => onChange({ maxDps: fnum(v) })}
                    />
                    <Range
                        label="Weapon speed"
                        step="0.1"
                        min={filter.minSpeed}
                        max={filter.maxSpeed}
                        onMin={(v) => onChange({ minSpeed: fnum(v) })}
                        onMax={(v) => onChange({ maxSpeed: fnum(v) })}
                    />
                    <Field label="Damage school">
                        <select
                            value={filter.damageSchool || 0}
                            onChange={(e) => onChange({ damageSchool: Number(e.target.value) })}
                            className={SELECT_CLS}
                        >
                            {DAMAGE_SCHOOLS.map((s) => (
                                <option key={s.id} value={s.id}>
                                    {s.name}
                                </option>
                            ))}
                        </select>
                    </Field>
                    <Range
                        label="Armor"
                        min={filter.minArmor}
                        max={filter.maxArmor}
                        onMin={(v) => onChange({ minArmor: num(v) })}
                        onMax={(v) => onChange({ maxArmor: num(v) })}
                    />
                    <Range
                        label="Block"
                        min={filter.minBlock}
                        max={filter.maxBlock}
                        onMin={(v) => onChange({ minBlock: num(v) })}
                        onMax={(v) => onChange({ maxBlock: num(v) })}
                    />
                    <Field label="Resistances (min)">
                        <div className="space-y-1">
                            {(filter.resists || []).map((row, i) => (
                                <div key={i} className="flex items-center gap-1">
                                    <select
                                        value={row.school}
                                        onChange={(e) => {
                                            const resists = [...filter.resists]
                                            resists[i] = { ...row, school: Number(e.target.value) }
                                            onChange({ resists })
                                        }}
                                        className={SELECT_CLS}
                                    >
                                        {RESIST_SCHOOLS.map((s) => (
                                            <option key={s.id} value={s.id}>
                                                {s.name}
                                            </option>
                                        ))}
                                    </select>
                                    <input
                                        type="number"
                                        min="0"
                                        value={row.min || ''}
                                        onChange={(e) => {
                                            const resists = [...filter.resists]
                                            resists[i] = { ...row, min: num(e.target.value) }
                                            onChange({ resists })
                                        }}
                                        placeholder="min"
                                        className={`${NUM_CLS} w-16 flex-shrink-0`}
                                    />
                                    <button
                                        onClick={() =>
                                            onChange({
                                                resists: filter.resists.filter((_, j) => j !== i),
                                            })
                                        }
                                        className="flex-shrink-0 px-1 text-gray-500 hover:text-red-400"
                                        title="Remove"
                                    >
                                        ✕
                                    </button>
                                </div>
                            ))}
                            <button
                                onClick={() =>
                                    onChange({
                                        resists: [...(filter.resists || []), { school: 2, min: 0 }],
                                    })
                                }
                                className="w-full rounded border border-dashed border-gray-700 px-2 py-1 text-[11px] text-gray-400 hover:border-gray-500 hover:text-white"
                            >
                                + Add resistance
                            </button>
                        </div>
                    </Field>
                </Group>

                {/* ---- Stats ---- */}
                <Group title="Stats (min)">
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
                </Group>

                {/* ---- Source ---- */}
                <Group title="Source">
                    <div className="flex flex-wrap gap-1">
                        {SOURCES.map((s) => {
                            const on = (filter.sources || []).includes(s.key)
                            return (
                                <button
                                    key={s.key}
                                    onClick={() =>
                                        onChange({ sources: toggle(filter.sources, s.key) })
                                    }
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
                </Group>
            </div>
        </SidebarPanel>
    )
}
