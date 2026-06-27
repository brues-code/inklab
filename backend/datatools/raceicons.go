package datatools

import (
	"fmt"
	"image"
	"image/draw"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// raceIconCoordRe matches one RACE_ICON_TCOORDS entry, e.g.
// ["HUMAN_MALE"] = {0, 0.125, 0, 0.25}.
var raceIconCoordRe = regexp.MustCompile(`\["(\w+)"\]\s*=\s*\{([^}]*)\}`)

// GenerateRaceIcons crops the per-race/gender character-create icons out of the
// client's UI-CharacterCreate-Races sprite sheet, using the RACE_ICON_TCOORDS
// table from GlueXML/CharacterCreate.lua, and writes one PNG per entry to
// outDir/<key>.png (lowercased, e.g. human_male.png). It adapts to whatever
// races the coord table defines — including server-custom ones.
func GenerateRaceIcons(cf ClientFiles, outDir string) (*IconGenResult, error) {
	var lua string
	for _, p := range []string{`BlizzardInterfaceCode\GlueXML\CharacterCreate.lua`, `Interface\GlueXML\CharacterCreate.lua`} {
		if b, err := cf.ReadFile(p); err == nil {
			lua = string(b)
			break
		}
	}
	if lua == "" {
		return nil, fmt.Errorf("CharacterCreate.lua not found")
	}

	// Parse just the RACE_ICON_TCOORDS block (CLASS_ICON_TCOORDS follows it).
	start := strings.Index(lua, "RACE_ICON_TCOORDS")
	if start < 0 {
		return nil, fmt.Errorf("RACE_ICON_TCOORDS not found")
	}
	block := lua[start:]
	if end := strings.Index(block, "};"); end > 0 {
		block = block[:end]
	}
	coords := map[string][4]float64{}
	for _, m := range raceIconCoordRe.FindAllStringSubmatch(block, -1) {
		parts := strings.Split(m[2], ",")
		if len(parts) < 4 {
			continue
		}
		var c [4]float64
		ok := true
		for i := 0; i < 4; i++ {
			v, err := strconv.ParseFloat(strings.TrimSpace(parts[i]), 64)
			if err != nil {
				ok = false
				break
			}
			c[i] = v
		}
		if ok {
			coords[strings.ToUpper(m[1])] = c
		}
	}
	if len(coords) == 0 {
		return nil, fmt.Errorf("no race icon coords parsed")
	}

	// Decode the sprite sheet (BlizzardInterfaceArt path in loose clients, or the
	// Interface\ path inside the MPQs).
	var sheet *image.RGBA
	for _, p := range []string{
		`BlizzardInterfaceArt\Glues\CharacterCreate\UI-CharacterCreate-Races.blp`,
		`Interface\Glues\CharacterCreate\UI-CharacterCreate-Races.blp`,
	} {
		if b, err := cf.ReadFile(p); err == nil {
			if img, err := decodeBLP2Bytes(b); err == nil {
				sheet = img
				break
			}
		}
	}
	if sheet == nil {
		return nil, fmt.Errorf("race sprite sheet not found or undecodable")
	}

	if err := os.MkdirAll(outDir, 0755); err != nil {
		return nil, err
	}
	b := sheet.Bounds()
	w, h := b.Dx(), b.Dy()
	res := &IconGenResult{}
	for key, c := range coords {
		x0, x1 := int(c[0]*float64(w)), int(c[1]*float64(w))
		y0, y1 := int(c[2]*float64(h)), int(c[3]*float64(h))
		if x1 <= x0 || y1 <= y0 {
			res.Skipped++
			continue
		}
		cell := image.NewRGBA(image.Rect(0, 0, x1-x0, y1-y0))
		draw.Draw(cell, cell.Bounds(), sheet, image.Pt(b.Min.X+x0, b.Min.Y+y0), draw.Src)
		if err := writePNG(filepath.Join(outDir, strings.ToLower(key)+".png"), cell); err != nil {
			res.Skipped++
			continue
		}
		res.Generated++
	}
	return res, nil
}
