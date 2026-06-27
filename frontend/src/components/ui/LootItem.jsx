import React, { useState } from 'react'
import { getQualityColor, QUESTION_MARK_ICON } from '../../utils/wow'
import { useIcon } from '../../services/useImage'
import { SyncSingleItem } from '../../../wailsjs/go/main/App'

/**
 * Loot item display with icon, name, and quality color
 */
export const LootItem = ({ 
    item,
    onClick,
    onMouseEnter,
    onMouseMove,
    onMouseLeave,
    showDropChance = false,
    className = '' 
}) => {
    const itemId = item.entry || item.itemId || item.id
    
    // Manage local name state to reflect updates immediately after sync
    const initialName = item.name || item.itemName
    const [localName, setLocalName] = useState(initialName)
    const [syncing, setSyncing] = useState(false)

    // Determine if item is "unknown" or missing data
    const isUnknown = !localName || localName === '' || localName.startsWith('Unknown Item') || localName.startsWith('Item ')

    const quality = item.quality || 0
    const qualityColor = getQualityColor(quality)
    const iconName = item.iconPath || item.iconName

    // Use unified icon loading
    const icon = useIcon(iconName)
    
    // Handle individual item sync
    const handleSync = async (e) => {
        e.stopPropagation() // Prevent navigating to item detail
        setSyncing(true)
        try {
            const res = await SyncSingleItem(itemId)
            if (res && res.success && res.name) {
                setLocalName(res.name) // Update name locally
            }
        } catch (err) {
            console.error("Failed to sync item:", itemId, err)
        } finally {
            setSyncing(false)
        }
    }

    return (
        <div 
            className={`
                flex items-center gap-2 p-1.5 
                bg-white/[0.02] hover:bg-white/5 
                border border-white/5 rounded 
                transition-all cursor-pointer group
                ${className}
            `}
            data-quality={quality}
            onClick={onClick}
            onMouseEnter={onMouseEnter}
            onMouseMove={onMouseMove}
            onMouseLeave={onMouseLeave}
        >
            {/* Icon */}
            <div 
                className="w-8 h-8 rounded border flex-shrink-0 bg-black/40 flex items-center justify-center overflow-hidden"
                style={{ borderColor: qualityColor }}
            >
                {icon.loading ? (
                    <div className="w-full h-full bg-white/5 animate-pulse" />
                ) : (
                    <img 
                        src={icon.src || QUESTION_MARK_ICON} // Fallback only for display
                        alt=""
                        className="w-full h-full object-cover"
                    />
                )}
            </div>
            
            {/* ID */}
            <span className="text-gray-600 text-[11px] font-mono min-w-[40px]">
                [{itemId}]
            </span>
            
            {/* Name and Sync UI */}
            <div className="flex-1 min-w-0 flex items-center justify-between">
                <span 
                    className={`
                        text-[13px] font-bold truncate pr-2
                        ${isUnknown ? 'text-gray-400 italic' : ''}
                    `}
                    style={!isUnknown ? { color: qualityColor } : {}}
                >
                    {localName || `Unknown Item #${itemId}`}
                </span>

                {isUnknown && (
                    <button 
                        className={`
                            px-2 py-0.5 text-[10px] font-bold uppercase tracking-wider rounded shadow-sm flex-shrink-0
                            transition-all duration-200
                            ${syncing 
                                ? 'bg-gray-600 text-gray-400 cursor-not-allowed' 
                                : 'bg-blue-600 hover:bg-blue-500 text-white'
                            }
                        `}
                        onClick={handleSync}
                        disabled={syncing}
                        title="Sync item data from Turtle WoW Database"
                    >
                        {syncing ? 'Syncing...' : 'Sync'}
                    </button>
                )}
            </div>
            
            {/* Drop Chance (optional) */}
            {showDropChance && item.dropChance && (
                <span className="text-gray-500 text-[10px] uppercase tracking-tight ml-2">
                    {item.dropChance}
                </span>
            )}
        </div>
    )
}

/**
 * Icon placeholder for non-item entities (NPC, Object, etc)
 */
export const EntityIcon = ({ 
    label, 
    color = '#555',
    size = 'md' 
}) => {
    const sizes = {
        sm: 'w-6 h-6 text-[10px]',
        md: 'w-8 h-8 text-[11px]',
        lg: 'w-10 h-10 text-xs',
    }
    
    return (
        <div 
            className={`${sizes[size]} rounded flex items-center justify-center font-bold text-white flex-shrink-0`}
            style={{ backgroundColor: color }}
        >
            {label}
        </div>
    )
}

export default { LootItem, EntityIcon }
