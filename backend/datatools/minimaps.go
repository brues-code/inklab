package datatools

import (
	"bufio"
	"bytes"
	"fmt"
	"image"
	"math"
	"os"
	"path/filepath"
	"strings"
)

// Minimap generation: the in-game minimap's terrain art, assembled per zone.
//
// The client stores one 256x256 BLP tile per ADT block (533.33 yds) under
// textures\Minimap\, indexed by the same (gx,gy) block coordinates the terrain
// uses (gx = 32 - worldY/533.33, gy = 32 - worldX/533.33 — see adtarea.go). Most
// tiles are content-addressed: textures\Minimap\md5translate.trs maps a logical
// "<MapDir>\map<gx>_<gy>.blp" to a hashed filename at the minimap root. A few
// custom (octo-added) maps keep un-hashed tiles in textures\Minimap\<MapDir>\.
//
// GenerateMinimaps stitches each zone's blocks and crops to that zone's
// WorldMapArea rect — the SAME world rectangle the painted atlas maps cover — so
// spawn pins (positioned by percentage) land identically on either view. The
// frontend can then toggle a zone between its atlas painting and this terrain
// minimap.

const (
	minimapTilePx = 256
	// maxMinimapTiles caps a single zone's stitch. Real outdoor zones are well
	// under this; the continent-overview WorldMapArea rows (areaID 0) are skipped
	// before this, but the guard still stops any pathologically large rect from
	// allocating a huge canvas.
	maxMinimapTiles = 400
)

// GenerateMinimaps builds data/minimaps/<areaName>.jpg for every WorldMapArea
// zone that has world bounds and minimap tiles in the client. Reads from any
// ClientFiles source (loose client root via NewDirSourceClient, or the MPQ set).
func GenerateMinimaps(cf ClientFiles, outDir string, progress func(zone string, i, total int)) (*MapGenResult, error) {
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return nil, err
	}

	dirByMap, err := mapDirsByID(cf)
	if err != nil {
		return nil, fmt.Errorf("Map.dbc: %w", err)
	}
	idx := parseMinimapIndex(cf)

	wma, err := openDBCFrom(cf, "WorldMapArea.dbc")
	if err != nil {
		return nil, fmt.Errorf("WorldMapArea.dbc: %w", err)
	}

	res := &MapGenResult{}
	// Maps that have at least one WorldMapArea zone are covered by the per-zone
	// pass below; the rest (e.g. the development map) get a raw whole-map minimap.
	covered := map[uint32]bool{}
	for r := 0; r < wma.RecordCount; r++ {
		covered[wma.U32(r, 1)] = true
		areaID := wma.U32(r, 2)
		if areaID == 0 {
			continue // continent overview row — not a real zone, and enormous
		}
		name := strings.TrimSpace(wma.Str(r, 3))
		if name == "" {
			continue
		}
		if progress != nil {
			progress(name, r+1, wma.RecordCount)
		}
		dir := dirByMap[wma.U32(r, 1)]
		if dir == "" {
			res.Skipped++
			continue
		}
		// WorldMapArea loc bounds, matching genZones / the pin projection:
		//   x_max=F32(6) x_min=F32(7)  (worldX, vertical)
		//   y_max=F32(4) y_min=F32(5)  (worldY, horizontal)
		xMax, xMin := float64(wma.F32(r, 6)), float64(wma.F32(r, 7))
		yMax, yMin := float64(wma.F32(r, 4)), float64(wma.F32(r, 5))
		if xMax == 0 && xMin == 0 && yMax == 0 && yMin == 0 {
			res.Skipped++
			continue // instance / no world bounds
		}

		img, warn := stitchZoneMinimap(cf, idx.lookup, dir, xMin, xMax, yMin, yMax)
		if warn != "" {
			res.Skipped++
			res.Warnings = append(res.Warnings, fmt.Sprintf("%s: %s", name, warn))
			continue
		}
		if err := writeJPEG(filepath.Join(outDir, sanitizeMapName(name)+".jpg"), img); err != nil {
			res.Skipped++
			res.Warnings = append(res.Warnings, fmt.Sprintf("%s: write %v", name, err))
			continue
		}
		res.Generated++
	}

	// Raw whole-map minimaps for maps with tiles but no WorldMapArea zone (e.g.
	// the development map). These have no world bounds, so they can't be pin-
	// aligned — they're keyed by map id ("map_<id>.jpg") and surfaced as a
	// terrain fallback on pages whose spawns land on such a map.
	for mapID, dir := range dirByMap {
		if covered[mapID] {
			continue
		}
		tiles := idx.tilesByDir[strings.ToLower(dir)]
		if len(tiles) == 0 {
			continue
		}
		img := stitchRawMinimap(cf, idx.lookup, dir, tiles)
		if img == nil {
			res.Skipped++
			continue
		}
		if err := writeJPEG(filepath.Join(outDir, fmt.Sprintf("map_%d.jpg", mapID)), img); err != nil {
			res.Skipped++
			res.Warnings = append(res.Warnings, fmt.Sprintf("map_%d: write %v", mapID, err))
			continue
		}
		res.Generated++
	}
	return res, nil
}

