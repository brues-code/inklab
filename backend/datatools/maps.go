// Package datatools holds in-process implementations of the data-import jobs
// (WDB cache overlays, zone-map generation) so both the CLI tools and the app's
// Tools tab can run them without a Go toolchain present.
package datatools

import (
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"path/filepath"
	"strings"
)

// MapGenResult summarizes a zone-map generation run.
type MapGenResult struct {
	Generated int      `json:"generated"`
	Skipped   int      `json:"skipped"`
	Warnings  []string `json:"warnings"`
}

// GenerateZoneMaps stitches each WorldMap/<zone>/ folder of 12 BLP tiles into a
// single data/maps/<zone>.jpg. worldMapDir is the client's
// Interface\WorldMap directory; outDir receives the JPGs.
func GenerateZoneMaps(worldMapDir, outDir string, progress func(zone string, i, total int)) (*MapGenResult, error) {
	entries, err := os.ReadDir(worldMapDir)
	if err != nil {
		return nil, fmt.Errorf("read WorldMap dir: %w", err)
	}
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return nil, err
	}

	// Collect zone folders that actually hold a tile set (<zone>1.blp).
	var zones []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		z := e.Name()
		if _, err := os.Stat(filepath.Join(worldMapDir, z, z+"1.blp")); err == nil {
			zones = append(zones, z)
		}
	}

	res := &MapGenResult{}
	for i, z := range zones {
		if progress != nil {
			progress(z, i+1, len(zones))
		}
		img, err := stitchZone(filepath.Join(worldMapDir, z), z)
		if err != nil {
			res.Skipped++
			res.Warnings = append(res.Warnings, fmt.Sprintf("%s: %v", z, err))
			continue
		}
		out := filepath.Join(outDir, z+".jpg")
		if err := writeJPEG(out, img); err != nil {
			res.Skipped++
			res.Warnings = append(res.Warnings, fmt.Sprintf("%s: write %v", z, err))
			continue
		}
		res.Generated++
	}
	return res, nil
}

// stitchZone assembles the 12 (4x3) 256px tiles into one 1024x768 image.
func stitchZone(zoneDir, name string) (*image.RGBA, error) {
	canvas := image.NewRGBA(image.Rect(0, 0, 256*4, 256*3))
	found := 0
	for i := 0; i < 12; i++ {
		tile, err := decodeBLP2(filepath.Join(zoneDir, fmt.Sprintf("%s%d.blp", name, i+1)))
		if err != nil {
			continue
		}
		found++
		ox, oy := (i%4)*256, (i/4)*256
		for y := 0; y < 256; y++ {
			for x := 0; x < 256; x++ {
				canvas.SetRGBA(ox+x, oy+y, tile.RGBAAt(x, y))
			}
		}
	}
	if found == 0 {
		return nil, fmt.Errorf("no decodable tiles")
	}
	return canvas, nil
}

func writeJPEG(path string, img image.Image) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return jpeg.Encode(f, img, &jpeg.Options{Quality: 82})
}

// decodeBLP2 decodes the top mip of a BLP2 / DXT1 texture (encoding=2,
// alphaEncoding=0) into RGBA. World-map tiles are all this variant.
func decodeBLP2(path string) (*image.RGBA, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(b) < 0x94 || string(b[0:4]) != "BLP2" {
		return nil, fmt.Errorf("not BLP2")
	}
	enc, alphaEnc := b[8], b[10]
	w := int(binary.LittleEndian.Uint32(b[12:16]))
	h := int(binary.LittleEndian.Uint32(b[16:20]))
	mipOff := int(binary.LittleEndian.Uint32(b[20:24]))
	if enc != 2 || alphaEnc != 0 {
		return nil, fmt.Errorf("unsupported enc=%d alphaEnc=%d", enc, alphaEnc)
	}
	if mipOff <= 0 || mipOff > len(b) {
		return nil, fmt.Errorf("bad mip offset")
	}
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	data := b[mipOff:]
	bi := 0
	for by := 0; by < h; by += 4 {
		for bx := 0; bx < w; bx += 4 {
			if bi+8 > len(data) {
				return img, nil
			}
			block := data[bi : bi+8]
			bi += 8
			c0 := binary.LittleEndian.Uint16(block[0:2])
			c1 := binary.LittleEndian.Uint16(block[2:4])
			var pal [4]color.RGBA
			pal[0] = rgb565(c0)
			pal[1] = rgb565(c1)
			if c0 > c1 {
				pal[2] = lerp(pal[0], pal[1], 2, 1)
				pal[3] = lerp(pal[0], pal[1], 1, 2)
			} else {
				pal[2] = lerp(pal[0], pal[1], 1, 1)
				pal[3] = color.RGBA{0, 0, 0, 255}
			}
			idx := binary.LittleEndian.Uint32(block[4:8])
			for py := 0; py < 4; py++ {
				for px := 0; px < 4; px++ {
					ci := (idx >> uint(2*(py*4+px))) & 3
					img.SetRGBA(bx+px, by+py, pal[ci])
				}
			}
		}
	}
	return img, nil
}

func rgb565(v uint16) color.RGBA {
	r := uint8((v>>11)&0x1f) << 3
	g := uint8((v>>5)&0x3f) << 2
	b := uint8(v&0x1f) << 3
	return color.RGBA{r | r>>5, g | g>>6, b | b>>5, 255}
}

func lerp(a, b color.RGBA, wa, wb int) color.RGBA {
	t := wa + wb
	return color.RGBA{
		uint8((int(a.R)*wa + int(b.R)*wb) / t),
		uint8((int(a.G)*wa + int(b.G)*wb) / t),
		uint8((int(a.B)*wa + int(b.B)*wb) / t),
		255,
	}
}

// NormalizeZoneName maps a stored zone name to its texture-folder base, e.g.
// trimming spaces so "Elwynn Forest" style values still resolve when callers
// pass localized names. Currently zones.json stores the folder name directly.
func NormalizeZoneName(zone string) string {
	return strings.TrimSpace(zone)
}
