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
	"regexp"
	"strings"
)

// MapGenResult summarizes a zone-map generation run.
type MapGenResult struct {
	Generated int      `json:"generated"`
	Skipped   int      `json:"skipped"`
	Warnings  []string `json:"warnings"`
}

// overlay describes a revealed sub-area painted onto a zone's base map.
type overlay struct {
	w, h, offX, offY int
}

// GenerateZoneMaps stitches each WorldMap/<zone>/ folder of 12 base BLP tiles
// into a single data/maps/<zone>.jpg, compositing the explored-area overlay
// textures on top (positions from WorldMapOverlay.dbc) so the result is a
// fully-revealed map rather than the unexplored base. overlayDBC may be empty
// to skip overlays (base-only).
func GenerateZoneMaps(worldMapDir, overlayDBC, outDir string, progress func(zone string, i, total int)) (*MapGenResult, error) {
	entries, err := os.ReadDir(worldMapDir)
	if err != nil {
		return nil, fmt.Errorf("read WorldMap dir: %w", err)
	}
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return nil, err
	}

	overlays := map[string]overlay{}
	if overlayDBC != "" {
		overlays = parseOverlays(overlayDBC)
	}

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
		img, err := stitchZone(filepath.Join(worldMapDir, z), z, overlays)
		if err != nil {
			res.Skipped++
			res.Warnings = append(res.Warnings, fmt.Sprintf("%s: %v", z, err))
			continue
		}
		if err := writeJPEG(filepath.Join(outDir, z+".jpg"), img); err != nil {
			res.Skipped++
			res.Warnings = append(res.Warnings, fmt.Sprintf("%s: write %v", z, err))
			continue
		}
		res.Generated++
	}
	return res, nil
}

var tileNumRe = regexp.MustCompile(`^(.+?)(\d+)\.blp$`)

// stitchZone assembles the 12 (4x3) 256px base tiles into a 1024x768 image,
// then alpha-composites each explored-area overlay at its map offset.
func stitchZone(zoneDir, name string, overlays map[string]overlay) (*image.RGBA, error) {
	canvas := image.NewRGBA(image.Rect(0, 0, 256*4, 256*3))
	found := 0
	for i := 0; i < 12; i++ {
		tile, err := decodeBLP2(filepath.Join(zoneDir, fmt.Sprintf("%s%d.blp", name, i+1)))
		if err != nil {
			continue
		}
		found++
		blit(canvas, tile, (i%4)*256, (i/4)*256)
	}
	if found == 0 {
		return nil, fmt.Errorf("no decodable base tiles")
	}

	// Composite overlays. Collect unique overlay base names present in the dir.
	files, _ := os.ReadDir(zoneDir)
	seen := map[string]bool{}
	lowerName := strings.ToLower(name)
	for _, f := range files {
		m := tileNumRe.FindStringSubmatch(f.Name())
		if m == nil {
			continue // no trailing tile number (e.g. <Zone>Highlight.blp)
		}
		base := m[1]
		if strings.EqualFold(base, name) || strings.HasSuffix(strings.ToLower(base), "highlight") {
			continue // base zone tiles / highlight outline
		}
		key := strings.ToLower(base)
		if seen[key] {
			continue
		}
		seen[key] = true
		ov, ok := overlays[key]
		if !ok {
			continue // no placement info
		}
		compositeOverlay(canvas, zoneDir, base, ov)
	}
	_ = lowerName
	return canvas, nil
}

// compositeOverlay decodes an overlay's tiles and alpha-blends them onto canvas
// at the overlay's map offset. Tiles are numbered row-major, ceil(w/256) wide.
// The read is capped to the DBC's cols*rows: some octo-custom zones ship more
// physical tiles than their WorldMapOverlay dimensions describe, and reading
// the extras would place them a row too low (bleeding into the next area).
func compositeOverlay(canvas *image.RGBA, zoneDir, base string, ov overlay) {
	cols := (ov.w + 255) / 256
	rows := (ov.h + 255) / 256
	if cols < 1 {
		cols = 1
	}
	if rows < 1 {
		rows = 1
	}
	for i := 1; i <= cols*rows; i++ {
		tile, err := decodeBLP2(filepath.Join(zoneDir, fmt.Sprintf("%s%d.blp", base, i)))
		if err != nil {
			break
		}
		col := (i - 1) % cols
		row := (i - 1) / cols
		blitAlpha(canvas, tile, ov.offX+col*256, ov.offY+row*256)
	}
}