// stitchRawMinimap stitches a no-bounds map's tiles into one image, cropped to
// the populated region. When that region's bounding box is mostly empty (the
// development map's tiles sit in two opposite grid corners), it falls back to the
// largest connected tile cluster so the result is filled terrain, not a vast
// black rect. Returns nil when nothing decodes.
func stitchRawMinimap(cf ClientFiles, lookup map[string]string, dir string, tiles [][2]int) *image.RGBA {
	gx0, gy0, gx1, gy1, ok := rawCropRegion(tiles)
	if !ok {
		return nil
	}
	w := (gx1 - gx0 + 1) * minimapTilePx
	h := (gy1 - gy0 + 1) * minimapTilePx
	canvas := image.NewRGBA(image.Rect(0, 0, w, h))
	found := 0
	for gy := gy0; gy <= gy1; gy++ {
		for gx := gx0; gx <= gx1; gx++ {
			tile, ok := readMinimapTile(cf, lookup, dir, gx, gy)
			if !ok {
				continue
			}
			found++
			blitAlpha(canvas, tile, (gx-gx0)*minimapTilePx, (gy-gy0)*minimapTilePx)
		}
	}
	if found == 0 {
		return nil
	}
	return canvas
}

// rawMaxBBoxTiles bounds a raw map's stitched bounding box; past it, the largest
// connected cluster is used instead (so a sparse two-corner layout doesn't yield
// a huge mostly-empty canvas).
const rawMaxBBoxTiles = 64

// rawCropRegion picks the grid rect to stitch for a no-bounds map: the bounding
// box of all populated tiles when it's compact enough, otherwise the bounding box
// of the largest 4-connected cluster.
func rawCropRegion(tiles [][2]int) (gx0, gy0, gx1, gy1 int, ok bool) {
	if len(tiles) == 0 {
		return 0, 0, 0, 0, false
	}
	bb := func(ts [][2]int) (int, int, int, int) {
		x0, y0, x1, y1 := ts[0][0], ts[0][1], ts[0][0], ts[0][1]
		for _, t := range ts {
			x0, y0 = min(x0, t[0]), min(y0, t[1])
			x1, y1 = max(x1, t[0]), max(y1, t[1])
		}
		return x0, y0, x1, y1
	}
	x0, y0, x1, y1 := bb(tiles)
	if (x1-x0+1)*(y1-y0+1) <= rawMaxBBoxTiles {
		return x0, y0, x1, y1, true
	}
	c := largestCluster(tiles)
	x0, y0, x1, y1 = bb(c)
	return x0, y0, x1, y1, true
}

