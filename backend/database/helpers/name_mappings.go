// Package helpers contains utility functions for database operations
package helpers

// GetClassName returns the item class name
func GetClassName(c int) string {
	classNames := map[int]string{
		0:  "Consumable",
		1:  "Container",
		2:  "Weapon",
		3:  "Gem",
		4:  "Armor",
		5:  "Reagent",
		6:  "Projectile",
		7:  "Trade Goods",
		8:  "Generic (OBSOLETE)",
		9:  "Recipe",
		10: "Money (OBSOLETE)",
		11: "Quiver",
		12: "Quest",
		13: "Key",
		14: "Permanent (OBSOLETE)",
		15: "Miscellaneous",
	}
	if name, ok := classNames[c]; ok {
		return name
	}
	return "Unknown"
}

// GetSubClassName returns the item subclass name
func GetSubClassName(c, sc int) string {
	// Weapon subclasses
	if c == 2 {
		weaponSubclasses := map[int]string{
			0:  "Axe",
			1:  "Two-Handed Axe",
			2:  "Bow",
			3:  "Gun",
			4:  "Mace",
			5:  "Two-Handed Mace",
			6:  "Polearm",
			7:  "Sword",
			8:  "Two-Handed Sword",
			9:  "Obsolete",
			10: "Staff",
			11: "Exotic",
			12: "Exotic",
			13: "Fist Weapon",
			14: "Miscellaneous",
			15: "Dagger",
			16: "Thrown",
			17: "Spear",
			18: "Crossbow",
			19: "Wand",
			20: "Fishing Pole",
		}
		if name, ok := weaponSubclasses[sc]; ok {
			return name
		}
	}

	// Armor subclasses
	if c == 4 {
		armorSubclasses := map[int]string{
			0:  "Miscellaneous",
			1:  "Cloth",
			2:  "Leather",
			3:  "Mail",
			4:  "Plate",
			5:  "Buckler (OBSOLETE)",
			6:  "Shield",
			7:  "Libram",
			8:  "Idol",
			9:  "Totem",
			10: "Sigil",
		}
		if name, ok := armorSubclasses[sc]; ok {
			return name
		}
	}

	// Container subclasses
	if c == 1 {
		containerSubclasses := map[int]string{
			0: "Bag",
			1: "Soul Bag",
			2: "Herb Bag",
			3: "Enchanting Bag",
			4: "Engineering Bag",
			5: "Gem Bag",
			6: "Mining Bag",
			7: "Leatherworking Bag",
			8: "Inscription Bag",
			9: "Fishing Bag",
		}
		if name, ok := containerSubclasses[sc]; ok {
			return name
		}
	}

	// Consumable subclasses
	if c == 0 {
		consumableSubclasses := map[int]string{
			0: "Consumable",
			1: "Potion",
			2: "Elixir",
			3: "Flask",
			4: "Scroll",
			5: "Food & Drink",
			6: "Item Enhancement",
			7: "Bandage",
			8: "Other",
		}
		if name, ok := consumableSubclasses[sc]; ok {
			return name
		}
	}

	// Reagent subclass
	if c == 5 {
		return "Reagent"
	}

	// Projectile subclasses
	if c == 6 {
		projectileSubclasses := map[int]string{
			0: "Wand (OBSOLETE)",
			1: "Bolt (OBSOLETE)",
			2: "Arrow",
			3: "Bullet",
			4: "Thrown (OBSOLETE)",
		}
		if name, ok := projectileSubclasses[sc]; ok {
			return name
		}
	}

	// Trade Goods subclasses
	if c == 7 {
		tradeSubclasses := map[int]string{
			0:  "Trade Goods",
			1:  "Parts",
			2:  "Explosives",
			3:  "Devices",
			4:  "Jewelcrafting",
			5:  "Cloth",
			6:  "Leather",
			7:  "Metal & Stone",
			8:  "Meat",
			9:  "Herb",
			10: "Elemental",
			11: "Other",
			12: "Enchanting",
			13: "Materials",
			14: "Armor Enchantment",
			15: "Weapon Enchantment",
		}
		if name, ok := tradeSubclasses[sc]; ok {
			return name
		}
	}

	// Recipe subclasses
	if c == 9 {
		recipeSubclasses := map[int]string{
			0:  "Book",
			1:  "Leatherworking",
			2:  "Tailoring",
			3:  "Engineering",
			4:  "Blacksmithing",
			5:  "Cooking",
			6:  "Alchemy",
			7:  "First Aid",
			8:  "Enchanting",
			9:  "Fishing",
			10: "Jewelcrafting",
		}
		if name, ok := recipeSubclasses[sc]; ok {
			return name
		}
	}

	// Quiver subclasses
	if c == 11 {
		quiverSubclasses := map[int]string{
			0: "Quiver (OBSOLETE)",
			1: "Quiver (OBSOLETE)",
			2: "Quiver",
			3: "Ammo Pouch",
		}
		if name, ok := quiverSubclasses[sc]; ok {
			return name
		}
	}

	// Quest subclass
	if c == 12 {
		return "Quest"
	}

	// Key subclasses
	if c == 13 {
		keySubclasses := map[int]string{
			0: "Key",
			1: "Lockpick",
		}
		if name, ok := keySubclasses[sc]; ok {
			return name
		}
	}

	// Miscellaneous subclasses
	if c == 15 {
		miscSubclasses := map[int]string{
			0: "Junk",
			1: "Reagent",
			2: "Pet",
			3: "Holiday",
			4: "Other",
			5: "Mount",
		}
		if name, ok := miscSubclasses[sc]; ok {
			return name
		}
	}

	return "Unknown"
}

