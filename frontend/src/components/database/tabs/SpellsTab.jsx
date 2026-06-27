import { useMemo } from 'react'
import { useStickyState } from '../../../hooks/useStickyState'
import {
    SidebarPanel,
    ContentPanel,
    ScrollList,
    SectionHeader,
    ListItem,
    EntityIcon,
} from '../../ui'
import { filterItems } from '../../../utils/databaseApi'
import {
    useSpellCategories,
    useSpellClasses,
    useSpellSkillsByCategory,
    useSpellSkillsByClass,
    useSpellsBySkill,
} from '../../../hooks/queries/spells'
import { useIcon } from '../../../services/useImage'
import { QUESTION_MARK_ICON } from '../../../utils/wow'

const SpellListItemIcon = ({ iconName, spellColor }) => {
    const icon = useIcon(iconName)

    return (
        <div
            className="flex h-8 w-8 flex-shrink-0 items-center justify-center overflow-hidden rounded border bg-black/40"
            style={{ borderColor: spellColor }}
        >
            {icon.loading ? (
                <div className="h-full w-full animate-pulse bg-white/5" />
            ) : (
                <img
                    src={icon.src || QUESTION_MARK_ICON}
                    alt=""
                    className="h-full w-full object-cover"
                />
            )}
        </div>
    )
}

// ... inside render:
// Replace the entire icon block with:
// <SpellListItemIcon iconName={spell.icon} spellColor={SPELL_COLOR} />

const SPELL_COLOR = '#772ce8'

