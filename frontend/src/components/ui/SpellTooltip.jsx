import React from 'react'
import { getSchoolName, getSchoolColor } from '../../utils/wow'

const POWER_TYPES = { 0: 'Mana', 1: 'Rage', 2: 'Focus', 3: 'Energy', 4: 'Happiness' }

/**
 * WoW-style spell tooltip. Renders a compact view of a SpellDetail (the same
 * payload the spell detail view uses), for the shared hover layer.
 */
const SpellTooltip = ({ spell, style }) => {
    if (!spell) {
        return (
            <div
                className="flex flex-col gap-1 p-2.5 bg-[#070707] border border-border-light rounded pointer-events-none min-w-[220px] shadow-xl"
                style={style}
            >
                <div className="font-bold text-sm text-wow-gold">Loading…</div>
            </div>
        )
    }

    const schoolName = spell.schoolName || getSchoolName(spell.school)
    const schoolColor = getSchoolColor(spell.school)
    const cost = spell.manaCost > 0 ? `${spell.manaCost} ${POWER_TYPES[spell.powerType] || 'Power'}` : ''
    // A line with an optional left/right pair; renders only if either side exists.
    const Row = ({ left, right }) =>
        (left || right) ? (
            <div className="flex justify-between gap-4 text-white leading-tight">
                <span>{left || ''}</span>
                <span className="text-gray-300">{right || ''}</span>
            </div>
        ) : null

    return (
        <div
            className="flex flex-col gap-0.5 p-2.5 bg-[#070707] border border-border-light rounded pointer-events-none select-none min-w-[220px] max-w-[320px] shadow-2xl font-sans text-xs"
            style={style}
        >
            {/* Name + rank */}
            <div className="font-bold text-[14px] leading-tight text-wow-gold">{spell.name}</div>
            {spell.nameSubtext && <div className="text-gray-400 leading-tight">{spell.nameSubtext}</div>}

            {/* Cost / cast time, range / duration */}
            <div className="mt-1 flex flex-col gap-0.5">
                <Row left={cost} right={spell.castTime} />
                <Row left={spell.range} right={spell.duration} />
            </div>

            {/* School */}
            {schoolName && (
                <div className="leading-tight font-medium" style={{ color: schoolColor }}>
                    {schoolName}
                </div>
            )}

            {/* Description (spellbook gold) */}
            {(spell.description || spell.toolTip) && (
                <div className="mt-1 text-[#ffd100] leading-snug whitespace-pre-wrap">
                    {spell.description || spell.toolTip}
                </div>
            )}
        </div>
    )
}

export default SpellTooltip
