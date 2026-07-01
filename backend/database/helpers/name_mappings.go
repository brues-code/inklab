// Package helpers contains utility functions for database operations
package helpers

import "fmt"

// Client-localized item type-name overrides, populated once at startup from the
// DBC-derived reference tables (item_class_names / item_subclass_names /
// inventory_type_names). When set, the Get* resolvers prefer these over their
// built-in English maps, so type/slot labels follow the client locale. Nil
// until loaded (or when a client import hasn't run) → built-in fallback.
var (
	itemClassNames      map[int]string
	itemSubclassShort   map[[2]int]string // (class,subclass) -> short name ("Axe")
	itemSubclassVerbose map[[2]int]string // (class,subclass) -> "One-Handed Axes"
	inventoryTypeNames  map[int]string
	creatureTypeNames   map[int]string    // CreatureType.dbc id -> name
	clientStrings       map[string]string // GlobalStrings key -> localized value
	schoolNames         map[int]string    // spell school index -> name (SPELL_SCHOOLn_CAP)
)

// SetSchoolNames installs client-localized spell-school names (from the
// spell_schools table, sourced from GlobalStrings). Nil/empty → built-in fallback.
func SetSchoolNames(m map[int]string) { schoolNames = m }

// SetCreatureTypeNames installs client-localized creature type names (from
// CreatureType.dbc). Nil/empty → built-in fallback.
func SetCreatureTypeNames(m map[int]string) { creatureTypeNames = m }

// SetClientStrings installs curated GlobalStrings UI values (item quality, bind
// type, spell-trigger prefix, creature rank). Nil/empty → built-in fallback.
func SetClientStrings(m map[string]string) { clientStrings = m }

// clientString returns a loaded GlobalStrings value, or "" when not present.
func clientString(key string) string {
	if clientStrings != nil {
		return clientStrings[key]
	}
	return ""
}

// SetItemNameTables installs the client-localized type-name lookups. Called once
// after the reference tables are imported; safe to call with nil maps (no-op
// override → built-in fallback). Not safe for concurrent use with the getters,
// so call during single-threaded startup before serving requests.
func SetItemNameTables(class map[int]string, subShort, subVerbose map[[2]int]string, inv map[int]string) {
	itemClassNames = class
	itemSubclassShort = subShort
	itemSubclassVerbose = subVerbose
	inventoryTypeNames = inv
}

