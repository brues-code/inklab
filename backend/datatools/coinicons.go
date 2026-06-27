package datatools

import (
	"fmt"
	"image"
	"image/draw"
	"os"
	"path/filepath"
)

// GenerateCoinIcons crops the gold/silver/copper coin cells out of the client's
// UI-MoneyIcons sprite (a 4-column strip where gold/silver/copper occupy the
// first three quarters) and writes gold.png/silver.png/copper.png to outDir.
func GenerateCoinIcons(cf ClientFiles, outDir string) (*IconGenResult, error) {
	var sheet *image.RGBA
	for _, p := range []string{
		`Interface\MoneyFrame\UI-MoneyIcons.blp`,
		`BlizzardInterfaceArt\MoneyFrame\UI-MoneyIcons.blp`,
	} {
		if b, err := cf.ReadFile(p); err == nil {
			if img, err := decodeBLP2Bytes(b); err == nil {
				sheet = img
				break
			}
		}
	}
	if sheet == nil {
		return nil, fmt.Errorf("UI-MoneyIcons not found or undecodable")
	}
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return nil, err
	}

	b := sheet.Bounds()
	w, h := b.Dx(), b.Dy()
	res := &IconGenResult{}
	cells := []struct {
		name     string
		x0f, x1f float64
	}{
		{"gold", 0.0, 0.25},
		{"silver", 0.25, 0.5},
		{"copper", 0.5, 0.75},
	}
	for _, c := range cells {
		x0, x1 := int(c.x0f*float64(w)), int(c.x1f*float64(w))
		if x1 <= x0 {
			res.Skipped++
			continue
		}
		cell := image.NewRGBA(image.Rect(0, 0, x1-x0, h))
		draw.Draw(cell, cell.Bounds(), sheet, image.Pt(b.Min.X+x0, b.Min.Y), draw.Src)
		if err := writePNG(filepath.Join(outDir, c.name+".png"), cell); err != nil {
			res.Skipped++
			continue
		}
		res.Generated++
	}
	return res, nil
}
