import React from 'react'
import { WowButton } from '../ui/Button'

/**
 * Shared layout for all detail views (Item, NPC, Quest, Spell, Faction)
 */
export const DetailPageLayout = ({ children, className = '' }) => (
    <div className={`flex-1 overflow-y-auto bg-bg-dark p-5 text-gray-200 ${className}`}>
        {children}
    </div>
)

/**
 * Detail page header with icon and title
 */
export const DetailHeader = ({
    icon,
    iconBorderColor,
    title,
    titleColor,
    subtitle,
    stats,
    action,
    children,
}) => (
    <header className="mb-8 border-b border-border-dark pb-5">
        <div className="flex items-start gap-5">
            {/* Icon */}
            {icon && (
                <div
                    className="flex h-14 w-14 flex-shrink-0 items-center justify-center overflow-hidden rounded border-2 bg-black/40 shadow-lg"
                    style={{ borderColor: iconBorderColor || '#666' }}
                >
                    {icon}
                </div>
            )}

            {/* Title & Subtitle */}
            <div className="min-w-0 flex-1">
                <div className="flex items-center gap-3">
                    <h1
                        className="m-0 text-2xl font-bold leading-tight"
                        style={{ color: titleColor || '#fff' }}
                    >
                        {title}
                    </h1>
                    {action && <div className="flex-shrink-0">{action}</div>}
                </div>
                {subtitle && <div className="mt-1 text-gray-500">{subtitle}</div>}
                {stats && <div className="mt-2 flex gap-4 text-sm">{stats}</div>}
            </div>
        </div>
        {children}
    </header>
)

/**
 * Section with gold header
 */
export const DetailSection = ({ title, children, className = '' }) => (
    <section className={`mb-8 ${className}`}>
        <h3 className="mb-3 border-b border-wow-gold/30 pb-1 text-sm font-bold uppercase tracking-wider text-wow-gold">
            {title}
        </h3>
        {children}
    </section>
)

/**
 * Two-column grid for detail content
 */
export const DetailGrid = ({ children, className = '' }) => (
    <div className={`grid grid-cols-1 gap-8 lg:grid-cols-2 ${className}`}>{children}</div>
)

/**
 * Side panel (right column typically)
 */
export const DetailSidePanel = ({ children, className = '' }) => (
    <div className={`self-start rounded-lg border border-border-dark bg-bg-main p-5 ${className}`}>
        {children}
    </div>
)

/**
 * Loot/reward grid
 */
export const LootGrid = ({ children, className = '' }) => (
    <div className={`grid grid-cols-1 gap-2 xl:grid-cols-2 ${className}`}>{children}</div>
)

/**
 * Stat badge (e.g., "HP: 1000")
 */
export const StatBadge = ({ label, value, color }) => (
    <span
        className="rounded border border-white/5 bg-black/30 px-2 py-0.5 text-sm"
        style={{ color: color || '#888' }}
    >
        {label}: <b className="text-gray-300">{value}</b>
    </span>
)

/**
 * Loading state for detail views
 */
export const DetailLoading = () => (
    <div className="flex flex-1 items-center justify-center bg-bg-dark">
        <div className="animate-pulse text-lg italic text-wow-gold">Loading...</div>
    </div>
)

/**
 * Error state for detail views
 */
export const DetailError = ({ message, onBack }) => (
    <div className="flex flex-1 flex-col items-center justify-center gap-4 bg-bg-dark">
        <div className="text-lg font-bold text-red-500">{message}</div>
        {onBack && (
            <WowButton variant="back" onClick={onBack}>
                ← Back
            </WowButton>
        )}
    </div>
)

export default {
    DetailPageLayout,
    DetailHeader,
    DetailSection,
    DetailGrid,
    DetailSidePanel,
    LootGrid,
    StatBadge,
    DetailLoading,
    DetailError,
}
