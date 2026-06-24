package main

import (
	"database/sql"
	"fmt"
	"hash/fnv"
	"os"
	"os/exec"
	"path/filepath"
	"sort"

	_ "modernc.org/sqlite"
)

// WhatsNewEntry is a single added/changed row, tagged with the entity type so
// the UI can link to it.
type WhatsNewEntry struct {
	Type   string `json:"type"` // item | npc | object | quest | spell
	ID     int64  `json:"id"`
	Name   string `json:"name"`
	Change string `json:"change"` // added | changed
}

// WhatsNewGroup buckets entries by entity type.
type WhatsNewGroup struct {
	Type    string          `json:"type"`
	Label   string          `json:"label"`
	Added   int             `json:"added"`
	Changed int             `json:"changed"`
	Entries []WhatsNewEntry `json:"entries"`
}

// WhatsNewReport is the result of diffing the live DB against a baseline.
type WhatsNewReport struct {
	Baseline string          `json:"baseline"` // human description of what we diffed against
	Groups   []WhatsNewGroup `json:"groups"`
	Error    string          `json:"error,omitempty"`
}

type diffSpec struct {
	table   string
	pk      string
	nameCol string
	typ     string
	label   string
}

var whatsNewTables = []diffSpec{
	{"item_template", "entry", "name", "item", "Items"},
	{"creature_template", "entry", "name", "npc", "NPCs"},
	{"gameobject_template", "entry", "name", "object", "Objects"},
	{"quest_template", "entry", "Title", "quest", "Quests"},
	{"spell_template", "entry", "name", "spell", "Spells"},
}

// perTypeCap limits how many entries we return per type so a huge first-import
// diff stays manageable in the UI.
const perTypeCap = 200

// WhatsNew compares the live data/inklab.db against a baseline (the last
// committed copy via `git show HEAD:data/inklab.db`, falling back to the
// database embedded in the binary) and reports the rows added or changed since.
func (a *App) WhatsNew() *WhatsNewReport {
	livePath := filepath.Join(a.DataDir, "inklab.db")

	baselinePath, desc, cleanup, err := a.baselineDB()
	if err != nil {
		return &WhatsNewReport{Error: "could not obtain a baseline to compare against: " + err.Error()}
	}
	defer cleanup()

	oldDB, err := sql.Open("sqlite", baselinePath+"?mode=ro")
	if err != nil {
		return &WhatsNewReport{Error: "open baseline: " + err.Error()}
	}
	defer oldDB.Close()
	newDB, err := sql.Open("sqlite", livePath+"?mode=ro")
	if err != nil {
		return &WhatsNewReport{Error: "open live db: " + err.Error()}
	}
	defer newDB.Close()

	rep := &WhatsNewReport{Baseline: desc}
	for _, spec := range whatsNewTables {
		g, err := diffTable(oldDB, newDB, spec)
		if err != nil {
			continue // table may not exist in one of them; skip quietly
		}
		if g.Added > 0 || g.Changed > 0 {
			rep.Groups = append(rep.Groups, g)
		}
	}
	return rep
}

// baselineDB returns a path to a baseline copy of the DB to diff against, a
// human description, and a cleanup func. Prefers the last git-committed copy;
// falls back to the embedded database.
func (a *App) baselineDB() (path, desc string, cleanup func(), err error) {
	noop := func() {}

	// Prefer the last committed inklab.db (matches the user's mental model of
	// "what changed since I last committed the DB").
	cmd := exec.Command("git", "show", "HEAD:data/inklab.db")
	cmd.Dir = repoRoot(a.DataDir)
	if out, gerr := cmd.Output(); gerr == nil && len(out) > 0 {
		tmp, terr := os.CreateTemp("", "inklab-baseline-*.db")
		if terr == nil {
			if _, werr := tmp.Write(out); werr == nil {
				tmp.Close()
				return tmp.Name(), "last committed database (git HEAD)", func() { os.Remove(tmp.Name()) }, nil
			}
			tmp.Close()
			os.Remove(tmp.Name())
		}
	}

	// Fallback: the database embedded in the binary at build time.
	if len(embeddedDB) > 0 {
		tmp, terr := os.CreateTemp("", "inklab-baseline-*.db")
		if terr != nil {
			return "", "", noop, terr
		}
		if _, werr := tmp.Write(embeddedDB); werr != nil {
			tmp.Close()
			os.Remove(tmp.Name())
			return "", "", noop, werr
		}
		tmp.Close()
		return tmp.Name(), "bundled baseline database (last build)", func() { os.Remove(tmp.Name()) }, nil
	}

	return "", "", noop, fmt.Errorf("no git copy and no embedded database")
}

