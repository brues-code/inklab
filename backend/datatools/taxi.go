package datatools

import (
	"encoding/json"
	"os"
	"path/filepath"
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
	PX       float64 `json:"px"` // 0..100 left->right
	PY       float64 `json:"py"` // 0..100 top->bottom
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

// TaxiData is the full flight-map data set.
type TaxiData struct {
	Continents []TaxiContinent   `json:"continents"`
	Nodes      []TaxiNodeOut     `json:"nodes"`
	Paths      []TaxiPathOut     `json:"paths"`
	Waypoints  []TaxiWaypointOut `json:"waypoints"`
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

// genTaxi builds the flight-map data from the taxi DBCs, projecting node and
// path-waypoint world coordinates onto their continent map as percentages.
func genTaxi(cf ClientFiles) (interface{}, error) {
	bounds := continentBounds(cf)

	nodesDBC, err := openDBCFrom(cf, "TaxiNodes.dbc")
	if err != nil {
		return nil, err
	}
	data := &TaxiData{}

	// Candidate flight masters on a known continent (keyed by id). Two filters
	// keep this to genuine gryphon/wyvern flight points:
	//   1. A real flight master always has a faction MOUNT creature (Horde and/or
	//      Alliance). The internal transport/boat/zeppelin waypoint nodes
	//      ("Transport, ...", "Boat: ...", "Zeppelin: ...", "Generic, World
	//      target ...", "Filming") have 0/0 mounts — they're the auto-transport
	//      pathing system, not flyable nodes, and create bogus cross-continent
	//      links, so they're excluded here.
	//   2. The node must actually be connected by a path (see below), which drops
	//      vestigial entries like "Northshire Abbey" that carry a stray mount id
	//      but no flight route.
	cands := map[int]TaxiNodeOut{}
	for r := 0; r < nodesDBC.RecordCount; r++ {
		mapID := int(nodesDBC.U32(r, 1))
		b, ok := bounds[mapID]
		if !ok {
			continue // node on a map without a continent overview (skip)
		}
		horde := nodesDBC.U32(r, 14) != 0
		alliance := nodesDBC.U32(r, 15) != 0
		if !horde && !alliance {
			continue // transport/system waypoint, not a flight master
		}
		px, py := b.pct(float64(nodesDBC.F32(r, 2)), float64(nodesDBC.F32(r, 3)))
		id := int(nodesDBC.U32(r, 0))
		cands[id] = TaxiNodeOut{
			ID:       id,
			MapID:    mapID,
			Name:     nodesDBC.Str(r, 5),
			Horde:    horde,
			Alliance: alliance,
			PX:       px,
			PY:       py,
		}
	}

	// Paths whose both endpoints are candidate nodes; these also tell us which
	// nodes are real flight masters.
	connected := map[int]bool{}
	pathsDBC, err := openDBCFrom(cf, "TaxiPath.dbc")
	if err == nil {
		for r := 0; r < pathsDBC.RecordCount; r++ {
			from := int(pathsDBC.U32(r, 1))
			to := int(pathsDBC.U32(r, 2))
			if _, ok := cands[from]; !ok {
				continue
			}
			if _, ok := cands[to]; !ok {
				continue
			}
			data.Paths = append(data.Paths, TaxiPathOut{ID: int(pathsDBC.U32(r, 0)), From: from, To: to})
			connected[from] = true
			connected[to] = true
		}
	}

	// Keep only connected nodes; record which continents end up with nodes.
	usedMap := map[int]bool{}
	for id, n := range cands {
		if !connected[id] {
			continue
		}
		data.Nodes = append(data.Nodes, n)
		usedMap[n.MapID] = true
	}
	for mapID := range usedMap {
		data.Continents = append(data.Continents, TaxiContinent{MapID: mapID, Name: bounds[mapID].name})
	}

	// Path waypoints (actual route), projected per their own map.
	wpDBC, err := openDBCFrom(cf, "TaxiPathNode.dbc")
	if err == nil {
		for r := 0; r < wpDBC.RecordCount; r++ {
			b, ok := bounds[int(wpDBC.U32(r, 3))]
			if !ok {
				continue
			}
			px, py := b.pct(float64(wpDBC.F32(r, 4)), float64(wpDBC.F32(r, 5)))
			data.Waypoints = append(data.Waypoints, TaxiWaypointOut{
				PathID: int(wpDBC.U32(r, 1)), Idx: int(wpDBC.U32(r, 2)), PX: px, PY: py,
			})
		}
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
