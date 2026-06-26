import { useState, useEffect, useRef, useCallback } from 'react'
import { AdvancedSearch } from '../../../wailsjs/go/main/App'
import { useIcon, useNpcPortrait } from '../../services/useImage'
import { getQualityColor } from '../../utils/wow'

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

const ResultIcon = ({ iconName, type, displayId }) => {
    const icon = useIcon(iconName)
    // NPCs prefer the head-shot portrait; bounded result set, so generate=true is safe.
    const portrait = useNpcPortrait(type === 'npc' ? displayId : 0, 0, 0, true)
    const fallback = '/local-icons/inv_misc_questionmark.jpg'
    const src = (type === 'npc' && portrait.src) ? portrait.src : (icon.src || fallback)

    return (
        <div className="w-7 h-7 mr-2 bg-black rounded overflow-hidden flex-shrink-0 relative">
            {!icon.loading && <img src={src} className="w-full h-full object-cover" alt="" />}
            {TYPE_BADGE[type] !== '' && (
                <div className="absolute bottom-0 right-0 text-[7px] font-bold px-0.5 bg-black/80 text-white uppercase leading-tight">
                    {TYPE_BADGE[type]}
                </div>
            )}
        </div>
    )
}

const subtitle = (item) => {
    switch (item.type) {
        case 'item': return `Item Lv ${item.itemLevel} (Req ${item.requiredLevel})`
        case 'npc': return `Level ${item.levelMin}${item.levelMin !== item.levelMax ? '-' + item.levelMax : ''}`
        case 'quest': return `Level ${item.questLevel} (Req ${item.minLevel})`
        case 'spell': return item.description ? item.description.slice(0, 60) + (item.description.length > 60 ? '…' : '') : 'Spell'
        case 'object': return item.typeName || 'Object'
        default: return ''
    }
}

// flatten the AdvancedSearch response (grouped by type) into one ranked list
const combine = (res) => {
    const out = []
    res.items?.forEach(i => out.push({ ...i, type: 'item' }))
    res.creatures?.forEach(c => out.push({ ...c, type: 'npc', iconPath: 'inv_misc_head_dragon_01', quality: 1 }))
    res.quests?.forEach(q => out.push({ ...q, type: 'quest', iconPath: 'inv_misc_book_11', quality: 1, name: q.title }))
    res.spells?.forEach(s => out.push({ ...s, type: 'spell', iconPath: s.icon || 'spell_nature_starfall', quality: 1 }))
    res.objects?.forEach(o => out.push({ ...o, type: 'object', iconPath: 'inv_box_01', quality: 1 }))
    return out
}

/**
 * GlobalSearch — header search box. Type to search every entity type; pick a
 * result to navigate to its Database detail page via onNavigate(type, entry).
 */
