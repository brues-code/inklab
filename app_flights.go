package main

import "strconv"

// Flight (taxi) network read from the taxi_node / taxi_path / taxi_path_node
// tables (DBC-derived, shipped in the embedded inklab.db). Nodes are positioned
// as percentages on their continent overview map; the frontend overlays them on
// the continent image and draws connections.

// FlightContinent is a continent that has flight nodes.
type FlightContinent struct {
	MapID int    `json:"mapId"`
	Name  string `json:"name"` // continent map image key (e.g. "Azeroth")
}

// FlightNode is a flight master positioned on its continent map.
type FlightNode struct {
	ID       int     `json:"id"`
	Name     string  `json:"name"`
	Alliance bool    `json:"alliance"`
	Horde    bool    `json:"horde"`
	Zone     string  `json:"zone"` // zone map key this node sits in (for drill-down)
	PX       float64 `json:"px"`
	PY       float64 `json:"py"`
}

// FlightConnection is a path between two nodes, with its waypoint route.
type FlightConnection struct {
	PathID    int       `json:"pathId"`
	From      int       `json:"from"`
	To        int       `json:"to"`
	Waypoints [][2]float64 `json:"waypoints"` // [px,py] route, empty if none
}

// TransportRoute is a boat/zeppelin leg as seen from one continent: its
// on-continent track, the destination hub, and whether the destination is on a
// different continent.
type TransportRoute struct {
	ID            int          `json:"id"`
	Type          string       `json:"type"` // boat | zeppelin
	Here          string       `json:"here"` // hub on this continent
	Dest          string       `json:"dest"` // hub at the other end
	DestContinent string       `json:"destContinent"`
	SameContinent bool         `json:"sameContinent"`
	Waypoints     [][2]float64 `json:"waypoints"` // this continent's leg
}

// FlightData is everything the map needs for one continent.
type FlightData struct {
	Nodes       []FlightNode       `json:"nodes"`
	Connections []FlightConnection `json:"connections"`
	Transports  []TransportRoute   `json:"transports"`
}

// WorldNode is a flight master with its continent, for the combined world view.
type WorldNode struct {
	ID       int     `json:"id"`
	Name     string  `json:"name"`
	Alliance bool    `json:"alliance"`
	Horde    bool    `json:"horde"`
	MapID    int     `json:"mapId"`
	Zone     string  `json:"zone"`
	PX       float64 `json:"px"`
	PY       float64 `json:"py"`
}

// WorldConn is a flight link between two nodes (node ids).
type WorldConn struct {
	From int `json:"from"`
	To   int `json:"to"`
}

// WorldTransport is a transport route with both hub endpoints resolved to their
// continent + position, so cross-continent boats/zeppelins draw as one line on
// the combined world map.
type WorldTransport struct {
	Type string  `json:"type"`
	AMap int     `json:"aMap"`
	APx  float64 `json:"aPx"`
	APy  float64 `json:"aPy"`
	AName string `json:"aName"`
	BMap int     `json:"bMap"`
	BPx  float64 `json:"bPx"`
	BPy  float64 `json:"bPy"`
	BName string `json:"bName"`
}

// WorldData is everything the combined world map needs: all continents, all
// flight nodes/links, and transports resolved to both endpoints.
type WorldData struct {
	Continents []FlightContinent `json:"continents"`
	Nodes      []WorldNode       `json:"nodes"`
	Connections []WorldConn      `json:"connections"`
	Transports []WorldTransport  `json:"transports"`
}

// GetWorldData returns the full flight + transport network across all continents
// for the combined world view. Transport endpoints are resolved to node
// positions so cross-continent routes render as a single connecting line.
func (a *App) GetWorldData() *WorldData {
	out := &WorldData{}
	if a.db == nil {
		return out
	}
	out.Continents = a.GetFlightContinents()

	// Nodes (all continents) + a (map,name)->pos index for transport resolution.
	type pos struct {
		mapID  int
		px, py float64
	}
	byKey := map[string]pos{}
	if rows, err := a.db.DB().Query("SELECT id, name, alliance, horde, map_id, px, py FROM taxi_node"); err == nil {
		for rows.Next() {
			var n WorldNode
			var al, ho int
			if rows.Scan(&n.ID, &n.Name, &al, &ho, &n.MapID, &n.PX, &n.PY) == nil {
				n.Alliance, n.Horde = al != 0, ho != 0
				if a.npcService != nil {
					n.Zone, _, _, _ = a.npcService.ResolveContinentPoint(n.MapID, n.PX, n.PY)
				}
				out.Nodes = append(out.Nodes, n)
				byKey[transportKey(n.MapID, n.Name)] = pos{n.MapID, n.PX, n.PY}
			}
		}
		rows.Close()
	}

	// Flight links, deduped to one per unordered node pair.
	if rows, err := a.db.DB().Query("SELECT from_node, to_node FROM taxi_path"); err == nil {
		seen := map[[2]int]bool{}
		for rows.Next() {
			var f, t int
			if rows.Scan(&f, &t) != nil {
				continue
			}
			key := [2]int{f, t}
			if f > t {
				key = [2]int{t, f}
			}
			if seen[key] {
				continue
			}
			seen[key] = true
			out.Connections = append(out.Connections, WorldConn{From: f, To: t})
		}
		rows.Close()
	}

	// Transports, both endpoints resolved to node positions.
	if rows, err := a.db.DB().Query("SELECT type, name_a, map_a, name_b, map_b FROM transport_route"); err == nil {
		for rows.Next() {
			var typ, na, nb string
			var ma, mb int
			if rows.Scan(&typ, &na, &ma, &nb, &mb) != nil {
				continue
			}
			pa, oka := byKey[transportKey(ma, na)]
			pb, okb := byKey[transportKey(mb, nb)]
			if !oka || !okb {
				continue // endpoint not a known node — skip rather than mislocate
			}
			out.Transports = append(out.Transports, WorldTransport{
				Type: typ,
				AMap: ma, APx: pa.px, APy: pa.py, AName: na,
				BMap: mb, BPx: pb.px, BPy: pb.py, BName: nb,
			})
		}
		rows.Close()
	}
	return out
}

