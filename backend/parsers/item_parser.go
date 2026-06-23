package parsers

import (
	"fmt"
	"html"
	"regexp"
	"strconv"
	"strings"

	"inklab/backend/database/models"
)

// ParseItemTitle extracts the item name from HTML content to check existence
func ParseItemTitle(content string) (bool, string) {
	// Check if item exists
	if strings.Contains(content, "Item not found") ||
		strings.Contains(content, "This item doesn't exist") ||
		strings.Contains(content, "404") {
		return false, ""
	}

	// Extract item name from title: "ItemName - Items - <Site> Database"
	titleRegex := regexp.MustCompile(`<title>([^<]+) - Items`)
	matches := titleRegex.FindStringSubmatch(content)
	if len(matches) > 1 {
		return true, strings.TrimSpace(matches[1])
	}

	// Fallback: check if page has item content
	if strings.Contains(content, "Display ID:") {
		return true, ""
	}

	return false, ""
}

// ParseItem extracts item details from HTML content
func ParseItem(content string, itemID int) (*models.ItemTemplateFull, *models.ItemSetEntry, error) {
	// Check if item exists
	if strings.Contains(content, "Item not found") || strings.Contains(content, "This item doesn't exist") {
		return nil, nil, fmt.Errorf("item not found: %d", itemID)
	}

	item := &models.ItemTemplateFull{Entry: itemID}
	var itemSet *models.ItemSetEntry

	// Extract item name from title
	titleRegex := regexp.MustCompile(`<title>([^<]+) - Items`)
	if matches := titleRegex.FindStringSubmatch(content); len(matches) > 1 {
		// Decode HTML entities like &#039; -> '
		item.Name = html.UnescapeString(strings.TrimSpace(matches[1]))
	}

	// Extract Display ID
	displayRegex := regexp.MustCompile(`Display ID:\s*</td>\s*<td[^>]*>(\d+)`)
	if matches := displayRegex.FindStringSubmatch(content); len(matches) > 1 {
		fmt.Sscanf(matches[1], "%d", &item.DisplayId)
	}
	// Try alternate format
	if item.DisplayId == 0 {
		displayRegex2 := regexp.MustCompile(`Display ID:\s*(\d+)`)
		if matches := displayRegex2.FindStringSubmatch(content); len(matches) > 1 {
			fmt.Sscanf(matches[1], "%d", &item.DisplayId)
		}
	}

	// Extract Item Level from "Level: X"
	levelRegex := regexp.MustCompile(`Level:\s*(\d+)`)
	if matches := levelRegex.FindStringSubmatch(content); len(matches) > 1 {
		fmt.Sscanf(matches[1], "%d", &item.ItemLevel)
	}

	// Try to determine quality from item name color class
	// Match the item title specifically: <b class="q3">ItemName</b> or <h1 class="q3">
	// The item name we already extracted should appear in a tag with the quality class
	item.Quality = 0 // Default to Poor
	if item.Name != "" {
		// Look for the item name within a quality-classed element
		// Pattern: class="qX">ItemName or class="qX" ...>ItemName
		qualityPatterns := []struct {
			pattern string
			quality int
		}{
			{`<b class="q6"[^>]*>` + regexp.QuoteMeta(item.Name), 6},
			{`<b class="q5"[^>]*>` + regexp.QuoteMeta(item.Name), 5},
			{`<b class="q4"[^>]*>` + regexp.QuoteMeta(item.Name), 4},
			{`<b class="q3"[^>]*>` + regexp.QuoteMeta(item.Name), 3},
			{`<b class="q2"[^>]*>` + regexp.QuoteMeta(item.Name), 2},
			{`<b class="q1"[^>]*>` + regexp.QuoteMeta(item.Name), 1},
			{`<b class="q0"[^>]*>` + regexp.QuoteMeta(item.Name), 0},
			// Also check h1 tag format
			{`<h1 class="q6"`, 6},
			{`<h1 class="q5"`, 5},
			{`<h1 class="q4"`, 4},
			{`<h1 class="q3"`, 3},
			{`<h1 class="q2"`, 2},
			{`<h1 class="q1"`, 1},
		}
		for _, qp := range qualityPatterns {
			if matched, _ := regexp.MatchString(qp.pattern, content); matched {
				item.Quality = qp.quality
				break
			}
		}
	}

	// Extract Unique / MaxCount
	if strings.Contains(content, "Unique") {
		// Check for "Unique (5)"
		uniqueRegex := regexp.MustCompile(`Unique \((\d+)\)`)
		if matches := uniqueRegex.FindStringSubmatch(content); len(matches) > 1 {
			fmt.Sscanf(matches[1], "%d", &item.MaxCount)
		} else {
			item.MaxCount = 1
		}
	}

	// Parse tooltip content for equipment info
	// Pattern: <td>Feet</td><th>Leather</th>
	slotTypeRegex := regexp.MustCompile(`<td>([^<]+)</td><th>([^<]+)</th>`)
	if matches := slotTypeRegex.FindStringSubmatch(content); len(matches) > 2 {
		slotName := strings.TrimSpace(matches[1])
		typeName := strings.TrimSpace(matches[2])
		item.InventoryType = parseInventoryType(slotName)
		item.Class, item.Subclass = parseArmorType(typeName)
	}

	// Fallback for slots that might not have a subtype (like Trinket, Neck, Finger, Back)
	if item.InventoryType == 0 {
		lowerContent := strings.ToLower(content)
		if strings.Contains(lowerContent, "<td>trinket") {
			item.InventoryType = 12 // Trinket
			item.Class = 4
			item.Subclass = 0 // Misc
		} else if strings.Contains(lowerContent, "<td>neck") {
			item.InventoryType = 2 // Neck
			item.Class = 4
			item.Subclass = 0
		} else if strings.Contains(lowerContent, "<td>finger") {
			item.InventoryType = 11 // Finger
			item.Class = 4
			item.Subclass = 0
		} else if strings.Contains(lowerContent, "<td>back") {
			item.InventoryType = 16 // Back
			item.Class = 4
			item.Subclass = 1 // Cloth
		} else if strings.Contains(lowerContent, "<td>shield") {
			item.InventoryType = 14 // Shield
			item.Class = 4
			item.Subclass = 6 // Shield
		} else if strings.Contains(lowerContent, "held in off-hand") {
			item.InventoryType = 23 // Held In Off-hand
			item.Class = 4
			item.Subclass = 0
		} else if strings.Contains(lowerContent, "<td>relic") || strings.Contains(lowerContent, "<td>libram") || strings.Contains(lowerContent, "<td>idol") || strings.Contains(lowerContent, "<td>totem") {
			item.InventoryType = 28 // Relic
			item.Class = 4
			item.Subclass = 0
		}
	}

	// Detect Bags/Containers: "12 Slot Soul Bag", "20 Slot Bag"
	// Pattern matches "X Slot [Type] Bag"
	bagRegex := regexp.MustCompile(`(\d+)\s+Slot\s+(.*)Bag`)
	if matches := bagRegex.FindStringSubmatch(content); len(matches) > 1 {
		item.InventoryType = 18 // Bag
		item.Class = 1          // Container

		// Extract container slots (the number before "Slot")
		fmt.Sscanf(matches[1], "%d", &item.ContainerSlots)

		// Determine subclass from bag type
		bagType := strings.TrimSpace(matches[2])
		switch {
		case strings.Contains(bagType, "Soul"):
			item.Subclass = 1 // Soul Bag
		case strings.Contains(bagType, "Herb"):
			item.Subclass = 2 // Herb Bag
		case strings.Contains(bagType, "Enchant"):
			item.Subclass = 3 // Enchanting Bag
		case strings.Contains(bagType, "Engineering"):
			item.Subclass = 4 // Engineering Bag
		case strings.Contains(bagType, "Gem"):
			item.Subclass = 5 // Gem Bag
		case strings.Contains(bagType, "Mining"):
			item.Subclass = 6 // Mining Bag
		case strings.Contains(bagType, "Leatherworking"):
			item.Subclass = 7 // Leatherworking Bag
		default:
			item.Subclass = 0 // Regular Bag
		}
	}

	// Extract Armor value: "69 Armor"
	armorRegex := regexp.MustCompile(`(\d+)\s*Armor`)
	if matches := armorRegex.FindStringSubmatch(content); len(matches) > 1 {
		fmt.Sscanf(matches[1], "%d", &item.Armor)
		// If armor > 0 and class not set, it's armor
		if item.Class == 0 && item.Armor > 0 {
			item.Class = 4 // Armor
		}
	}

	// Extract Required Level: "Requires Level 24"
	reqLevelRegex := regexp.MustCompile(`Requires Level\s*(\d+)`)
	if matches := reqLevelRegex.FindStringSubmatch(content); len(matches) > 1 {
		fmt.Sscanf(matches[1], "%d", &item.RequiredLevel)
	}

	// Extract Durability: "Durability 50 / 50"
	durabilityRegex := regexp.MustCompile(`Durability\s*(\d+)\s*/\s*(\d+)`)
	if matches := durabilityRegex.FindStringSubmatch(content); len(matches) > 2 {
		fmt.Sscanf(matches[2], "%d", &item.MaxDurability)
	}

	// Extract Bonding
	if strings.Contains(content, "Binds when picked up") {
		item.Bonding = 1
	} else if strings.Contains(content, "Binds when equipped") {
		item.Bonding = 2
	} else if strings.Contains(content, "Binds when used") {
		item.Bonding = 3
	}

	// Extract stats: "+7 Stamina", "+5 Agility"
	statIdx := 1
	statPatterns := map[string]int{
		"Stamina":   7,
		"Intellect": 5,
		"Spirit":    6,
		"Agility":   3,
		"Strength":  4,
	}
	for statName, statType := range statPatterns {
		statRegex := regexp.MustCompile(`\+(\d+)\s*` + statName)
		if matches := statRegex.FindStringSubmatch(content); len(matches) > 1 {
			var value int
			fmt.Sscanf(matches[1], "%d", &value)
			switch statIdx {
			case 1:
				item.StatType1 = statType
				item.StatValue1 = value
			case 2:
				item.StatType2 = statType
				item.StatValue2 = value
			case 3:
				item.StatType3 = statType
				item.StatValue3 = value
			case 4:
				item.StatType4 = statType
				item.StatValue4 = value
			case 5:
				item.StatType5 = statType
				item.StatValue5 = value
			case 6:
				item.StatType6 = statType
				item.StatValue6 = value
			case 7:
				item.StatType7 = statType
				item.StatValue7 = value
			case 8:
				item.StatType8 = statType
				item.StatValue8 = value
			case 9:
				item.StatType9 = statType
				item.StatValue9 = value
			case 10:
				item.StatType10 = statType
				item.StatValue10 = value
			}
			statIdx++
			if statIdx > 10 {
				break
			}
		}
	}

	// Extract Speed: "Speed 2.30"
	speedRegex := regexp.MustCompile(`Speed\s*(\d+\.?\d*)`)
	if matches := speedRegex.FindStringSubmatch(content); len(matches) > 1 {
		var speed float64
		fmt.Sscanf(matches[1], "%f", &speed)
		item.Delay = int(speed * 1000)
	}

	// Extract Damage: "15 - 28 Damage" or "15 - 28 Fire Damage"
	dmgRegex := regexp.MustCompile(`(\d+)\s*-\s*(\d+)\s*(\w*)\s*Damage`)
	if matches := dmgRegex.FindStringSubmatch(content); len(matches) > 2 {
		fmt.Sscanf(matches[1], "%f", &item.DmgMin1)
		fmt.Sscanf(matches[2], "%f", &item.DmgMax1)

		if len(matches) > 3 {
			item.DmgType1 = parseDamageType(matches[3])
		}
	}

	// Extract spell effects from links like: href="?spell=18384"
	// Use a more robust regex to catch "?spell=" or "&spell="
	// IMPORTANT: Only extract spells that are part of item effects (Equip/Use/Chance on hit)
	// Exclude set bonus spells which appear after itemset links
	// NEW: Also extract spell description from link text since spell pages often don't have it
	spellWithDescRegex := regexp.MustCompile(`<a href="[?&]spell=(\d+)"[^>]*>([^<]+)</a>`)
	spellWithDescMatches := spellWithDescRegex.FindAllStringSubmatchIndex(content, -1)

	// First, find the position of itemset link to know where set bonuses start
	itemsetIdx := -1
	itemsetRegex := regexp.MustCompile(`\?itemset=\d+`)
	if itemsetMatch := itemsetRegex.FindStringIndex(content); itemsetMatch != nil {
		itemsetIdx = itemsetMatch[0]
	}

	// Initialize spell descriptions map
	item.SpellDescriptions = make(map[int]string)

	spellCount := 0
	for _, match := range spellWithDescMatches {
		// match indices: [0:1] full match, [2:3] spell ID, [4:5] description text
		if len(match) >= 6 && spellCount < 5 {
			spellPos := match[0]

			// Skip spells that appear after the itemset link (they are set bonuses)
			if itemsetIdx >= 0 && spellPos > itemsetIdx {
				continue
			}

			// Extract spell ID
			spellIDStr := content[match[2]:match[3]]
			var spellID int
			fmt.Sscanf(spellIDStr, "%d", &spellID)

			// Extract spell description from link text
			spellDesc := cleanSpellDescription(strings.TrimSpace(content[match[4]:match[5]]))

			if spellID > 0 {
				// Detect trigger type by looking backwards from the match
				// We look at the preceding ~200 characters for keywords
				startLookback := match[0] - 200
				if startLookback < 0 {
					startLookback = 0
				}
				contextStr := content[startLookback:match[0]]

				// Skip if context contains set bonus indicators
				if strings.Contains(contextStr, ") Set:") || strings.Contains(contextStr, "Set Bonus:") {
					continue
				}

				// Check for valid item effect triggers
				hasValidTrigger := strings.Contains(contextStr, "Equip:") ||
					strings.Contains(contextStr, "Use:") ||
					strings.Contains(contextStr, ">Use<") ||
					strings.Contains(contextStr, "Chance on hit:")

				// If no valid trigger found, skip this spell
				if !hasValidTrigger {
					continue
				}

				// Default to Equip (1)
				trigger := 1

				// Check for explicit "Use:" or "Chance on hit:"
				if strings.Contains(contextStr, "Use:") || strings.Contains(contextStr, ">Use<") {
					trigger = 0 // Use
				} else if strings.Contains(contextStr, "Chance on hit:") {
					trigger = 2 // Chance on hit
				}

				// Store spell description for later sync
				if spellDesc != "" {
					item.SpellDescriptions[spellID] = spellDesc
				}

				switch spellCount {
				case 0:
					item.Spellid1 = spellID
					item.Spelltrigger1 = trigger
				case 1:
					item.Spellid2 = spellID
					item.Spelltrigger2 = trigger
				case 2:
					item.Spellid3 = spellID
					item.Spelltrigger3 = trigger
				case 3:
					item.Spellid4 = spellID
					item.Spelltrigger4 = trigger
				case 4:
					item.Spellid5 = spellID
					item.Spelltrigger5 = trigger
				}
				spellCount++
			}
		}
	}

	// Extract Description (Gold text)
	descRegex := regexp.MustCompile(`<span class="q">"([^"]+)"</span>`)
	if matches := descRegex.FindStringSubmatch(content); len(matches) > 1 {
		item.Description = matches[1]
	}

	// Extract Sell Price
	if strings.Contains(content, "Sells for") {
		var gold, silver, copper int

		gRegex := regexp.MustCompile(`(\d+)<span class="moneygold">`)
		if m := gRegex.FindStringSubmatch(content); len(m) > 1 {
			fmt.Sscanf(m[1], "%d", &gold)
		}

		sRegex := regexp.MustCompile(`(\d+)<span class="moneysilver">`)
		if m := sRegex.FindStringSubmatch(content); len(m) > 1 {
			fmt.Sscanf(m[1], "%d", &silver)
		}

		cRegex := regexp.MustCompile(`(\d+)<span class="moneycopper">`)
		if m := cRegex.FindStringSubmatch(content); len(m) > 1 {
			fmt.Sscanf(m[1], "%d", &copper)
		}

		item.SellPrice = copper + (silver * 100) + (gold * 10000)
	}

	// Extract Resistances
	resMap := map[string]*int{
		"Holy Resistance":   &item.HolyRes,
		"Fire Resistance":   &item.FireRes,
		"Nature Resistance": &item.NatureRes,
		"Frost Resistance":  &item.FrostRes,
		"Shadow Resistance": &item.ShadowRes,
		"Arcane Resistance": &item.ArcaneRes,
	}
	for name, ptr := range resMap {
		regex := regexp.MustCompile(`\+(\d+)\s*` + name)
		if matches := regex.FindStringSubmatch(content); len(matches) > 1 {
			fmt.Sscanf(matches[1], "%d", ptr)
		}
	}

	// Extract Classes/Races
	if strings.Contains(content, "Classes:") {
		classMask := 0
		if strings.Contains(content, "Warrior") {
			classMask |= 1
		}
		if strings.Contains(content, "Paladin") {
			classMask |= 2
		}
		if strings.Contains(content, "Hunter") {
			classMask |= 4
		}
		if strings.Contains(content, "Rogue") {
			classMask |= 8
		}
		if strings.Contains(content, "Priest") {
			classMask |= 16
		}
		if strings.Contains(content, "Shaman") {
			classMask |= 64
		}
		if strings.Contains(content, "Mage") {
			classMask |= 128
		}
		if strings.Contains(content, "Warlock") {
			classMask |= 256
		}
		if strings.Contains(content, "Druid") {
			classMask |= 1024
		}
		item.AllowableClass = classMask
	}

	// Extract Set ID and Set Info
	setRegex := regexp.MustCompile(`\?itemset=(\d+)`)
	if matches := setRegex.FindStringSubmatch(content); len(matches) > 1 {
		fmt.Sscanf(matches[1], "%d", &item.SetId)

		// Parse full set info
		itemSet = parseItemSet(content, item.SetId)
	}

	// Extract Dropped By NPCs
	item.DroppedByNpcs = extractListViewIDs(content, "dropped-by")

	// Item class/subclass come authoritatively from the breadcrumb path the
	// site emits for every item: g_initPath([0, 0, <class>, <subclass>]).
	// The equipment text only covers wearable/wieldable items, so without this
	// trade goods, consumables, recipes, etc. fall through as class 0.
	pathRegex := regexp.MustCompile(`g_initPath\(\[\s*\d+\s*,\s*\d+\s*,\s*(\d+)\s*,\s*(\d+)`)
	if matches := pathRegex.FindStringSubmatch(content); len(matches) > 2 {
		item.Class, _ = strconv.Atoi(matches[1])
		item.Subclass, _ = strconv.Atoi(matches[2])
	}

	return item, itemSet, nil
}

