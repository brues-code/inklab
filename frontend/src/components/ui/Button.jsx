import React from 'react'

/**
 * WoW-styled button component
 * @param {string} variant - 'primary' | 'secondary' | 'back' | 'tab'
 */
export const WowButton = ({
    children,
    onClick,
    variant = 'primary',
    active = false,
    className = '',
    ...props
}) => {
    const baseClasses =
        'px-4 py-2 font-bold text-sm cursor-pointer transition-all duration-200 border'

    const variants = {
        primary: `bg-wow-rare border-wow-rare/30 text-white rounded hover:brightness-110 active:scale-95`,
        secondary: `bg-bg-hover border-border-light text-white rounded hover:bg-bg-active`,
        back: `bg-bg-panel border-border-light text-gray-400 rounded hover:bg-bg-hover hover:text-white`,
        tab: `bg-transparent border-transparent text-wow-gold uppercase text-[13px] rounded-none hover:bg-bg-hover ${
            active ? '!bg-bg-active !text-white !border-border-light' : ''
        }`,
    }

    return (
        <button
            onClick={onClick}
            className={`${baseClasses} ${variants[variant]} ${className}`}
            {...props}
        >
            {children}
        </button>
    )
}

/**
 * Clickable list item for sidebars
 */
export const ListItem = ({ children, onClick, active = false, className = '' }) => (
    <div
        onClick={onClick}
        className={`cursor-pointer px-3 py-2 text-[13px] transition-colors ${
            active
                ? 'border-l-2 border-wow-gold bg-bg-hover text-white'
                : 'border-l-2 border-transparent text-gray-400 hover:bg-bg-panel'
        } ${className} `}
    >
        {children}
    </div>
)

/**
 * Tab button for tab navigation
 */
export const TabButton = ({ children, onClick, active = false }) => (
    <WowButton variant="tab" active={active} onClick={onClick}>
        {children}
    </WowButton>
)

/**
 * Tab container
 */
export const TabBar = ({ children, className = '' }) => (
    <div className={`flex gap-0 border-b border-border-light bg-bg-main px-2.5 ${className}`}>
        {children}
    </div>
)

export default { WowButton, ListItem, TabButton, TabBar }
