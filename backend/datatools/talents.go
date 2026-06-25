package datatools

import (
	"encoding/json"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"strings"
)

// GenerateTalentsJSON regenerates just data/talents.json from a client source
// (without running the full gen-all pass).
func GenerateTalentsJSON(cf ClientFiles, dataDir string) error {
	v, err := genTalents(cf)
	if err != nil {
		return err
	}
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dataDir, "talents.json"), b, 0644)
}

// Talent.dbc (21 fields, 1.12): id(0), tabID(1), row(2), col(3),
// spellRank[9](4-12), prereqTalent[3](13-15), prereqRank[3](16-18), flags(19),
// reqSpell(20).
//
// TalentTab.dbc (15 fields): id(0), name[8](1-8), flags(9), spellIconID(10),
// raceMask(11), classMask(12), orderIndex(13), background(14).

// TalentOut is one talent: its grid position, the spell id of each rank (length
// = max rank), and an optional prerequisite (another talent at a given rank).
type TalentOut struct {
	ID        int   `json:"id"`
	Row       int   `json:"row"`
	Col       int   `json:"col"`
	Ranks     []int `json:"ranks"` // spell id per rank; len == maxRank
	ReqTalent int   `json:"reqTalent"`
	ReqRank   int   `json:"reqRank"`
}

// TalentTabOut is one of a class's three talent trees.
type TalentTabOut struct {
	ID         int         `json:"id"`
	Name       string      `json:"name"`
	Class      string      `json:"class"`     // e.g. "MAGE"
	ClassMask  int         `json:"classMask"` // single class bit
	Order      int         `json:"order"`     // 0..2, tree position for the class
	Background string      `json:"background"`// TalentFrame art base name, e.g. "MageArcane"
	Talents    []TalentOut `json:"talents"`
}

// classByMask maps a TalentTab classMask (a single class bit) to a class token.
var classByMask = map[uint32]string{
	1:    "WARRIOR",
	2:    "PALADIN",
	4:    "HUNTER",
	8:    "ROGUE",
	16:   "PRIEST",
	64:   "SHAMAN",
	128:  "MAGE",
	256:  "WARLOCK",
	1024: "DRUID",
}

// genTalents builds the class talent trees from Talent.dbc + TalentTab.dbc. The
// per-rank spell ids are emitted raw; name/description/icon are resolved later
// from the (already $-resolved) spell_template rows at query time.
func genTalents(cf ClientFiles) (interface{}, error) {
	tabDBC, err := openDBCFrom(cf, "TalentTab.dbc")
	if err != nil {
		return nil, err
	}
	talDBC, err := openDBCFrom(cf, "Talent.dbc")
	if err != nil {
		return nil, err
	}

	tabs := make([]*TalentTabOut, 0, tabDBC.RecordCount)
	byID := make(map[int]*TalentTabOut, tabDBC.RecordCount)
	for r := 0; r < tabDBC.RecordCount; r++ {
		mask := tabDBC.U32(r, 12)
		t := &TalentTabOut{
			ID:         int(tabDBC.U32(r, 0)),
			Name:       tabDBC.Str(r, 1),
			Class:      classByMask[mask],
			ClassMask:  int(mask),
			Order:      int(tabDBC.U32(r, 13)),
			Background: tabDBC.Str(r, 14),
		}
		tabs = append(tabs, t)
		byID[t.ID] = t
	}

	for r := 0; r < talDBC.RecordCount; r++ {
		tabID := int(talDBC.U32(r, 1))
		tab := byID[tabID]
		if tab == nil {
			continue
		}
		var ranks []int
		for f := 4; f <= 12; f++ {
			s := int(talDBC.U32(r, f))
			if s == 0 {
				break
			}
			ranks = append(ranks, s)
		}
		if len(ranks) == 0 {
			continue
		}
		tab.Talents = append(tab.Talents, TalentOut{
			ID:        int(talDBC.U32(r, 0)),
			Row:       int(talDBC.U32(r, 2)),
			Col:       int(talDBC.U32(r, 3)),
			Ranks:     ranks,
			ReqTalent: int(talDBC.U32(r, 13)),
			ReqRank:   int(talDBC.U32(r, 16)),
		})
	}
	return tabs, nil
}

// GenerateTalentBackgrounds composites the four TalentFrame art tiles for every
// talent tree into outDir/<lowercase background>.png (320x384). Mirrors the icon
// extraction model: the art is built locally from the client, not embedded.
func GenerateTalentBackgrounds(cf ClientFiles, outDir string, progress func(name string, i, total int)) (*IconGenResult, error) {
	v, err := genTalents(cf)
	if err != nil {
		return nil, err
	}
	tabs := v.([]*TalentTabOut)
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return nil, err
	}

	// Unique background names (3 trees per class share none, but be safe).
	seen := map[string]bool{}
	var bgs []string
	for _, t := range tabs {
		if t.Background == "" || seen[strings.ToLower(t.Background)] {
			continue
		}
		seen[strings.ToLower(t.Background)] = true
		bgs = append(bgs, t.Background)
	}

	res := &IconGenResult{}
	for i, bg := range bgs {
		if progress != nil {
			progress(bg, i+1, len(bgs))
		}
		img := compositeTalentBG(cf, bg)
		if img == nil {
			res.Skipped++
			if len(res.Warnings) < 20 {
				res.Warnings = append(res.Warnings, fmt.Sprintf("%s: tiles missing", bg))
			}
			continue
		}
		if err := writePNG(filepath.Join(outDir, strings.ToLower(bg)+".png"), img); err != nil {
			res.Skipped++
			continue
		}
		res.Generated++
	}
	return res, nil
}

// compositeTalentBG stitches the four TalentFrame tiles into a 320x384 RGBA, or
// returns nil if the primary tile is unreadable. Tile layout:
//   TopLeft  256x256 @ (0,0)     TopRight    64x256 @ (256,0)
//   BottomLeft 256x128 @ (0,256) BottomRight 64x128 @ (256,256)
func compositeTalentBG(cf ClientFiles, bg string) *image.RGBA {
	tile := func(suffix string) *image.RGBA {
		return loadBLPTexture(cf, `Interface\TalentFrame\`+bg+suffix+".blp")
	}
	tl := tile("-TopLeft")
	if tl == nil {
		return nil
	}
	canvas := image.NewRGBA(image.Rect(0, 0, 320, 384))
	blit(canvas, tl, 0, 0)
	if tr := tile("-TopRight"); tr != nil {
		blit(canvas, tr, 256, 0)
	}
	if bl := tile("-BottomLeft"); bl != nil {
		blit(canvas, bl, 0, 256)
	}
	if br := tile("-BottomRight"); br != nil {
		blit(canvas, br, 256, 256)
	}
	return canvas
}
