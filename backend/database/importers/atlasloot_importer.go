package importers

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// AtlasLootImporter handles AtlasLoot data imports
type AtlasLootImporter struct {
	db *sql.DB
}

// NewAtlasLootImporter creates a new AtlasLoot importer
func NewAtlasLootImporter(db *sql.DB) *AtlasLootImporter {
	return &AtlasLootImporter{db: db}
}

// LoadDataFromLua loads all data required for import using the Lua Parser
func (a *AtlasLootImporter) LoadDataFromLua(addonDir string) (map[string][]LuaLootItem, map[string][]SpecialMenuItem, map[string]string, error) {
	parser := NewAtlasLootLuaParser(addonDir)

	// 1. Display Names
	displayNames, err := parser.ParseTableRegister()
	if err != nil {
		fmt.Println("Warning: Could not parse TableRegister", err)
	}

	// 2. Special Menus
	specialMenus, err := parser.ParseSpecialMenus()
	if err != nil {
		fmt.Println("Warning: Could not parse SpecialMenus", err)
	}

	// 3. Loot Tables
	files := []string{"Instances.lua", "Sets.lua", "Factions.lua", "PvP.lua", "WorldBosses.lua", "WorldEvents.lua", "Crafting.lua"}
	allTables := make(map[string][]LuaLootItem)

	for _, f := range files {
		tables, err := parser.ParseLootTables(f)
		if err != nil {
			fmt.Printf("Info: Could not parse %s (might be missing), skipping.\n", f)
			continue
		}
		for k, v := range tables {
			allTables[k] = v
		}
	}

	return allTables, specialMenus, displayNames, nil
}

func (a *AtlasLootImporter) LoadItemNameMap() (map[string]int, error) {
	// Check if item_template exists first to avoid error
	var count int
	err := a.db.QueryRow("SELECT count(*) FROM sqlite_master WHERE type='table' AND name='item_template'").Scan(&count)
	if err != nil || count == 0 {
		return make(map[string]int), nil
	}

	rows, err := a.db.Query("SELECT entry, name FROM item_template")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	m := make(map[string]int)
	for rows.Next() {
		var id int
		var name string
		if err := rows.Scan(&id, &name); err == nil {
			m[strings.ToLower(name)] = id
		}
	}
	fmt.Printf("Loaded %d items for name resolution\n", len(m))
	return m, nil
}

