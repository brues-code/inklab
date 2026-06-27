import { useState, useEffect, useRef, useMemo } from 'react'
import { useIcon, useNpcPortrait } from '../../services/useImage'
import { getQualityColor, QUESTION_MARK_ICON } from '../../utils/wow'
import { useEntityNavigate } from '../../utils/entityNav'
import { useDebouncedValue } from '../../hooks/useDebouncedValue'
import { useGlobalSearch } from '../../hooks/queries/search'

const TYPE_BADGE = { npc: 'NPC', quest: 'Q', spell: 'S', object: 'OBJ', item: '' }
const TYPE_COLOR = { npc: '#FFD100', spell: '#a855f7', object: '#00B4FF', quest: '#fff' }
// category tabs in display order; `key` matches the result item's `type`
const CATEGORIES = [
    { key: 'item', label: 'Items' },
    { key: 'npc', label: 'NPCs' },
    { key: 'quest', label: 'Quests' },
    { key: 'spell', label: 'Spells' },
    { key: 'object', label: 'Objects' },
]

const ResultIcon = ({ iconName, type, displayId, activeCategory }) => {
    const icon = useIcon(iconName)
    // NPCs prefer the head-shot portrait; bounded result set, so generate=true is safe.
    const portrait = useNpcPortrait(type === 'npc' ? displayId : 0, 0, 0, true)
    const src = type === 'npc' && portrait.src ? portrait.src : icon.src || QUESTION_MARK_ICON
    // The type badge disambiguates the mixed "All" list; it's redundant on a
    // single-type tab, so hide it there.
    const showBadge = TYPE_BADGE[type] !== '' && activeCategory !== type

    return (
        <div className="relative mr-2 h-7 w-7 flex-shrink-0 overflow-hidden rounded bg-black">
            {!icon.loading && <img src={src} className="h-full w-full object-cover" alt="" />}
            {showBadge && (
                <div className="absolute bottom-0 right-0 bg-black/80 px-0.5 text-[7px] font-bold uppercase leading-tight text-white">
                    {TYPE_BADGE[type]}
                </div>
            )}
        </div>
    )
}

const subtitle = (item) => {
    switch (item.type) {
        case 'item':
            return `Item Lv ${item.itemLevel} (Req ${item.requiredLevel})`
        case 'npc':
            return `Level ${item.levelMin}${item.levelMin !== item.levelMax ? '-' + item.levelMax : ''}`
        case 'quest':
            return `Level ${item.questLevel} (Req ${item.minLevel})`
        case 'spell':
            return item.description
                ? item.description.slice(0, 60) + (item.description.length > 60 ? '…' : '')
                : 'Spell'
        case 'object':
            return item.typeName || 'Object'
        default:
            return ''
    }
}

// flatten the AdvancedSearch response (grouped by type) into one ranked list
const combine = (res) => {
    const out = []
    res.items?.forEach((i) => out.push({ ...i, type: 'item' }))
    res.creatures?.forEach((c) =>
        out.push({ ...c, type: 'npc', iconPath: 'inv_misc_head_dragon_01', quality: 1 }),
    )
    res.quests?.forEach((q) =>
        out.push({ ...q, type: 'quest', iconPath: 'inv_misc_book_11', quality: 1, name: q.title }),
    )
    res.spells?.forEach((s) =>
        out.push({ ...s, type: 'spell', iconPath: s.icon || 'spell_nature_starfall', quality: 1 }),
    )
    res.objects?.forEach((o) =>
        out.push({ ...o, type: 'object', iconPath: 'inv_box_01', quality: 1 }),
    )
    return out
}

/**
 * GlobalSearch — header search box. Type to search every entity type; pick a
 * result to navigate to its Database detail route.
 */
