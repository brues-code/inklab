import { useMemo, useState } from 'react'
import { useStickyState } from '../../../hooks/useStickyState'
import { SidebarPanel, ContentPanel, ScrollList, SectionHeader, ListItem } from '../../ui'
import { getQualityColor, QUESTION_MARK_ICON } from '../../../utils/wow'
import { useIcon } from '../../../services/useImage'
import { useProfessions, useProfessionRecipes } from '../../../hooks/queries/professions'

// Skill-up threshold colors (the in-game recipe-difficulty colors).
const SKILL_COLORS = ['#ff8040', '#ffff00', '#40bf40', '#9d9d9d']

// The [learn/yellow/green/grey] chips: at what skill the recipe is learnable,
// stops always skilling up, turns green, and turns grey.
function Thresholds({ r }) {
    return (
        <div
            className="flex w-[150px] shrink-0 justify-end gap-1 font-mono text-[11px]"
            title="Learnable / yellow / green / grey"
        >
            {[r.learn, r.yellow, r.green, r.grey].map((v, i) => (
                <span key={i} className="min-w-[30px] text-right" style={{ color: SKILL_COLORS[i] }}>
                    {v}
                </span>
            ))}
        </div>
    )
}

// Small reagent icon with count badge; hover = item tooltip, click = item page.
function ReagentChip({ item, onNavigate, tooltipHook }) {
    const icon = useIcon(item.icon)
    const handlers = tooltipHook?.getItemHandlers?.(item.entry) || {}
    return (
        <div
            className="relative h-6 w-6 shrink-0 cursor-pointer overflow-hidden rounded border border-gray-700 bg-black/40"
            onClick={(e) => {
                e.stopPropagation()
                onNavigate?.('item', item.entry)
            }}
            {...handlers}
        >
            <img src={icon.src || QUESTION_MARK_ICON} alt="" className="h-full w-full object-cover" />
            {item.count > 1 && (
                <span className="absolute bottom-0 right-0 rounded-tl bg-black/80 px-0.5 text-[9px] font-bold leading-tight text-white">
                    {item.count}
                </span>
            )}
        </div>
    )
}

function RecipeRow({ r, onNavigate, tooltipHook }) {
    const icon = useIcon(r.crafts?.icon || r.icon)
    // The row hovers as the CRAFT (item tooltip when it makes an item, spell
    // tooltip otherwise, e.g. enchants) and clicks through to the spell page.
    const hoverHandlers = r.crafts
        ? tooltipHook?.getItemHandlers?.(r.crafts.entry) || {}
        : tooltipHook?.getSpellHandlers?.(r.spellId) || {}
    const nameColor = r.crafts ? getQualityColor(r.crafts.quality) : '#71d5ff'

    return (
        <div
            className="flex cursor-pointer items-center gap-2 border-b border-white/5 px-2 py-1.5 transition-colors hover:bg-white/5"
            onClick={() => onNavigate?.('spell', r.spellId)}
            {...hoverHandlers}
        >
            <div className="h-8 w-8 shrink-0 overflow-hidden rounded border border-gray-700 bg-black/40">
                <img
                    src={icon.src || QUESTION_MARK_ICON}
                    alt=""
                    className="h-full w-full object-cover"
                />
            </div>

            <div className="min-w-0 flex-1">
                <div className="truncate text-[13px] font-bold" style={{ color: nameColor }}>
                    {r.name}
                    {r.crafts?.count > 1 && (
                        <span className="ml-1 text-[11px] text-gray-400">x{r.crafts.count}</span>
                    )}
                </div>
                {/* Source line: how the recipe is learned */}
                <div className="flex items-center gap-2 text-[10px] text-gray-500">
                    {r.trainer && <span className="text-gray-400">Trainer</span>}
                    {r.teachItem && (
                        <span
                            className="cursor-pointer hover:underline"
                            style={{ color: getQualityColor(r.teachItem.quality) }}
                            onClick={(e) => {
                                e.stopPropagation()
                                onNavigate?.('item', r.teachItem.entry)
                            }}
                            {...(tooltipHook?.getItemHandlers?.(r.teachItem.entry) || {})}
                        >
                            {r.teachItem.name}
                        </span>
                    )}
                    {r.quest && <span className="text-wow-gold">Quest</span>}
                    {!r.trainer && !r.teachItem && !r.quest && <span>—</span>}
                </div>
            </div>

            {/* Reagents */}
            <div className="flex max-w-[220px] shrink-0 flex-wrap justify-end gap-1">
                {r.reagents.map((re) => (
                    <ReagentChip
                        key={re.entry}
                        item={re}
                        onNavigate={onNavigate}
                        tooltipHook={tooltipHook}
                    />
                ))}
            </div>

            <Thresholds r={r} />
        </div>
    )
}

/**
 * Professions tab: primary professions + crafting secondaries (from
 * SkillLine/SkillLineAbility) in the sidebar; the selected profession's full
 * recipe list on the right, ordered by learnable rank, with skill-up threshold
 * chips, reagent icons, the crafted item, and how each recipe is learned
 * (trainer / recipe item / quest). Rows click through to the spell page.
 */