// largestCluster returns the largest 4-connected group of tiles. Ties break
// toward the cluster with the smallest top-left coordinate for determinism.
func largestCluster(tiles [][2]int) [][2]int {
	set := make(map[[2]int]bool, len(tiles))
	for _, t := range tiles {
		set[t] = true
	}
	seen := map[[2]int]bool{}
	var best [][2]int
	for _, start := range tiles {
		if seen[start] {
			continue
		}
		var cluster [][2]int
		stack := [][2]int{start}
		seen[start] = true
		for len(stack) > 0 {
			t := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			cluster = append(cluster, t)
			for _, d := range [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
				n := [2]int{t[0] + d[0], t[1] + d[1]}
				if set[n] && !seen[n] {
					seen[n] = true
					stack = append(stack, n)
				}
			}
		}
		if len(cluster) > len(best) {
			best = cluster
		}
	}
	return best
}

// stitchZoneMinimap assembles the minimap blocks covering a zone's world rect and
// crops to it. The global minimap pixel space puts each ADT block at 256px with
// gpx(worldY)=(32-worldY/533.33)*256 (left->right as worldY falls) and
// gpy(worldX)=(32-worldX/533.33)*256 (top->bottom as worldX falls) — the same
// orientation as the atlas, so the result aligns with spawn pins. Returns a
// non-empty warning string instead of an image when nothing is produced.
func stitchZoneMinimap(cf ClientFiles, trs map[string]string, dir string, xMin, xMax, yMin, yMax float64) (*image.RGBA, string) {
	gpx := func(worldY float64) float64 { return (32.0 - worldY/adtTileSize) * minimapTilePx }
	gpy := func(worldX float64) float64 { return (32.0 - worldX/adtTileSize) * minimapTilePx }

	left := int(math.Round(gpx(yMax)))
	right := int(math.Round(gpx(yMin)))
	top := int(math.Round(gpy(xMax)))
	bot := int(math.Round(gpy(xMin)))
	w, h := right-left, bot-top
	if w <= 0 || h <= 0 {
		return nil, "degenerate bounds"
	}

	gx0, gx1 := floorDiv(left, minimapTilePx), floorDiv(right-1, minimapTilePx)
	gy0, gy1 := floorDiv(top, minimapTilePx), floorDiv(bot-1, minimapTilePx)
	if (gx1-gx0+1)*(gy1-gy0+1) > maxMinimapTiles {
		return nil, fmt.Sprintf("rect too large (%dx%d tiles)", gx1-gx0+1, gy1-gy0+1)
	}

	canvas := image.NewRGBA(image.Rect(0, 0, w, h))
	found := 0
	for gy := gy0; gy <= gy1; gy++ {
		for gx := gx0; gx <= gx1; gx++ {
			tile, ok := readMinimapTile(cf, trs, dir, gx, gy)
			if !ok {
				continue
			}
			found++
			// Opaque DXT1 tiles; blitAlpha is bounds-checked so edge tiles that
			// overhang the crop are clipped rather than panicking.
			blitAlpha(canvas, tile, gx*minimapTilePx-left, gy*minimapTilePx-top)
		}
	}
	if found == 0 {
		return nil, "no minimap tiles"
	}
	return canvas, ""
}

