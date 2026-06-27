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
                className="pointer-events-none flex min-w-[220px] flex-col gap-1 rounded border border-border-light bg-[#070707] p-2.5 shadow-xl"
                style={style}
            >
                <div className="text-sm font-bold text-wow-gold">Loading…</div>
            </div>
        )
    }

    const schoolName = spell.schoolName || getSchoolName(spell.school)
    const schoolColor = getSchoolColor(spell.school)
    const cost =
        spell.manaCost > 0 ? `${spell.manaCost} ${POWER_TYPES[spell.powerType] || 'Power'}` : ''
    // A line with an optional left/right pair; renders only if either side exists.
    const Row = ({ left, right }) =>
        left || right ? (
            <div className="flex justify-between gap-4 leading-tight text-white">
                <span>{left || ''}</span>
                <span className="text-gray-300">{right || ''}</span>
            </div>
        ) : null

    return (
        <div
            className="pointer-events-none flex min-w-[220px] max-w-[320px] select-none flex-col gap-0.5 rounded border border-border-light bg-[#070707] p-2.5 font-sans text-xs shadow-2xl"
            style={style}
        >
            {/* Name + rank */}
            <div className="text-[14px] font-bold leading-tight text-wow-gold">{spell.name}</div>
            {spell.nameSubtext && (
                <div className="leading-tight text-gray-400">{spell.nameSubtext}</div>
            )}

            {/* Cost / cast time, range / duration */}
            <div className="mt-1 flex flex-col gap-0.5">
                <Row left={cost} right={spell.castTime} />
                <Row left={spell.range} right={spell.duration} />
            </div>

            {/* School */}
            {schoolName && (
                <div className="font-medium leading-tight" style={{ color: schoolColor }}>
                    {schoolName}
                </div>
            )}

            {/* Description (spellbook gold) */}
            {(spell.description || spell.toolTip) && (
                <div className="mt-1 whitespace-pre-wrap leading-snug text-[#ffd100]">
                    {spell.description || spell.toolTip}
                </div>
            )}
        </div>
    )
}

export default SpellTooltip
