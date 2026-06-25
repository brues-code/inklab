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
	ReadFile(path string) ([]byte, error)       // arbitrary client path (models, textures)
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
	rootDir     string // client root, for arbitrary ReadFile paths
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

// NewDirSourceClient builds a ClientFiles rooted at a loose client base dir, able
// to read both DBCs (from baseDir/DBFilesClient) and arbitrary client paths such
// as World\Maps terrain ADTs. Used by the area-grid generator.
func NewDirSourceClient(baseDir string) ClientFiles {
	return &dirSource{rootDir: baseDir, dbcDir: filepath.Join(baseDir, "DBFilesClient")}
}

func (d *dirSource) ReadDBC(name string) ([]byte, error) {
	return os.ReadFile(filepath.Join(d.dbcDir, name))
}

// ReadFile resolves an arbitrary client path against the root dir. Backslashes
// (client convention) are normalized to the OS separator.
func (d *dirSource) ReadFile(path string) ([]byte, error) {
	if d.rootDir == "" {
		return nil, fmt.Errorf("dirSource has no root dir for %q", path)
	}
	p := strings.ReplaceAll(path, `\`, string(filepath.Separator))
	return os.ReadFile(filepath.Join(d.rootDir, p))
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

	iconOnce  sync.Once
	iconNames []string // icon base names recovered from SpellIcon/ItemDisplayInfo DBCs
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

// ReadFile reads an arbitrary client path from the MPQ set (by name, via the
// hash table — works regardless of listfile health).
func (m *mpqSource) ReadFile(path string) ([]byte, error) {
	return m.read(path)
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

	// Augment with icon names referenced by SpellIcon.dbc / ItemDisplayInfo.dbc.
	// Custom icons added in a patch archive whose (listfile) failed to scan
	// (e.g. octo's inv_misc_octopuss_purple) are readable by name via the hash
	// table but absent from the enumeration above, so they never get extracted.
	// Recover them from those DBCs, verifying each is actually present in this
	// client before adding it (so the generator's skip count stays honest).
	for _, base := range m.dbcIconNames() {
		key := strings.ToLower(base)
		if seen[key] {
			continue
		}
		if _, err := m.ReadIcon(base); err != nil {
			continue // referenced but not present in this client
		}
		seen[key] = true
		out = append(out, base)
	}
	return out, nil
}

// dbcIconNames recovers icon base names referenced by SpellIcon.dbc (full
// "Interface\Icons\<name>" texture paths) and ItemDisplayInfo.dbc (bare
// inventoryIcon names). Both DBCs are readable by name even when a patch
// archive's (listfile) can't be scanned, so they surface custom icons the
// listfile enumeration misses. Cached after first build.
func (m *mpqSource) dbcIconNames() []string {
	m.iconOnce.Do(func() {
		seen := map[string]bool{}
		add := func(s string) {
			// SpellIcon stores a full path; ItemDisplayInfo stores a bare name.
			if i := strings.LastIndexAny(s, `\/`); i >= 0 {
				s = s[i+1:]
			}
			s = strings.TrimSuffix(s, filepath.Ext(s))
			if s == "" {
				return
			}
			k := strings.ToLower(s)
			if seen[k] {
				return
			}
			seen[k] = true
			m.iconNames = append(m.iconNames, s)
		}
		m.dbcStringField("SpellIcon.dbc", []int{1}, add)         // field 1: texture path
		m.dbcStringField("ItemDisplayInfo.dbc", []int{5, 6}, add) // fields 5,6: inventoryIcon[0,1]
	})
	return m.iconNames
}

// dbcStringField invokes add with the string value of each of the given field
// indices for every record in a DBC. Empty / out-of-range strings are skipped.
func (m *mpqSource) dbcStringField(name string, fields []int, add func(string)) {
	b, err := m.ReadDBC(name)
	if err != nil || len(b) < 20 || string(b[0:4]) != "WDBC" {
		return
	}
	rc := int(binary.LittleEndian.Uint32(b[4:8]))
	fc := int(binary.LittleEndian.Uint32(b[8:12]))
	rs := int(binary.LittleEndian.Uint32(b[12:16]))
	if rc <= 0 || fc <= 0 || rs < fc*4 {
		return
	}
	sb := 20 + rc*rs
	for r := 0; r < rc; r++ {
		rec := 20 + r*rs
		for _, f := range fields {
			if f < 0 || f >= fc {
				continue
			}
			off := rec + f*4
			if off+4 > len(b) {
				continue
			}
			so := sb + int(binary.LittleEndian.Uint32(b[off:off+4]))
			if so < sb || so >= len(b) {
				continue
			}
			e := so
			for e < len(b) && b[e] != 0 {
				e++
			}
			if s := string(b[so:e]); s != "" {
				add(s)
			}
		}
	}
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

	// Augment with WorldMapOverlay.dbc overlays. Some overlay tiles (often
	// octo-added sub-areas like Wetlands' DunAgrath / HawksVigil) live in a
	// patch archive whose (listfile) failed to scan, so they're absent from the
	// enumeration above even though they're readable by name. Without this their
	// explored-area overlay never composites, leaving a blank spot on the map.
	// One file per overlay base name is enough — stitchZone reads the remaining
	// tiles by name. Already-listed overlays are skipped via `seen`.
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
	return out, nil
}

func (m *mpqSource) ReadZoneFile(zone, file string) ([]byte, error) {
	return m.read(`Interface\WorldMap\` + zone + `\` + file)
}

func (m *mpqSource) Close() error { return nil }
