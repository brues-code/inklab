import { useLayoutEffect, useRef, useState } from 'react'
import { QUESTION_MARK_ICON } from '../../utils/wow'
import { useIcon } from '../../services/useImage'
import { useIconUsage } from '../../hooks/queries/icons'

/**
 * Icon hover popup: larger icon, copyable name, and item/spell usage counts
 * that deep-link to the icon detail page. Shared by the Icons tab wall and the
 * icon images on entity detail views (via IconPopupAnchor below).
 *
 * `counts` ({ itemCount, spellCount }) is optional — the Icons tab already has
 * them from its list query; other callers omit it and the counts resolve from
 * the usage query (cached, and it pre-warms the icon detail page).
 *
 * Positioning: centered on the anchor and floating entirely ABOVE it, so it
 * never covers the neighboring tiles the pointer is heading for; flips below
 * when there's no room above, and clamps to the viewport. Stays open while the
 * pointer is over the anchor or the popup itself (the copy button is
 * clickable) — reaching it is a straight move up from the anchor.
 */
export function IconHoverPopup({ name, counts, anchor, onEnter, onLeave, onOpen, onWheel }) {
    const img = useIcon(name)
    const [copied, setCopied] = useState(false)
    const ref = useRef(null)
    const [top, setTop] = useState(null)

    const { data: usage } = useIconUsage(counts ? null : name)
    const itemCount = counts ? counts.itemCount : usage?.items?.length
    const spellCount = counts ? counts.spellCount : usage?.spells?.length
    const loaded = itemCount !== undefined && spellCount !== undefined

    const copy = (e) => {
        e.stopPropagation()
        navigator.clipboard?.writeText(name)
        setCopied(true)
        setTimeout(() => setCopied(false), 1200)
    }

    const width = 190
    const left = Math.min(Math.max(anchor.x, width / 2 + 8), window.innerWidth - width / 2 - 8)
    useLayoutEffect(() => {
        const h = ref.current?.offsetHeight || 0
        let t = anchor.y - h - 6 // fully above the anchor
        if (t < 8) t = (anchor.bottom ?? anchor.y) + 6 // no room above → below
        setTop(Math.min(t, window.innerHeight - h - 8))
    }, [anchor, name, loaded])

    return (
        <div
            ref={ref}
            className="fixed z-50 -translate-x-1/2 rounded-lg border border-gray-600 bg-bg-panel p-3 shadow-2xl"
            // Render offscreen until measured so the first frame can't flash at
            // an unclamped position.
            style={{ left, top: top ?? -9999, width }}
            onMouseEnter={onEnter}
            onMouseLeave={onLeave}
            onWheel={onWheel}
        >
            <div className="flex flex-col items-center gap-2">
                <div
                    className="h-14 w-14 cursor-pointer overflow-hidden rounded border border-gray-600 bg-black/40"
                    // stopPropagation: the anchor may sit inside a row with its
                    // own onClick (e.g. NPC ability rows navigating to the spell).
                    onClick={(e) => {
                        e.stopPropagation()
                        onOpen()
                    }}
                    title="Open icon page"
                >
                    <img
                        src={img.src || QUESTION_MARK_ICON}
                        alt=""
                        className="h-full w-full object-cover"
                    />
                </div>

                <div className="flex w-full items-center gap-1">
                    <div className="min-w-0 flex-1 truncate rounded border border-gray-700 bg-black/50 px-2 py-1 font-mono text-[11px] text-gray-200">
                        {name}
                    </div>
                    <button
                        className="shrink-0 rounded border border-gray-700 bg-black/50 px-1.5 py-1 text-[11px] text-gray-400 transition-colors hover:border-wow-gold/60 hover:text-white"
                        onClick={copy}
                        title="Copy icon name"
                    >
                        {copied ? '✓' : '⧉'}
                    </button>
                </div>

                <div className="text-center text-[12px] leading-tight text-wow-gold">
                    {!loaded ? (
                        <span className="italic text-gray-600">...</span>
                    ) : itemCount + spellCount === 0 ? (
                        <span className="italic text-gray-500">unused</span>
                    ) : (
                        <>
                            {itemCount > 0 && (
                                <div
                                    className="cursor-pointer hover:text-yellow-300 hover:underline"
                                    onClick={(e) => {
                                        e.stopPropagation()
                                        onOpen('items')
                                    }}
                                >
                                    {itemCount} item{itemCount !== 1 ? 's' : ''}
                                </div>
                            )}
                            {spellCount > 0 && (
                                <div
                                    className="cursor-pointer hover:text-yellow-300 hover:underline"
                                    onClick={(e) => {
                                        e.stopPropagation()
                                        onOpen('spells')
                                    }}
                                >
                                    {spellCount} spell{spellCount !== 1 ? 's' : ''}
                                </div>
                            )}
                        </>
                    )}
                </div>
            </div>
        </div>
    )
}

/**
 * Hover wrapper for any icon image: wraps `children`, and while hovered shows
 * the IconHoverPopup for `name`. Opening (popup icon or count links) navigates
 * via onNavigate('icon', name, rel?) — the icon detail page. Wheel over the
 * popup dismisses it so the page keeps scrolling naturally.
 */
export function IconPopupAnchor({ name, onNavigate, className = '', children }) {
    const [anchor, setAnchor] = useState(null)
    const closeTimer = useRef(null)
    const openTimer = useRef(null)

    if (!name) return children

    // Hover intent: open only after a short dwell, so passing the pointer over
    // the icon doesn't flash a popup.
    const open = (e) => {
        clearTimeout(closeTimer.current)
        clearTimeout(openTimer.current)
        const rect = e.currentTarget.getBoundingClientRect()
        const a = { x: rect.left + rect.width / 2, y: rect.top, bottom: rect.bottom }
        openTimer.current = setTimeout(() => setAnchor(a), 250)
    }
    const scheduleClose = () => {
        clearTimeout(openTimer.current)
        clearTimeout(closeTimer.current)
        closeTimer.current = setTimeout(() => setAnchor(null), 150)
    }
    const cancelClose = () => clearTimeout(closeTimer.current)

    return (
        <>
            <div className={className} onMouseEnter={open} onMouseLeave={scheduleClose}>
                {children}
            </div>
            {anchor && (
                <IconHoverPopup
                    name={name}
                    anchor={anchor}
                    onEnter={cancelClose}
                    onLeave={scheduleClose}
                    onOpen={(rel) => {
                        setAnchor(null)
                        onNavigate?.('icon', name, rel)
                    }}
                    onWheel={() => setAnchor(null)}
                />
            )}
        </>
    )
}

export default IconHoverPopup