// parseItemSet parses the item set info from the content
func parseItemSet(content string, setID int) *models.ItemSetEntry {
	set := &models.ItemSetEntry{
		ID: setID,
	}

	// 1. Extract Set Name
	// <a href="?itemset=123" class="q">Set Name</a>
	nameRegex := regexp.MustCompile(fmt.Sprintf(`\?itemset=%d[^>]*>([^<]+)</a>`, setID))
	if matches := nameRegex.FindStringSubmatch(content); len(matches) > 1 {
		set.Name = html.UnescapeString(matches[1])
	}

	// Find the itemset block start to constrain search
	setLinkIdx := strings.Index(content, fmt.Sprintf("?itemset=%d", setID))
	if setLinkIdx == -1 {
		return set
	}
	setBlock := content[setLinkIdx:]

	// Truncate block at some reasonable point or next major section to avoid false positives?
	// The set items usually come immediately after.
	// But the bonuses come after the items.
	// Let's rely on standard WoW DB formatting.

	// 2. Extract Set Items
	// They usually appear as links in the same block
	// Look for links that are NOT the set link itself, but item links
	// Logic: Find ?item=ID links that appear within the "List of items in set" context
	// Usually: <span><a href="?item=123">Item Name</a></span>

	// We'll limit the search window for items to the "Regalia of Faith (0/9)" section
	// which ends before the bonuses "(2) Set: ..."

	// Let's just scan for all ?item=ID links that are NOT the current item ID (optional, but they are all in the set)
	// We need to be careful not to pick up "Related Items" or "Dropped By" items.
	// The set items are usually listed in a <div> or <ul> immediately following the set header.

	// Strategy: Find strings like `<span><a href="?item=(\d+)">` that appear after the set header
	// and before the first "Set:" bonus or end of reasonable block

	// Find start of bonuses
	bonusStartIdx := -1
	bonusRegex := regexp.MustCompile(`\(\d+\) Set:`)
	if loc := bonusRegex.FindStringIndex(setBlock); loc != nil {
		bonusStartIdx = loc[0]
	}

	itemsBlock := setBlock
	if bonusStartIdx != -1 {
		itemsBlock = setBlock[:bonusStartIdx]
	} else {
		// Fallback: search next 2000 chars
		if len(itemsBlock) > 2000 {
			itemsBlock = itemsBlock[:2000]
		}
	}

	itemRegex := regexp.MustCompile(`\?item=(\d+)`)
	itemMatches := itemRegex.FindAllStringSubmatch(itemsBlock, -1)

	itemIDs := []int{}
	seen := make(map[int]bool)
	for _, m := range itemMatches {
		var id int
		fmt.Sscanf(m[1], "%d", &id)
		if id > 0 && !seen[id] {
			itemIDs = append(itemIDs, id)
			seen[id] = true
		}
	}

	// Assign to fields
	for i, id := range itemIDs {
		switch i {
		case 0:
			set.Item1 = id
		case 1:
			set.Item2 = id
		case 2:
			set.Item3 = id
		case 3:
			set.Item4 = id
		case 4:
			set.Item5 = id
		case 5:
			set.Item6 = id
		case 6:
			set.Item7 = id
		case 7:
			set.Item8 = id
		case 8:
			set.Item9 = id
		case 9:
			set.Item10 = id
		}
	}

	// 3. Extract Set Bonuses
	// Pattern: (2) Set: <a href="?spell=123">...</a> OR (2) Set: Plain Text
	// We search in setBlock (starting from set name)

	// Regex needs to capture: Threshold, SpellID (optional)
	// We'll iterate through matches of `(\d+) Set:`

	// Create a regex that finds the threshold and looks for a spell link shortly after
	bonusMatches := bonusRegex.FindAllStringIndex(setBlock, -1)

	for i, loc := range bonusMatches {
		if i >= 8 {
			break
		} // Max 8 bonuses supported by schema

		start := loc[0]
		end := loc[1]

		// Extract threshold
		thresholdStr := setBlock[start+1 : strings.Index(setBlock[start:], ")")+start]
		var threshold int
		fmt.Sscanf(thresholdStr, "%d", &threshold)

		// Look for spell link in the immediate following text (e.g., next 200 chars or until next <br>)
		// <a href="?spell=123">

		searchEnd := start + 300
		if i+1 < len(bonusMatches) {
			if bonusMatches[i+1][0] < searchEnd {
				searchEnd = bonusMatches[i+1][0]
			}
		}
		if searchEnd > len(setBlock) {
			searchEnd = len(setBlock)
		}

		segment := setBlock[end:searchEnd]

		spellLinkRegex := regexp.MustCompile(`\?spell=(\d+)`)
		var spellID int
		if sMatch := spellLinkRegex.FindStringSubmatch(segment); len(sMatch) > 1 {
			fmt.Sscanf(sMatch[1], "%d", &spellID)
		}

		// Assign to fields
		switch i {
		case 0:
			set.Bonus1 = threshold
			set.Spell1 = spellID
		case 1:
			set.Bonus2 = threshold
			set.Spell2 = spellID
		case 2:
			set.Bonus3 = threshold
			set.Spell3 = spellID
		case 3:
			set.Bonus4 = threshold
			set.Spell4 = spellID
		case 4:
			set.Bonus5 = threshold
			set.Spell5 = spellID
		case 5:
			set.Bonus6 = threshold
			set.Spell6 = spellID
		case 6:
			set.Bonus7 = threshold
			set.Spell7 = spellID
		case 7:
			set.Bonus8 = threshold
			set.Spell8 = spellID
		}
	}

	return set
}

