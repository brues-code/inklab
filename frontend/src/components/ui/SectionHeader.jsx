import React, { useState } from 'react'

/**
 * Section header with title and filter input
 */
export const SectionHeader = ({
    title,
    placeholder = 'Filter...',
    onFilterChange,
    titleColor,
    className = '',
    noSearch = false,
    actions = null,
}) => {
    const [value, setValue] = useState('')

    const handleChange = (e) => {
        const newValue = e.target.value
        setValue(newValue)
        onFilterChange?.(newValue)
    }

    const handleClear = () => {
        setValue('')
        onFilterChange?.('')
    }

    return (
        <div
            className={`flex min-h-[70px] flex-col justify-end gap-2 border-b border-border-dark bg-bg-hover p-3 ${className}`}
        >
            <div className="flex min-h-[26px] w-full items-end justify-between">
                <h3
                    className="m-0 text-xs font-bold uppercase tracking-wider"
                    style={{ color: titleColor || '#ffd100' }}
                >
                    {title}
                </h3>
                {actions}
            </div>
            <div
                className={`flex items-center overflow-hidden rounded border border-border-dark bg-bg-main ${noSearch ? 'pointer-events-none invisible select-none' : ''}`}
            >
                <span className="flex items-center px-2 text-gray-600">
                    <svg
                        width="14"
                        height="14"
                        viewBox="0 0 24 24"
                        fill="none"
                        stroke="currentColor"
                        strokeWidth="2"
                    >
                        <polygon points="22 3 2 3 10 12.46 10 19 14 21 14 12.46 22 3" />
                    </svg>
                </span>
                <input
                    type="text"
                    value={value}
                    onChange={handleChange}
                    placeholder={placeholder}
                    className="min-w-[80px] flex-1 border-none bg-transparent px-2 py-1.5 text-[13px] text-white outline-none placeholder:text-gray-600"
                    disabled={noSearch}
                />
                {value && !noSearch && (
                    <button
                        onClick={handleClear}
                        className="cursor-pointer border-none bg-transparent px-2 py-1 text-sm text-gray-500 transition-colors hover:text-white"
                    >
                        ✕
                    </button>
                )}
            </div>
        </div>
    )
}

export default SectionHeader
