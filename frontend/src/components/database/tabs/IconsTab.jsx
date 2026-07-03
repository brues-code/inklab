import { useMemo, useRef, useState } from 'react'
import { ContentPanel, IconHoverPopup, ScrollList, SectionHeader } from '../../ui'
import { QUESTION_MARK_ICON } from '../../../utils/wow'
import { useIcon } from '../../../services/useImage'
import { useLocalIcons } from '../../../hooks/queries/icons'

const PAGE_SIZE = 400

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
 * Tiles render incrementally as you scroll so a 10k-icon set doesn't kick off
 * 10k image loads at once.
 */
function IconsTab({ onNavigate }) {
    const [filter, setFilter] = useState('')
    const [usedOnly, setUsedOnly] = useState(false)
    const [visible, setVisible] = useState(PAGE_SIZE)
    const [hover, setHover] = useState(null) // { icon, x, y, bottom }
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

    const shown = filtered.slice(0, visible)

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

    // Grow the wall as the user nears the bottom (no button to click). Any
    // scroll also dismisses the popup — its anchor is viewport-fixed and the
    // tiles have moved out from under it.
    const onScroll = (e) => {
        const el = e.currentTarget
        setHover(null)
        clearTimeout(closeTimer.current)
        clearTimeout(openTimer.current)
        if (el.scrollHeight - el.scrollTop - el.clientHeight < 600 && visible < filtered.length) {
            setVisible((v) => v + PAGE_SIZE)
        }
    }

    return (
        <ContentPanel>
            <SectionHeader
                title={`Icons (${filtered.length})`}
                placeholder="Filter icons..."
                onFilterChange={(v) => {
                    setFilter(v)
                    setVisible(PAGE_SIZE)
                }}
                actions={
                    <label className="flex cursor-pointer select-none items-center gap-1.5 text-[11px] text-gray-400">
                        <input
                            type="checkbox"
                            checked={usedOnly}
                            onChange={(e) => {
                                setUsedOnly(e.target.checked)
                                setVisible(PAGE_SIZE)
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
                    <div className="flex flex-wrap content-start gap-3">
                        {shown.map((icon) => (
                            <IconTile
                                key={icon.name}
                                icon={icon}
                                onClick={() => onNavigate?.('icon', icon.name)}
                                onHover={openHover}
                                onLeave={scheduleClose}
                            />
                        ))}
                    </div>
                    {visible < filtered.length && (
                        <div className="py-3 text-center text-xs italic text-gray-600">
                            Scroll for more ({filtered.length - visible} remaining)
                        </div>
                    )}
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
