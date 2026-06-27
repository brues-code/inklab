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
    className = '',
}) => {
    const itemId = item.entry || item.itemId || item.id

    // Manage local name state to reflect updates immediately after sync
    const initialName = item.name || item.itemName
    const [localName, setLocalName] = useState(initialName)
    const [syncing, setSyncing] = useState(false)

    // Determine if item is "unknown" or missing data
    const isUnknown =
        !localName ||
        localName === '' ||
        localName.startsWith('Unknown Item') ||
        localName.startsWith('Item ')

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
            console.error('Failed to sync item:', itemId, err)
        } finally {
            setSyncing(false)
        }
    }

    return (
        <div
            className={`group flex cursor-pointer items-center gap-2 rounded border border-white/5 bg-white/[0.02] p-1.5 transition-all hover:bg-white/5 ${className} `}
            data-quality={quality}
            onClick={onClick}
            onMouseEnter={onMouseEnter}
            onMouseMove={onMouseMove}
            onMouseLeave={onMouseLeave}
        >
            {/* Icon */}
            <div
                className="flex h-8 w-8 flex-shrink-0 items-center justify-center overflow-hidden rounded border bg-black/40"
                style={{ borderColor: qualityColor }}
            >
                {icon.loading ? (
                    <div className="h-full w-full animate-pulse bg-white/5" />
                ) : (
                    <img
                        src={icon.src || QUESTION_MARK_ICON} // Fallback only for display
                        alt=""
                        className="h-full w-full object-cover"
                    />
                )}
            </div>

            {/* ID */}
            <span className="min-w-[40px] font-mono text-[11px] text-gray-600">[{itemId}]</span>

            {/* Name and Sync UI */}
            <div className="flex min-w-0 flex-1 items-center justify-between">
                <span
                    className={`truncate pr-2 text-[13px] font-bold ${isUnknown ? 'italic text-gray-400' : ''} `}
                    style={!isUnknown ? { color: qualityColor } : {}}
                >
                    {localName || `Unknown Item #${itemId}`}
                </span>

                {isUnknown && (
                    <button
                        className={`flex-shrink-0 rounded px-2 py-0.5 text-[10px] font-bold uppercase tracking-wider shadow-sm transition-all duration-200 ${
                            syncing
                                ? 'cursor-not-allowed bg-gray-600 text-gray-400'
                                : 'bg-blue-600 text-white hover:bg-blue-500'
                        } `}
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
                <span className="ml-2 text-[10px] uppercase tracking-tight text-gray-500">
                    {item.dropChance}
                </span>
            )}
        </div>
    )
}

/**
 * Icon placeholder for non-item entities (NPC, Object, etc)
 */
export const EntityIcon = ({ label, color = '#555', size = 'md' }) => {
    const sizes = {
        sm: 'w-6 h-6 text-[10px]',
        md: 'w-8 h-8 text-[11px]',
        lg: 'w-10 h-10 text-xs',
    }

    return (
        <div
            className={`${sizes[size]} flex flex-shrink-0 items-center justify-center rounded font-bold text-white`}
            style={{ backgroundColor: color }}
        >
            {label}
        </div>
    )
}

export default { LootItem, EntityIcon }