// ImportFromLua parses the Core/AtlasLoot.lua file for structure and uses loaded data to populate DB
func (a *AtlasLootImporter) ImportFromLua(luaPath string,
	lootTables map[string][]LuaLootItem,
	specialMenus map[string][]SpecialMenuItem,
	displayNames map[string]string) error {

	// Load item name mapping for checking spell-based items
	itemNameMap, err := a.LoadItemNameMap()
	if err != nil {
		fmt.Printf("Warning: Failed to load item name map: %v\n", err)
	}

	contentBytes, err := os.ReadFile(luaPath)
	if err != nil {
		return fmt.Errorf("failed to read Lua file: %w", err)
	}
	content := string(contentBytes)

	tx, err := a.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Clear existing
	tx.Exec("DELETE FROM atlasloot_items")
	tx.Exec("DELETE FROM atlasloot_tables")
	tx.Exec("DELETE FROM atlasloot_modules")
	tx.Exec("DELETE FROM atlasloot_categories")

	stmtCat, _ := tx.Prepare("INSERT INTO atlasloot_categories (name, display_name, sort_order) VALUES (?, ?, ?)")
	stmtMod, _ := tx.Prepare("INSERT INTO atlasloot_modules (category_id, name, display_name, sort_order) VALUES (?, ?, ?, ?)")
	stmtTbl, _ := tx.Prepare("INSERT INTO atlasloot_tables (module_id, table_key, display_name, sort_order) VALUES (?, ?, ?, ?)")
	stmtItem, _ := tx.Prepare("INSERT INTO atlasloot_items (table_id, item_id, drop_chance, sort_order, spell_id, override_name, override_icon, quality) VALUES (?, ?, ?, ?, ?, ?, ?, ?)")

	// Regex for Categories: {[AL["Dungeons & Raids"]] = {
	reCategory := regexp.MustCompile(`\{\[AL\["(.*?)"\]\]\s*=\s*\{`)

	// Regex for Menu Items. The display part may be a concatenation with a
	// color/level-range (or [RAID]) prefix before AL["Name"], e.g.
	//   {{ WHITE.."[13-18]|r "..AL["Ragefire Chasm"], "RagefireChasm", "Submenu" },},
	//   {{ RED.."[RAID]|r "..AL["Molten Core"], "MoltenCore", "Submenu" },},
	// Group 1 = prefix (used to detect [RAID]); 2 = name, 3 = key, 4 = type.
	reMenuItem := regexp.MustCompile(`\{+\s*([^{}]*?)AL\["(.*?)"\]\s*,\s*"(.*?)"\s*,\s*"(.*?)"\s*\}+,?`)

	// Regex for SubTables start: ["RagefireChasm"] = {
	reSubTableStart := regexp.MustCompile(`\["(.*?)"\]\s*=\s*\{`)

	// Regex for SubTable Entry: { AL["Taragaman the Hungerer"], "RFCTaragaman" },
	reSubTableEntry := regexp.MustCompile(`\{\s*AL\["(.*?)"\],\s*"(.*?)"\s*\},`)

	// 1. Parse SubTables first to map ModuleKey -> []TableEntry (Static Submenus)
	type SubTableEntry struct {
		DisplayName string
		TableKey    string
	}
	subTables := make(map[string][]SubTableEntry)

	parts := strings.Split(content, "AtlasLoot_HewdropDown_SubTables = {")
	if len(parts) > 1 {
		subTableBlock := parts[1]
		subTableMatches := reSubTableStart.FindAllStringSubmatchIndex(subTableBlock, -1)
		for i, match := range subTableMatches {
			key := subTableBlock[match[2]:match[3]]
			start := match[1]
			end := len(subTableBlock)
			if i < len(subTableMatches)-1 {
				end = subTableMatches[i+1][0]
			}
			block := subTableBlock[start:end]
			entriesMatches := reSubTableEntry.FindAllStringSubmatch(block, -1)
			var entries []SubTableEntry
			for _, em := range entriesMatches {
				entries = append(entries, SubTableEntry{
					DisplayName: em[1],
					TableKey:    em[2],
				})
			}
			subTables[key] = entries
		}
	}

	// 2. Parse Main Menu (Categories)
	mainMenuPart := strings.Split(content, "AtlasLoot_HewdropDown = {")[1]
	mainMenuPart = strings.Split(mainMenuPart, "};")[0]

	currentIndex := 0
	for currentIndex < len(mainMenuPart) {
		match := reCategory.FindStringSubmatchIndex(mainMenuPart[currentIndex:])
		if match == nil {
			break
		}

		nameStartRel := match[2]
		nameEndRel := match[3]
		catName := mainMenuPart[currentIndex+nameStartRel : currentIndex+nameEndRel]
		blockStartAbs := currentIndex + match[1]

		braceCount := 1
		blockEndAbs := -1

		for k := blockStartAbs; k < len(mainMenuPart); k++ {
			if mainMenuPart[k] == '{' {
				braceCount++
			} else if mainMenuPart[k] == '}' {
				braceCount--
				if braceCount == 0 {
					blockEndAbs = k
					break
				}
			}
		}

		if blockEndAbs == -1 {
			break
		}

		block := mainMenuPart[blockStartAbs:blockEndAbs]
		fmt.Printf("  Processing Category: %s\n", catName)

		var currentCatID int64
		var dungeonsID, raidsID int64
		isSplit := (catName == "Dungeons & Raids")

		if isSplit {
			res, _ := stmtCat.Exec("Dungeons", "Dungeons", 10)
			dungeonsID, _ = res.LastInsertId()
			res, _ = stmtCat.Exec("Raids", "Raids", 11)
			raidsID, _ = res.LastInsertId()
		} else {
			res, err := stmtCat.Exec(catName, catName, 99)
			if err != nil {
				fmt.Printf("Error inserting category: %v\n", err)
			} else {
				currentCatID, _ = res.LastInsertId()
			}
		}

		insertedModules := make(map[string]bool)

		var parseModules func(string)
		parseModules = func(contentBlock string) {
			matches := reMenuItem.FindAllStringSubmatch(contentBlock, -1)
			for _, mod := range matches {
				modPrefix := mod[1]
				modName := mod[2]
				modKey := mod[3]
				modType := mod[4]

				if insertedModules[modKey] {
					continue
				}
				insertedModules[modKey] = true

				targetCatID := currentCatID
				if isSplit {
					if strings.Contains(modPrefix, "[RAID]") {
						targetCatID = raidsID
					} else {
						targetCatID = dungeonsID
					}
				}

				fmt.Printf("    Module: %s (Type: %s, Key: %s)\n", modName, modType, modKey)
				res, err := stmtMod.Exec(targetCatID, modKey, modName, 0)
				if err != nil {
					continue
				}
				modID, _ := res.LastInsertId()

				// Helper to insert items for a table
				insertItems := func(tblID int64, key string) {
					if items, ok := lootTables[key]; ok {
						seenKeys := make(map[string]bool)
						for k, item := range items {
							// Try to resolve Item ID if it's a Spell or Name based lookup
							resolvedID := item.ID
							if (item.SpellID > 0 || resolvedID == 0) && item.Name != "" {
								if foundID, ok := itemNameMap[strings.ToLower(item.Name)]; ok {
									resolvedID = foundID
								}
							}

							// Generate unique key for deduplication
							uniqueKey := fmt.Sprintf("%d-%d", resolvedID, item.SpellID)
							if seenKeys[uniqueKey] {
								continue
							}
							seenKeys[uniqueKey] = true

							_, err := stmtItem.Exec(tblID, resolvedID, item.DropRate, k, item.SpellID, item.Name, item.Icon, item.Quality)
							if err != nil {
								fmt.Printf("Error inserting item [%s] (ID=%d, Spell=%d) to table %d: %v\n", item.Name, resolvedID, item.SpellID, tblID, err)
							} else if item.SpellID > 0 {
								// Debug log for successful spell insertion
								// fmt.Printf("Inserted Spell [%s] (Spell=%d) to table %d\n", item.Name, item.SpellID, tblID)
							}
						}
					} else {
						fmt.Printf("Warning: Table key '%s' not found inside lootTables map during validation.\n", key)
					}
				}

				// Resolve Tables/Submenus
				if menuItems, isSpecial := specialMenus[modKey]; isSpecial {
					for i, entry := range menuItems {
						dName := entry.Name
						if resolved, ok := displayNames[entry.Key]; ok {
							_ = resolved
						}

						if dName == "" || dName == entry.Key {
							if val, ok := displayNames[entry.Key]; ok {
								dName = val
							}
						}

						res, err := stmtTbl.Exec(modID, entry.Key, dName, i)
						if err != nil {
							continue
						}
						tblID, _ := res.LastInsertId()
						insertItems(tblID, entry.Key)
					}
				} else if modType == "Submenu" {
					if entries, ok := subTables[modKey]; ok {
						for k, entry := range entries {
							res, err := stmtTbl.Exec(modID, entry.TableKey, entry.DisplayName, k)
							if err != nil {
								continue
							}
							tblID, _ := res.LastInsertId()
							insertItems(tblID, entry.TableKey)
						}
					} else {
						fmt.Printf("      Warning: Submenu key '%s' not found in subTables/SpecialMenus\n", modKey)
					}
				} else if modType == "Table" {
					tblKey := modKey
					displayName := modName
					res, err := stmtTbl.Exec(modID, tblKey, displayName, 0)
					if err != nil {
						continue
					}
					tblID, _ := res.LastInsertId()
					insertItems(tblID, tblKey)
				}
			}

			reNestedStart := regexp.MustCompile(`\{\[\s*AL\["(.*?)"\]\]\s*=\s*\{`)
			nestedMatches := reNestedStart.FindAllStringSubmatchIndex(contentBlock, -1)
			for _, match := range nestedMatches {
				startOfInner := match[1]
				braceCount := 1
				endOfInner := -1
				for k := startOfInner; k < len(contentBlock); k++ {
					if contentBlock[k] == '{' {
						braceCount++
					} else if contentBlock[k] == '}' {
						braceCount--
						if braceCount == 0 {
							endOfInner = k
							break
						}
					}
				}
				if endOfInner != -1 {
					parseModules(contentBlock[startOfInner:endOfInner])
				}
			}
		}

		if currentCatID != 0 || isSplit {
			parseModules(block)
		}
		currentIndex = blockEndAbs + 1
	}

	fmt.Println("Committing AtlasLoot import...")
	return tx.Commit()
}

