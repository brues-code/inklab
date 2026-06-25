package datatools

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/Gophercraft/mpq"
)

// ClientFiles abstracts where the generators read client assets from: either
// loose extracted folders (dirSource) or, in-memory, the client's MPQ archives
// (mpqSource). Methods are category-specific so neither backend has to
// reconcile path schemes. Names are bare ("ItemSet.dbc", an icon base name, a
// WorldMap zone folder and file). All access is read-only.
type ClientFiles interface {
	ReadDBC(name string) ([]byte, error)        // DBFilesClient\<name>
	ListIcons() ([]string, error)               // Interface\Icons\*.blp -> base names (no ext)
	ReadIcon(base string) ([]byte, error)       // Interface\Icons\<base>.blp
	ListZones() ([]string, error)               // Interface\WorldMap\<zone> folder names
	ListZoneFiles(zone string) ([]string, error) // file names within a zone folder
	ReadZoneFile(zone, file string) ([]byte, error)
	Close() error
}

// ---- loose-folder backend ----

type dirSource struct {
	dbcDir      string
	iconsDir    string
	worldMapDir string
}

// NewDirSourceDBC builds a ClientFiles that reads DBCs from a loose
// DBFilesClient directory.
func NewDirSourceDBC(dbcDir string) ClientFiles { return &dirSource{dbcDir: dbcDir} }

// NewDirSourceIcons builds a ClientFiles that reads icon BLPs from a loose
// Icons directory.
func NewDirSourceIcons(iconsDir string) ClientFiles { return &dirSource{iconsDir: iconsDir} }

// NewDirSourceMaps builds a ClientFiles that reads WorldMap tiles from a loose
// WorldMap directory and WorldMapOverlay.dbc from dbcDir.
func NewDirSourceMaps(worldMapDir, dbcDir string) ClientFiles {
	return &dirSource{worldMapDir: worldMapDir, dbcDir: dbcDir}
}

func (d *dirSource) ReadDBC(name string) ([]byte, error) {
	return os.ReadFile(filepath.Join(d.dbcDir, name))
}

func (d *dirSource) ListIcons() ([]string, error) {
	ents, err := os.ReadDir(d.iconsDir)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, e := range ents {
		if !e.IsDir() && strings.EqualFold(filepath.Ext(e.Name()), ".blp") {
			out = append(out, strings.TrimSuffix(e.Name(), filepath.Ext(e.Name())))
		}
	}
	return out, nil
}

func (d *dirSource) ReadIcon(base string) ([]byte, error) {
	return os.ReadFile(filepath.Join(d.iconsDir, base+".blp"))
}

func (d *dirSource) ListZones() ([]string, error) {
	ents, err := os.ReadDir(d.worldMapDir)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, e := range ents {
		if e.IsDir() {
			out = append(out, e.Name())
		}
	}
	return out, nil
}

func (d *dirSource) ListZoneFiles(zone string) ([]string, error) {
	ents, err := os.ReadDir(filepath.Join(d.worldMapDir, zone))
	if err != nil {
		return nil, err
	}
	var out []string
	for _, e := range ents {
		if !e.IsDir() {
			out = append(out, e.Name())
		}
	}
	return out, nil
}

func (d *dirSource) ReadZoneFile(zone, file string) ([]byte, error) {
	return os.ReadFile(filepath.Join(d.worldMapDir, zone, file))
}

func (d *dirSource) Close() error { return nil }

// ---- in-memory MPQ backend ----

type mpqSource struct {
	set      *mpq.Set
	listOnce sync.Once
	entries  []string // combined listfile paths (original case)

	ovOnce   sync.Once
	ovByZone map[string][]string // lower(WorldMap folder name) -> overlay texture names
}

// NewMpqSource opens the client's MPQ archives under dataDir (e.g.
// "C:\WoW\Octo\Data") as a read-only patch-chained set. Archives are added base
// first, then patches in ascending order, so later patches override earlier
// ones (matching the client load order). The client directory is never written.
func NewMpqSource(dataDir string) (ClientFiles, error) {
	ents, err := os.ReadDir(dataDir)
	if err != nil {
		return nil, fmt.Errorf("read Data dir: %w", err)
	}
	var archives []string
	for _, e := range ents {
		if e.IsDir() || !strings.EqualFold(filepath.Ext(e.Name()), ".mpq") {
			continue
		}
		archives = append(archives, e.Name())
	}
	if len(archives) == 0 {
		return nil, fmt.Errorf("no .mpq archives in %s", dataDir)
	}
	sort.Slice(archives, func(i, j int) bool { return mpqPriority(archives[i]) < mpqPriority(archives[j]) })

	set := mpq.NewSet()
	added := 0
	for _, name := range archives {
		if err := set.Add(filepath.Join(dataDir, name)); err == nil {
			added++
		}
	}
	if added == 0 {
		return nil, fmt.Errorf("could not open any MPQ in %s", dataDir)
	}
	return &mpqSource{set: set}, nil
}

