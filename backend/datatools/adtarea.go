package datatools

import (
	"encoding/binary"
	"fmt"
	"math"
	"os"
)

// Area-grid resolution: the in-game zone of a world coordinate, read straight from
// the client's terrain. Each ADT tile holds a 16x16 grid of MCNK chunks, and every
// chunk header carries the AreaTable.dbc id of the area it sits in. Walking that
// area's parent chain yields the top-level zone — exactly how the game (and Octo)
// assign a spawn to a zone. This is precise where the axis-aligned zone bounding
// boxes overlap (e.g. the Barrens vs Dustwallow Marsh), which smallest-area box
// selection gets wrong.
//
// GenerateAreaGrid bakes this into a compact binary (data/area_grid.bin) so runtime
// resolution needs only that file plus zones.json — no MPQ access during sync.

const (
	adtTileSize  = 533.33333    // yards per ADT tile
	adtChunkSize = adtTileSize / 16
	areaGridMagic = "IAG1"
)

// areaGridTile is one ADT tile's resolved top-zone areatableIDs, indexed iy*16+ix.
type areaGridTile struct {
	mapID  uint16
	gx, gy uint8
	zones  [256]uint16
}

// AreaGrid is the loaded grid: (mapID,gx,gy) -> 256 top-zone areatableIDs.
type AreaGrid struct {
	tiles map[uint32][256]uint16
}

func tileKey(mapID, gx, gy int) uint32 {
	return uint32(mapID)<<16 | uint32(gx)<<8 | uint32(gy)
}

// GenerateAreaGrid scans every terrain map's ADT tiles, resolves each chunk's
// AreaTable id to its top-level zone, and writes the binary grid to outPath.
// Maps that are WMO-based (instances, no ADT terrain) contribute no tiles.
func GenerateAreaGrid(cf ClientFiles, outPath string, progress func(mapName string, i, total int)) error {
	at, err := openDBCFrom(cf, "AreaTable.dbc")
	if err != nil {
		return fmt.Errorf("AreaTable.dbc: %w", err)
	}
	parent := make(map[uint32]uint32, at.RecordCount)
	for r := 0; r < at.RecordCount; r++ {
		parent[at.U32(r, 0)] = at.U32(r, 2)
	}
	topZone := func(area uint32) uint32 {
		for i := 0; i < 16; i++ {
			p, ok := parent[area]
			if !ok || p == 0 {
				break
			}
			area = p
		}
		return area
	}

	mp, err := openDBCFrom(cf, "Map.dbc")
	if err != nil {
		return fmt.Errorf("Map.dbc: %w", err)
	}
	type mapDir struct {
		id  int
		dir string
	}
	var maps []mapDir
	for r := 0; r < mp.RecordCount; r++ {
		if dir := mp.Str(r, 1); dir != "" {
			maps = append(maps, mapDir{id: int(mp.U32(r, 0)), dir: dir})
		}
	}

	var tiles []areaGridTile
	var totListed, totRead, totParse, totOK int
	for i, m := range maps {
		if progress != nil {
			progress(m.dir, i+1, len(maps))
		}
		present := wdtTiles(cf, m.dir)
		var readFail, parseFail, ok int
		var firstErr string
		for _, t := range present {
			path := fmt.Sprintf(`World\Maps\%s\%s_%d_%d.adt`, m.dir, m.dir, t.gx, t.gy)
			b, err := cf.ReadFile(path)
			if err != nil {
				readFail++
				if firstErr == "" {
					firstErr = fmt.Sprintf("%s: %v", path, err)
				}
				continue
			}
			zones, parsed := tileZones(b, topZone)
			if !parsed {
				parseFail++
				continue
			}
			ok++
			tiles = append(tiles, areaGridTile{mapID: uint16(m.id), gx: uint8(t.gx), gy: uint8(t.gy), zones: zones})
		}
		totListed += len(present)
		totRead += readFail
		totParse += parseFail
		totOK += ok
		// Only report maps that actually have terrain tiles, and flag any drops.
		if len(present) > 0 && (readFail > 0 || parseFail > 0) {
			fmt.Printf("[area-grid] map %-20s (id %d): %d listed, %d ok, %d read-fail, %d parse-fail; first read-fail: %s\n",
				m.dir, m.id, len(present), ok, readFail, parseFail, firstErr)
		}
	}
	fmt.Printf("[area-grid] TOTAL: %d tiles listed in WDTs, %d written; %d read failures, %d parse failures\n",
		totListed, totOK, totRead, totParse)

	return writeAreaGrid(outPath, tiles)
}

type tileXY struct{ gx, gy int }