// readMinimapTile loads the (gx,gy) block for a map directory: the hashed tile
// named by md5translate.trs first, then an un-hashed textures\Minimap\<dir>\
// tile (octo's custom maps), decoding the BLP to RGBA.
func readMinimapTile(cf ClientFiles, trs map[string]string, dir string, gx, gy int) (*image.RGBA, bool) {
	if hash := trs[minimapKey(dir, gx, gy)]; hash != "" {
		if b, err := cf.ReadFile(`textures\Minimap\` + hash); err == nil {
			if img, err := decodeBLP2Bytes(b); err == nil {
				return img, true
			}
		}
	}
	direct := fmt.Sprintf(`textures\Minimap\%s\map%d_%d.blp`, dir, gx, gy)
	if b, err := cf.ReadFile(direct); err == nil {
		if img, err := decodeBLP2Bytes(b); err == nil {
			return img, true
		}
	}
	return nil, false
}

// minimapKey is the lookup key for the trs map: lowercased "<dir>\map<gx>_<gy>.blp".
func minimapKey(dir string, gx, gy int) string {
	return strings.ToLower(fmt.Sprintf(`%s\map%d_%d.blp`, dir, gx, gy))
}

// minimapIndex is the parsed md5translate.trs: a logical->hash lookup plus, per
// lowercased map directory, the (gx,gy) block coordinates that have tiles.
type minimapIndex struct {
	lookup     map[string]string // lower("<dir>\map<gx>_<gy>.blp") -> hashed filename
	tilesByDir map[string][][2]int
}

// parseMinimapIndex reads textures\Minimap\md5translate.trs. Lines are
// tab-separated (logical\thash); "dir:" header lines and blanks are ignored.
// Returns empty maps when the file is absent (clients with only un-hashed tiles
// still resolve via readMinimapTile's direct-path fallback, though such maps
// won't appear in tilesByDir).
func parseMinimapIndex(cf ClientFiles) minimapIndex {
	idx := minimapIndex{lookup: map[string]string{}, tilesByDir: map[string][][2]int{}}
	b, err := cf.ReadFile(`textures\Minimap\md5translate.trs`)
	if err != nil {
		return idx
	}
	sc := bufio.NewScanner(bytes.NewReader(b))
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := strings.TrimRight(sc.Text(), "\r")
		tab := strings.IndexByte(line, '\t')
		if tab <= 0 {
			continue
		}
		logical := strings.TrimSpace(line[:tab])
		hash := strings.TrimSpace(line[tab+1:])
		if logical == "" || hash == "" || !strings.Contains(strings.ToLower(logical), `\map`) {
			continue
		}
		idx.lookup[strings.ToLower(logical)] = hash
		if dir, gx, gy, ok := parseMinimapLogical(logical); ok {
			d := strings.ToLower(dir)
			idx.tilesByDir[d] = append(idx.tilesByDir[d], [2]int{gx, gy})
		}
	}
	return idx
}

// parseMinimapLogical splits a trs logical path "<dir>\map<gx>_<gy>.blp" into its
// directory and block coordinates. dir may itself contain backslashes.
func parseMinimapLogical(logical string) (dir string, gx, gy int, ok bool) {
	slash := strings.LastIndexByte(logical, '\\')
	if slash < 0 {
		return "", 0, 0, false
	}
	dir, file := logical[:slash], logical[slash+1:]
	if _, err := fmt.Sscanf(strings.ToLower(file), "map%d_%d.blp", &gx, &gy); err != nil {
		return "", 0, 0, false
	}
	return dir, gx, gy, true
}

// mapDirsByID parses Map.dbc into mapID -> directory name (the minimap/ADT folder).
func mapDirsByID(cf ClientFiles) (map[uint32]string, error) {
	d, err := openDBCFrom(cf, "Map.dbc")
	if err != nil {
		return nil, err
	}
	out := make(map[uint32]string, d.RecordCount)
	for r := 0; r < d.RecordCount; r++ {
		if dir := d.Str(r, 1); dir != "" {
			out[d.U32(r, 0)] = dir
		}
	}
	return out, nil
}

// floorDiv is integer floor division that handles negative numerators (a block
// index can be negative for far-north/west bounds before clamping by the canvas).
func floorDiv(a, b int) int {
	q := a / b
	if (a%b != 0) && ((a < 0) != (b < 0)) {
		q--
	}
	return q
}

// sanitizeMapName strips path separators from a WorldMapArea name so it's a safe
// single-segment filename (names like "Zul'gurub" keep their apostrophe).
func sanitizeMapName(s string) string {
	return strings.NewReplacer(`\`, "", "/", "").Replace(s)
}
