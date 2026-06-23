package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// factionOut matches the shape the importer expects in data/factions.json.
type factionOut struct {
	FactionID uint32 `json:"factionID"`
	NameLoc0  string `json:"name_loc0"`
	DescLoc0  string `json:"description1_loc0"`
	Side      int    `json:"side"`
	Team      uint32 `json:"team"`
}

// Alliance/Horde grouping. side is not a Faction.dbc column; in the curated
// data it follows the team (parent) hierarchy, with the player races assigned
// by identity. These root group IDs are stable vanilla faction IDs.
var (
	allianceParents = map[uint32]bool{469: true, 891: true} // Alliance, Alliance Forces
	hordeParents    = map[uint32]bool{67: true, 892: true}  // Horde, Horde Forces
	alliancePlayers = map[uint32]bool{1: true, 3: true, 4: true, 8: true}
	hordePlayers    = map[uint32]bool{2: true, 5: true, 6: true, 9: true}

	// Parentless factions whose side is hand-curated (not inferable from the
	// hierarchy or player race). IDs are stable vanilla faction IDs.
	allianceSpecial = map[uint32]bool{469: true, 189: true, 61: true, 71: true, 49: true, 269: true, 589: true}
	hordeSpecial    = map[uint32]bool{67: true, 66: true}
)

func factionSide(id, parent uint32) int {
	switch {
	case alliancePlayers[id] || allianceParents[parent] || allianceSpecial[id]:
		return 1
	case hordePlayers[id] || hordeParents[parent] || hordeSpecial[id]:
		return 2
	default:
		return 0
	}
}

func genFactions(dbcDir string) ([]factionOut, error) {
	d, err := Open(filepath.Join(dbcDir, "Faction.dbc"))
	if err != nil {
		return nil, err
	}
	out := make([]factionOut, 0, d.RecordCount)
	for rec := 0; rec < d.RecordCount; rec++ {
		id := d.Uint32(rec, facID)
		parent := d.Uint32(rec, facParent)
		out = append(out, factionOut{
			FactionID: id,
			NameLoc0:  d.Str(rec, facName),
			DescLoc0:  d.Str(rec, facDesc),
			Side:      factionSide(id, parent),
			Team:      parent,
		})
	}
	// Match the committed file's ordering: by side, then name.
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Side != out[j].Side {
			return out[i].Side < out[j].Side
		}
		return out[i].NameLoc0 < out[j].NameLoc0
	})
	return out, nil
}

func writeFactions(dbcDir, outPath string) error {
	out, err := genFactions(dbcDir)
	if err != nil {
		return err
	}
	b, err := json.Marshal(out)
	if err != nil {
		return err
	}
	if err := os.WriteFile(outPath, b, 0644); err != nil {
		return err
	}
	fmt.Printf("wrote %d factions to %s\n", len(out), outPath)
	return nil
}

// validateFactions compares DBC-generated factions against the committed JSON,
// keyed by factionID, and reports per-field mismatches.
func validateFactions(dbcDir, committedPath string) error {
	gen, err := genFactions(dbcDir)
	if err != nil {
		return err
	}
	raw, err := os.ReadFile(committedPath)
	if err != nil {
		return err
	}
	var have []factionOut
	if err := json.Unmarshal(raw, &have); err != nil {
		return err
	}

	haveByID := make(map[uint32]factionOut, len(have))
	for _, f := range have {
		haveByID[f.FactionID] = f
	}
	genByID := make(map[uint32]factionOut, len(gen))
	for _, f := range gen {
		genByID[f.FactionID] = f
	}

	var nameMiss, descMiss, descMissRaw, sideMiss, teamMiss, missing, extra int
	trim := func(s string) string { return strings.TrimRight(s, "\r\n ") }
	for id, h := range haveByID {
		g, ok := genByID[id]
		if !ok {
			missing++
			continue
		}
		if g.NameLoc0 != h.NameLoc0 {
			nameMiss++
			if nameMiss <= 15 {
				fmt.Printf("  name  mismatch id=%d generated=%q committed=%q\n", id, g.NameLoc0, h.NameLoc0)
			}
		}
		if g.DescLoc0 != h.DescLoc0 {
			descMissRaw++
		}
		if trim(g.DescLoc0) != trim(h.DescLoc0) {
			descMiss++
		}
		if g.Team != h.Team {
			teamMiss++
		}
		if g.Side != h.Side {
			sideMiss++
		}
	}
	for id := range genByID {
		if _, ok := haveByID[id]; !ok {
			extra++
		}
	}

	fmt.Printf("committed=%d generated=%d\n", len(have), len(gen))
	fmt.Printf("mismatches  name=%d  description(raw)=%d  description(trimmed)=%d  team=%d  side=%d\n",
		nameMiss, descMissRaw, descMiss, teamMiss, sideMiss)
	fmt.Printf("in committed but not generated=%d   in generated but not committed=%d\n", missing, extra)
	return nil
}
