import { useState } from 'react'
import { AdvancedSearch } from '../../../wailsjs/go/main/App'
import { useIcon, useNpcPortrait } from '../../services/useImage'
import { getQualityColor } from '../../utils/wow'

const SearchResultIcon = ({ iconName, type, displayId }) => {
    const icon = useIcon(iconName)
    // For NPCs, prefer the head-shot portrait, falling back to the generic NPC
    // icon. Search is bounded (≤50 results) and renders serialize through the
    // backend mutex, so generate=true here trickles portraits in over a few
    // seconds rather than storming (the infinite-scroll browse list stays
    // cached-only).
    const portrait = useNpcPortrait(type === 'npc' ? displayId : 0, 0, 0, true)

    const fallback = '/local-icons/inv_misc_questionmark.jpg'
    const src = (type === 'npc' && portrait.src) ? portrait.src : (icon.src || fallback)

    return (
        <div className="w-8 h-8 mr-2 bg-black rounded overflow-hidden flex-shrink-0 relative">
            {!icon.loading && (
                <img
                    src={src}
                    className="w-full h-full object-cover"
                    alt=""
                />
            )}
            {/* Type Badge */}
            <div className="absolute bottom-0 right-0 text-[8px] font-bold px-1 bg-black/80 text-white uppercase leading-tight">
                {type === 'npc' ? 'NPC' : type === 'quest' ? 'Q' : type === 'spell' ? 'S' : type === 'object' ? 'OBJ' : ''}
            </div>
        </div>
    )
}

// ... inside render:
// Instead of:
// <div className="w-8 h-8 mr-2 bg-black rounded overflow-hidden flex-shrink-0 relative"> ... </div>
// Use:
// <SearchResultIcon iconName={item.iconPath} type={item.type} />

// Removed unused QUALITIES constant and filtering logic

function SearchPage({ onItemClick, onNavigate }) {
    const [query, setQuery] = useState('')
    const [results, setResults] = useState([])
    const [loading, setLoading] = useState(false)
    const [totalCount, setTotalCount] = useState(0)

    const handleSearch = () => {
        setLoading(true)
        setResults([])
        
        // Use simplified filter for unified search
        const filter = {
            query: query.trim(),
            minLevel: 0,
            maxLevel: 0,
            quality: [],
            limit: 50,
            offset: 0
        }

        AdvancedSearch(filter)
            .then(res => {
                const combined = [];
                
                // Process Creatures
                if (res.creatures) {
                    res.creatures.forEach(c => {
                        combined.push({
                            ...c,
                            type: 'npc',
                            iconPath: 'inv_misc_head_dragon_01', // Generic NPC icon
                            quality: 1
                        })
                    })
                }

                // Process Quests
                if (res.quests) {
                    res.quests.forEach(q => {
                        combined.push({
                            ...q,
                            type: 'quest',
                            iconPath: 'inv_misc_book_11', // Generic Quest icon
                            quality: 1,
                            name: q.title // Map title to name
                        })
                    })
                }

                // Process Spells
                if (res.spells) {
                    res.spells.forEach(s => {
                        combined.push({
                            ...s,
                            type: 'spell',
                            iconPath: s.icon || 'spell_nature_starfall', // Use spell icon or fallback
                            quality: 1,
                            name: s.name
                        })
                    })
                }

                // Process Objects
                if (res.objects) {
                    res.objects.forEach(o => {
                        combined.push({
                            ...o,
                            type: 'object',
                            iconPath: 'inv_box_01', // Generic object icon
                            quality: 1
                        })
                    })
                }

                // Process Items
                if (res.items) {
                    res.items.forEach(i => {
                        combined.push({ ...i, type: 'item' })
                    })
                }

                setResults(combined)
                setTotalCount((res.items?.length || 0) + (res.creatures?.length || 0) + (res.quests?.length || 0) + (res.spells?.length || 0) + (res.objects?.length || 0))
                setLoading(false)
            })
            .catch(err => {
                console.error("Search failed:", err)
                setLoading(false)
            })
    }

    // Navigate to detail page
    const handleItemClick = (entry) => {
        if (onNavigate) {
            // Determine type based on entry in results list
            const item = results.find(r => r.entry === entry.entry && r.type === entry.type);
            if (item) {
                onNavigate(item.type, item.entry)
            }
        }
    }

    return (
        <div className="h-full flex flex-col bg-bg-dark p-4">
            {/* Search Header */}
            <div className="mb-5 p-4 bg-bg-panel rounded-lg border border-border-dark">
                {/* Search Input */}
                <div className="flex gap-3">
                    <input 
                        type="text" 
                        value={query} 
                        onChange={e => setQuery(e.target.value)}
                        placeholder="Search Items, NPCs, Quests, Objects, Spells (ID supported)..."
                        className="flex-1 px-4 py-2 bg-bg-main border border-border-dark rounded text-white text-base outline-none focus:border-wow-rare transition-colors"
                        onKeyDown={e => e.key === 'Enter' && handleSearch()}
                    />
                    <button 
                        onClick={handleSearch}
                        className="px-6 py-2 bg-wow-rare text-white rounded font-bold hover:bg-wow-rare/80 transition-colors"
                    >
                        Search
                    </button>
                </div>
            </div>

            {/* Results Info */}
            <div className="mb-3 text-sm text-gray-400">
                {loading ? (
                    <span className="text-wow-gold animate-pulse">Searching...</span>
                ) : (
                    <span>Found <b className="text-white">{totalCount}</b> results</span>
                )}
            </div>

            {/* Results Grid */}
            <div className="flex-1 overflow-y-auto grid grid-cols-[repeat(auto-fill,minmax(300px,1fr))] gap-2 content-start">
                {results.map((item, idx) => (
                    <div 
                        key={`${item.type}-${item.entry}-${idx}`} 
                        className="flex items-center bg-bg-panel p-2 border border-border-dark rounded hover:bg-bg-hover hover:border-wow-rare/50 transition-colors cursor-pointer"
                        onClick={() => handleItemClick(item)}
                        onMouseEnter={() => item.type === 'item' && onItemClick?.(item.entry, true)}
                        onMouseLeave={() => item.type === 'item' && onItemClick?.(item.entry, false)}
                    >
                        {/* Icon */}
                        <SearchResultIcon iconName={item.iconPath} type={item.type} displayId={item.displayId1} />
                        
                        {/* ID */}
                        <span className="text-gray-500 text-xs font-mono mr-2 min-w-[50px]">
                            #{item.entry}
                        </span>
                        
                        {/* Name & Details */}
                        <div className="flex-1 min-w-0">
                            <div 
                                className="font-bold truncate"
                                style={{ color: item.type === 'item' ? getQualityColor(item.quality) : (item.type === 'npc' ? '#FFD100' : (item.type === 'spell' ? '#a855f7' : (item.type === 'object' ? '#00B4FF' : '#fff'))) }}
                            >
                                {item.name}
                            </div>
                            <div className="text-xs text-gray-500">
                                {item.type === 'item' && `Item Lv ${item.itemLevel} (Req ${item.requiredLevel})`}
                                {item.type === 'npc' && `Level ${item.levelMin}${item.levelMin !== item.levelMax ? '-'+item.levelMax : ''}`}
                                {item.type === 'quest' && `Level ${item.questLevel} (Req ${item.minLevel})`}
                                {item.type === 'spell' && (item.description ? item.description.substring(0, 50) + (item.description.length > 50 ? '...' : '') : 'Spell')}
                                {item.type === 'object' && (item.typeName || 'Object')}
                            </div>
                        </div>
                    </div>
                ))}
            </div>
        </div>
    )
}

export default SearchPage
