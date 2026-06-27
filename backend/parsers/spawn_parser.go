package parsers

import (
	"io"
	"regexp"
	"strconv"
	"strings"
)

// SpawnPoint is one spawn location scraped from an octowow.st (aowow) detail
// page: a zone areatableID plus already-computed 0-100 map-percentage coords.
// ZoneName is the location-link label (e.g. "Kalimdor", "Azeroth", or a real
// zone name) — needed to pick the right continent when ZoneID is 0, since aowow
// buckets unzoned spawns under the continent name (which can be either continent,
// not just Azeroth — custom Kalimdor zones like Blackstone Island land here too).
type SpawnPoint struct {
	ZoneID   int     `json:"zoneId"`
	ZoneName string  `json:"zoneName"`
	X        float64 `json:"x"`
	Y        float64 `json:"y"`
}

// aowow embeds spawn data on object/npc pages as per-zone JS calls inside a
// location-list anchor:
//
//	<a ... onclick="myMapper.update({zone: 1519,coords: [[39.95,84.36,...]]})...">Zone Name</a>
//
// One call per zone the entity spawns in. blockRe captures (zoneID, coordsBody,
// linkName); coordRe pulls each [x,y, ...] leading pair out of that body. The
// link name is optional so a bare myMapper.update still matches.
var (
	spawnBlockRe = regexp.MustCompile(`(?s)zone:\s*(\d+)\s*,\s*coords:\s*\[(.*?)\]\}\)(?:[^>]*>([^<]*)</a>)?`)
	spawnCoordRe = regexp.MustCompile(`\[(-?\d+(?:\.\d+)?),(-?\d+(?:\.\d+)?),`)
)

// ParseSpawnPoints extracts spawn points from an aowow detail page body.
func ParseSpawnPoints(r io.Reader) ([]SpawnPoint, error) {
	body, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	html := string(body)

	var points []SpawnPoint
	for _, block := range spawnBlockRe.FindAllStringSubmatch(html, -1) {
		zoneID, err := strconv.Atoi(block[1])
		if err != nil {
			continue
		}
		name := strings.TrimSpace(block[3])
		for _, c := range spawnCoordRe.FindAllStringSubmatch(block[2], -1) {
			x, err1 := strconv.ParseFloat(c[1], 64)
			y, err2 := strconv.ParseFloat(c[2], 64)
			if err1 != nil || err2 != nil {
				continue
			}
			points = append(points, SpawnPoint{ZoneID: zoneID, ZoneName: name, X: x, Y: y})
		}
	}
	return points, nil
}
