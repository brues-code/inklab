import React from 'react'
import { SidebarPanel, SectionHeader, ScrollList } from '../../ui'
import { getQualityColor } from '../../../utils/wow'
import { useItemStatTypes } from '../../../hooks/queries/items'

// Column-based stats — these live in dedicated item columns (not the stat_typeN
// slots), so they're fixed rather than derived from the data.
const OTHER_STAT_OPTIONS = [
    { value: '', label: '- Select -' },
    { value: 'armor', label: 'Armor' },
    { value: 'fire_res', label: 'Fire Resistance' },
    { value: 'frost_res', label: 'Frost Resistance' },
    { value: 'nature_res', label: 'Nature Resistance' },
    { value: 'shadow_res', label: 'Shadow Resistance' },
    { value: 'arcane_res', label: 'Arcane Resistance' },
]

function FilterSection({ title, children }) {
    return (
        <div className="mb-4 px-2">
            <div className="text-xs font-bold text-wow-gold uppercase mb-1">{title}</div>
            <div className="space-y-2">
                {children}
            </div>
        </div>
    )
}

function RangeInput({ label, minVal, maxVal, onMinChange, onMaxChange }) {
    return (
        <div className="flex flex-col gap-1">
            {label && <span className="text-gray-400 text-xs">{label}</span>}
            <div className="flex items-center gap-2">
                <input
                    type="number"
                    value={minVal}
                    onChange={(e) => onMinChange(e.target.value)}
                    placeholder="0"
                    className="w-full bg-black/40 border border-gray-700 rounded text-xs px-2 py-1 text-white focus:border-wow-gold outline-none"
                    min="0"
                    max="100"
                />
                <span className="text-gray-500">-</span>
                <input
                    type="number"
                    value={maxVal}
                    onChange={(e) => onMaxChange(e.target.value)}
                    placeholder="60"
                    className="w-full bg-black/40 border border-gray-700 rounded text-xs px-2 py-1 text-white focus:border-wow-gold outline-none"
                    min="0"
                    max="100"
                />
            </div>
        </div>
    )
}

function StatRow({ stat, minValue, maxValue, onStatChange, onMinValueChange, onMaxValueChange, onRemove, options }) {
    return (
        <div className="flex gap-1 items-center">
            <select
                value={stat}
                onChange={(e) => onStatChange(e.target.value)}
                className="flex-1 min-w-0 bg-black/40 border border-gray-700 rounded text-xs px-1 py-1 text-gray-300 focus:border-wow-gold outline-none"
            >
                {options.map((opt, idx) => (
                    <option
                        key={opt.value}
                        value={opt.value}
                        style={{ backgroundColor: idx % 2 === 0 ? '#181818' : '#242424', color: '#e0e0e0' }}
                    >
                        {opt.label}
                    </option>
                ))}
            </select>
            <input
                type="number"
                value={minValue}
                onChange={(e) => onMinValueChange(e.target.value)}
                placeholder="Min"
                className="w-14 bg-black/40 border border-gray-700 rounded text-xs px-1 py-1 text-white focus:border-wow-gold outline-none"
                step="0.1"
                min="0"
            />
            <span className="text-gray-500 text-xs">-</span>
            <input
                type="number"
                value={maxValue}
                onChange={(e) => onMaxValueChange(e.target.value)}
                placeholder="Max"
                className="w-14 bg-black/40 border border-gray-700 rounded text-xs px-1 py-1 text-white focus:border-wow-gold outline-none"
                step="0.1"
                min="0"
            />
            {onRemove && (
                <button
                    type="button"
                    onClick={onRemove}
                    aria-label="Remove stat"
                    className="shrink-0 w-5 h-5 flex items-center justify-center rounded text-gray-500 hover:text-red-400 hover:bg-red-900/30 transition-colors"
                >
                    ×
                </button>
            )}
        </div>
    )
}

// StatFilterSection renders a dynamic, add/remove list of StatRows backed by
// filters[fieldKey]. It always shows at least one row; the remove button only
// appears once there's more than one.
function StatFilterSection({ title, fieldKey, rows, options, onUpdate, onAdd, onRemove }) {
    const list = rows && rows.length ? rows : [{ stat: '', minVal: '', maxVal: '' }]
    return (
        <FilterSection title={title}>
            {list.map((row, i) => (
                <StatRow
                    key={i}
                    stat={row?.stat || ''}
                    minValue={row?.minVal || ''}
                    maxValue={row?.maxVal || ''}
                    onStatChange={(v) => onUpdate(fieldKey, i, 'stat', v)}
                    onMinValueChange={(v) => onUpdate(fieldKey, i, 'minVal', v)}
                    onMaxValueChange={(v) => onUpdate(fieldKey, i, 'maxVal', v)}
                    onRemove={list.length > 1 ? () => onRemove(fieldKey, i) : undefined}
                    options={options}
                />
            ))}
            <button
                type="button"
                onClick={() => onAdd(fieldKey)}
                className="text-[11px] text-wow-gold/80 hover:text-wow-gold transition-colors"
            >
                + Add stat
            </button>
        </FilterSection>
    )
}

