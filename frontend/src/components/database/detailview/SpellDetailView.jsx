import React, { useState } from 'react'
import { queryClient } from '../../../queryClient'
import { SyncSingleSpell } from '../../../services/api'
import { useSpellDetail } from '../../../hooks/queries/spells'
import { 
    DetailPageLayout, 
    DetailHeader, 
    DetailSection,
    DetailLoading,
    DetailError
} from '../../ui'
import { useIcon } from '../../../services/useImage'
import { getQualityColor, getSchoolName, getSchoolColor } from '../../../utils/wow'
import { DATABASE_BASE_URL } from '../../../utils/constants'

// Helper component for Spell Icon
const SpellIcon = ({ iconName }) => {
    const icon = useIcon(iconName)
    
    if (icon.loading) {
        return <div className="w-full h-full bg-white/5 animate-pulse" />
    }

    return (
        <img 
            src={icon.src || '/local-icons/inv_misc_questionmark.jpg'} 
            className="w-full h-full object-cover" 
            alt=""
            onError={(e) => { e.target.style.display = 'none' }}
        />
    )
}

// Helper component for Item Icon in Used By list
const ItemIcon = ({ iconName }) => {
    const icon = useIcon(iconName)
    
    if (icon.loading) {
        return <div className="w-full h-full bg-white/5 animate-pulse" />
    }

    return (
        <img 
            src={icon.src || '/local-icons/inv_misc_questionmark.jpg'} 
            className="w-full h-full object-cover" 
            alt=""
        />
    )
}

// One label/value row in the Properties grid; renders nothing when value is empty.
const PropRow = ({ label, value, color, capitalize }) =>
    (value === 0 || value) ? (
        <>
            <span className="text-gray-500">{label}:</span>
            <span
                className={`text-gray-300 text-right ${capitalize ? 'capitalize' : ''}`}
                style={color ? { color, fontWeight: 500 } : undefined}
            >
                {value}
            </span>
        </>
    ) : null