func transportKey(mapID int, name string) string {
	return strconv.Itoa(mapID) + "|" + name
}

// --- zone drill-down --------------------------------------------------------
//
// Zone resolution reuses the authoritative spawn resolver
// (NpcService.ResolveContinentPoint: client area grid, then zones.json), so
// flight masters land in the same zone — and at the same spot — as creature
// spawns, including custom octo zones (e.g. Thalassian Highlands).

// ZoneNode is a flight master projected onto a zone map, with its destinations.
type ZoneNode struct {
	ID       int      `json:"id"`
	Name     string   `json:"name"`
	Alliance bool     `json:"alliance"`
	Horde    bool     `json:"horde"`
	PX       float64  `json:"px"`
	PY       float64  `json:"py"`
	Dests    []string `json:"dests"`
}

// ZoneData is the detail view for one zone: its map key and the flight masters
// in it, positioned on the zone map.
type ZoneData struct {
	MapKey string     `json:"mapKey"`
	MapID  int        `json:"mapId"`
	Zone   string     `json:"zone"`
	Nodes  []ZoneNode `json:"nodes"`
}

// GetZoneData returns the flight masters within a zone, projected onto the zone
// map via the shared spawn resolver, for the drill-down view.
func (a *App) GetZoneData(mapID int, zone string) *ZoneData {
	out := &ZoneData{MapKey: zone, MapID: mapID, Zone: zone}
	if a.db == nil || a.npcService == nil || zone == "" {
		return out
	}

	destsFor := func(id int) []string {
		r, err := a.db.DB().Query(
			"SELECT DISTINCT t.name FROM taxi_path p JOIN taxi_node t ON p.to_node = t.id WHERE p.from_node = ? ORDER BY t.name", id)
		if err != nil {
			return nil
		}
		defer r.Close()
		var ds []string
		for r.Next() {
			var n string
			if r.Scan(&n) == nil {
				ds = append(ds, n)
			}
		}
		return ds
	}

	rows, err := a.db.DB().Query("SELECT id, name, alliance, horde, px, py FROM taxi_node WHERE map_id = ?", mapID)
	if err != nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var id, al, ho int
		var name string
		var px, py float64
		if rows.Scan(&id, &name, &al, &ho, &px, &py) != nil {
			continue
		}
		zName, zx, zy, ok := a.npcService.ResolveContinentPoint(mapID, px, py)
		if !ok || zName != zone {
			continue
		}
		out.Nodes = append(out.Nodes, ZoneNode{
			ID: id, Name: name, Alliance: al != 0, Horde: ho != 0, PX: zx, PY: zy, Dests: destsFor(id),
		})
	}
	return out
}

// GetFlightContinents lists the continents that have flight nodes, ordered by
// map id (Azeroth=0, Kalimdor=1, then any custom).
func (a *App) GetFlightContinents() []FlightContinent {
	if a.db == nil {
		return nil
	}
	rows, err := a.db.DB().Query(
		"SELECT map_id, continent FROM taxi_node WHERE continent != '' GROUP BY map_id, continent ORDER BY map_id")
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []FlightContinent
	for rows.Next() {
		var c FlightContinent
		if rows.Scan(&c.MapID, &c.Name) == nil {
			out = append(out, c)
		}
	}
	return out
}