// mpqPriority orders archives low->high. Non-patch base archives sort first;
// "patch.MPQ" next; then "patch-2"..."patch-9", "patch-A"... by their suffix.
func mpqPriority(name string) int {
	base := strings.ToLower(strings.TrimSuffix(name, filepath.Ext(name)))
	if !strings.HasPrefix(base, "patch") {
		return 0
	}
	suffix := strings.TrimPrefix(base, "patch")
	suffix = strings.TrimPrefix(suffix, "-")
	if suffix == "" {
		return 1000 // plain patch.MPQ
	}
	// "2".."9","a","b"... -> rank by first rune.
	r := suffix[0]
	switch {
	case r >= '0' && r <= '9':
		return 1001 + int(r-'0')
	case r >= 'a' && r <= 'z':
		return 1011 + int(r-'a')
	default:
		return 1100
	}
}

func (m *mpqSource) read(path string) ([]byte, error) {
	f, err := m.set.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return io.ReadAll(f)
}

func (m *mpqSource) loadList() {
	m.listOnce.Do(func() {
		// The mpq lib prints non-fatal "(listfile)" scan warnings via
		// fmt.Println for the few archives whose listfile sector table it can't
		// fully read. Enumeration still succeeds from the other archives, so
		// silence that stdout noise for the duration of the scan.
		restore := silenceStdout()
		defer restore()

		list, err := m.set.List()
		if err != nil {
			return
		}
		defer list.Close()
		for list.Next() {
			m.entries = append(m.entries, list.Path())
		}
	})
}

// silenceStdout temporarily redirects os.Stdout to the null device, returning a
// function that restores it. Used to suppress a dependency's unconditional
// fmt.Println diagnostics during a tightly-scoped call.
func silenceStdout() func() {
	old := os.Stdout
	devnull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		return func() {}
	}
	os.Stdout = devnull
	return func() {
		os.Stdout = old
		_ = devnull.Close()
	}
}

