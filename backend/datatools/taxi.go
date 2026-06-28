package datatools

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// TaxiNodes.dbc (16 fields): id(0), mapID(1), x(2), y(3), z(4), name[8](5-12),
// nameFlags(13), hordeMountID(14), allianceMountID(15).
// TaxiPath.dbc (4 fields): id(0), fromNode(1), toNode(2), cost(3).
// TaxiPathNode.dbc (9 fields): id(0), pathID(1), index(2), mapID(3), x(4), y(5), z(6),...
// WorldMapArea.dbc continent extent: areaID(2)==0 row per map carries the
// continent's world bounds — y_max(4), y_min(5), x_max(6), x_min(7).
//
// World->map% (matches WoW: +X is North=up, +Y is West=left):
//   px = (y_max - worldY) / (y_max - y_min)   [0 left .. 1 right]
//   py = (x_max - worldX) / (x_max - x_min)   [0 top  .. 1 bottom]

// TaxiContinent is a map that has flight nodes and a continent overview image.
type TaxiContinent struct {
	MapID int    `json:"mapId"`
	Name  string `json:"name"` // WorldMapArea continent area name == map image key
}

// TaxiNodeOut is a flight master, positioned as a percentage on its continent
// map. Alliance/Horde availability comes from the per-faction mount creature ids.
type TaxiNodeOut struct {
	ID       int     `json:"id"`
	MapID    int     `json:"mapId"`
	Name     string  `json:"name"`
	Alliance bool    `json:"alliance"`
	Horde    bool    `json:"horde"`
	PX       float64 `json:"px"`     // 0..100 left->right
	PY       float64 `json:"py"`     // 0..100 top->bottom
	WorldX   float64 `json:"worldX"` // node world coords (to match the flightmaster NPC)
	WorldY   float64 `json:"worldY"`
}

// TaxiPathOut is a directed connection between two nodes.
type TaxiPathOut struct {
	ID   int `json:"id"`
	From int `json:"from"`
	To   int `json:"to"`
}

// TaxiWaypointOut is one point along a path's actual flight route (continent %).
type TaxiWaypointOut struct {
	PathID int     `json:"pathId"`
	Idx    int     `json:"idx"`
	PX     float64 `json:"px"`
	PY     float64 `json:"py"`
}

// TransportRouteOut is a boat/zeppelin route between two hubs (possibly on
// different continents). Endpoints are the nearest flight hub to each end of the
// route's physical waypoint track.
type TransportRouteOut struct {
	ID    int    `json:"id"`
	Type  string `json:"type"` // boat | zeppelin
	NameA string `json:"nameA"`
	MapA  int    `json:"mapA"`
	NameB string `json:"nameB"`
	MapB  int    `json:"mapB"`
}

// TransportWaypointOut is one point of a transport route's track (continent %),
// tagged with the continent it falls on so a single continent can draw its leg.
type TransportWaypointOut struct {
	RouteID int     `json:"routeId"`
	Idx     int     `json:"idx"`
	MapID   int     `json:"mapId"`
	PX      float64 `json:"px"`
	PY      float64 `json:"py"`
}

// TaxiData is the full flight-map data set.
type TaxiData struct {
	Continents   []TaxiContinent        `json:"continents"`
	Nodes        []TaxiNodeOut          `json:"nodes"`
	Paths        []TaxiPathOut          `json:"paths"`
	Waypoints    []TaxiWaypointOut      `json:"waypoints"`
	Transports   []TransportRouteOut    `json:"transports"`
	TransportWps []TransportWaypointOut `json:"transportWps"`
}

type contBounds struct {
	name                       string
	yMax, yMin, xMax, xMin     float64
}

// continentBounds reads the WorldMapArea areaID==0 rows (continent overview) and
// returns mapID -> bounds, skipping degenerate (all-zero) entries like "World".
func continentBounds(cf ClientFiles) map[int]contBounds {
	out := map[int]contBounds{}
	d, err := openDBCFrom(cf, "WorldMapArea.dbc")
	if err != nil {
		return out
	}
	for r := 0; r < d.RecordCount; r++ {
		if d.U32(r, 2) != 0 { // areaID 0 == whole-continent map
			continue
		}
		b := contBounds{
			name: d.Str(r, 3),
			yMax: float64(d.F32(r, 4)), yMin: float64(d.F32(r, 5)),
			xMax: float64(d.F32(r, 6)), xMin: float64(d.F32(r, 7)),
		}
		if b.yMax == b.yMin || b.xMax == b.xMin || b.name == "" {
			continue // no usable extent (e.g. "World")
		}
		out[int(d.U32(r, 1))] = b
	}
	return out
}

func (b contBounds) pct(worldX, worldY float64) (px, py float64) {
	px = (b.yMax - worldY) / (b.yMax - b.yMin) * 100
	py = (b.xMax - worldX) / (b.xMax - b.xMin) * 100
	return clamp01(px), clamp01(py)
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 100 {
		return 100
	}
	return v
}

