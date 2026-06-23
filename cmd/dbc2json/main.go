// Command dbc2json regenerates the data/*.json files InkLab imports from the
// client DBC files (DBFilesClient). See docs/ARCHITECTURE_SPEC.md.
//
// Usage:
//
//	go run ./cmd/dbc2json verify-factions <DBFilesClient dir>
package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("usage: dbc2json <command> <DBFilesClient dir>")
		fmt.Println("commands: verify-factions")
		os.Exit(2)
	}
	cmd, dbcDir := os.Args[1], os.Args[2]

	switch cmd {
	case "verify-factions":
		if err := verifyFactions(dbcDir); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
	case "validate-factions":
		// args: validate-factions <dbcDir> <committed factions.json>
		if len(os.Args) < 4 {
			fmt.Fprintln(os.Stderr, "usage: dbc2json validate-factions <DBFilesClient dir> <factions.json>")
			os.Exit(2)
		}
		if err := validateFactions(dbcDir, os.Args[3]); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
	case "factions":
		// args: factions <dbcDir> <out factions.json>
		if len(os.Args) < 4 {
			fmt.Fprintln(os.Stderr, "usage: dbc2json factions <DBFilesClient dir> <out.json>")
			os.Exit(2)
		}
		if err := writeFactions(dbcDir, os.Args[3]); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
	case "gen-all":
		// args: gen-all <DBFilesClient dir> <data dir>
		if len(os.Args) < 4 {
			fmt.Fprintln(os.Stderr, "usage: dbc2json gen-all <DBFilesClient dir> <data dir>")
			os.Exit(2)
		}
		outDir := os.Args[3]
		if err := writeFactions(dbcDir, filepath.Join(outDir, "factions.json")); err != nil {
			fmt.Fprintln(os.Stderr, "error (factions):", err)
			os.Exit(1)
		}
		jobs := []struct{ name, file string }{
			{"itemsets", "item_sets.json"},
			{"skills", "skills.json"},
			{"sla", "skill_line_abilities.json"},
			{"zones", "zones.json"},
			{"icons", "item_icons.json"},
			{"spells", "spells_enhanced.json"},
		}
		for _, j := range jobs {
			if err := runGen(j.name, dbcDir, filepath.Join(outDir, j.file)); err != nil {
				fmt.Fprintf(os.Stderr, "error (%s): %v\n", j.name, err)
				os.Exit(1)
			}
		}
	case "gen":
		// args: gen <name> <dbcDir> <outPath>
		if len(os.Args) < 5 {
			fmt.Fprintln(os.Stderr, "usage: dbc2json gen <name> <DBFilesClient dir> <out.json>")
			fmt.Fprintln(os.Stderr, "names: itemsets skills sla zones icons spells")
			os.Exit(2)
		}
		// here dbcDir is actually os.Args[3]; os.Args[2] is the name
		name, dir, out := os.Args[2], os.Args[3], os.Args[4]
		if err := runGen(name, dir, out); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
	case "headers":
		for _, name := range []string{
			"ItemSet.dbc", "SkillLine.dbc", "SkillLineAbility.dbc", "AreaTable.dbc",
			"WorldMapArea.dbc", "ItemDisplayInfo.dbc", "Spell.dbc", "SpellIcon.dbc",
		} {
			d, err := Open(filepath.Join(dbcDir, name))
			if err != nil {
				fmt.Printf("%-22s MISSING/ERROR: %v\n", name, err)
				continue
			}
			fmt.Printf("%-22s records=%-6d fields=%-4d recordSize=%d\n", name, d.RecordCount, d.FieldCount, d.RecordSize)
		}
	default:
		fmt.Fprintln(os.Stderr, "unknown command:", cmd)
		os.Exit(2)
	}
}

// Expected 1.12 Faction.dbc layout (37 fields, 148-byte records):
//
//	0       ID
//	1       reputationIndex
//	2..5    reputationRaceMask[4]
//	6..9    reputationClassMask[4]
//	10..13  reputationBase[4]
//	14..17  reputationFlags[4]
//	18      parentFactionID
//	19..26  name_lang[8]   (19 = enUS)
//	27      name_flags
//	28..35  description_lang[8] (28 = enUS)
//	36      description_flags
const (
	facID       = 0
	facRaceMask = 2 // first of 4
	facParent   = 18
	facName     = 19 // enUS
	facDesc     = 28 // enUS
)

func verifyFactions(dbcDir string) error {
	d, err := Open(filepath.Join(dbcDir, "Faction.dbc"))
	if err != nil {
		return err
	}
	fmt.Printf("Faction.dbc: records=%d fields=%d recordSize=%d (expected fields=37 recordSize=148)\n",
		d.RecordCount, d.FieldCount, d.RecordSize)
	fmt.Println("--- sample records (verify name/parent/raceMask look sane) ---")
	want := map[uint32]bool{72: true, 76: true, 891: true, 529: true, 1: true, 530: true}
	for rec := 0; rec < d.RecordCount; rec++ {
		id := d.Uint32(rec, facID)
		if rec < 3 || want[id] {
			rm := [4]uint32{
				d.Uint32(rec, facRaceMask), d.Uint32(rec, facRaceMask+1),
				d.Uint32(rec, facRaceMask+2), d.Uint32(rec, facRaceMask+3),
			}
			fmt.Printf("id=%-4d parent=%-4d raceMask=%v name=%q\n",
				id, d.Uint32(rec, facParent), rm, d.Str(rec, facName))
		}
	}
	return nil
}