// repoRoot returns the repo root given the data dir (data/ lives at the root).
func repoRoot(dataDir string) string {
	return filepath.Dir(dataDir)
}

// diffTable diffs one table between two DBs by primary key, hashing every column
// to detect content changes.
func diffTable(oldDB, newDB *sql.DB, spec diffSpec) (WhatsNewGroup, error) {
	g := WhatsNewGroup{Type: spec.typ, Label: spec.label}

	oldHashes, err := rowHashes(oldDB, spec)
	if err != nil {
		return g, err
	}
	// Iterate the new DB, classifying each row.
	rows, err := newDB.Query(fmt.Sprintf("SELECT * FROM %s", spec.table))
	if err != nil {
		return g, err
	}
	defer rows.Close()
	cols, err := rows.Columns()
	if err != nil {
		return g, err
	}
	pkIdx, nameIdx := indexOf(cols, spec.pk), indexOf(cols, spec.nameCol)
	if pkIdx < 0 {
		return g, fmt.Errorf("no pk column %s", spec.pk)
	}

	var added, changed []WhatsNewEntry
	for rows.Next() {
		raw := make([]sql.RawBytes, len(cols))
		ptrs := make([]interface{}, len(cols))
		for i := range raw {
			ptrs[i] = &raw[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			continue
		}
		id := bytesToInt64(raw[pkIdx])
		name := ""
		if nameIdx >= 0 {
			name = string(raw[nameIdx])
		}
		h := hashRow(raw)
		oldH, existed := oldHashes[id]
		switch {
		case !existed:
			added = append(added, WhatsNewEntry{spec.typ, id, name, "added"})
		case oldH != h:
			changed = append(changed, WhatsNewEntry{spec.typ, id, name, "changed"})
		}
	}

	g.Added = len(added)
	g.Changed = len(changed)
	// Show added first, then changed; cap the combined list.
	sort.Slice(added, func(i, j int) bool { return added[i].ID < added[j].ID })
	sort.Slice(changed, func(i, j int) bool { return changed[i].ID < changed[j].ID })
	g.Entries = append(g.Entries, added...)
	g.Entries = append(g.Entries, changed...)
	if len(g.Entries) > perTypeCap {
		g.Entries = g.Entries[:perTypeCap]
	}
	return g, nil
}

// rowHashes builds pk -> full-row hash for a table.
func rowHashes(db *sql.DB, spec diffSpec) (map[int64]uint64, error) {
	rows, err := db.Query(fmt.Sprintf("SELECT * FROM %s", spec.table))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	pkIdx := indexOf(cols, spec.pk)
	if pkIdx < 0 {
		return nil, fmt.Errorf("no pk column %s", spec.pk)
	}
	out := make(map[int64]uint64, 16384)
	for rows.Next() {
		raw := make([]sql.RawBytes, len(cols))
		ptrs := make([]interface{}, len(cols))
		for i := range raw {
			ptrs[i] = &raw[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			continue
		}
		out[bytesToInt64(raw[pkIdx])] = hashRow(raw)
	}
	return out, nil
}

func hashRow(raw []sql.RawBytes) uint64 {
	h := fnv.New64a()
	for _, c := range raw {
		h.Write(c)
		h.Write([]byte{0})
	}
	return h.Sum64()
}

func indexOf(s []string, v string) int {
	for i, x := range s {
		if x == v {
			return i
		}
	}
	return -1
}

func bytesToInt64(b []byte) int64 {
	var n int64
	neg := false
	for i, c := range b {
		if i == 0 && c == '-' {
			neg = true
			continue
		}
		if c < '0' || c > '9' {
			break
		}
		n = n*10 + int64(c-'0')
	}
	if neg {
		n = -n
	}
	return n
}