// extractListViewIDs parses the data array from a Listview definition
func extractListViewIDs(content string, viewID string) []int {
	target := fmt.Sprintf("id: '%s'", viewID)
	idx := strings.Index(content, target)
	if idx == -1 {
		return nil
	}

	// Look for "data: [" after the ID
	rest := content[idx:]
	dataIdx := strings.Index(rest, "data: [")
	if dataIdx == -1 {
		return nil
	}

	// Find the end of the Listview block to constrain search
	// Usually ends with "});"
	endIdx := strings.Index(rest, "});")
	if endIdx == -1 || endIdx < dataIdx {
		// If can't find end or end is before data (impossible if logic holds), just take a large chunk
		endIdx = dataIdx + 5000
		if endIdx > len(rest) {
			endIdx = len(rest)
		}
	}

	dataBlock := rest[dataIdx:endIdx]

	// Extract IDs using regex
	// Pattern: id:123
	var ids []int
	idRegex := regexp.MustCompile(`id:\s*(\d+)`)
	matches := idRegex.FindAllStringSubmatch(dataBlock, -1)

	seen := make(map[int]bool)
	for _, m := range matches {
		var id int
		fmt.Sscanf(m[1], "%d", &id)
		if id > 0 && !seen[id] {
			ids = append(ids, id)
			seen[id] = true
		}
	}
	return ids
}