const SpellDetailView = ({ entry, onBack, onNavigate, tooltipHook }) => {
    const [syncing, setSyncing] = useState(false)

    const { data: detail, isLoading: loading, isError, error } = useSpellDetail(entry)

    const handleSync = async () => {
        setSyncing(true)
        try {
            const result = await SyncSingleSpell(parseInt(entry))
            if (result?.success) {
                // Drop the cache so lists/search/tooltips referencing this spell refetch.
                await queryClient.invalidateQueries()
            } else {
                console.error("Sync failed:", result?.error)
            }
        } catch (err) {
            console.error("Sync error:", err)
        }
        setSyncing(false)
    }

    if (loading) return <DetailLoading />
    if (isError) return <DetailError message={String(error)} onBack={onBack} />
    if (!detail) return <DetailError message="Spell not found" onBack={onBack} />
    
    // Localized school name from the client (spell_schools), English fallback.
    // Color is a client UI constant keyed by school index (Physical has none).
    const schoolName = detail.schoolName || getSchoolName(detail.school)
    const schoolColor = getSchoolColor(detail.school)

    // Format power type
    const powerTypes = {
        0: 'Mana', 1: 'Rage', 2: 'Focus', 3: 'Energy', 4: 'Happiness'
    }
    const powerType = powerTypes[detail.powerType] || 'Power'

    return (
        <DetailPageLayout onBack={onBack}>
            <DetailHeader
                title={`${detail.name} [${detail.entry}]`}
                icon={<SpellIcon iconName={detail.icon} />}
                titleColor="#FFD100" 
                subtitle={
                    <>
                        {[detail.nameSubtext, `Level ${detail.spellLevel}`].filter(Boolean).join(' • ')}
                        {' • '}
                        <span style={{ color: schoolColor }}>{schoolName}</span>
                    </>
                }
                action={
                    <div className="flex gap-2">
                        <button
                            onClick={handleSync}
                            disabled={syncing}
                            className={`px-3 py-1.5 text-xs font-bold uppercase rounded transition-colors ${
                                syncing 
                                    ? 'bg-gray-600 text-gray-400 cursor-not-allowed' 
                                    : 'bg-green-700 hover:bg-green-600 text-white'
                            }`}
                            title="Resolve this spell's description from local DBC data"
                        >
                            {syncing ? '⏳ Resolving...' : '🔄 Resolve'}
                        </button>
                        <a
                            href={`${DATABASE_BASE_URL}/?spell=${entry}`}
                            target="_blank"
                            rel="noreferrer"
                            className="px-3 py-1.5 text-xs font-bold uppercase rounded transition-colors bg-purple-700 hover:bg-purple-600 text-white"
                            title="View on Turtle WoW Database"
                        >
                            🔗 OctoHead
                        </a>
                    </div>
                }
            />
            
            <div className="grid grid-cols-1 lg:grid-cols-[2fr_1fr] gap-10">
                {/* Main Content */}
                <div className="space-y-8">
                     <DetailSection title="Description">
                        <p className="text-gray-300 leading-relaxed whitespace-pre-wrap">
                            {detail.description || 'No description available.'}
                        </p>
                    </DetailSection>

                    {detail.toolTip && detail.toolTip !== detail.description && (
                        <DetailSection title="Tooltip">
                            <p className="text-gray-300 leading-relaxed whitespace-pre-wrap">
                                {detail.toolTip}
                            </p>
                        </DetailSection>
                    )}

                    {/* Spell effects */}
                    {detail.effects?.length > 0 && (
                        <DetailSection title="Effects">
                            <div className="space-y-2">
                                {detail.effects.map((e) => (
                                    <div key={e.index} className="rounded border border-border-dark/50 bg-white/[0.02] p-2.5">
                                        <div className="text-sm text-gray-200 font-semibold">
                                            Effect #{e.index}: {e.effect}{e.auraName ? `: ${e.auraName}` : ''}
                                        </div>
                                        {e.value && (
                                            <div className="text-xs text-gray-400 mt-0.5">Value: {e.value}</div>
                                        )}
                                        {(e.radius || e.mechanic || e.triggerSpell > 0) && (
                                            <div className="flex flex-wrap gap-x-4 gap-y-0.5 mt-0.5 text-[11px] text-gray-500">
                                                {e.radius && <span>Radius: {e.radius}</span>}
                                                {e.mechanic && <span className="capitalize">Mechanic: {e.mechanic}</span>}
                                                {e.triggerSpell > 0 && (
                                                    <span
                                                        className="cursor-pointer text-wow-rare/80 hover:text-wow-rare"
                                                        onClick={() => onNavigate?.('spell', e.triggerSpell)}
                                                        {...(tooltipHook?.getSpellHandlers?.(e.triggerSpell) || {})}
                                                    >
                                                        Triggers spell #{e.triggerSpell}
                                                    </span>
                                                )}
                                            </div>
                                        )}
                                    </div>
                                ))}
                            </div>
                        </DetailSection>
                    )}

                    {/* Used By Items */}
                    {detail.usedByItems && detail.usedByItems.length > 0 && (
                        <DetailSection title={`Used By (${detail.usedByItems.length})`}>
                            <div className="space-y-1">
                                {detail.usedByItems.map(item => (
                                    <div 
                                        key={item.entry}
                                        className="flex items-center gap-2 p-1.5 rounded hover:bg-white/5 cursor-pointer transition-colors"
                                        onClick={() => onNavigate?.('item', item.entry)}
                                        onMouseEnter={() => tooltipHook?.onHover?.(item.entry)}
                                        onMouseLeave={() => tooltipHook?.onLeave?.()}
                                    >
                                        <div className="w-6 h-6 bg-black rounded overflow-hidden flex-shrink-0">
                                            <ItemIcon iconName={item.iconPath} />
                                        </div>
                                        <span 
                                            className="text-sm font-medium truncate"
                                            style={{ color: getQualityColor(item.quality) }}
                                        >
                                            {item.name}
                                        </span>
                                        <span className="text-xs text-gray-500 ml-auto">
                                            {item.triggerType === 0 ? 'Use' : 
                                             item.triggerType === 1 ? 'Equip' : 
                                             item.triggerType === 2 ? 'Chance on Hit' : 
                                             item.triggerType === 5 ? 'Learn' : ''}
                                        </span>
                                    </div>
                                ))}
                            </div>
                        </DetailSection>
                    )}
                </div>
                
                {/* Side Panel */}
                <div className="space-y-6">
                    <DetailSection title="Properties">
                        <div className="grid grid-cols-2 gap-y-2 text-sm">
                            <PropRow label="Cost" value={detail.manaCost > 0 ? `${detail.manaCost} ${powerType}` : 'None'} />
                            <PropRow label="Cast Time" value={detail.castTime} />
                            <PropRow label="Cooldown" value={detail.cooldown} />
                            <PropRow label="GCD" value={detail.gcd} />
                            <PropRow label="Range" value={detail.range} />
                            <PropRow label="Duration" value={detail.duration} />
                            <PropRow label="School" value={schoolName} color={schoolColor} />
                            <PropRow label="Mechanic" value={detail.mechanicName} capitalize />
                            <PropRow label="Dispel type" value={detail.dispelType} />
                            {/* Real proc rate (PPM / %) from the world DB proc tables; "" when none. */}
                            <PropRow label="Proc" value={detail.proc} />
                            <PropRow label="Max targets" value={detail.maxAffectedTargets > 0 ? detail.maxAffectedTargets : ''} />
                            <PropRow label="Level" value={detail.spellLevel} />
                        </div>
                    </DetailSection>

                    {detail.flags?.length > 0 && (
                        <DetailSection title="Flags">
                            <ul className="text-xs text-amber-300/80 space-y-1 list-disc list-inside">
                                {detail.flags.map((f, i) => <li key={i}>{f}</li>)}
                            </ul>
                        </DetailSection>
                    )}
                </div>
            </div>
        </DetailPageLayout>
    )
}

export default SpellDetailView
