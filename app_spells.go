package main

import (
	"fmt"
	"inklab/backend/database"
)

// GetSpellSkillCategories returns spell skill categories (Class Skills, Professions, etc.)
func (a *App) GetSpellSkillCategories() []*database.SpellSkillCategory {
	fmt.Println("[API] GetSpellSkillCategories called")
	cats, err := a.spellRepo.GetSpellSkillCategories()
	if err != nil {
		fmt.Printf("[API] Error: %v\n", err)
		return []*database.SpellSkillCategory{}
	}
	return cats
}

// GetSpellSkillsByCategory returns skills for a category
func (a *App) GetSpellSkillsByCategory(categoryID int) []*database.SpellSkill {
	skills, err := a.spellRepo.GetSpellSkillsByCategory(categoryID)
	if err != nil {
		fmt.Printf("[API] Error: %v\n", err)
		return []*database.SpellSkill{}
	}
	return skills
}

// GetSpellClasses returns the classes under the Class Skills category
func (a *App) GetSpellClasses() []*database.SpellClass {
	classes, err := a.spellRepo.GetSpellClasses()
	if err != nil {
		fmt.Printf("[API] Error: %v\n", err)
		return []*database.SpellClass{}
	}
	return classes
}

// GetSpellSkillsByClass returns the class skill lines for one class
func (a *App) GetSpellSkillsByClass(classID int) []*database.SpellSkill {
	skills, err := a.spellRepo.GetSpellSkillsByClass(classID)
	if err != nil {
		fmt.Printf("[API] Error: %v\n", err)
		return []*database.SpellSkill{}
	}
	return skills
}

// GetSpellsBySkill returns spells for a skill
func (a *App) GetSpellsBySkill(skillID int, nameFilter string) []*database.Spell {
	spells, err := a.spellRepo.GetSpellsBySkill(skillID, nameFilter)
	if err != nil {
		fmt.Printf("[API] Error: %v\n", err)
		return []*database.Spell{}
	}
	return spells
}

// GetSpellDetail returns full details for a spell
func (a *App) GetSpellDetail(entry int) (*database.SpellDetail, error) {
	s := a.spellRepo.GetSpellDetail(entry)
	if s == nil {
		return nil, fmt.Errorf("spell not found")
	}
	return s, nil
}

// SearchSpells searches for spells by name
func (a *App) SearchSpells(query string) []*database.Spell {
	spells, err := a.spellRepo.SearchSpells(query)
	if err != nil {
		fmt.Printf("Error searching spells: %v\n", err)
		return []*database.Spell{}
	}
	return spells
}
