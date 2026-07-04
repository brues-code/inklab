import { useLayoutEffect, useMemo, useRef, useState } from 'react'
import { useVirtualizer } from '@tanstack/react-virtual'
import { ContentPanel, IconHoverPopup, ScrollList, SectionHeader } from '../../ui'
import { QUESTION_MARK_ICON } from '../../../utils/wow'
import { useIcon } from '../../../services/useImage'
import { useLocalIcons } from '../../../hooks/queries/icons'

// Fixed tile geometry: the wall is a uniform grid, virtualized by row.
const TILE = 48 // h-12/w-12
const GAP = 12 // gap between tiles
const STRIDE = TILE + GAP
const PAD = 12 // ScrollList p-3

// Bare icon tile — the wall shows only images; everything else lives in the
// hover popup. Tiles share the app-wide icon data-URL cache via useIcon.
function IconTile({ icon, onClick, onHover, onLeave }) {
    const img = useIcon(icon.name)
    return (
        <div
            className="h-12 w-12 cursor-pointer overflow-hidden rounded border border-gray-800 bg-black/40 transition-transform hover:scale-105 hover:border-wow-gold/60"
            onClick={onClick}
            onMouseEnter={(e) => onHover(icon, e.currentTarget.getBoundingClientRect())}
            onMouseLeave={onLeave}
        >
            {img.loading ? (
                <div className="h-full w-full animate-pulse bg-white/5" />
            ) : (
                <img
                    src={img.src || QUESTION_MARK_ICON}
                    alt=""
                    className="h-full w-full object-cover"
                />
            )}
        </div>
    )
}

/**
 * Icons tab: a dense wall of every unique icon in the local client-imported set
 * (data/icons). Hovering a tile pops up the icon's name (copyable) and how many
 * items/spells use it; clicking opens the icon's detail page listing them all.
 *
 * The wall is row-virtualized with @tanstack/react-virtual: only the rows in
 * (and just around) the viewport are mounted, so a 10k-icon set renders a few
 * hundred tiles at any moment and images only ever load for what's been on
 * screen, while the scrollbar reflects the true list length.
 */
function IconsTab({ onNavigate }) {
    const [filter, setFilter] = useState('')
    const [usedOnly, setUsedOnly] = useState(false)
    const [hover, setHover] = useState(null) // { icon, x, y, bottom }
    const [width, setWidth] = useState(800)
    const closeTimer = useRef(null)
    const openTimer = useRef(null)
    const scrollRef = useRef(null)

    const { data: icons = [], isLoading } = useLocalIcons()

    const filtered = useMemo(() => {
        const f = filter.trim().toLowerCase()
        return icons.filter(
            (i) => (!usedOnly || i.itemCount + i.spellCount > 0) && (!f || i.name.includes(f)),
        )
    }, [icons, filter, usedOnly])

    // Track the scroll container's width — the column count re-flows with it.
    useLayoutEffect(() => {
        const el = scrollRef.current
        if (!el) return
        const measure = () => setWidth(el.clientWidth - PAD * 2)
        measure()
        const ro = new ResizeObserver(measure)
        ro.observe(el)
        return () => ro.disconnect()
    }, [isLoading, icons.length])

    const cols = Math.max(1, Math.floor((width + GAP) / STRIDE))
    const rows = Math.ceil(filtered.length / cols)

    const rowVirtualizer = useVirtualizer({
        count: rows,
        getScrollElement: () => scrollRef.current,
        estimateSize: () => STRIDE,
        overscan: 3,
    })

    // Hover intent: the first popup opens only after a short dwell so sweeping
    // the wall doesn't flash popups in the pointer's path; once one is open,
    // moving to another tile switches it instantly.
    const openHover = (icon, rect) => {
        clearTimeout(closeTimer.current)
        clearTimeout(openTimer.current)
        const a = { icon, x: rect.left + rect.width / 2, y: rect.top, bottom: rect.bottom }
        if (hover) setHover(a)
        else openTimer.current = setTimeout(() => setHover(a), 250)
    }
    const scheduleClose = () => {
        clearTimeout(openTimer.current)
        clearTimeout(closeTimer.current)
        closeTimer.current = setTimeout(() => setHover(null), 150)
    }
    const cancelClose = () => clearTimeout(closeTimer.current)

    // Scrolling dismisses the popup — its anchor is viewport-fixed and the
    // tiles have moved out from under it. (The virtualizer tracks scroll itself.)
    const onScroll = () => {
        setHover(null)
        clearTimeout(closeTimer.current)
        clearTimeout(openTimer.current)
    }

    return (
        <ContentPanel>
            <SectionHeader
                title={`Icons (${filtered.length})`}
                placeholder="Filter icons..."
                onFilterChange={(v) => {
                    setFilter(v)
                    scrollRef.current?.scrollTo(0, 0)
                }}
                actions={
                    <label className="flex cursor-pointer select-none items-center gap-1.5 text-[11px] text-gray-400">
                        <input
                            type="checkbox"
                            checked={usedOnly}
                            onChange={(e) => {
                                setUsedOnly(e.target.checked)
                                scrollRef.current?.scrollTo(0, 0)
                            }}
                        />
                        used only
                    </label>
                }
            />

            {isLoading && (
                <div className="flex flex-1 animate-pulse items-center justify-center italic text-wow-gold">
                    Scanning local icons...
                </div>
            )}

            {!isLoading && icons.length === 0 && (
                <div className="flex flex-1 flex-col items-center justify-center gap-1 italic text-gray-600">
                    <span>No local icons found.</span>
                    <span className="text-xs">
                        Run the Client Data import (Tools → Import) to extract them from your WoW
                        client.
                    </span>
                </div>
            )}

            {!isLoading && icons.length > 0 && (
                <ScrollList ref={scrollRef} className="p-3" onScroll={onScroll}>
                    {/* Spacer with the grid's full height; only the virtual
                        rows are mounted, translated to their offsets. */}
                    <div className="relative" style={{ height: rowVirtualizer.getTotalSize() }}>
                        {rowVirtualizer.getVirtualItems().map((vRow) => (
                            <div
                                key={vRow.key}
                                className="absolute left-0 flex gap-3"
                                style={{ top: 0, transform: `translateY(${vRow.start}px)` }}
                            >
                                {filtered
                                    .slice(vRow.index * cols, (vRow.index + 1) * cols)
                                    .map((icon) => (
                                        <IconTile
                                            key={icon.name}
                                            icon={icon}
                                            onClick={() => onNavigate?.('icon', icon.name)}
                                            onHover={openHover}
                                            onLeave={scheduleClose}
                                        />
                                    ))}
                            </div>
                        ))}
                    </div>
                </ScrollList>
            )}

            {hover && (
                <IconHoverPopup
                    name={hover.icon.name}
                    counts={hover.icon}
                    anchor={hover}
                    onEnter={cancelClose}
                    onLeave={scheduleClose}
                    onOpen={(rel) => onNavigate?.('icon', hover.icon.name, rel)}
                    // Forward wheel to the icon wall so the popup never traps
                    // scrolling; the resulting scroll dismisses it.
                    onWheel={(e) => scrollRef.current?.scrollBy(0, e.deltaY)}
                />
            )}
        </ContentPanel>
    )
}

export default IconsTab