// parseOverlays reads WorldMapOverlay.dbc -> lowercased textureName -> placement.
func parseOverlays(path string) map[string]overlay {
	out := map[string]overlay{}
	b, err := os.ReadFile(path)
	if err != nil || len(b) < 20 || string(b[0:4]) != "WDBC" {
		return out
	}
	rc := int(binary.LittleEndian.Uint32(b[4:8]))
	rs := int(binary.LittleEndian.Uint32(b[12:16]))
	sb := 20 + rc*rs
	field := func(rec, f int) uint32 {
		o := 20 + rec*rs + f*4
		if o+4 > len(b) {
			return 0
		}
		return binary.LittleEndian.Uint32(b[o : o+4])
	}
	str := func(rec, f int) string {
		off := sb + int(field(rec, f))
		e := off
		for e < len(b) && b[e] != 0 {
			e++
		}
		if off < 0 || off > len(b) {
			return ""
		}
		return string(b[off:e])
	}
	// Layout (17 fields): id, mapAreaID, areaID[6], textureName(8), w(9), h(10),
	// offsetX(11), offsetY(12), hitRect[4].
	for r := 0; r < rc; r++ {
		name := strings.ToLower(strings.TrimSpace(str(r, 8)))
		if name == "" {
			continue
		}
		out[name] = overlay{
			w: int(field(r, 9)), h: int(field(r, 10)),
			offX: int(field(r, 11)), offY: int(field(r, 12)),
		}
	}
	return out
}

// blit copies an opaque tile onto dst at (ox,oy).
func blit(dst *image.RGBA, src *image.RGBA, ox, oy int) {
	b := src.Bounds()
	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			dst.SetRGBA(ox+x, oy+y, src.RGBAAt(x, y))
		}
	}
}

// blitAlpha alpha-blends src onto dst at (ox,oy).
func blitAlpha(dst *image.RGBA, src *image.RGBA, ox, oy int) {
	b := src.Bounds()
	for y := 0; y < b.Dy(); y++ {
		dy := oy + y
		for x := 0; x < b.Dx(); x++ {
			dx := ox + x
			if dx < 0 || dy < 0 || dx >= dst.Bounds().Dx() || dy >= dst.Bounds().Dy() {
				continue
			}
			s := src.RGBAAt(x, y)
			if s.A == 0 {
				continue
			}
			if s.A == 255 {
				dst.SetRGBA(dx, dy, s)
				continue
			}
			d := dst.RGBAAt(dx, dy)
			a := int(s.A)
			dst.SetRGBA(dx, dy, color.RGBA{
				uint8((int(s.R)*a + int(d.R)*(255-a)) / 255),
				uint8((int(s.G)*a + int(d.G)*(255-a)) / 255),
				uint8((int(s.B)*a + int(d.B)*(255-a)) / 255),
				255,
			})
		}
	}
}

func writeJPEG(path string, img image.Image) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return jpeg.Encode(f, img, &jpeg.Options{Quality: 82})
}

// decodeBLP2 decodes the top mip of a BLP2 texture into RGBA. Handles the two
// variants world maps use: DXT1 (alphaEncoding 0, opaque base tiles) and DXT3
// (alphaEncoding 1, overlay tiles with an explicit alpha block).
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
	if enc != 2 || (alphaEnc != 0 && alphaEnc != 1) {
		return nil, fmt.Errorf("unsupported enc=%d alphaEnc=%d", enc, alphaEnc)
	}
	if mipOff <= 0 || mipOff > len(b) {
		return nil, fmt.Errorf("bad mip offset")
	}
	dxt3 := alphaEnc == 1
	blockBytes := 8
	if dxt3 {
		blockBytes = 16
	}
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	data := b[mipOff:]
	bi := 0
	for by := 0; by < h; by += 4 {
		for bx := 0; bx < w; bx += 4 {
			if bi+blockBytes > len(data) {
				return img, nil
			}
			var alpha [16]uint8
			colorBlock := data[bi : bi+8]
			if dxt3 {
				ab := data[bi : bi+8]
				for p := 0; p < 16; p++ {
					nib := (ab[p/2] >> uint(4*(p&1))) & 0xF
					alpha[p] = nib * 17
				}
				colorBlock = data[bi+8 : bi+16]
			} else {
				for p := range alpha {
					alpha[p] = 255
				}
			}
			bi += blockBytes

			c0 := binary.LittleEndian.Uint16(colorBlock[0:2])
			c1 := binary.LittleEndian.Uint16(colorBlock[2:4])
			var pal [4]color.RGBA
			pal[0] = rgb565(c0)
			pal[1] = rgb565(c1)
			if dxt3 || c0 > c1 {
				pal[2] = lerp(pal[0], pal[1], 2, 1)
				pal[3] = lerp(pal[0], pal[1], 1, 2)
			} else {
				pal[2] = lerp(pal[0], pal[1], 1, 1)
				pal[3] = color.RGBA{0, 0, 0, 0}
			}
			idx := binary.LittleEndian.Uint32(colorBlock[4:8])
			for py := 0; py < 4; py++ {
				for px := 0; px < 4; px++ {
					p := py*4 + px
					ci := (idx >> uint(2*p)) & 3
					c := pal[ci]
					c.A = alpha[p]
					if ci == 3 && !dxt3 && c0 <= c1 {
						c.A = 0 // DXT1 transparent index
					}
					img.SetRGBA(bx+px, by+py, c)
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

// NormalizeZoneName trims a stored zone name to its texture-folder base.
func NormalizeZoneName(zone string) string { return strings.TrimSpace(zone) }