type taxiNodeRaw struct {
	mapID           int
	name            string
	x, y            float64
	horde, alliance bool
}

type rawWP struct {
	idx, mapID int
	x, y       float64
}

// transportType classifies a route from any descriptive strings available (the
// transport node names + snapped hub names). Octo customs carry "Boat:"/
// "Zeppelin:" prefixes; vanilla uses "Transport, X - Y", so fall back to the
// Horde zeppelin hubs to tell zeppelins from boats.
func transportType(names ...string) string {
	s := strings.ToLower(strings.Join(names, " "))
	if strings.Contains(s, "zeppelin") || strings.Contains(s, "zepplin") {
		return "zeppelin"
	}
	if strings.Contains(s, "boat") || strings.Contains(s, "ship") || strings.Contains(s, "ferry") {
		return "boat"
	}
	for _, z := range []string{"orgrimmar", "undercity", "grom'gol", "sparkwater", "durotar", "tirisfal", "kargath"} {
		if strings.Contains(s, z) {
			return "zeppelin"
		}
	}
	return "boat"
}

// descriptiveTokens extracts the endpoint names a transport node encodes in an
// "A - B" route name (e.g. "Transport, Grom'gol - Orgrimmar" -> grom'gol,
// orgrimmar). Returns nil for uninformative names ("Transport, Menethil Ships",
// "Generic ...", "").
func descriptiveTokens(name string) []string {
	n := name
	for _, p := range []string{"Transport,", "Boat:", "Zeppelin:"} {
		n = strings.TrimSpace(strings.TrimPrefix(n, p))
	}
	if !strings.Contains(n, " - ") {
		return nil
	}
	var out []string
	for _, part := range strings.Split(n, " - ") {
		if part = strings.TrimSpace(strings.ToLower(part)); part != "" {
			out = append(out, part)
		}
	}
	return out
}

func beforeComma(s string) string {
	if i := strings.Index(s, ","); i >= 0 {
		s = s[:i]
	}
	return strings.TrimSpace(strings.ToLower(s))
}

// transportEndpointsOK guards against spurious nearest-hub snaps. When a route
// names its endpoints ("A - B"), at least one named endpoint must match a
// snapped hub; otherwise the route doesn't actually serve those hubs (e.g. the
// custom "Spadowprey Village - Moonhoof Village" boat snapping to a far-off
// Feathermoon) and is dropped. Routes with no descriptive name are trusted.
func transportEndpointsOK(fromName, toName, snapA, snapB string) bool {
	tokens := append(descriptiveTokens(fromName), descriptiveTokens(toName)...)
	if len(tokens) == 0 {
		return true
	}
	hubA, hubB := beforeComma(snapA), beforeComma(snapB)
	for _, t := range tokens {
		for _, h := range []string{hubA, hubB} {
			if h != "" && (strings.Contains(t, h) || strings.Contains(h, t)) {
				return true
			}
		}
	}
	return false
}