function GlobalSearch() {
    const [query, setQuery] = useState('')
    const [open, setOpen] = useState(false)
    const [category, setCategory] = useState('all')
    const boxRef = useRef(null)
    const entityNavigate = useEntityNavigate()

    // Debounce the input, then let Query run/cache the search. Query handles
    // request dedup and drops out-of-order responses, so no manual race guard.
    const debounced = useDebouncedValue(query.trim(), 250)
    const searchQuery = useGlobalSearch(debounced)
    const loading = searchQuery.isFetching
    const results = useMemo(
        () => (searchQuery.data ? combine(searchQuery.data) : []),
        [searchQuery.data],
    )

    // A fresh query starts unfiltered (render-time reset, no effect).
    const [categoryKey, setCategoryKey] = useState(debounced)
    if (debounced !== categoryKey) {
        setCategoryKey(debounced)
        setCategory('all')
    }

    // Close the dropdown on outside click.
    useEffect(() => {
        const onDown = (e) => {
            if (boxRef.current && !boxRef.current.contains(e.target)) setOpen(false)
        }
        document.addEventListener('mousedown', onDown)
        return () => document.removeEventListener('mousedown', onDown)
    }, [])

    const pick = (item) => {
        entityNavigate(item.type, item.entry)
        setOpen(false)
        setQuery('')
        setCategory('all')
    }

    const counts = results.reduce((acc, r) => {
        acc[r.type] = (acc[r.type] || 0) + 1
        return acc
    }, {})
    const visible = category === 'all' ? results : results.filter((r) => r.type === category)

    return (
        <div ref={boxRef} className="relative w-72">
            <div className="relative">
                <span className="pointer-events-none absolute left-2.5 top-1/2 -translate-y-1/2 text-sm text-gray-500">
                    ⌕
                </span>
                <input
                    type="text"
                    value={query}
                    onChange={(e) => {
                        setQuery(e.target.value)
                        setOpen(true)
                    }}
                    onFocus={() => setOpen(true)}
                    onKeyDown={(e) => {
                        if (e.key === 'Escape') setOpen(false)
                    }}
                    placeholder="Search items, NPCs, quests…"
                    className="w-full rounded border border-border-dark bg-bg-main py-1.5 pl-7 pr-3 text-sm text-white outline-none transition-colors focus:border-wow-rare"
                />
            </div>

            {open && query.trim().length >= 2 && (
                <div className="absolute right-0 z-[9999] mt-1 max-h-[70vh] w-[26rem] overflow-y-auto rounded-lg border border-border-dark bg-bg-panel shadow-2xl">
                    {loading && results.length === 0 ? (
                        <div className="animate-pulse px-4 py-3 text-sm text-wow-gold">
                            Searching…
                        </div>
                    ) : results.length === 0 ? (
                        <div className="px-4 py-3 text-sm text-gray-500">
                            No results for “{query.trim()}”
                        </div>
                    ) : (
                        <>
                            {/* Category filter tabs */}
                            <div className="sticky top-0 z-10 flex flex-wrap gap-1 border-b border-border-dark bg-bg-panel px-2 py-1.5">
                                <button
                                    onClick={() => setCategory('all')}
                                    className={`rounded px-2 py-0.5 text-[11px] font-bold transition-colors ${category === 'all' ? 'bg-wow-rare text-white' : 'text-gray-400 hover:bg-bg-hover'}`}
                                >
                                    All <span className="opacity-60">{results.length}</span>
                                </button>
                                {CATEGORIES.filter((c) => counts[c.key]).map((c) => (
                                    <button
                                        key={c.key}
                                        onClick={() => setCategory(c.key)}
                                        className={`rounded px-2 py-0.5 text-[11px] font-bold transition-colors ${category === c.key ? 'bg-wow-rare text-white' : 'text-gray-400 hover:bg-bg-hover'}`}
                                    >
                                        {c.label}{' '}
                                        <span className="opacity-60">{counts[c.key]}</span>
                                    </button>
                                ))}
                            </div>
                            {visible.map((item, idx) => (
                                <button
                                    key={`${item.type}-${item.entry}-${idx}`}
                                    onClick={() => pick(item)}
                                    className="flex w-full items-center border-b border-border-dark/50 px-2.5 py-1.5 text-left transition-colors last:border-b-0 hover:bg-bg-hover"
                                >
                                    <ResultIcon
                                        iconName={item.iconPath}
                                        type={item.type}
                                        displayId={item.displayId1}
                                        activeCategory={category}
                                    />
                                    <span className="mr-2 min-w-[46px] font-mono text-[11px] text-gray-500">
                                        #{item.entry}
                                    </span>
                                    <span className="min-w-0 flex-1">
                                        <span
                                            className="block truncate text-sm font-bold"
                                            style={{
                                                color:
                                                    item.type === 'item'
                                                        ? getQualityColor(item.quality)
                                                        : TYPE_COLOR[item.type] || '#fff',
                                            }}
                                        >
                                            {item.name}
                                            {item.type === 'spell' && item.subname && (
                                                <span className="ml-1.5 text-[11px] font-normal text-gray-500">
                                                    {item.subname}
                                                </span>
                                            )}
                                        </span>
                                        <span className="block truncate text-[11px] text-gray-500">
                                            {subtitle(item)}
                                        </span>
                                    </span>
                                </button>
                            ))}
                        </>
                    )}
                </div>
            )}
        </div>
    )
}

export default GlobalSearch