// wdtTiles parses a map's WDT MAIN chunk for the (gx,gy) tiles that have ADTs.
// Falls back to brute-force probing all 64x64 if the WDT is missing/odd.
func wdtTiles(cf ClientFiles, dir string) []tileXY {
	b, err := cf.ReadFile(fmt.Sprintf(`World\Maps\%s\%s.wdt`, dir, dir))
	if err != nil {
		return nil
	}
	// Locate MAIN (stored reversed as "NIAM"): 64*64 entries of 8 bytes
	// (flags uint32, asyncId uint32). flags&1 => tile present. Index = gy*64+gx.
	for i := 0; i+8 < len(b); {
		magic := string(b[i : i+4])
		size := binary.LittleEndian.Uint32(b[i+4 : i+8])
		if magic == "NIAM" {
			data := b[i+8:]
			var out []tileXY
			for idx := 0; idx < 64*64; idx++ {
				o := idx * 8
				if o+4 > len(data) {
					break
				}
				if binary.LittleEndian.Uint32(data[o:o+4])&1 != 0 {
					out = append(out, tileXY{gx: idx % 64, gy: idx / 64})
				}
			}
			return out
		}
		i += 8 + int(size)
		if size == 0 {
			i++
		}
	}
	return nil
}

// tileZones reads an ADT's MCNK chunks and returns each chunk's top-level zone id,
// indexed by iy*16+ix from the chunk's own IndexX/IndexY header fields.
func tileZones(b []byte, topZone func(uint32) uint32) ([256]uint16, bool) {
	var out [256]uint16
	any := false
	for i := 0; i+8 < len(b); {
		magic := string(b[i : i+4])
		size := binary.LittleEndian.Uint32(b[i+4 : i+8])
		if magic == "KNCM" && i+8+0x38 <= len(b) { // MCNK header (>=0x38 for areaid)
			h := i + 8
			ix := binary.LittleEndian.Uint32(b[h+0x04 : h+0x08])
			iy := binary.LittleEndian.Uint32(b[h+0x08 : h+0x0C])
			area := binary.LittleEndian.Uint32(b[h+0x34 : h+0x38])
			if ix < 16 && iy < 16 && area != 0 {
				out[iy*16+ix] = uint16(topZone(area))
				any = true
			}
		}
		i += 8 + int(size)
		if size == 0 {
			i++
		}
	}
	return out, any
}

func writeAreaGrid(path string, tiles []areaGridTile) error {
	buf := make([]byte, 0, 8+len(tiles)*(4+512))
	buf = append(buf, areaGridMagic...)
	var u32 [4]byte
	binary.LittleEndian.PutUint32(u32[:], uint32(len(tiles)))
	buf = append(buf, u32[:]...)
	for _, t := range tiles {
		var hdr [4]byte
		binary.LittleEndian.PutUint16(hdr[0:2], t.mapID)
		hdr[2] = t.gx
		hdr[3] = t.gy
		buf = append(buf, hdr[:]...)
		var z [512]byte
		for j := 0; j < 256; j++ {
			binary.LittleEndian.PutUint16(z[j*2:j*2+2], t.zones[j])
		}
		buf = append(buf, z[:]...)
	}
	return os.WriteFile(path, buf, 0644)
}

// LoadAreaGrid reads a grid written by GenerateAreaGrid. Returns nil (no error)
// when the file is absent, so callers can fall back to bounding-box resolution.
func LoadAreaGrid(path string) (*AreaGrid, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	if len(b) < 8 || string(b[0:4]) != areaGridMagic {
		return nil, fmt.Errorf("area grid: bad header")
	}
	count := int(binary.LittleEndian.Uint32(b[4:8]))
	g := &AreaGrid{tiles: make(map[uint32][256]uint16, count)}
	off := 8
	for i := 0; i < count; i++ {
		if off+4+512 > len(b) {
			return nil, fmt.Errorf("area grid: truncated at tile %d", i)
		}
		mapID := binary.LittleEndian.Uint16(b[off : off+2])
		gx := b[off+2]
		gy := b[off+3]
		off += 4
		var z [256]uint16
		for j := 0; j < 256; j++ {
			z[j] = binary.LittleEndian.Uint16(b[off+j*2 : off+j*2+2])
		}
		off += 512
		g.tiles[tileKey(int(mapID), int(gx), int(gy))] = z
	}
	return g, nil
}

// ZoneAt returns the top-level zone areatableID for a world coordinate, or ok=false
// if the tile/chunk has no area data (ocean, missing tile, WMO-only map).
func (g *AreaGrid) ZoneAt(mapID int, worldX, worldY float64) (areatableID uint32, ok bool) {
	if g == nil {
		return 0, false
	}
	gx := int(32.0 - worldY/adtTileSize)
	gy := int(32.0 - worldX/adtTileSize)
	if gx < 0 || gx > 63 || gy < 0 || gy > 63 {
		return 0, false
	}
	tile, found := g.tiles[tileKey(mapID, gx, gy)]
	if !found {
		return 0, false
	}
	ix := int(math.Floor((float64(32-gx)*adtTileSize - worldY) / adtChunkSize))
	iy := int(math.Floor((float64(32-gy)*adtTileSize - worldX) / adtChunkSize))
	if ix < 0 || ix > 15 || iy < 0 || iy > 15 {
		return 0, false
	}
	z := tile[iy*16+ix]
	if z == 0 {
		return 0, false
	}
	return uint32(z), true
}

// TileCount reports how many tiles the grid holds (diagnostics).
func (g *AreaGrid) TileCount() int {
	if g == nil {
		return 0
	}
	return len(g.tiles)
}