function ProfessionsTab({ onNavigate, tooltipHook }) {
    const [selectedId, setSelectedId] = useStickyState('professions.selected', null)
    const [filter, setFilter] = useState('')
    // Sort: 'skill' (leveling-path order, the backend default) or 'name';
    // clicking the active column header flips direction.
    const [sort, setSort] = useState({ key: 'skill', dir: 'asc' })
    const toggleSort = (key) =>
        setSort((s) =>
            s.key === key ? { key, dir: s.dir === 'asc' ? 'desc' : 'asc' } : { key, dir: 'asc' },
        )

    const { data: professions = [], isLoading: loadingProfs } = useProfessions()
    const { data: recipes = [], isLoading: loadingRecipes } = useProfessionRecipes(selectedId)

    const selected = professions.find((p) => p.id === selectedId)

    const filtered = useMemo(() => {
        const f = filter.trim().toLowerCase()
        let list = recipes
        if (f) {
            list = recipes.filter(
                (r) =>
                    r.name.toLowerCase().includes(f) ||
                    r.crafts?.name?.toLowerCase().includes(f) ||
                    r.reagents.some((re) => re.name.toLowerCase().includes(f)),
            )
        }
        // Effective difficulty, mirroring the backend's sort key (learn rank,
        // or the yellow threshold when learn is a placeholder 1).
        const skillKey = (r) => Math.max(r.learn, r.yellow)
        const sorted = [...list].sort((a, b) =>
            sort.key === 'name' ? a.name.localeCompare(b.name) : skillKey(a) - skillKey(b),
        )
        if (sort.dir === 'desc') sorted.reverse()
        return sorted
    }, [recipes, filter, sort])

    return (
        <>
            <SidebarPanel className="col-span-1">
                <SectionHeader title={`Professions (${professions.length})`} noSearch />
                <ScrollList>
                    {loadingProfs && (
                        <div className="animate-pulse p-3 text-xs italic text-gray-500">
                            Loading...
                        </div>
                    )}
                    {professions.map((p) => (
                        <ListItem
                            key={p.id}
                            active={selectedId === p.id}
                            onClick={() => setSelectedId(p.id)}
                        >
                            <span className="flex w-full items-center justify-between">
                                <span>{p.name}</span>
                                <span className="text-[10px] text-gray-500">{p.count}</span>
                            </span>
                        </ListItem>
                    ))}
                </ScrollList>
            </SidebarPanel>

            <ContentPanel className="col-span-3">
                <SectionHeader
                    title={
                        selected
                            ? `${selected.name} (${filtered.length})`
                            : 'Select a Profession'
                    }
                    placeholder="Filter by recipe, item, or reagent..."
                    onFilterChange={setFilter}
                />

                {selectedId && loadingRecipes && (
                    <div className="flex flex-1 animate-pulse items-center justify-center italic text-wow-gold">
                        Loading recipes...
                    </div>
                )}

                {selectedId && !loadingRecipes && (
                    <>
                        {/* Column headers: Name and Skill sort the list; the
                            active column shows the direction arrow. */}
                        <div className="flex items-center justify-between border-b border-border-dark bg-black/20 px-2 py-1 font-mono text-[9px] uppercase text-gray-600">
                            <button
                                className={`uppercase hover:text-white ${sort.key === 'name' ? 'text-gray-300' : ''}`}
                                onClick={() => toggleSort('name')}
                            >
                                Name {sort.key === 'name' && (sort.dir === 'asc' ? '▲' : '▼')}
                            </button>
                            <button
                                className="flex items-center gap-1 hover:brightness-125"
                                onClick={() => toggleSort('skill')}
                                title="Sort by skill (learn / yellow / green / grey)"
                            >
                                <span
                                    className={`mr-[2px] uppercase ${sort.key === 'skill' ? 'text-gray-300' : ''}`}
                                >
                                    skill {sort.key === 'skill' && (sort.dir === 'asc' ? '▲' : '▼')}
                                </span>
                                {['learn', 'yellow', 'green', 'grey'].map((label, i) => (
                                    <span
                                        key={label}
                                        className="min-w-[30px] text-right"
                                        style={{ color: SKILL_COLORS[i] }}
                                    >
                                        {label}
                                    </span>
                                ))}
                            </button>
                        </div>
                        <ScrollList>
                            {filtered.map((r) => (
                                <RecipeRow
                                    key={r.spellId}
                                    r={r}
                                    onNavigate={onNavigate}
                                    tooltipHook={tooltipHook}
                                />
                            ))}
                        </ScrollList>
                    </>
                )}

                {!selectedId && (
                    <div className="flex flex-1 items-center justify-center italic text-gray-600">
                        Select a profession to browse its recipes
                    </div>
                )}
            </ContentPanel>
        </>
    )
}

export default ProfessionsTab
