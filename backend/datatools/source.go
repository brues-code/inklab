package datatools

import (
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
	const pre = `interface\worldmap\`
	seen := map[string]bool{}
	var out []string
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
		zone := rest[:i]
		if seen[strings.ToLower(zone)] {
			continue
		}
		seen[strings.ToLower(zone)] = true
		out = append(out, zone)
	}
	return out, nil
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
	return out, nil
}

func (m *mpqSource) ReadZoneFile(zone, file string) ([]byte, error) {
	return m.read(`Interface\WorldMap\` + zone + `\` + file)
}

func (m *mpqSource) Close() error { return nil }