function GlobalSearch({ onNavigate }) {
    const [query, setQuery] = useState('')
    const [results, setResults] = useState([])
    const [loading, setLoading] = useState(false)
    const [open, setOpen] = useState(false)
    const [category, setCategory] = useState('all')
    const boxRef = useRef(null)
    const reqId = useRef(0)

    const runSearch = useCallback((q) => {
        const trimmed = q.trim()
        if (trimmed.length < 2) {
            setResults([])
            setLoading(false)
            return
        }
        const id = ++reqId.current
        setLoading(true)
        AdvancedSearch({ query: trimmed, minLevel: 0, maxLevel: 0, quality: [], limit: 50, offset: 0 })
            .then(res => {
                if (id !== reqId.current) return // a newer query superseded this one
                setResults(combine(res))
                setCategory('all') // a fresh query starts unfiltered
                setLoading(false)
            })
            .catch(err => {
                if (id !== reqId.current) return
                console.error('Search failed:', err)
                setResults([])
                setLoading(false)
            })
    }, [])

    // Debounce typing so results stream in without a click.
    useEffect(() => {
        const t = setTimeout(() => runSearch(query), 250)
        return () => clearTimeout(t)
    }, [query, runSearch])

    // Close the dropdown on outside click.
    useEffect(() => {
        const onDown = (e) => {
            if (boxRef.current && !boxRef.current.contains(e.target)) setOpen(false)
        }
        document.addEventListener('mousedown', onDown)
        return () => document.removeEventListener('mousedown', onDown)
    }, [])

    const pick = (item) => {
        onNavigate?.(item.type, item.entry)
        setOpen(false)
        setQuery('')
        setResults([])
        setCategory('all')
    }

    const counts = results.reduce((acc, r) => { acc[r.type] = (acc[r.type] || 0) + 1; return acc }, {})
    const visible = category === 'all' ? results : results.filter(r => r.type === category)

    return (
        <div ref={boxRef} className="relative w-72">
            <div className="relative">
                <span className="absolute left-2.5 top-1/2 -translate-y-1/2 text-gray-500 text-sm pointer-events-none">⌕</span>
                <input
                    type="text"
                    value={query}
                    onChange={e => { setQuery(e.target.value); setOpen(true) }}
                    onFocus={() => setOpen(true)}
                    onKeyDown={e => { if (e.key === 'Escape') setOpen(false) }}
                    placeholder="Search items, NPCs, quests…"
                    className="w-full pl-7 pr-3 py-1.5 bg-bg-main border border-border-dark rounded text-white text-sm outline-none focus:border-wow-rare transition-colors"
                />
            </div>

            {open && query.trim().length >= 2 && (
                <div className="absolute right-0 mt-1 w-[26rem] max-h-[70vh] overflow-y-auto bg-bg-panel border border-border-dark rounded-lg shadow-2xl z-[9999]">
                    {loading && results.length === 0 ? (
                        <div className="px-4 py-3 text-sm text-wow-gold animate-pulse">Searching…</div>
                    ) : results.length === 0 ? (
                        <div className="px-4 py-3 text-sm text-gray-500">No results for “{query.trim()}”</div>
                    ) : (
                      <>
                        {/* Category filter tabs */}
                        <div className="sticky top-0 z-10 flex flex-wrap gap-1 px-2 py-1.5 bg-bg-panel border-b border-border-dark">
                            <button
                                onClick={() => setCategory('all')}
                                className={`px-2 py-0.5 rounded text-[11px] font-bold transition-colors ${category === 'all' ? 'bg-wow-rare text-white' : 'text-gray-400 hover:bg-bg-hover'}`}
                            >
                                All <span className="opacity-60">{results.length}</span>
                            </button>
                            {CATEGORIES.filter(c => counts[c.key]).map(c => (
                                <button
                                    key={c.key}
                                    onClick={() => setCategory(c.key)}
                                    className={`px-2 py-0.5 rounded text-[11px] font-bold transition-colors ${category === c.key ? 'bg-wow-rare text-white' : 'text-gray-400 hover:bg-bg-hover'}`}
                                >
                                    {c.label} <span className="opacity-60">{counts[c.key]}</span>
                                </button>
                            ))}
                        </div>
                        {visible.map((item, idx) => (
                            <button
                                key={`${item.type}-${item.entry}-${idx}`}
                                onClick={() => pick(item)}
                                className="w-full flex items-center text-left px-2.5 py-1.5 hover:bg-bg-hover border-b border-border-dark/50 last:border-b-0 transition-colors"
                            >
                                <ResultIcon iconName={item.iconPath} type={item.type} displayId={item.displayId1} />
                                <span className="text-gray-500 text-[11px] font-mono mr-2 min-w-[46px]">#{item.entry}</span>
                                <span className="flex-1 min-w-0">
                                    <span
                                        className="block font-bold text-sm truncate"
                                        style={{ color: item.type === 'item' ? getQualityColor(item.quality) : (TYPE_COLOR[item.type] || '#fff') }}
                                    >
                                        {item.name}
                                    </span>
                                    <span className="block text-[11px] text-gray-500 truncate">{subtitle(item)}</span>
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
