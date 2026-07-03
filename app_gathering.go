package main

import (
	"fmt"
	"sort"
)

// The gathering map: type-3 gameobjects (veins, herbs, chests) whose lock
// (gameobject_template.data0 -> locks) demands a skill (lock slot type 2),
// joined with their spawn points. The "profession" is the lock TYPE
// (LockType.dbc: Herbalism, Mining, Pick Lock, and custom lines like
// Survival), so custom client data appears automatically.

// gatherReqExprFor resolves the required skill for the lock type given by
// expr, across the five lock slots (slot type 2 = requires skill, prop =
// lock type id).
func gatherReqExprFor(expr string) string {
	s := "CASE "
	for i := 1; i <= 5; i++ {
		s += fmt.Sprintf("WHEN l.type%d = 2 AND l.prop%d = %s THEN l.req%d ", i, i, expr, i)
	}
	return s + "END"
}

// GatheringProfession is one gathering skill in the sidebar picker.
type GatheringProfession struct {
	ID     int    `json:"id"` // LockType id
	Name   string `json:"name"`
	Nodes  int    `json:"nodes"`  // distinct node names
	Spawns int    `json:"spawns"` // total known spawn points
}

// GetGatheringProfessions lists the lock types worth a gathering map: ones
// with real skill requirements (not "anyone can open") and enough known
// spawns to plot.
func (a *App) GetGatheringProfessions() []GatheringProfession {
	out := []GatheringProfession{}
	rows, err := a.db.DB().Query(`
		SELECT lt.id, lt.name, COUNT(DISTINCT gt.name),
		       MAX(` + gatherReqExprFor("lt.id") + `),
		       (SELECT COUNT(*) FROM gameobject_spawn gs
		        JOIN gameobject_template gt2 ON gt2.entry = gs.gameobject_entry
		        JOIN locks l ON l.id = gt2.data0
		        WHERE gt2.type = 3 AND ` + gatherLockPred("lt.id") + `)
		FROM lock_types lt
		JOIN locks l ON ` + gatherLockPred("lt.id") + `
		JOIN gameobject_template gt ON gt.data0 = l.id AND gt.type = 3
		GROUP BY lt.id
		ORDER BY 5 DESC`)
	if err != nil {
		fmt.Printf("[API] GetGatheringProfessions: %v\n", err)
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var p GatheringProfession
		var maxReq int
		if rows.Scan(&p.ID, &p.Name, &p.Nodes, &maxReq, &p.Spawns) != nil {
			continue
		}
		// Real gathering lines demand skill somewhere and have enough spawns
		// to plot; "Open"/"Open Kneeling" quest objects (req 0) drop out.
		if maxReq >= 25 && p.Spawns >= 20 {
			out = append(out, p)
		}
	}
	return out
}

// gatherLockPred builds the "this lock demands skill <lockTypeExpr>" predicate
// over the five lock slots.
func gatherLockPred(lockTypeExpr string) string {
	pred := ""
	for i := 1; i <= 5; i++ {
		if pred != "" {
			pred += " OR "
		}
		pred += fmt.Sprintf("(l.type%d = 2 AND l.prop%d = %s)", i, i, lockTypeExpr)
	}
	return "(" + pred + ")"
}

// GatheringSpawn is one plotted node spawn (zone-map percentages).
type GatheringSpawn struct {
	Zone string  `json:"zone"`
	X    float64 `json:"x"`
	Y    float64 `json:"y"`
}

// GatheringNode is one node type (e.g. "Tin Vein") with its skill requirement
// and every known spawn point.
type GatheringNode struct {
	Name     string           `json:"name"`
	ReqSkill int              `json:"reqSkill"`
	Entries  []int            `json:"entries"`
	Spawns   []GatheringSpawn `json:"spawns"`
}

// GetGatheringNodes returns every node of a gathering lock type, grouped by
// name + required skill (a node name often spans several gameobject entries),
// with all known spawn points.
func (a *App) GetGatheringNodes(lockType int) []GatheringNode {
	out := []GatheringNode{}
	db := a.db.DB()

	type key struct {
		name string
		req  int
	}
	byKey := map[key]*GatheringNode{}
	entryNode := map[int]*GatheringNode{}

	rows, err := db.Query(`
		SELECT gt.entry, gt.name, `+gatherReqExprFor("?1")+`
		FROM gameobject_template gt
		JOIN locks l ON l.id = gt.data0
		WHERE gt.type = 3 AND `+gatherLockPred("?1"), lockType)
	if err != nil {
		fmt.Printf("[API] GetGatheringNodes(%d): %v\n", lockType, err)
		return out
	}
	for rows.Next() {
		var entry, req int
		var name string
		if rows.Scan(&entry, &name, &req) != nil {
			continue
		}
		k := key{name, req}
		n := byKey[k]
		if n == nil {
			n = &GatheringNode{Name: name, ReqSkill: req, Spawns: []GatheringSpawn{}}
			byKey[k] = n
		}
		n.Entries = append(n.Entries, entry)
		entryNode[entry] = n
	}
	rows.Close()
	if len(entryNode) == 0 {
		return out
	}

	// All spawns for those entries in one query.
	ph := ""
	args := make([]any, 0, len(entryNode))
	for e := range entryNode {
		if ph != "" {
			ph += ","
		}
		ph += "?"
		args = append(args, e)
	}
	sRows, err := db.Query(`
		SELECT gameobject_entry, COALESCE(zone_name, ''), position_x, position_y
		FROM gameobject_spawn
		WHERE gameobject_entry IN (`+ph+`)`, args...)
	if err == nil {
		for sRows.Next() {
			var entry int
			var s GatheringSpawn
			if sRows.Scan(&entry, &s.Zone, &s.X, &s.Y) != nil {
				continue
			}
			if s.Zone == "" || s.X <= 0 || s.X > 100 || s.Y <= 0 || s.Y > 100 {
				continue
			}
			if n := entryNode[entry]; n != nil {
				n.Spawns = append(n.Spawns, s)
			}
		}
		sRows.Close()
	}

	for _, n := range byKey {
		if len(n.Spawns) > 0 {
			out = append(out, *n)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].ReqSkill != out[j].ReqSkill {
			return out[i].ReqSkill < out[j].ReqSkill
		}
		return out[i].Name < out[j].Name
	})
	return out
}