// genTaxi builds the flight-map data from the taxi DBCs, projecting node and
// path-waypoint world coordinates onto their continent map as percentages, and
// derives boat/zeppelin transport routes from the non-flight paths.
func genTaxi(cf ClientFiles) (interface{}, error) {
	bounds := continentBounds(cf)

	nodesDBC, err := openDBCFrom(cf, "TaxiNodes.dbc")
	if err != nil {
		return nil, err
	}
	data := &TaxiData{}

	// All nodes with world coords (needed to snap transport route ends to a hub),
	// plus the flight-master candidates: a node with a faction MOUNT creature on a
	// known continent. Mountless nodes are the transport/system waypoints
	// ("Transport, ...", "Boat: ...", "Generic ...") — not flyable themselves.
	allNodes := map[int]taxiNodeRaw{}
	cands := map[int]TaxiNodeOut{}
	for r := 0; r < nodesDBC.RecordCount; r++ {
		id := int(nodesDBC.U32(r, 0))
		mapID := int(nodesDBC.U32(r, 1))
		horde := nodesDBC.U32(r, 14) != 0
		alliance := nodesDBC.U32(r, 15) != 0
		x, y := float64(nodesDBC.F32(r, 2)), float64(nodesDBC.F32(r, 3))
		name := nodesDBC.Str(r, 5)
		allNodes[id] = taxiNodeRaw{mapID, name, x, y, horde, alliance}
		b, ok := bounds[mapID]
		if !ok || (!horde && !alliance) {
			continue
		}
		px, py := b.pct(x, y)
		cands[id] = TaxiNodeOut{ID: id, MapID: mapID, Name: name, Horde: horde, Alliance: alliance, PX: px, PY: py, WorldX: x, WorldY: y}
	}

	// All paths.
	type rawPath struct{ id, from, to int }
	var paths []rawPath
	if pathsDBC, perr := openDBCFrom(cf, "TaxiPath.dbc"); perr == nil {
		for r := 0; r < pathsDBC.RecordCount; r++ {
			paths = append(paths, rawPath{int(pathsDBC.U32(r, 0)), int(pathsDBC.U32(r, 1)), int(pathsDBC.U32(r, 2))})
		}
	}

	// Waypoints grouped by path (world coords + map), index-ordered.
	wpByPath := map[int][]rawWP{}
	if wpDBC, werr := openDBCFrom(cf, "TaxiPathNode.dbc"); werr == nil {
		for r := 0; r < wpDBC.RecordCount; r++ {
			p := int(wpDBC.U32(r, 1))
			wpByPath[p] = append(wpByPath[p], rawWP{int(wpDBC.U32(r, 2)), int(wpDBC.U32(r, 3)), float64(wpDBC.F32(r, 4)), float64(wpDBC.F32(r, 5))})
		}
		for p := range wpByPath {
			sort.Slice(wpByPath[p], func(i, j int) bool { return wpByPath[p][i].idx < wpByPath[p][j].idx })
		}
	}

	// Flight paths: both endpoints are flight masters. Emit the path + its
	// projected waypoints, and mark its endpoints as real (connected) nodes.
	connected := map[int]bool{}
	for _, p := range paths {
		if _, ok := cands[p.from]; !ok {
			continue
		}
		if _, ok := cands[p.to]; !ok {
			continue
		}
		data.Paths = append(data.Paths, TaxiPathOut{ID: p.id, From: p.from, To: p.to})
		connected[p.from] = true
		connected[p.to] = true
		for _, w := range wpByPath[p.id] {
			if b, ok := bounds[w.mapID]; ok {
				px, py := b.pct(w.x, w.y)
				data.Waypoints = append(data.Waypoints, TaxiWaypointOut{PathID: p.id, Idx: w.idx, PX: px, PY: py})
			}
		}
	}

	usedMap := map[int]bool{}
	for id, n := range cands {
		if !connected[id] {
			continue // drops vestigial mount nodes with no route (e.g. Northshire)
		}
		data.Nodes = append(data.Nodes, n)
		usedMap[n.MapID] = true
	}

	// Transport routes: non-flight paths whose physical track links two distinct
	// flight hubs (one per end), within range. The nearest-hub snap labels each
	// end; requiring two different hubs drops partial legs and junk (Naxxramas,
	// Filming, same-hub loops). Reverse duplicates are deduped.
	snap := func(mapID int, x, y float64) (string, float64) {
		best, bestD := "", math.MaxFloat64
		for _, n := range allNodes {
			if n.mapID != mapID || (!n.horde && !n.alliance) {
				continue
			}
			d := (n.x-x)*(n.x-x) + (n.y-y)*(n.y-y)
			if d < bestD {
				bestD, best = d, n.name
			}
		}
		return best, math.Sqrt(bestD)
	}
	const snapMax = 2500.0
	seen := map[string]bool{}
	for _, p := range paths {
		_, fromHub := cands[p.from]
		_, toHub := cands[p.to]
		if fromHub && toHub {
			continue // a flight path
		}
		w := wpByPath[p.id]
		if len(w) < 2 {
			continue
		}
		a, b := w[0], w[len(w)-1]
		if _, ok := bounds[a.mapID]; !ok {
			continue
		}
		if _, ok := bounds[b.mapID]; !ok {
			continue
		}
		nameA, dA := snap(a.mapID, a.x, a.y)
		nameB, dB := snap(b.mapID, b.x, b.y)
		if nameA == "" || nameB == "" || nameA == nameB || dA > snapMax || dB > snapMax {
			continue
		}
		if !transportEndpointsOK(allNodes[p.from].name, allNodes[p.to].name, nameA, nameB) {
			continue // snap is spurious — route names hubs it doesn't serve
		}
		if seen[nameA+"|"+nameB] || seen[nameB+"|"+nameA] {
			continue
		}
		seen[nameA+"|"+nameB] = true
		data.Transports = append(data.Transports, TransportRouteOut{
			ID: p.id,
			// Classify from the clean snapped hub names only — the raw path
			// node names route through shared junctions ("... Zeppelin Paths",
			// "Orgrimmar Zepplins") that would misclassify boats as zeppelins.
			Type:  transportType(nameA, nameB),
			NameA: nameA, MapA: a.mapID,
			NameB: nameB, MapB: b.mapID,
		})
		usedMap[a.mapID] = true
		usedMap[b.mapID] = true
		for _, ww := range w {
			if bb, ok := bounds[ww.mapID]; ok {
				px, py := bb.pct(ww.x, ww.y)
				data.TransportWps = append(data.TransportWps, TransportWaypointOut{RouteID: p.id, Idx: ww.idx, MapID: ww.mapID, PX: px, PY: py})
			}
		}
	}

	for mapID := range usedMap {
		data.Continents = append(data.Continents, TaxiContinent{MapID: mapID, Name: bounds[mapID].name})
	}
	return data, nil
}

// GenerateTaxiJSON regenerates just data/taxi.json from a client source.
func GenerateTaxiJSON(cf ClientFiles, dataDir string) error {
	v, err := genTaxi(cf)
	if err != nil {
		return err
	}
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dataDir, "taxi.json"), b, 0644)
}
