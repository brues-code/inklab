import React from 'react'
import { getQualityColor } from '../../utils/wow'
import { Money } from './Money'

/**
 * WoW-style item tooltip component
 */
const ItemTooltip = ({
    item,
    tooltip,
    style,
    onMouseEnter = undefined,
    onMouseLeave = undefined,
    interactive = false,
    onSpellClick = undefined,
    onItemClick = undefined,
    tooltipHook = undefined,
    // Rendered inside another tooltip (a recipe's crafted item): drop the box
    // chrome (border/bg/shadow/width) and just separate with a top divider.
    nested = false,
    // Selected random enchantment ({ suffix, effects }): appends the suffix to
    // the name and its stat lines (green, like the in-game suffix enchant).
    suffix = undefined,
}) => {
    // Loading state
    if (!tooltip) {
        return (
            <div
                className="pointer-events-none z-[1000] flex min-w-[200px] flex-col gap-1 rounded border border-border-light bg-[#070707] p-2.5 shadow-xl"
                style={style}
            >
                <div
                    className="text-sm font-bold leading-tight"
                    style={{ color: getQualityColor(item?.quality) }}
                >
                    {item?.name || item?.itemName || 'Unknown Item'}
                </div>
                <div className="animate-pulse text-[11px] italic text-gray-500">Loading...</div>
            </div>
        )
    }

    const qualityColor = getQualityColor(tooltip.quality)
    const interactionClass = interactive
        ? 'select-text pointer-events-auto cursor-auto'
        : 'select-none pointer-events-none'
    const boxClass = nested
        ? 'mt-2 border-t border-white/10 pt-2'
        : 'rounded border border-border-light bg-[#070707] p-2.5 z-[1000] min-w-[240px] max-w-[320px] shadow-2xl'

    return (
        <div
            className={`flex flex-col gap-0.5 font-sans text-xs ${interactionClass} ${boxClass}`}
            style={style}
            onMouseEnter={onMouseEnter}
            onMouseLeave={onMouseLeave}
        >
            {/* Name. In a nested subtooltip (a recipe's crafted item) it links to
                that item's page; the top-level name isn't linked (you're already
                on/hovering it). */}
            <div className="text-[14px] font-bold leading-tight" style={{ color: qualityColor }}>
                {nested && onItemClick && tooltip.entry ? (
                    <span
                        className="cursor-pointer hover:underline"
                        onClick={() => onItemClick(tooltip.entry)}
                    >
                        {tooltip.name}
                    </span>
                ) : suffix ? (
                    `${tooltip.name} ${suffix.suffix}`
                ) : (
                    tooltip.name
                )}
            </div>

            {/* Set Name */}
            {tooltip.setName && (
                <div className="leading-tight text-wow-gold">{tooltip.setName}</div>
            )}

            {/* Binding */}
            {tooltip.binding && <div className="leading-tight text-white">{tooltip.binding}</div>}

            {/* Unique */}
            {tooltip.unique && <div className="leading-tight text-white">Unique</div>}

            {/* Slot / Type */}
            {(() => {
                // Don't show type if it's "Consumable" as it's redundant for Trinkets/etc
                const shouldShowType =
                    tooltip.typeName &&
                    tooltip.typeName !== 'Consumable' &&
                    tooltip.typeName !== 'Junk' &&
                    tooltip.typeName !== 'Miscellaneous'
                const hasContent = tooltip.slotName || shouldShowType
                return hasContent ? (
                    <div className="flex w-full flex-row items-center justify-between leading-tight text-white">
                        {tooltip.slotName && <span>{tooltip.slotName}</span>}
                        {shouldShowType && <span>{tooltip.typeName}</span>}
                    </div>
                ) : null
            })()}

            {/* Races */}
            {tooltip.races && <div className="leading-tight text-white">{tooltip.races}</div>}

            {/* Damage */}
            {tooltip.damageText && (
                <div className="flex w-full flex-row items-center justify-between leading-tight text-white">
                    <span>{tooltip.damageText}</span>
                    <span className="font-medium">{tooltip.speedText}</span>
                </div>
            )}

            {/* DPS */}
            {tooltip.dps && <div className="leading-tight text-white">{tooltip.dps}</div>}

            {/* Armor */}
            {tooltip.armor > 0 && (
                <div className="leading-tight text-white">{tooltip.armor} Armor</div>
            )}

            {/* Stats */}
            {tooltip.stats?.length > 0 && (
                <div className="flex flex-col">
                    {tooltip.stats.map((stat, i) => (
                        <div key={i} className="leading-tight text-white">
                            {stat}
                        </div>
                    ))}
                </div>
            )}

            {/* Selected random suffix stat lines (green, like an enchant) */}
            {suffix?.effects?.length > 0 && (
                <div className="flex flex-col">
                    {suffix.effects.map((line, i) => (
                        <div key={i} className="leading-tight text-wow-uncommon">
                            {line}
                        </div>
                    ))}
                </div>
            )}

            {/* Resistances */}
            {tooltip.resistances?.length > 0 && (
                <div className="flex flex-col">
                    {tooltip.resistances.map((res, i) => (
                        <div key={i} className="leading-tight text-white">
                            {res}
                        </div>
                    ))}
                </div>
            )}

            {/* Durability */}
            {tooltip.durability && (
                <div className="text-[11px] leading-tight text-white">{tooltip.durability}</div>
            )}

            {/* Class restriction. classReqs is the colored, structured form (one
                span per allowed class, in its class color); fall back to the
                legacy plain string if present. */}
            {tooltip.classReqs?.length > 0 ? (
                <div className="leading-tight text-white">
                    Classes:{' '}
                    {tooltip.classReqs.map((c, i) => (
                        <span key={c.name}>
                            <span style={{ color: c.color || undefined }}>{c.name}</span>
                            {i < tooltip.classReqs.length - 1 ? ', ' : ''}
                        </span>
                    ))}
                </div>
            ) : (
                tooltip.classes && <div className="leading-tight text-white">{tooltip.classes}</div>
            )}

            {/* Required Level */}
            {tooltip.requiredLevel > 1 && (
                <div className="leading-tight text-white">
                    Requires Level {tooltip.requiredLevel}
                </div>
            )}

            {/* Reputation requirement, e.g. "Requires The League of Arathor - Revered" */}
            {tooltip.reqRepFaction && (
                <div className="leading-tight text-white">
                    Requires {tooltip.reqRepFaction}
                    {tooltip.reqRepStanding ? ` - ${tooltip.reqRepStanding}` : ''}
                </div>
            )}

            {/* Profession/skill requirement, e.g. "Requires Blacksmithing (300)"
                and a specialization like "Requires Armorsmith" (recipes). */}
            {tooltip.reqSkill && (
                <div className="leading-tight text-white">
                    Requires {tooltip.reqSkill}
                    {tooltip.reqSkillRank > 0 ? ` (${tooltip.reqSkillRank})` : ''}
                </div>
            )}
            {tooltip.reqSpell && (
                <div className="leading-tight text-white">Requires {tooltip.reqSpell}</div>
            )}

            {/* Spell Effects (green) - WoW style: after stats/durability. In the
                interactive (detail-page) tooltip the spell is a link to its page. */}
            {tooltip.effects?.length > 0 && (
                <div className="mt-1 flex flex-col gap-0.5">
                    {tooltip.effects.map((effect, i) => {
                        const clickable = interactive && onSpellClick && effect.spellId > 0
                        return (
                            <div key={i} className="leading-tight text-wow-uncommon">
                                {clickable ? (
                                    <span
                                        className="cursor-pointer hover:underline"
                                        onClick={() => onSpellClick(effect.spellId)}
                                        {...(tooltipHook?.getSpellHandlers?.(effect.spellId) || {})}
                                    >
                                        {effect.text}
                                    </span>
                                ) : (
                                    effect.text
                                )}
                            </div>
                        )
                    })}
                </div>
            )}

            {/* Set Info */}
            {tooltip.setInfo && (
                <div className="mt-2 flex flex-col gap-0.5 border-t border-white/10 pt-2">
                    <div className="font-bold text-wow-gold">{tooltip.setInfo.name}</div>
                    {tooltip.setInfo.items?.map((setItem, i) => (
                        <div key={i} className="ml-2 text-[11px] leading-tight text-gray-500">
                            {setItem}
                        </div>
                    ))}
                    <div className="mt-1">
                        {tooltip.setInfo.bonuses?.map((bonus, i) => (
                            <div key={i} className="text-[11px] leading-tight text-wow-uncommon">
                                {bonus}
                            </div>
                        ))}
                    </div>
                </div>
            )}

            {/* Description */}
            {tooltip.description && (
                <div className="mt-1 italic leading-snug text-wow-gold">
                    "{tooltip.description}"
                </div>
            )}

            {/* Sell Price */}
            {tooltip.sellPrice > 0 && (
                <div className="mt-1 flex items-center gap-1 text-[11px] leading-tight text-white">
                    <span className="text-gray-500">Sell Price:</span>
                    <Money copper={tooltip.sellPrice} />
                </div>
            )}

            {/* Crafts: the full tooltip of the item this recipe produces, nested
                last (after the recipe's own fields) so it reads as its own block,
                like the in-game recipe tooltip. */}
            {tooltip.crafts && (
                <ItemTooltip
                    nested
                    item={tooltip.crafts}
                    tooltip={tooltip.crafts}
                    interactive={interactive}
                    onSpellClick={onSpellClick}
                    onItemClick={onItemClick}
                    tooltipHook={tooltipHook}
                />
            )}
        </div>
    )
}

export default ItemTooltip
