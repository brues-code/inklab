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

// FlightData is everything the map needs for one continent.
type FlightData struct {
	Nodes       []FlightNode       `json:"nodes"`
	Connections []FlightConnection `json:"connections"`
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
	return out
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