export default function ItemFilters({ filters, onChange, onSearch, onReset }) {
    // Stat dropdown options come from the stat types actually present in the data
    // (id -> name), so the filter adapts to whatever stats the items use.
    const { data: statTypes } = useItemStatTypes()
    const statOptions = [
        { value: '', label: '- Select -' },
        ...(statTypes || []).map(s => ({ value: String(s.id), label: s.name })),
    ]

    const updateFilter = (key, value) => {
        onChange({ ...filters, [key]: value })
    }

    // Stat rows are dynamic per section (key = 'stats' | 'otherStats').
    const updateStat = (key, index, field, value) => {
        const rows = [...(filters[key] || [])]
        if (!rows[index]) rows[index] = { stat: '', minVal: '', maxVal: '' }
        rows[index] = { ...rows[index], [field]: value }
        onChange({ ...filters, [key]: rows })
    }

    const addStat = (key) => {
        const rows = [...(filters[key] || [])]
        // One implicit row is always shown, so grow past it to actually add one.
        const target = Math.max(1, rows.length) + 1
        while (rows.length < target) rows.push({ stat: '', minVal: '', maxVal: '' })
        onChange({ ...filters, [key]: rows })
    }

    const removeStat = (key, index) => {
        const rows = [...(filters[key] || [])]
        rows.splice(index, 1)
        onChange({ ...filters, [key]: rows })
    }

    const handleReset = () => {
        if (onReset) onReset()
    }

    return (
        <SidebarPanel>
             <SectionHeader 
                title="Filters" 
                noSearch={true}
            />
            <ScrollList className="p-2 space-y-4">
                {/* Item Level */}
                <FilterSection title="Item Level">
                    <RangeInput
                        minVal={filters.minIlvl || ''}
                        maxVal={filters.maxIlvl || ''}
                        onMinChange={(v) => onChange({...filters, minIlvl: v})}
                        onMaxChange={(v) => onChange({...filters, maxIlvl: v})}
                    />
                </FilterSection>

                {/* Required Level */}
                <FilterSection title="Required Level">
                    <RangeInput
                        minVal={filters.minRl || ''}
                        maxVal={filters.maxRl || ''}
                        onMinChange={(v) => onChange({...filters, minRl: v})}
                        onMaxChange={(v) => onChange({...filters, maxRl: v})}
                    />
                </FilterSection>

                {/* Quality */}
                <FilterSection title="Quality">
                    <div className="flex flex-wrap gap-1">
                        {['Poor', 'Common', 'Uncommon', 'Rare', 'Epic', 'Legendary'].map((q, i) => {
                            const currentQualities = Array.isArray(filters.quality) ? filters.quality : []
                            const isSelected = currentQualities.includes(i)
                            const color = getQualityColor(i)
                            const isHighContrast = ['Rare', 'Epic'].includes(q)
                            
                            return (
                                <button
                                    key={q}
                                    onClick={() => {
                                        const newQualities = isSelected
                                            ? currentQualities.filter(q => q !== i)
                                            : [...currentQualities, i]
                                        onChange({...filters, quality: newQualities})
                                    }}
                                    className={`
                                        px-2 py-1 text-xs rounded border transition-all duration-200 flex-1 min-w-[45%] text-center font-medium
                                        ${isSelected 
                                            ? (isHighContrast ? 'text-white' : 'text-black') 
                                            : 'bg-black/40 border-gray-700 hover:bg-black/60'}
                                    `}
                                    style={{
                                        backgroundColor: isSelected ? color : undefined,
                                        borderColor: isSelected ? color : undefined,
                                        color: isSelected ? undefined : color,
                                        textShadow: isSelected && isHighContrast ? '0 1px 2px rgba(0,0,0,0.8)' : 'none',
                                        boxShadow: isSelected ? `0 0 15px ${color}66` : 'none'
                                    }}
                                >
                                    {q}
                                </button>
                            )
                        })}
                    </div>
                </FilterSection>

                {/* Basic Stats (stat_typeN slots, dynamic from the data) */}
                <StatFilterSection
                    title="Stats (Min-Max)"
                    fieldKey="stats"
                    rows={filters.stats}
                    options={statOptions}
                    onUpdate={updateStat}
                    onAdd={addStat}
                    onRemove={removeStat}
                />

                {/* Other Stats (armor / resistances — dedicated columns) */}
                <StatFilterSection
                    title="Other Stats"
                    fieldKey="otherStats"
                    rows={filters.otherStats}
                    options={OTHER_STAT_OPTIONS}
                    onUpdate={updateStat}
                    onAdd={addStat}
                    onRemove={removeStat}
                />
                
                {/* Reset Button */}
                <div className="pt-2 flex justify-center">
                    <button 
                        onClick={handleReset}
                        className="w-1/2 bg-red-900/30 border border-red-800 text-red-400 hover:bg-red-800/50 hover:text-white text-xs py-2 rounded transition-colors uppercase font-semibold"
                    >
                        Reset Filters
                    </button>
                </div>
            </ScrollList>
        </SidebarPanel>
    )
}
