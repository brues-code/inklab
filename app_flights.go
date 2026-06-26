package main

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