// GetClassName returns the item class name
func GetClassName(c int) string {
	if itemClassNames != nil {
		if n, ok := itemClassNames[c]; ok && n != "" {
			return n
		}
	}
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

// GetSubClassName returns the item subclass name — the client's short form
// ("Sword", "Mace"), matching how tooltips/lists pair it with the equip slot
// ("Two-Hand" + "Sword"), like the game and Wowhead. The 1H/2H distinction
// comes from the slot, not this name. (itemSubclassVerbose holds the long
// "One-/Two-Handed Swords" form for a future standalone filter that lacks slot
// context.) Falls back to built-in names.
func GetSubClassName(c, sc int) string {
	if itemSubclassShort != nil {
		if n, ok := itemSubclassShort[[2]int{c, sc}]; ok && n != "" {
			return n
		}
	}
	// Weapon subclasses (short/family names; the slot carries One-/Two-Hand).
	if c == 2 {
		weaponSubclasses := map[int]string{
			0:  "Axe",
			1:  "Axe",
			2:  "Bow",
			3:  "Gun",
			4:  "Mace",
			5:  "Mace",
			6:  "Polearm",
			7:  "Sword",
			8:  "Sword",
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

	// Miscellaneous subclasses: the 1.12 client DBC only names subclass 0 (Junk) —
	// companion pets and mounts predate those item categories, so the client has
	// no name and they'd collapse to "Miscellaneous". Provide them, matching the
	// data's layout (subclass 2 = companion pets, 4 = mounts).
	if c == 15 {
		miscSubclasses := map[int]string{
			0: "Junk",
			1: "Reagent",
			2: "Companion",
			3: "Holiday",
			4: "Mount",
		}
		if name, ok := miscSubclasses[sc]; ok {
			return name
		}
	}

	// Other item classes: their subclass names come from the client table
	// (item_subclass_names); pre-import we fall back to the class name rather
	// than maintaining exhaustive hardcoded maps.
	return GetClassName(c)
}

// GetSubClassFamilyName returns the short, family-level subclass name ("Sword",
// "Axe") rather than the verbose 1H/2H form — for the filter sidebar, which
// groups 1H/2H weapons under one family. Prefers the client's short name.
func GetSubClassFamilyName(c, sc int) string {
	if itemSubclassShort != nil {
		if n, ok := itemSubclassShort[[2]int{c, sc}]; ok && n != "" {
			return n
		}
	}
	return GetSubClassName(c, sc)
}

// GetInventoryTypeName returns the inventory slot name
func GetInventoryTypeName(invType int) string {
	if inventoryTypeNames != nil {
		if n, ok := inventoryTypeNames[invType]; ok && n != "" {
			return n
		}
	}
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

// bondingKey maps a bonding id to its GlobalStrings key.
var bondingKey = map[int]string{1: "ITEM_BIND_ON_PICKUP", 2: "ITEM_BIND_ON_EQUIP", 3: "ITEM_BIND_ON_USE", 4: "ITEM_BIND_QUEST"}

// GetBondingName returns the bonding type name (client-localized when loaded).
func GetBondingName(bonding int) string {
	if v := clientString(bondingKey[bonding]); v != "" {
		return v
	}
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

// GetQualityName returns the quality name (client-localized when loaded).
func GetQualityName(quality int) string {
	if v := clientString(fmt.Sprintf("ITEM_QUALITY%d_DESC", quality)); v != "" {
		return v
	}
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

// GetCreatureTypeName returns the creature type name (client-localized when
// loaded from CreatureType.dbc).
func GetCreatureTypeName(t int) string {
	if creatureTypeNames != nil {
		if n, ok := creatureTypeNames[t]; ok && n != "" {
			return n
		}
	}
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

// GetCreatureRankName returns the creature rank name. "Elite"/"Boss" come from
// the client (ELITE/BOSS GlobalStrings) when loaded; the composite ranks
// ("Rare Elite") and "Normal"/"Rare" have no clean client string, so they keep
// a minimal built-in fallback.
func GetCreatureRankName(r int) string {
	elite := clientString("ELITE")
	if elite == "" {
		elite = "Elite"
	}
	boss := clientString("BOSS")
	if boss == "" {
		boss = "Boss"
	}
	switch r {
	case 1:
		return elite
	case 2:
		return "Rare " + elite
	case 3:
		return boss
	case 4:
		return "Rare"
	default:
		return "Normal"
	}
}

// triggerKey maps a spell-trigger id to its GlobalStrings prefix key (the three
// the client defines). Other triggers keep a built-in fallback below.
var triggerKey = map[int]string{0: "ITEM_SPELL_TRIGGER_ONUSE", 1: "ITEM_SPELL_TRIGGER_ONEQUIP", 2: "ITEM_SPELL_TRIGGER_ONPROC"}

// GetTriggerPrefix returns the spell trigger prefix (client-localized when
// loaded for Use/Equip/Chance-on-hit).
func GetTriggerPrefix(trigger int) string {
	if v := clientString(triggerKey[trigger]); v != "" {
		return v + " "
	}
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

// Faction group masks from FactionTemplate.dbc (FACTION_MASK_*).
const (
	FactionMaskPlayer   = 1
	FactionMaskAlliance = 2
	FactionMaskHorde    = 4
	FactionMaskMonster  = 8
)

// GetFactionReaction derives an NPC's reaction toward a player faction group
// (target = FactionMaskAlliance or FactionMaskHorde) from its FactionTemplate
// group masks. Returns "hostile", "friendly", or "neutral". Enemy takes
// precedence over friend; an NPC in the target's own group counts as friendly.
func GetFactionReaction(ourMask, friendMask, enemyMask, target int) string {
	switch {
	case enemyMask&target != 0:
		return "hostile"
	case friendMask&target != 0:
		return "friendly"
	case ourMask&target != 0:
		return "friendly"
	default:
		return "neutral"
	}
}

// GetSchoolName returns the magic school name (client-localized from
// spell_schools when loaded; built-in English otherwise).
func GetSchoolName(school int) string {
	if schoolNames != nil {
		if n, ok := schoolNames[school]; ok && n != "" {
			return n
		}
	}
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