// GetInventoryTypeName returns the inventory slot name
func GetInventoryTypeName(invType int) string {
	invTypeNames := map[int]string{
		0:  "Non-equippable",
		1:  "Head",
		2:  "Neck",
		3:  "Shoulder",
		4:  "Shirt",
		5:  "Chest",
		6:  "Waist",
		7:  "Legs",
		8:  "Feet",
		9:  "Wrists",
		10: "Hands",
		11: "Finger",
		12: "Trinket",
		13: "One-Hand",
		14: "Shield",
		15: "Ranged",
		16: "Back",
		17: "Two-Hand",
		18: "Bag",
		19: "Tabard",
		20: "Robe",
		21: "Main Hand",
		22: "Off Hand",
		23: "Holdable",
		24: "Ammo",
		25: "Thrown",
		26: "Ranged Right",
		27: "Quiver",
		28: "Relic",
	}
	if name, ok := invTypeNames[invType]; ok {
		return name
	}
	return "Unknown"
}

// GetBondingName returns the bonding type name
func GetBondingName(bonding int) string {
	switch bonding {
	case 1:
		return "Binds when picked up"
	case 2:
		return "Binds when equipped"
	case 3:
		return "Binds when used"
	case 4:
		return "Quest Item"
	default:
		return ""
	}
}

// GetQualityName returns the quality name
func GetQualityName(quality int) string {
	switch quality {
	case 0:
		return "Poor"
	case 1:
		return "Common"
	case 2:
		return "Uncommon"
	case 3:
		return "Rare"
	case 4:
		return "Epic"
	case 5:
		return "Legendary"
	case 6:
		return "Artifact"
	default:
		return "Unknown"
	}
}

// GetCreatureTypeName returns the creature type name
func GetCreatureTypeName(t int) string {
	typeNames := map[int]string{
		0:  "None",
		1:  "Beast",
		2:  "Dragonkin",
		3:  "Demon",
		4:  "Elemental",
		5:  "Giant",
		6:  "Undead",
		7:  "Humanoid",
		8:  "Critter",
		9:  "Mechanical",
		10: "Not Specified",
		11: "Totem",
	}
	if name, ok := typeNames[t]; ok {
		return name
	}
	return "Unknown"
}

// GetCreatureRankName returns the creature rank name
func GetCreatureRankName(r int) string {
	rankNames := map[int]string{
		0: "Normal",
		1: "Elite",
		2: "Rare Elite",
		3: "Boss",
		4: "Rare",
	}
	if name, ok := rankNames[r]; ok {
		return name
	}
	return "Normal"
}

// GetTriggerPrefix returns the spell trigger prefix
func GetTriggerPrefix(trigger int) string {
	switch trigger {
	case 0:
		return "Use: "
	case 1:
		return "Equip: "
	case 2:
		return "Chance on hit: "
	case 4:
		return "Soulstone: "
	case 5:
		return "Use: (no cooldown) "
	case 6:
		return "Learn: "
	default:
		return ""
	}
}

// StatNames is the canonical item stat_type -> display name map (the ITEM_MOD
// enum). It's the single source of truth for stat names across the app: the
// stat_types table is seeded from it, then the base stats (Strength, Agility,
// Stamina, Intellect, Spirit) are overlaid with the client's localized names at
// import time. The secondary/rating stats (12+) have no string in the 1.12
// client, so their English names live here and stay built-in.
var StatNames = map[int]string{
	0: "Mana", 1: "Health", 3: "Agility", 4: "Strength",
	5: "Intellect", 6: "Spirit", 7: "Stamina",
	12: "Defense Rating", 13: "Dodge Rating", 14: "Parry Rating",
	15: "Shield Block Rating", 16: "Melee Hit Rating", 17: "Ranged Hit Rating",
	18: "Spell Hit Rating", 19: "Melee Critical Rating", 20: "Ranged Critical Rating",
	21: "Spell Critical Rating", 35: "Resilience Rating", 36: "Haste Rating",
	37: "Expertise Rating", 38: "Attack Power", 39: "Ranged Attack Power",
	41: "Spell Healing", 42: "Spell Damage", 43: "Mana Regeneration",
	44: "Armor Penetration Rating", 45: "Spell Power",
}

// GetStatName returns the canonical (built-in English) name for a stat_type id,
// or "" if unknown. Runtime callers should prefer the stat_types table (which
// may carry localized base-stat names); this is the fallback/seed source.
func GetStatName(statType int) string {
	return StatNames[statType]
}

// DecodeSpellAttributes turns the spell_template attribute bitfields into a list
// of human labels using the server-source flag table (SpellAttrFlags). fields is
// indexed: 0=attributes, 1=attributesEx .. 4=attributesEx4, 5=customFlags.
func DecodeSpellAttributes(fields [6]uint32) []string {
	var out []string
	for _, f := range SpellAttrFlags {
		if f.Field < len(fields) && fields[f.Field]&f.Mask != 0 {
			out = append(out, f.Name)
		}
	}
	return out
}

// GetSchoolName returns the magic school name for damage types
func GetSchoolName(school int) string {
	switch school {
	case 0:
		return "Physical"
	case 1:
		return "Holy"
	case 2:
		return "Fire"
	case 3:
		return "Nature"
	case 4:
		return "Frost"
	case 5:
		return "Shadow"
	case 6:
		return "Arcane"
	default:
		return "Physical"
	}
}
