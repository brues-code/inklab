import React, { useState, useEffect } from 'react'
import { GetSpellDetail, SyncSingleSpell } from '../../../services/api'
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

const SpellDetailView = ({ entry, onBack, onNavigate, tooltipHook }) => {
    const [detail, setDetail] = useState(null)
    const [loading, setLoading] = useState(true)
    const [error, setError] = useState(null)
    const [syncing, setSyncing] = useState(false)

    const loadSpell = () => {
        setLoading(true)
        setError(null)
        
        GetSpellDetail(parseInt(entry))
            .then(res => {
                if (!res) {
                    setError("Spell data not found");
                } else {
                    setDetail(res)
                }
                setLoading(false)
            })
            .catch(err => {
                setError(err.toString());
                setLoading(false)
            })
    }

    useEffect(() => {
        loadSpell()
    }, [entry])

    const handleSync = async () => {
        setSyncing(true)
        try {
            const result = await SyncSingleSpell(parseInt(entry))
            if (result?.success) {
                // Reload spell data after sync
                loadSpell()
            } else {
                console.error("Sync failed:", result?.error)
            }
        } catch (err) {
            console.error("Sync error:", err)
        }
        setSyncing(false)
    }

    if (loading) return <DetailLoading />
    if (error) return <DetailError message={error} onBack={onBack} />
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
                            <span className="text-gray-500">Duration:</span>
                            <span className="text-gray-300 text-right">{detail.duration}</span>
                            
                            <span className="text-gray-500">Range:</span>
                            <span className="text-gray-300 text-right">{detail.range}</span>
                            
                            <span className="text-gray-500">Cost:</span>
                            <span className="text-gray-300 text-right">
                                {detail.manaCost > 0 ? `${detail.manaCost} ${powerType}` : 'None'}
                            </span>
                            
                            <span className="text-gray-500">Cast Time:</span>
                            <span className="text-gray-300 text-right">{detail.castTime}</span>

                            <span className="text-gray-500">School:</span>
                            <span className="text-gray-300 text-right font-medium" style={{ color: schoolColor }}>{schoolName}</span>
                            
                            <span className="text-gray-500">Level:</span>
                            <span className="text-gray-300 text-right">{detail.spellLevel}</span>
                        </div>
                    </DetailSection>
                </div>
            </div>
        </DetailPageLayout>
    )
}

export default SpellDetailView