// CheckAndImport checks if AtlasLoot data exists and imports it
func (a *AtlasLootImporter) CheckAndImport(dataDir string) error {
	var countRaids int
	var countWrongInRaids int
	var countT0Items int

	// 1. Check if Raids category exists
	a.db.QueryRow("SELECT COUNT(*) FROM atlasloot_categories WHERE name = 'Raids'").Scan(&countRaids)

	// 2. Check for potential misconfiguration
	a.db.QueryRow(`
		SELECT COUNT(*) 
		FROM atlasloot_modules m 
		JOIN atlasloot_categories c ON m.category_id = c.id 
		WHERE c.name = 'Raids' AND m.display_name NOT LIKE '[RAID]%'
	`).Scan(&countWrongInRaids)

	// Check if Special Menus (e.g. T0 Set) are missing children
	a.db.QueryRow(`
        SELECT COUNT(*) 
        FROM atlasloot_items i
        JOIN atlasloot_tables t ON i.table_id = t.id
        JOIN atlasloot_modules m ON t.module_id = m.id
        WHERE m.name = 'T0SET'
    `).Scan(&countT0Items)

	// Check Alchemy
	var countAlchemyItems int
	a.db.QueryRow(`
        SELECT COUNT(*) 
        FROM atlasloot_items i
        JOIN atlasloot_tables t ON i.table_id = t.id
        JOIN atlasloot_modules m ON t.module_id = m.id
        WHERE m.name = 'ALCHEMYMENU'
    `).Scan(&countAlchemyItems)

	// Re-import if data looks suspicious
	if countRaids == 0 || countWrongInRaids > 0 || countT0Items == 0 || countAlchemyItems == 0 {
		addonDir := filepath.Join(dataDir, "../addons/AtlasLoot")
		corePath := filepath.Join(addonDir, "Core/AtlasLoot.lua")

		if _, err := os.Stat(corePath); err == nil {
			fmt.Println("Importing AtlasLoot from Lua...")

			tables, specialMenus, displayNames, err := a.LoadDataFromLua(addonDir)
			if err != nil {
				return err
			}

			return a.ImportFromLua(corePath, tables, specialMenus, displayNames)
		} else {
			fmt.Printf("Lua file not found at %s. Skipping AtlasLoot import.\n", corePath)
		}
	}
	return nil
}