// parseInventoryType converts slot name to inventory type ID
func parseInventoryType(slot string) int {
	slots := map[string]int{
		"Head": 1, "Neck": 2, "Shoulder": 3, "Shirt": 4, "Chest": 5,
		"Waist": 6, "Legs": 7, "Feet": 8, "Wrists": 9, "Hands": 10,
		"Finger": 11, "Trinket": 12, "One-Hand": 13, "Shield": 14,
		"Ranged": 15, "Back": 16, "Two-Hand": 17, "Bag": 18,
		"Tabard": 19, "Robe": 20, "Main Hand": 21, "Off Hand": 22,
		"Held In Off-hand": 23, "Ammo": 24, "Thrown": 25,
		"One-hand": 13, "Two-hand": 17, // Add lowercase variants explicitly
	}
	if id, ok := slots[slot]; ok {
		return id
	}
	return 0
}

// parseArmorType converts armor type name to class/subclass
func parseArmorType(typeName string) (int, int) {
	armorTypes := map[string]int{
		"Cloth": 1, "Leather": 2, "Mail": 3, "Plate": 4,
		"Shield": 6, "Libram": 7, "Idol": 8, "Totem": 9,
	}
	if subClass, ok := armorTypes[typeName]; ok {
		return 4, subClass // Class 4 = Armor
	}

	// Weapon types
	weaponTypes := map[string]int{
		"Axe": 0, "Two-Handed Axe": 1, "Bow": 2, "Gun": 3,
		"Mace": 4, "Two-Handed Mace": 5, "Polearm": 6,
		"Sword": 7, "Two-Handed Sword": 8, "Staff": 10,
		"Fist Weapon": 13, "Dagger": 15, "Thrown": 16,
		"Crossbow": 18, "Wand": 19, "Fishing Pole": 20,
	}
	if subClass, ok := weaponTypes[typeName]; ok {
		return 2, subClass // Class 2 = Weapon
	}

	return 0, 0
}