// GetFlightData returns the nodes and connections for one continent (by map id).
// Connections are limited to paths whose both endpoints are on this continent,
// and carry their waypoint route (for the "actual route" rendering mode).
func (a *App) GetFlightData(mapID int) *FlightData {
	out := &FlightData{}
	if a.db == nil {
		return out
	}

	nodeRows, err := a.db.DB().Query(
		"SELECT id, name, alliance, horde, px, py FROM taxi_node WHERE map_id = ? ORDER BY name", mapID)
	if err != nil {
		return out
	}
	onMap := map[int]bool{}
	for nodeRows.Next() {
		var n FlightNode
		var al, ho int
		if nodeRows.Scan(&n.ID, &n.Name, &al, &ho, &n.PX, &n.PY) == nil {
			n.Alliance = al != 0
			n.Horde = ho != 0
			if a.npcService != nil {
				n.Zone, _, _, _ = a.npcService.ResolveContinentPoint(mapID, n.PX, n.PY)
			}
			out.Nodes = append(out.Nodes, n)
			onMap[n.ID] = true
		}
	}
	nodeRows.Close()
	if len(out.Nodes) == 0 {
		return out
	}

	// Paths between nodes on this continent.
	pathRows, err := a.db.DB().Query("SELECT id, from_node, to_node FROM taxi_path")
	if err != nil {
		return out
	}
	var pathIDs []int
	connByID := map[int]*FlightConnection{}
	for pathRows.Next() {
		var c FlightConnection
		if pathRows.Scan(&c.PathID, &c.From, &c.To) != nil {
			continue
		}
		if !onMap[c.From] || !onMap[c.To] {
			continue
		}
		conn := c
		out.Connections = append(out.Connections, conn)
	}
	pathRows.Close()

	// Attach waypoint routes for the kept paths.
	for idx := range out.Connections {
		connByID[out.Connections[idx].PathID] = &out.Connections[idx]
		pathIDs = append(pathIDs, out.Connections[idx].PathID)
	}
	if len(pathIDs) > 0 {
		wpRows, err := a.db.DB().Query(
			"SELECT path_id, px, py FROM taxi_path_node WHERE path_id IN ("+placeholders(len(pathIDs))+") ORDER BY path_id, idx",
			toArgs(pathIDs)...)
		if err == nil {
			for wpRows.Next() {
				var pid int
				var px, py float64
				if wpRows.Scan(&pid, &px, &py) == nil {
					if c := connByID[pid]; c != nil {
						c.Waypoints = append(c.Waypoints, [2]float64{px, py})
					}
				}
			}
			wpRows.Close()
		}
	}

	out.Transports = a.loadTransports(mapID)
	return out
}

// loadTransports returns the boat/zeppelin legs that touch a continent: each
// route's on-continent waypoint track plus where it goes.
func (a *App) loadTransports(mapID int) []TransportRoute {
	// Continent map_id -> name (for the destination continent label).
	contName := map[int]string{}
	if cr, err := a.db.DB().Query("SELECT map_id, continent FROM taxi_node WHERE continent != '' GROUP BY map_id"); err == nil {
		for cr.Next() {
			var m int
			var n string
			if cr.Scan(&m, &n) == nil {
				contName[m] = n
			}
		}
		cr.Close()
	}

	rows, err := a.db.DB().Query(
		"SELECT id, type, name_a, map_a, name_b, map_b FROM transport_route WHERE map_a = ? OR map_b = ?", mapID, mapID)
	if err != nil {
		return nil
	}
	var routes []TransportRoute
	byID := map[int]*TransportRoute{}
	for rows.Next() {
		var id, mapA, mapB int
		var typ, nameA, nameB string
		if rows.Scan(&id, &typ, &nameA, &mapA, &nameB, &mapB) != nil {
			continue
		}
		tr := TransportRoute{ID: id, Type: typ, SameContinent: mapA == mapB}
		if mapA == mapID {
			tr.Here, tr.Dest = nameA, nameB
			if mapB != mapID {
				tr.DestContinent = contName[mapB]
			}
		} else {
			tr.Here, tr.Dest = nameB, nameA
			if mapA != mapID {
				tr.DestContinent = contName[mapA]
			}
		}
		routes = append(routes, tr)
	}
	rows.Close()
	if len(routes) == 0 {
		return nil
	}
	for i := range routes {
		byID[routes[i].ID] = &routes[i]
	}

	ids := make([]int, 0, len(routes))
	for _, r := range routes {
		ids = append(ids, r.ID)
	}
	wpRows, err := a.db.DB().Query(
		"SELECT route_id, px, py FROM transport_waypoint WHERE map_id = ? AND route_id IN ("+placeholders(len(ids))+") ORDER BY route_id, idx",
		append([]interface{}{mapID}, toArgs(ids)...)...)
	if err == nil {
		for wpRows.Next() {
			var rid int
			var px, py float64
			if wpRows.Scan(&rid, &px, &py) == nil {
				if r := byID[rid]; r != nil {
					r.Waypoints = append(r.Waypoints, [2]float64{px, py})
				}
			}
		}
		wpRows.Close()
	}
	return routes
}

func placeholders(n int) string {
	if n <= 0 {
		return ""
	}
	b := make([]byte, 0, n*2)
	for i := 0; i < n; i++ {
		if i > 0 {
			b = append(b, ',')
		}
		b = append(b, '?')
	}
	return string(b)
}

func toArgs(ids []int) []interface{} {
	a := make([]interface{}, len(ids))
	for i, v := range ids {
		a[i] = v
	}
	return a
}
