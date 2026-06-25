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
	"math"
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
	dbcDir := ""
	if overlayDBC != "" {
		dbcDir = filepath.Dir(overlayDBC)
	}
	return GenerateZoneMapsFrom(NewDirSourceMaps(worldMapDir, dbcDir), outDir, progress)
}

// GenerateZoneMapsFrom stitches each WorldMap zone's base tiles + explored-area
// overlays into data/maps/<zone>.jpg, reading from any ClientFiles source (loose
// folder or in-memory MPQ).
func GenerateZoneMapsFrom(cf ClientFiles, outDir string, progress func(zone string, i, total int)) (*MapGenResult, error) {
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return nil, err
	}

	// Overlay placements are scoped to each zone's WorldMapArea id. octo's data
	// sometimes drops one zone's overlay tiles into another's folder (e.g. Hyjal
	// ships Tel Abim tiles like TheJaggedIsles); keying placement by area id — not
	// just texture name — keeps those foreign tiles from bleeding onto the map.
	areaIDByName := map[string]uint32{}
	if b, err := cf.ReadDBC("WorldMapArea.dbc"); err == nil {
		areaIDByName = parseWorldMapAreaIDs(b)
	}
	overlaysByArea := map[uint32]map[string]overlay{}
	if b, err := cf.ReadDBC("WorldMapOverlay.dbc"); err == nil {
		overlaysByArea = parseOverlaysByArea(b)
	}

	allZones, err := cf.ListZones()
	if err != nil {
		return nil, fmt.Errorf("list WorldMap zones: %w", err)
	}
	// Keep only zones that actually have a first base tile (skip overlay-only or
	// chrome folders silently rather than warning).
	var zones []string
	for _, z := range allZones {
		if _, err := cf.ReadZoneFile(z, z+"1.blp"); err == nil {
			zones = append(zones, z)
		}
	}

	res := &MapGenResult{}
	for i, z := range zones {
		if progress != nil {
			progress(z, i+1, len(zones))
		}
		img, err := stitchZone(cf, z, overlaysByArea[areaIDByName[strings.ToLower(z)]])
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
func stitchZone(cf ClientFiles, name string, overlays map[string]overlay) (*image.RGBA, error) {
	canvas := image.NewRGBA(image.Rect(0, 0, 256*4, 256*3))
	found := 0
	for i := 0; i < 12; i++ {
		b, err := cf.ReadZoneFile(name, fmt.Sprintf("%s%d.blp", name, i+1))
		if err != nil {
			continue
		}
		tile, err := decodeBLP2Bytes(b)
		if err != nil {
			continue
		}
		found++
		blit(canvas, tile, (i%4)*256, (i/4)*256)
	}
	if found == 0 {
		return nil, fmt.Errorf("no decodable base tiles")
	}

	// Composite overlays. Collect unique overlay base names present in the folder.
	files, _ := cf.ListZoneFiles(name)
	seen := map[string]bool{}
	for _, f := range files {
		m := tileNumRe.FindStringSubmatch(f)
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
		compositeOverlay(canvas, cf, name, base, ov)
	}

	// The 4x3 256px tile grid is 1024x768, but the actual map content (and the
	// in-game WorldMapDetailFrame) is 1002x668 — the extra right/bottom strips
	// are black tile padding. Crop them off.
	cropped := image.NewRGBA(image.Rect(0, 0, mapWidth, mapHeight))
	blit(cropped, canvas.SubImage(image.Rect(0, 0, mapWidth, mapHeight)).(*image.RGBA), 0, 0)
	return cropped, nil
}

// mapWidth/mapHeight are the rendered zone-map dimensions (the visible
// WorldMapDetailFrame region within the 1024x768 tile grid).
const (
	mapWidth  = 1002
	mapHeight = 668
)

// compositeOverlay decodes an overlay's tiles (numbered row-major from 1) and
// alpha-blends them onto canvas at the overlay's map offset, with 256px tile
// spacing.
//
// The challenge is that octo frequently ships MORE tiles than an overlay's
// WorldMapOverlay.dbc rect calls for, in two unrelated ways that must be told
// apart:
//
//   - Appended foreign tiles: unrelated map art dumped into the folder after the
//     real overlay (e.g. Icepoint's "kaneqnuun", Lapidis's "caelansrest" /
//     "toweroflapidis"). Drawing these scatters duplicate landmass and labels
//     across the map. The real overlay must be isolated and the rest dropped.
//   - Upscaled art: the same rect re-exported at higher resolution (e.g.
//     Stonetalon's WindshearCrag ships 4 tiles for a 1-tile rect). Every tile is
//     real and should be drawn, scaled implicitly by the 256px grid.
//
// An overlay's true grid is cols=ceil(w/256) x rows=ceil(h/256) tiles, where only
// the last column may be <256 wide and only the last row <256 tall. So a
// partial-width tile marks the real column count (and a sliver column the DBC w
// rounds away, as in kaneqnuun), and a partial-height tile marks the last row.
// Tiles past that grid are foreign and dropped. Only when the art has no partial
// edge at all AND the DBC rect is a whole number of 256px tiles do we treat the
// extras as an upscale and keep them.
func compositeOverlay(canvas *image.RGBA, cf ClientFiles, zone, base string, ov overlay) {
	var tiles []*image.RGBA
	for i := 1; i <= 64; i++ {
		b, err := cf.ReadZoneFile(zone, fmt.Sprintf("%s%d.blp", base, i))
		if err != nil {
			break
		}
		tile, err := decodeBLP2Bytes(b)
		if err != nil {
			break
		}
		tiles = append(tiles, tile)
	}
	if len(tiles) == 0 {
		return
	}

	dbcCols := (ov.w + 255) / 256
	dbcRows := (ov.h + 255) / 256
	if dbcCols < 1 {
		dbcCols = 1
	}
	if dbcRows < 1 {
		dbcRows = 1
	}

	// Real column count: the first partial-width tile is a row's right edge. Search
	// one past dbcCols so a sliver column the DBC width rounds off (kaneqnuun's 8px
	// 3rd column) is still found.
	cols, widthEdge := dbcCols, false
	for i := 0; i < len(tiles) && i <= dbcCols; i++ {
		if tiles[i].Bounds().Dx() < 256 {
			cols, widthEdge = i+1, true
			break
		}
	}

	// Real row count: the row holding the first partial-height tile is the last row.
	rows, heightEdge := dbcRows, false
	for i, t := range tiles {
		if t.Bounds().Dy() < 256 {
			rows, heightEdge = i/cols+1, true
			break
		}
	}

	limit := cols * rows
	// Upscale exception: no partial edge anywhere and a whole-tile DBC rect means
	// the extras are higher-res art for the same rect, not foreign — keep them all.
	if !widthEdge && !heightEdge && ov.w%256 == 0 && ov.h%256 == 0 && len(tiles) > limit {
		cols = overlayCols(len(tiles), ov.w, ov.h)
		limit = len(tiles)
	}
	if cols < 1 {
		cols = 1
	}
	if limit > len(tiles) {
		limit = len(tiles)
	}

	for i := 0; i < limit; i++ {
		col := i % cols
		row := i / cols
		blitAlpha(canvas, tiles[i], ov.offX+col*256, ov.offY+row*256)
	}
}

// overlayCols infers an overlay's column count from the n tiles present and its
// DBC dimensions. When the dimensions account for the tiles
// (ceil(w/256)*ceil(h/256) >= n) they're trusted directly; otherwise the art
// was upscaled past its stale DBC entry, so the grid shape is recovered from the
// aspect ratio: cols ≈ round(sqrt(n * w/h)).
func overlayCols(n, w, h int) int {
	if n <= 1 || w <= 0 || h <= 0 {
		return 1
	}
	cc := (w + 255) / 256
	cr := (h + 255) / 256
	if cc < 1 {
		cc = 1
	}
	if cr < 1 {
		cr = 1
	}
	if n <= cc*cr {
		return cc
	}
	cols := int(math.Round(math.Sqrt(float64(n) * float64(w) / float64(h))))
	if cols < 1 {
		cols = 1
	}
	if cols > n {
		cols = n
	}
	return cols
}

// parseWorldMapAreaIDs parses WorldMapArea.dbc -> lowercased areaName (the
// WorldMap texture-folder name) -> record id, so overlays can be matched to the
// zone they belong to. Layout (8 fields): id(0), mapID(1), areaID(2), name(3), ...
func parseWorldMapAreaIDs(b []byte) map[string]uint32 {
	out := map[string]uint32{}
	if len(b) < 20 || string(b[0:4]) != "WDBC" {
		return out
	}
	rc := int(binary.LittleEndian.Uint32(b[4:8]))
	rs := int(binary.LittleEndian.Uint32(b[12:16]))
	if rc <= 0 || rs < 16 {
		return out
	}
	sb := 20 + rc*rs
	for r := 0; r < rc; r++ {
		rec := 20 + r*rs
		if rec+16 > len(b) {
			break
		}
		id := binary.LittleEndian.Uint32(b[rec : rec+4])
		off := sb + int(binary.LittleEndian.Uint32(b[rec+12:rec+16]))
		if off < sb || off >= len(b) {
			continue
		}
		e := off
		for e < len(b) && b[e] != 0 {
			e++
		}
		if name := strings.ToLower(strings.TrimSpace(string(b[off:e]))); name != "" {
			out[name] = id
		}
	}
	return out
}

// parseOverlaysByArea parses WorldMapOverlay.dbc bytes into
// mapAreaID -> lowercased textureName -> placement. Keying by area id (not just
// texture name) means a zone only composites the overlays that actually belong
// to it. Layout (17 fields): id, mapAreaID(1), areaID[2..7], textureName(8),
// w(9), h(10), offsetX(11), offsetY(12), hitRect[4].
func parseOverlaysByArea(b []byte) map[uint32]map[string]overlay {
	out := map[uint32]map[string]overlay{}
	if len(b) < 20 || string(b[0:4]) != "WDBC" {
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
	for r := 0; r < rc; r++ {
		name := strings.ToLower(strings.TrimSpace(str(r, 8)))
		if name == "" {
			continue
		}
		area := field(r, 1)
		if out[area] == nil {
			out[area] = map[string]overlay{}
		}
		out[area][name] = overlay{
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
	return decodeBLP2Bytes(b)
}

// decodeBLP2Bytes decodes a BLP2 blob already in memory (e.g. read from an MPQ).
func decodeBLP2Bytes(b []byte) (*image.RGBA, error) {
	if len(b) < 0x94 || string(b[0:4]) != "BLP2" {
		return nil, fmt.Errorf("not BLP2")
	}
	enc, alphaDepth, alphaEnc := b[8], b[9], b[10]
	w := int(binary.LittleEndian.Uint32(b[12:16]))
	h := int(binary.LittleEndian.Uint32(b[16:20]))
	mipOff := int(binary.LittleEndian.Uint32(b[20:24]))
	if mipOff <= 0 || mipOff > len(b) {
		return nil, fmt.Errorf("bad mip offset")
	}
	// enc 1 = palettized; enc 2 = DXT (alphaEnc 0=DXT1, 1=DXT3, 7=DXT5).
	if enc == 1 {
		return decodePalettized(b, w, h, mipOff, alphaDepth)
	}
	if enc != 2 || (alphaEnc != 0 && alphaEnc != 1 && alphaEnc != 7) {
		return nil, fmt.Errorf("unsupported enc=%d alphaEnc=%d", enc, alphaEnc)
	}
	dxt3 := alphaEnc == 1
	dxt5 := alphaEnc == 7
	blockBytes := 8
	if dxt3 || dxt5 {
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
			switch {
			case dxt3:
				ab := data[bi : bi+8]
				for p := 0; p < 16; p++ {
					alpha[p] = ((ab[p/2] >> uint(4*(p&1))) & 0xF) * 17
				}
				colorBlock = data[bi+8 : bi+16]
			case dxt5:
				dxt5Alpha(data[bi:bi+8], &alpha)
				colorBlock = data[bi+8 : bi+16]
			default:
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
			alphaMode := dxt3 || dxt5
			if alphaMode || c0 > c1 {
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
					if ci == 3 && !alphaMode && c0 <= c1 {
						c.A = 0 // DXT1 1-bit transparent index
					}
					img.SetRGBA(bx+px, by+py, c)
				}
			}
		}
	}
	return img, nil
}

// decodePalettized decodes a BLP2 enc=1 image: a 256-entry BGRA palette at
// offset 0x94, then mip data of 1-byte palette indices, optionally followed by
// an 8-bit alpha plane (alphaDepth==8).
func decodePalettized(b []byte, w, h, mipOff int, alphaDepth byte) (*image.RGBA, error) {
	const palOff = 0x94
	if palOff+256*4 > len(b) {
		return nil, fmt.Errorf("palette out of range")
	}
	pixels := w * h
	if mipOff+pixels > len(b) {
		return nil, fmt.Errorf("index data out of range")
	}
	hasAlpha := alphaDepth == 8 && mipOff+pixels*2 <= len(b)
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for i := 0; i < pixels; i++ {
		idx := int(b[mipOff+i])
		p := palOff + idx*4
		a := uint8(255)
		if hasAlpha {
			a = b[mipOff+pixels+i]
		}
		img.SetRGBA(i%w, i/w, color.RGBA{b[p+2], b[p+1], b[p], a}) // BGRA -> RGBA
	}
	return img, nil
}

// dxt5Alpha decodes a DXT5 alpha block (8 bytes: a0, a1, then 16x 3-bit indices).
func dxt5Alpha(blk []byte, out *[16]uint8) {
	a0, a1 := int(blk[0]), int(blk[1])
	var av [8]int
	av[0], av[1] = a0, a1
	if a0 > a1 {
		for i := 1; i < 7; i++ {
			av[i+1] = ((7-i)*a0 + i*a1) / 7
		}
	} else {
		for i := 1; i < 5; i++ {
			av[i+1] = ((5-i)*a0 + i*a1) / 5
		}
		av[6], av[7] = 0, 255
	}
	bits := uint64(blk[2]) | uint64(blk[3])<<8 | uint64(blk[4])<<16 |
		uint64(blk[5])<<24 | uint64(blk[6])<<32 | uint64(blk[7])<<40
	for p := 0; p < 16; p++ {
		out[p] = uint8(av[(bits>>uint(3*p))&7])
	}
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