function SpellsTab({ onNavigate, tooltipHook }) {
    const [selectedCategory, setSelectedCategory] = useStickyState('spells.selectedCategory', null)
    const [selectedClass, setSelectedClass] = useStickyState('spells.selectedClass', null)
    const [selectedSkill, setSelectedSkill] = useStickyState('spells.selectedSkill', null)

    const [categoryFilter, setCategoryFilter] = useStickyState('spells.categoryFilter', '')
    const [skillFilter, setSkillFilter] = useStickyState('spells.skillFilter', '')
    const [spellFilter, setSpellFilter] = useStickyState('spells.spellFilter', '')

    // "Class Skills" gets an extra Class tier: Class Skills -> Warlock -> Affliction
    const isClassCategory = selectedCategory?.name === 'Class Skills'

    // Data via domain query hooks, keyed by the current selection. Each cascading
    // query enables only once its parent is selected; switching selection swaps
    // the cached entry. Downstream selection is reset in the click handlers.
    const categoriesQuery = useSpellCategories()
    const classesQuery = useSpellClasses(isClassCategory)
    const categorySkillsQuery = useSpellSkillsByCategory(
        selectedCategory?.id,
        !!selectedCategory && !isClassCategory,
    )
    const classSkillsQuery = useSpellSkillsByClass(selectedClass?.id, !!selectedClass)
    const spellsQuery = useSpellsBySkill(selectedSkill?.id, !!selectedSkill)

    const categories = categoriesQuery.data || []
    const classes = classesQuery.data || []
    const skills = (selectedClass ? classSkillsQuery.data : categorySkillsQuery.data) || []
    const spells = spellsQuery.data || []

    // Selecting a level resets everything below it (handler-driven, no effects).
    const pickCategory = (cat) => {
        setSelectedCategory(cat)
        setSelectedClass(null)
        setSelectedSkill(null)
        setSkillFilter('')
        setSpellFilter('')
    }
    const pickClass = (cls) => {
        setSelectedClass(cls)
        setSelectedSkill(null)
        setSkillFilter('')
        setSpellFilter('')
    }
    const pickSkill = (skill) => {
        setSelectedSkill(skill)
        setSpellFilter('')
    }
    const backToClasses = () => {
        setSelectedClass(null)
        setSelectedSkill(null)
        setSkillFilter('')
    }

    const filteredCategories = useMemo(
        () => filterItems(categories, categoryFilter),
        [categories, categoryFilter],
    )
    const filteredClasses = useMemo(() => filterItems(classes, skillFilter), [classes, skillFilter])
    const filteredSkills = useMemo(() => filterItems(skills, skillFilter), [skills, skillFilter])
    const filteredSpells = useMemo(() => filterItems(spells, spellFilter), [spells, spellFilter])

    // Whether the middle pane is showing the class picker (vs. skill lines)
    const showingClassPicker = isClassCategory && !selectedClass

    let middleTitle = 'Select Category'
    if (selectedCategory) {
        if (showingClassPicker) middleTitle = `${selectedCategory.name} (${filteredClasses.length})`
        else if (isClassCategory && selectedClass)
            middleTitle = `${selectedClass.name} (${filteredSkills.length})`
        else middleTitle = `${selectedCategory.name} (${filteredSkills.length})`
    }

    return (
        <>
            {/* 1. Categories */}
            <SidebarPanel>
                <SectionHeader
                    title={`Categories (${filteredCategories.length})`}
                    placeholder="Filter categories..."
                    onFilterChange={setCategoryFilter}
                />
                <ScrollList>
                    {filteredCategories.map((cat) => (
                        <ListItem
                            key={cat.id}
                            active={selectedCategory?.id === cat.id}
                            onClick={() => pickCategory(cat)}
                        >
                            {cat.name}
                        </ListItem>
                    ))}
                </ScrollList>
            </SidebarPanel>

            {/* 2. Classes / Skills */}
            <SidebarPanel>
                <SectionHeader
                    title={middleTitle}
                    placeholder={showingClassPicker ? 'Filter classes...' : 'Filter skills...'}
                    onFilterChange={setSkillFilter}
                />
                <ScrollList>
                    {/* Class picker (Class Skills only) */}
                    {showingClassPicker &&
                        filteredClasses.map((cls) => (
                            <ListItem key={cls.id} onClick={() => pickClass(cls)}>
                                <span className="flex w-full justify-between">
                                    <span style={{ color: cls.color || undefined }}>
                                        {cls.name}
                                    </span>
                                    <span className="text-xs text-gray-600">
                                        ({cls.skillCount})
                                    </span>
                                </span>
                            </ListItem>
                        ))}

                    {/* Back to classes */}
                    {isClassCategory && selectedClass && (
                        <ListItem onClick={backToClasses}>
                            <span className="text-gray-400">← All classes</span>
                        </ListItem>
                    )}

                    {/* Skill lines */}
                    {!showingClassPicker &&
                        filteredSkills.map((skill) => (
                            <ListItem
                                key={skill.id}
                                active={selectedSkill?.id === skill.id}
                                onClick={() => pickSkill(skill)}
                            >
                                <span className="flex w-full justify-between">
                                    <span>{skill.name}</span>
                                    <span className="text-xs text-gray-600">
                                        ({skill.spellCount})
                                    </span>
                                </span>
                            </ListItem>
                        ))}
                </ScrollList>
            </SidebarPanel>

            {/* 3. Spells List */}
            <ContentPanel className="col-span-2">
                <SectionHeader
                    title={
                        selectedSkill
                            ? `${selectedSkill.name} (${filteredSpells.length})`
                            : 'Select Skill'
                    }
                    placeholder="Filter spells..."
                    onFilterChange={setSpellFilter}
                    titleColor={SPELL_COLOR}
                />

                {spellsQuery.isLoading && (
                    <div
                        className="flex flex-1 animate-pulse items-center justify-center italic"
                        style={{ color: SPELL_COLOR }}
                    >
                        Loading spells...
                    </div>
                )}

                {!selectedSkill && (
                    <div className="flex flex-1 items-center justify-center italic text-gray-600">
                        Select a skill to browse spells.
                    </div>
                )}

                {!spellsQuery.isLoading && spells.length > 0 && (
                    <ScrollList className="space-y-1 p-2">
                        {filteredSpells.map((spell) => (
                            <div
                                key={spell.entry}
                                className="flex cursor-pointer items-center gap-3 rounded-r border-l-[3px] bg-white/[0.02] p-2 transition-colors hover:bg-white/5"
                                style={{ borderLeftColor: SPELL_COLOR }}
                                onClick={() => onNavigate && onNavigate('spell', spell.entry)}
                                {...(tooltipHook?.getSpellHandlers?.(spell.entry) || {})}
                            >
                                {spell.icon ? (
                                    <SpellListItemIcon
                                        iconName={spell.icon}
                                        spellColor={SPELL_COLOR}
                                    />
                                ) : (
                                    <EntityIcon label="SPL" color={SPELL_COLOR} size="md" />
                                )}

                                <span className="min-w-[50px] font-mono text-[11px] text-gray-600">
                                    [{spell.entry}]
                                </span>

                                <div className="flex min-w-0 flex-1 flex-col">
                                    <span
                                        className="truncate font-bold"
                                        style={{ color: SPELL_COLOR }}
                                    >
                                        {spell.name}
                                        {spell.subname && (
                                            <span className="ml-1.5 text-[11px] font-normal text-gray-500">
                                                {spell.subname}
                                            </span>
                                        )}
                                    </span>
                                    {spell.description && (
                                        <span className="mt-0.5 truncate text-xs text-gray-500">
                                            {spell.description.length > 100
                                                ? spell.description.substring(0, 100) + '...'
                                                : spell.description}
                                        </span>
                                    )}
                                </div>
                            </div>
                        ))}
                    </ScrollList>
                )}
            </ContentPanel>
        </>
    )
}

export default SpellsTab