// parseDamageType converts damage type string to int ID
func parseDamageType(typeStr string) int {
	typeStr = strings.TrimSpace(typeStr)
	switch typeStr {
	case "Holy":
		return 1
	case "Fire":
		return 2
	case "Nature":
		return 3
	case "Frost":
		return 4
	case "Shadow":
		return 5
	case "Arcane":
		return 6
	default:
		return 0 // Physical
	}
}

// cleanSpellDescription processes Aowow/Turtle formatting variables like $leffect:effects;
func cleanSpellDescription(desc string) string {
	// Handle $lvariant:plural;
	// Example: Removes 1 poison $leffect:effects;
	pluralRegex := regexp.MustCompile(`\$l([^:]+):([^;]+);`)
	matches := pluralRegex.FindAllStringSubmatchIndex(desc, -1)

	if len(matches) == 0 {
		return desc
	}

	var sb strings.Builder
	lastIndex := 0

	for _, match := range matches {
		// match[0]-match[1] is the full match range
		// match[2]-match[3] is singular
		// match[4]-match[5] is plural

		start := match[0]
		end := match[1]

		// Append content before match
		sb.WriteString(desc[lastIndex:start])

		// Look back for number in desc[0:start]
		// We want the *last* number before this match.
		// However, we must be careful not to pick a number from a previous sentence or unrelated part if possible.
		// But in "Removes 1 poison $leffect:effects;", 1 is the quantity.
		preceding := desc[0:start]

		// Find all numbers
		numRegex := regexp.MustCompile(`(\d+)`)
		numMatches := numRegex.FindAllStringSubmatch(preceding, -1)

		count := 0
		if len(numMatches) > 0 {
			// Take the last one
			lastNumStr := numMatches[len(numMatches)-1][1]
			fmt.Sscanf(lastNumStr, "%d", &count)
		}

		singular := desc[match[2]:match[3]]
		plural := desc[match[4]:match[5]]

		if count == 1 {
			sb.WriteString(singular)
		} else {
			sb.WriteString(plural)
		}

		lastIndex = end
	}

	sb.WriteString(desc[lastIndex:])
	return sb.String()
}
