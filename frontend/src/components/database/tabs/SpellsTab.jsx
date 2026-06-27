import { useState, useMemo } from 'react'
import { SidebarPanel, ContentPanel, ScrollList, SectionHeader, ListItem, EntityIcon } from '../../ui'
import { filterItems } from '../../../utils/databaseApi'
import {
    useSpellCategories,
    useSpellClasses,
    useSpellSkillsByCategory,
    useSpellSkillsByClass,
    useSpellsBySkill,
} from '../../../hooks/queries/spells'
import { useIcon } from '../../../services/useImage'

const SpellListItemIcon = ({ iconName, spellColor }) => {
    const icon = useIcon(iconName)

    // Fallback based on type
    let fallback = '/local-icons/inv_misc_questionmark.jpg'

    return (
        <div
            className="w-8 h-8 rounded border flex-shrink-0 bg-black/40 flex items-center justify-center overflow-hidden"
            style={{ borderColor: spellColor }}
        >
            {icon.loading ? (
                <div className="w-full h-full bg-white/5 animate-pulse" />
            ) : (
                <img
                    src={icon.src || fallback}
                    alt=""
                    className="w-full h-full object-cover"
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
    const [selectedCategory, setSelectedCategory] = useState(null)
    const [selectedClass, setSelectedClass] = useState(null)
    const [selectedSkill, setSelectedSkill] = useState(null)

    const [categoryFilter, setCategoryFilter] = useState('')
    const [skillFilter, setSkillFilter] = useState('')
    const [spellFilter, setSpellFilter] = useState('')

    // "Class Skills" gets an extra Class tier: Class Skills -> Warlock -> Affliction
    const isClassCategory = selectedCategory?.name === 'Class Skills'

    // Data via domain query hooks, keyed by the current selection. Each cascading
    // query enables only once its parent is selected; switching selection swaps
    // the cached entry. Downstream selection is reset in the click handlers.
    const categoriesQuery = useSpellCategories()
    const classesQuery = useSpellClasses(isClassCategory)
    const categorySkillsQuery = useSpellSkillsByCategory(selectedCategory?.id, !!selectedCategory && !isClassCategory)
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

    const filteredCategories = useMemo(() => filterItems(categories, categoryFilter), [categories, categoryFilter])
    const filteredClasses = useMemo(() => filterItems(classes, skillFilter), [classes, skillFilter])
    const filteredSkills = useMemo(() => filterItems(skills, skillFilter), [skills, skillFilter])
    const filteredSpells = useMemo(() => filterItems(spells, spellFilter), [spells, spellFilter])

    // Whether the middle pane is showing the class picker (vs. skill lines)
    const showingClassPicker = isClassCategory && !selectedClass

    let middleTitle = 'Select Category'
    if (selectedCategory) {
        if (showingClassPicker) middleTitle = `${selectedCategory.name} (${filteredClasses.length})`
        else if (isClassCategory && selectedClass) middleTitle = `${selectedClass.name} (${filteredSkills.length})`
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
                    {filteredCategories.map(cat => (
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
                    placeholder={showingClassPicker ? "Filter classes..." : "Filter skills..."}
                    onFilterChange={setSkillFilter}
                />
                <ScrollList>
                    {/* Class picker (Class Skills only) */}
                    {showingClassPicker && filteredClasses.map(cls => (
                        <ListItem
                            key={cls.id}
                            onClick={() => pickClass(cls)}
                        >
                            <span className="flex justify-between w-full">
                                <span style={{ color: cls.color || undefined }}>{cls.name}</span>
                                <span className="text-gray-600 text-xs">({cls.skillCount})</span>
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
                    {!showingClassPicker && filteredSkills.map(skill => (
                        <ListItem
                            key={skill.id}
                            active={selectedSkill?.id === skill.id}
                            onClick={() => pickSkill(skill)}
                        >
                            <span className="flex justify-between w-full">
                                <span>{skill.name}</span>
                                <span className="text-gray-600 text-xs">({skill.spellCount})</span>
                            </span>
                        </ListItem>
                    ))}
                </ScrollList>
            </SidebarPanel>

            {/* 3. Spells List */}
            <ContentPanel className="col-span-2">
                <SectionHeader
                    title={selectedSkill ? `${selectedSkill.name} (${filteredSpells.length})` : 'Select Skill'}
                    placeholder="Filter spells..."
                    onFilterChange={setSpellFilter}
                    titleColor={SPELL_COLOR}
                />

                {spellsQuery.isLoading && (
                    <div className="flex-1 flex items-center justify-center italic animate-pulse" style={{ color: SPELL_COLOR }}>
                        Loading spells...
                    </div>
                )}

                {!selectedSkill && (
                    <div className="flex-1 flex items-center justify-center text-gray-600 italic">
                        Select a skill to browse spells.
                    </div>
                )}

                {!spellsQuery.isLoading && spells.length > 0 && (
                    <ScrollList className="p-2 space-y-1">
                        {filteredSpells.map(spell => (
                            <div
                                key={spell.entry}
                                className="flex items-center gap-3 p-2 bg-white/[0.02] hover:bg-white/5 border-l-[3px] transition-colors rounded-r cursor-pointer"
                                style={{ borderLeftColor: SPELL_COLOR }}
                                onClick={() => onNavigate && onNavigate('spell', spell.entry)}
                                {...(tooltipHook?.getSpellHandlers?.(spell.entry) || {})}
                            >
                                {spell.icon ? (
                                    <SpellListItemIcon iconName={spell.icon} spellColor={SPELL_COLOR} />
                                ) : (
                                    <EntityIcon
                                        label="SPL"
                                        color={SPELL_COLOR}
                                        size="md"
                                    />
                                )}

                                <span className="text-gray-600 text-[11px] font-mono min-w-[50px]">
                                    [{spell.entry}]
                                </span>

                                <div className="flex flex-col flex-1 min-w-0">
                                    <span
                                        className="font-bold truncate"
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
                                        <span className="text-gray-500 text-xs truncate mt-0.5">
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
