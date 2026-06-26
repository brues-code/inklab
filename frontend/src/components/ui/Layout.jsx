import React from 'react'

/**
 * Main page layout container - full height flex column
 */
export const PageLayout = ({ children, className = '' }) => (
    <div className={`h-full flex flex-col bg-bg-dark ${className}`}>
        {children}
    </div>
)

/**
 * Content grid for multi-column layouts. `columns` is a CSS
 * grid-template-columns value (e.g. "180px 1fr").
 */
export const ContentGrid = ({ children, columns, className = '' }) => (
    <div 
        className={`flex-1 grid gap-0 overflow-hidden ${className}`}
        style={columns ? { gridTemplateColumns: columns } : undefined}
    >
        {children}
    </div>
)

/**
 * Sidebar panel - left column with dark bg and border
 */
export const SidebarPanel = ({ children, className = '' }) => (
    <aside className={`flex flex-col h-full bg-bg-main border-r border-border-dark overflow-hidden ${className}`}>
        {children}
    </aside>
)

/**
 * Main content panel - flexible content area
 */
export const ContentPanel = ({ children, className = '' }) => (
    <section className={`flex flex-col h-full bg-bg-panel overflow-hidden flex-1 ${className}`}>
        {children}
    </section>
)

/**
 * Scrollable list container inside panels
 */
export const ScrollList = React.forwardRef(({ children, className = '', ...props }, ref) => (
    <div 
        ref={ref}
        className={`flex-1 overflow-y-auto p-1 space-y-px ${className}`}
        {...props}
    >
        {children}
    </div>
))

ScrollList.displayName = 'ScrollList'

export default { PageLayout, ContentGrid, SidebarPanel, ContentPanel, ScrollList }