func (m *mpqSource) ReadDBC(name string) ([]byte, error) {
	return m.read(`DBFilesClient\` + name)
}

func (m *mpqSource) ListIcons() ([]string, error) {
	m.loadList()
	const pre = `interface\icons\`
	seen := map[string]bool{}
	var out []string
	for _, p := range m.entries {
		lp := strings.ToLower(p)
		if !strings.HasPrefix(lp, pre) || !strings.HasSuffix(lp, ".blp") {
			continue
		}
		base := p[len(pre) : len(p)-len(".blp")]
		if strings.ContainsAny(base, `\/`) { // nested, not a direct icon
			continue
		}
		key := strings.ToLower(base)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, base)
	}
	return out, nil
}

func (m *mpqSource) ReadIcon(base string) ([]byte, error) {
	return m.read(`Interface\Icons\` + base + ".blp")
}

func (m *mpqSource) ListZones() ([]string, error) {
	m.loadList()
	seen := map[string]bool{}
	var out []string
	add := func(zone string) {
		if zone == "" {
			return
		}
		k := strings.ToLower(zone)
		if seen[k] {
			return
		}
		seen[k] = true
		out = append(out, zone)
	}

	// 1. WorldMap folders enumerated from the combined (listfile).
	const pre = `interface\worldmap\`
	for _, p := range m.entries {
		lp := strings.ToLower(p)
		if !strings.HasPrefix(lp, pre) {
			continue
		}
		rest := p[len(pre):]
		i := strings.IndexAny(rest, `\/`)
		if i <= 0 {
			continue // file directly under WorldMap, not a zone folder
		}
		add(rest[:i])
	}

	// 2. WorldMapArea.dbc area names. This is authoritative and readable by name
	//    via the hash table even when an archive's (listfile) failed to scan, so
	//    it surfaces zones the listfile pass missed (e.g. octo-custom zones like
	//    TimbermawTunnels added in a patch MPQ with an unreadable listfile). The
	//    caller already filters to zones that have a readable base tile.
	for _, z := range m.worldMapAreas() {
		add(z)
	}
	return out, nil
}

// worldMapAreas parses WorldMapArea.dbc and returns recordID -> AreaName (the
// WorldMap texture-folder name). Returns nil if the DBC is missing or malformed.
// Layout (1.12, 8 fields): id, mapID, areaID, areaName(3), locLeft, locRight,
// locTop, locBottom.
func (m *mpqSource) worldMapAreas() map[uint32]string {
	b, err := m.ReadDBC("WorldMapArea.dbc")
	if err != nil || len(b) < 20 || string(b[0:4]) != "WDBC" {
		return nil
	}
	rc := int(binary.LittleEndian.Uint32(b[4:8]))
	rs := int(binary.LittleEndian.Uint32(b[12:16]))
	if rc <= 0 || rs < 16 {
		return nil
	}
	sb := 20 + rc*rs
	out := make(map[uint32]string, rc)
	for r := 0; r < rc; r++ {
		rec := 20 + r*rs
		if rec+16 > len(b) {
			break
		}
		id := binary.LittleEndian.Uint32(b[rec : rec+4])
		off := sb + int(binary.LittleEndian.Uint32(b[rec+3*4:rec+4*4])) // AreaName
		if off < sb || off >= len(b) {
			continue
		}
		e := off
		for e < len(b) && b[e] != 0 {
			e++
		}
		if name := string(b[off:e]); name != "" {
			out[id] = name
		}
	}
	return out
}

// overlayTextures returns the WorldMapOverlay texture base names that belong to
// the given WorldMap folder, joining WorldMapOverlay.dbc (mapAreaID -> texture)
// with WorldMapArea.dbc (id -> folder name). Used to composite explored-area
// overlays for zones whose folder isn't in the (listfile), so ListZoneFiles can
// still surface them. Cached after first build.
func (m *mpqSource) overlayTextures(zone string) []string {
	m.ovOnce.Do(func() {
		m.ovByZone = map[string][]string{}
		areas := m.worldMapAreas()
		if len(areas) == 0 {
			return
		}
		b, err := m.ReadDBC("WorldMapOverlay.dbc")
		if err != nil || len(b) < 20 || string(b[0:4]) != "WDBC" {
			return
		}
		rc := int(binary.LittleEndian.Uint32(b[4:8]))
		rs := int(binary.LittleEndian.Uint32(b[12:16]))
		if rc <= 0 || rs < 9*4 {
			return
		}
		sb := 20 + rc*rs
		// Layout: id(0), mapAreaID(1), areaID[2..7], textureName(8), ...
		for r := 0; r < rc; r++ {
			rec := 20 + r*rs
			if rec+9*4 > len(b) {
				break
			}
			mapAreaID := binary.LittleEndian.Uint32(b[rec+1*4 : rec+2*4])
			off := sb + int(binary.LittleEndian.Uint32(b[rec+8*4:rec+9*4]))
			if off < sb || off >= len(b) {
				continue
			}
			e := off
			for e < len(b) && b[e] != 0 {
				e++
			}
			tex := string(b[off:e])
			zoneName, ok := areas[mapAreaID]
			if !ok || tex == "" {
				continue
			}
			k := strings.ToLower(zoneName)
			m.ovByZone[k] = append(m.ovByZone[k], tex)
		}
	})
	return m.ovByZone[strings.ToLower(zone)]
}

func (m *mpqSource) ListZoneFiles(zone string) ([]string, error) {
	m.loadList()
	pre := strings.ToLower(`interface\worldmap\` + zone + `\`)
	seen := map[string]bool{}
	var out []string
	for _, p := range m.entries {
		lp := strings.ToLower(p)
		if !strings.HasPrefix(lp, pre) {
			continue
		}
		file := p[len(pre):]
		if strings.ContainsAny(file, `\/`) {
			continue // nested deeper
		}
		if seen[strings.ToLower(file)] {
			continue
		}
		seen[strings.ToLower(file)] = true
		out = append(out, file)
	}

	// Fallback: this zone's folder isn't in the (listfile) (its archive's
	// listfile failed to scan), so the loop above found nothing. The base tiles
	// are read directly by name elsewhere; here we synthesize the overlay tile
	// names from WorldMapOverlay.dbc so explored-area overlays still composite.
	// One file per overlay base name is enough — stitchZone reads the rest by
	// name via the overlay's DBC dimensions.
	if len(out) == 0 {
		for _, tex := range m.overlayTextures(zone) {
			name := tex + "1.blp"
			if seen[strings.ToLower(name)] {
				continue
			}
			if _, err := m.ReadZoneFile(zone, name); err != nil {
				continue // overlay not actually present in this client
			}
			seen[strings.ToLower(name)] = true
			out = append(out, name)
		}
	}
	return out, nil
}

func (m *mpqSource) ReadZoneFile(zone, file string) ([]byte, error) {
	return m.read(`Interface\WorldMap\` + zone + `\` + file)
}

func (m *mpqSource) Close() error { return nil }
