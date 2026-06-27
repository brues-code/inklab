package parsers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// ScrapedNpcData holds data scraped from Wowhead
type ScrapedNpcData struct {
	Infobox       map[string]string `json:"infobox"`
	MapURL        string            `json:"mapUrl"`
	ModelImageURL string            `json:"modelImageUrl"`
	ZoneName      string            `json:"zoneName"`
	X             float64           `json:"x"`
	Y             float64           `json:"y"`
	Spawns        []ScrapedSpawn    `json:"spawns"` // every spawn point across every zone (octowow mapper data)
	Sells         []VendorSale      `json:"sells"`  // items this NPC sells (octowow "sells" tab)
	TrainerSpells []int             `json:"trainerSpells"` // spell ids this NPC trains (octowow "teaches" tab)
}

// aowow embeds a trainer's taught spells in a "teaches-ability" Listview:
//
//	new Listview({template:'spell',id:'teaches-ability',...,data:[{name:'@X',...,id: 1303},...]})
//
// trainerBlockRe isolates that listview's data array; trainerIdRe pulls each
// spell id out of it.
var (
	trainerBlockRe = regexp.MustCompile(`(?s)id:'teaches-ability'.*?data:\s*\[(.*?)\]`)
	trainerIDRe    = regexp.MustCompile(`,\s*id:\s*(\d+)`)
)

// ParseTrainerSpells extracts the spell ids an NPC trains from an aowow NPC page.
func ParseTrainerSpells(html string) []int {
	block := trainerBlockRe.FindStringSubmatch(html)
	if block == nil {
		return nil
	}
	var out []int
	seen := map[int]bool{}
	for _, m := range trainerIDRe.FindAllStringSubmatch(block[1], -1) {
		if id, err := strconv.Atoi(m[1]); err == nil && id > 0 && !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	return out
}

// ScrapedSpawn is one spawn point: its zone (id + resolved name) and the
// zone-relative 0-100 map coordinates octowow reports for it.
type ScrapedSpawn struct {
	ZoneID   int     `json:"zoneId"`
	ZoneName string  `json:"zoneName"`
	X        float64 `json:"x"`
	Y        float64 `json:"y"`
}

// VendorSale is one item an NPC sells: the item entry, money cost (copper, 0 for
// extended-cost currencies) and stock (-1 = unlimited).
type VendorSale struct {
	ItemEntry int `json:"itemEntry"`
	Cost      int `json:"cost"`
	Stock     int `json:"stock"`
}

// ParseNpcData parses the HTML content of a Wowhead NPC page
func ParseNpcData(r io.Reader) (*ScrapedNpcData, error) {
	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		return nil, err
	}

	data := &ScrapedNpcData{
		Infobox: make(map[string]string),
	}

	// 1. Parse Infobox (Standard + Scripted)
	doc.Find("table.infobox tr").Each(func(i int, s *goquery.Selection) {
		header := strings.TrimSpace(s.Find("th").Text())
		contentCell := s.Find("td")
		content := strings.TrimSpace(contentCell.Text())

		// Skip "screenshots", "videos" rows which confuse the parser
		if strings.EqualFold(header, "Screenshots") || strings.EqualFold(header, "Videos") || strings.Contains(content, "ScreenshotsVideos") {
			return
		}

		// Handle "Quick Facts" specifically if it contains script markup
		if strings.Contains(header, "Quick Facts") || strings.Contains(content, "WH.markup.printHtml") {
			scriptContent := contentCell.Text() // This gets the script text
			if strings.Contains(scriptContent, "WH.markup.printHtml") {
				// Extract the markup string: printHtml(" ... ", ...)
				// Heuristic regex to grab the content inside standard quotes
				re := regexp.MustCompile(`printHtml\(\"(.+?)\",`)
				match := re.FindStringSubmatch(scriptContent)
				if len(match) > 1 {
					markup := match[1]
					// Parse [li]Key: Value[/li] pairs from the markup
					// We unescape the string first if needed (usually Go handles basic, but this is raw JS source text)
					// Handle escaped quotes if any: \" -> "
					markup = strings.ReplaceAll(markup, `\"`, `"`)

					// Regex to find list items
					liRe := regexp.MustCompile(`\[li\](.*?):?\s*(.*?)\[\/li\]`)
					items := liRe.FindAllStringSubmatch(markup, -1)

					for _, item := range items {
						if len(item) == 3 {
							key := stripTags_Local(item[1])
							val := stripTags_Local(item[2])
							if key != "" && val != "" {
								data.Infobox[key] = val
							}
						}
					}
				}
			}
			return // Done with this special row
		}

		// Standard rows
		if header != "" && content != "" {
			// Clean up content just in case
			if !strings.Contains(content, "WH.markup") {
				data.Infobox[header] = content
			}
		}
	})

	// 2. Parse Map
	doc.Find("span.mapper-map").Each(func(i int, s *goquery.Selection) {
		style, exists := s.Attr("style")
		if exists {
			re := regexp.MustCompile(`url\(["']?([^"']+)["']?\)`)
			matches := re.FindStringSubmatch(style)
			if len(matches) > 1 {
				data.MapURL = matches[1]
			}
		}
	})

	// 3. Parse Model Image (Screenshot)
	// Try meta og:image first which is usually high quality
	doc.Find("meta[property='og:image']").Each(func(i int, s *goquery.Selection) {
		if content, exists := s.Attr("content"); exists {
			data.ModelImageURL = content
			fmt.Printf("[DEBUG] Found og:image: %s\n", content)
		}
	})

	// Also try twitter:image as fallback
	if data.ModelImageURL == "" {
		doc.Find("meta[name='twitter:image']").Each(func(i int, s *goquery.Selection) {
			if content, exists := s.Attr("content"); exists {
				data.ModelImageURL = content
				fmt.Printf("[DEBUG] Found twitter:image: %s\n", content)
			}
		})
	}

	// 3.5 Parse Zone Name from page header (fallback for instance NPCs)
	// Look for the zone link near the title - usually in a heading-size-1 or similar
	doc.Find("h1.heading-size-1").Parent().Find("a[href*='/zone=']").Each(func(i int, s *goquery.Selection) {
		zoneName := strings.TrimSpace(s.Text())
		if zoneName != "" && data.ZoneName == "" {
			data.ZoneName = zoneName
			fmt.Printf("[DEBUG] Found zone from header link: %s\n", zoneName)
		}
	})

	// Also try to find zone in the breadcrumb or subheader area
	doc.Find(".text a[href*='/zone=']").Each(func(i int, s *goquery.Selection) {
		zoneName := strings.TrimSpace(s.Text())
		if zoneName != "" && data.ZoneName == "" {
			data.ZoneName = zoneName
			fmt.Printf("[DEBUG] Found zone from text link: %s\n", zoneName)
		}
	})

	// 4. Parse Mapper Data (g_mapperData)
	doc.Find("script").Each(func(i int, s *goquery.Selection) {
		scriptContent := s.Text()
		if strings.Contains(scriptContent, "g_mapperData") {
			fmt.Printf("[DEBUG] Found g_mapperData in script block (len=%d)\n", len(scriptContent))

			// Show context around g_mapperData
			idx := strings.Index(scriptContent, "g_mapperData")
			if idx >= 0 {
				start := idx
				if start > 50 {
					start = idx - 50
				}
				end := idx + 300
				if end > len(scriptContent) {
					end = len(scriptContent)
				}
				fmt.Printf("[DEBUG] Context: ...%s...\n", scriptContent[start:end])
			}

			// Try multiple patterns
			patterns := []string{
				`(?s)var g_mapperData\s*=\s*(\{.+?\});`,     // standard with whitespace
				`(?s)g_mapperData\s*=\s*(\{.+?\});`,         // without var
				`(?s)WH\.setPageData\("map",\s*(\{.+?\})\)`, // alternative format
			}

			var jsonStr string
			for _, pattern := range patterns {
				re := regexp.MustCompile(pattern)
				match := re.FindStringSubmatch(scriptContent)
				if len(match) > 1 {
					jsonStr = match[1]
					fmt.Printf("[DEBUG] Matched pattern: %s\n", pattern)
					break
				}
			}

			if jsonStr != "" {
				previewLen := 300
				if len(jsonStr) < previewLen {
					previewLen = len(jsonStr)
				}
				fmt.Printf("[DEBUG] Extracted mapperData JSON: %s\n", jsonStr[:previewLen])

				// Define a struct to map the JSON structure
				// Structure is keys (ZoneIDs) -> list of objects
				// We just want to grab the first zone and its name/coords
				type MapperZoneData struct {
					UIMapName string      `json:"uiMapName"`
					Coords    [][]float64 `json:"coords"`
				}

				// Use generic map to handle dynamic keys (Zone IDs)
				var mapperData map[string][]MapperZoneData
				if err := json.Unmarshal([]byte(jsonStr), &mapperData); err == nil {
					for zoneID, zones := range mapperData {
						for _, zone := range zones {
							if zone.UIMapName != "" {
								data.ZoneName = zone.UIMapName
								if len(zone.Coords) > 0 {
									// Just take the first coordinate for simplicity
									// Wowhead coords are usually 0-100, we might want to keep them as is or format them
									data.X = zone.Coords[0][0]
									data.Y = zone.Coords[0][1]
								}
								// Construct map URL from zone ID if we don't have one already
								if data.MapURL == "" && zoneID != "" {
									data.MapURL = "https://wow.zamimg.com/images/wow/classic/maps/enus/original/" + zoneID + ".jpg"
								}
								return // Found valid data, stop
							}
						}
					}
				}
			}
		}
	})

	// 5. Fallback: If we have a zone name but no map URL, try to use known instance maps
	if data.ZoneName != "" && data.MapURL == "" {
		instanceMaps := map[string]string{
			"Zul'Gurub":             "https://wow.zamimg.com/images/wow/maps/enus/original/1977.jpg",
			"Molten Core":           "https://wow.zamimg.com/images/wow/maps/enus/original/2717.jpg",
			"Blackwing Lair":        "https://wow.zamimg.com/images/wow/maps/enus/original/2677.jpg",
			"Onyxia's Lair":         "https://wow.zamimg.com/images/wow/maps/enus/original/2159.jpg",
			"Temple of Ahn'Qiraj":   "https://wow.zamimg.com/images/wow/maps/enus/original/3428.jpg",
			"Ruins of Ahn'Qiraj":    "https://wow.zamimg.com/images/wow/maps/enus/original/3429.jpg",
			"Naxxramas":             "https://wow.zamimg.com/images/wow/maps/enus/original/3456.jpg",
			"Stratholme":            "https://wow.zamimg.com/images/wow/maps/enus/original/2017.jpg",
			"Scholomance":           "https://wow.zamimg.com/images/wow/maps/enus/original/2057.jpg",
			"Dire Maul":             "https://wow.zamimg.com/images/wow/maps/enus/original/2557.jpg",
			"Upper Blackrock Spire": "https://wow.zamimg.com/images/wow/maps/enus/original/1583.jpg",
			"Lower Blackrock Spire": "https://wow.zamimg.com/images/wow/maps/enus/original/1584.jpg",
			"Blackrock Depths":      "https://wow.zamimg.com/images/wow/maps/enus/original/1584.jpg",
			"Maraudon":              "https://wow.zamimg.com/images/wow/maps/enus/original/2100.jpg",
			"Sunken Temple":         "https://wow.zamimg.com/images/wow/maps/enus/original/1477.jpg",
			"Zul'Farrak":            "https://wow.zamimg.com/images/wow/maps/enus/original/1176.jpg",
			"Uldaman":               "https://wow.zamimg.com/images/wow/maps/enus/original/1337.jpg",
			"Razorfen Downs":        "https://wow.zamimg.com/images/wow/maps/enus/original/722.jpg",
			"Razorfen Kraul":        "https://wow.zamimg.com/images/wow/maps/enus/original/761.jpg",
			"Scarlet Monastery":     "https://wow.zamimg.com/images/wow/maps/enus/original/796.jpg",
			"Gnomeregan":            "https://wow.zamimg.com/images/wow/maps/enus/original/721.jpg",
			"Shadowfang Keep":       "https://wow.zamimg.com/images/wow/maps/enus/original/764.jpg",
			"Blackfathom Deeps":     "https://wow.zamimg.com/images/wow/maps/enus/original/719.jpg",
			"The Stockade":          "https://wow.zamimg.com/images/wow/maps/enus/original/717.jpg",
			"Wailing Caverns":       "https://wow.zamimg.com/images/wow/maps/enus/original/718.jpg",
			"Deadmines":             "https://wow.zamimg.com/images/wow/maps/enus/original/756.jpg",
			"Ragefire Chasm":        "https://wow.zamimg.com/images/wow/maps/enus/original/680.jpg",
		}

		if mapURL, ok := instanceMaps[data.ZoneName]; ok {
			data.MapURL = mapURL
			fmt.Printf("[DEBUG] Using hardcoded instance map for %s\n", data.ZoneName)
		}
	}

	return data, nil
}

// Helper to strip BBCode/HTML-like tags from specific Wowhead strings
func stripTags_Local(input string) string {
	// Remove [tag] and [/tag]
	re := regexp.MustCompile(`\[\/?[^\]]+\]`)
	return strings.TrimSpace(re.ReplaceAllString(input, ""))
}

// ParseNpcDataTurtlecraft parses the HTML content of a TurtleCraft NPC page
// soldByNpcSellsRe isolates the "sells" Listview's data array on an NPC page
// (template:'item', id:'sells'). The array closes the Listview with "]})",
// which never appears inside an item object (its sub-arrays like cost use "],"),
// so a non-greedy capture to the first "]})" grabs exactly this array.
var npcSellsRe = regexp.MustCompile(`(?s)id:'sells'.*?data: ?\[(.*?)\]\}\)`)
var sellObjRe = regexp.MustCompile(`\{[^{}]*\}`)

// spawnZoneRe captures every octowow mapper block that carries coordinates:
// `myMapper.update({zone: 267,coords: [[22.55,43.17,{..}],[56.36,58.31,{..}]]})`.
// The coords array never contains "]]" until it closes (inner items end with
// "}]," ), so a non-greedy capture to the first "]]" grabs the whole array.
var spawnZoneRe = regexp.MustCompile(`(?:myMapper\.update|new Mapper)\(\{[^{}]*?\bzone:\s*'?(\d+)'?\s*,\s*coords:\s*(\[\[.*?\]\])`)

// zoneNameByID maps the AoWoW/octowow numeric zone id to its display name, the
// reverse of the name->id table used for map URLs. Covers the classic 1.12
// open-world zones spawns are reported in.
var zoneNameByID = map[int]string{
	1422: "Western Plaguelands", 1423: "Eastern Plaguelands", 85: "Tirisfal Glades",
	130: "Silverpine Forest", 267: "Hillsbrad Foothills", 36: "Alterac Mountains",
	33: "Stranglethorn Vale", 10: "Duskwood", 40: "Westfall", 12: "Elwynn Forest",
	44: "Redridge Mountains", 46: "Burning Steppes", 51: "Searing Gorge", 3: "Badlands",
	8: "Swamp of Sorrows", 4: "Blasted Lands", 41: "Deadwind Pass", 1: "Dun Morogh",
	38: "Loch Modan", 11: "Wetlands", 45: "Arathi Highlands", 47: "The Hinterlands",
	14: "Durotar", 17: "The Barrens", 215: "Mulgore", 406: "Stonetalon Mountains",
	331: "Ashenvale", 400: "Thousand Needles", 405: "Desolace", 357: "Feralas",
	15: "Dustwallow Marsh", 440: "Tanaris", 490: "Un'Goro Crater", 1377: "Silithus",
	361: "Felwood", 618: "Winterspring", 493: "Moonglade", 16: "Azshara",
	141: "Teldrassil", 148: "Darkshore",
}

// parseNpcSells extracts the items an NPC sells from its page's "sells" tab.
func parseNpcSells(content string) []VendorSale {
	m := npcSellsRe.FindStringSubmatch(content)
	if len(m) < 2 {
		return nil
	}
	intField := func(obj, key string) (int, bool) {
		re := regexp.MustCompile(key + `: ?(-?\d+)`)
		if mm := re.FindStringSubmatch(obj); len(mm) > 1 {
			v, _ := strconv.Atoi(mm[1])
			return v, true
		}
		return 0, false
	}
	var out []VendorSale
	for _, obj := range sellObjRe.FindAllString(m[1], -1) {
		id, ok := intField(obj, "id")
		if !ok || id == 0 {
			continue
		}
		sale := VendorSale{ItemEntry: id, Stock: -1}
		if st, ok := intField(obj, "stock"); ok {
			sale.Stock = st
		}
		// cost: [money, ...] — first element is the copper price.
		if cm := regexp.MustCompile(`cost: ?\[(-?\d+)`).FindStringSubmatch(obj); len(cm) > 1 {
			sale.Cost, _ = strconv.Atoi(cm[1])
		}
		out = append(out, sale)
	}
	return out
}

func ParseNpcDataTurtlecraft(r io.Reader) (*ScrapedNpcData, error) {
	raw, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}

	data := &ScrapedNpcData{
		Infobox: make(map[string]string),
		Sells:   parseNpcSells(string(raw)),
	}

	// Parse infobox items (li elements with label: value format)
	doc.Find("li").Each(func(i int, s *goquery.Selection) {
		text := strings.TrimSpace(s.Text())
		// Parse "Label: Value" format
		if idx := strings.Index(text, ":"); idx > 0 {
			label := strings.TrimSpace(text[:idx])
			value := strings.TrimSpace(text[idx+1:])
			if label != "" && value != "" {
				data.Infobox[label] = value
			}
		}
	})

	// Parse zone name from location section
	// TurtleCraft shows zone as a link like [Western Plaguelands](javascript:;)
	doc.Find("a[href='javascript:;']").Each(func(i int, s *goquery.Selection) {
		zoneName := strings.TrimSpace(s.Text())
		if zoneName != "" && data.ZoneName == "" {
			// Filter out non-zone links
			if !strings.Contains(zoneName, "Wowhead") && !strings.Contains(zoneName, "Monster") {
				data.ZoneName = zoneName
				fmt.Printf("[DEBUG] TurtleCraft: Found zone name: %s\n", zoneName)
			}
		}
	})

	// Try to find zone from h2/h3 headers or specific sections
	if data.ZoneName == "" {
		doc.Find("h1, h2, h3").Each(func(i int, s *goquery.Selection) {
			// Look for zone patterns after NPC name
			text := strings.TrimSpace(s.Text())
			// TurtleCraft wraps zone in specific sections - capture if pattern matches known zones
			if strings.Contains(text, "Plaguelands") || strings.Contains(text, "Forest") ||
				strings.Contains(text, "Valley") || strings.Contains(text, "Mountains") ||
				strings.Contains(text, "Marsh") || strings.Contains(text, "Lands") {
				if data.ZoneName == "" {
					data.ZoneName = text
				}
			}
		})
	}

	// Parse model image from og:image meta tag
	doc.Find("meta[property='og:image']").Each(func(i int, s *goquery.Selection) {
		if content, exists := s.Attr("content"); exists {
			data.ModelImageURL = content
			fmt.Printf("[DEBUG] TurtleCraft: Found og:image: %s\n", content)
		}
	})

	// Parse map data. octowow (AoWoW) emits the zone via
	// `new Mapper({...zone:'12'})` in a <script> and the spawn coords via
	// `myMapper.update({zone: 12, coords: [[26.58,93.94,...]]})` in an onclick
	// attribute, so scan the full serialized document, not just <script> tags.
	var mapZoneID string
	if fullHTML, herr := doc.Html(); herr == nil {
		zoneRe := regexp.MustCompile(`(?:new Mapper|myMapper\.update)\(\s*\{[^}]*?\bzone:\s*'?(\d+)'?`)
		if m := zoneRe.FindStringSubmatch(fullHTML); len(m) > 1 {
			mapZoneID = m[1]
		}
		if data.X == 0 && data.Y == 0 {
			coordRe := regexp.MustCompile(`coords:\s*\[\[\s*(-?[\d.]+)\s*,\s*(-?[\d.]+)`)
			if m := coordRe.FindStringSubmatch(fullHTML); len(m) > 2 {
				data.X, _ = strconv.ParseFloat(m[1], 64)
				data.Y, _ = strconv.ParseFloat(m[2], 64)
			}
		}

		// Capture EVERY spawn point across EVERY zone. An NPC can spawn in
		// multiple zones (e.g. Hillsbrad + Arathi), each with several points;
		// the single X/Y above only describes the first one.
		for _, zb := range spawnZoneRe.FindAllStringSubmatch(fullHTML, -1) {
			zoneID, _ := strconv.Atoi(zb[1])
			for _, c := range spawnCoordRe.FindAllStringSubmatch(zb[2], -1) {
				x, ex := strconv.ParseFloat(c[1], 64)
				y, ey := strconv.ParseFloat(c[2], 64)
				if ex != nil || ey != nil || (x == 0 && y == 0) {
					continue
				}
				data.Spawns = append(data.Spawns, ScrapedSpawn{
					ZoneID:   zoneID,
					ZoneName: zoneNameByID[zoneID],
					X:        x,
					Y:        y,
				})
			}
		}
	}

	// Legacy turtlecraft map data (g_mapperData = {...};)
	doc.Find("script").Each(func(i int, s *goquery.Selection) {
		scriptContent := s.Text()
		if strings.Contains(scriptContent, "g_mapperData") {
			// Extract g_mapperData = {...};
			re := regexp.MustCompile(`g_mapperData\s*=\s*(\{.+?\});`)
			match := re.FindStringSubmatch(scriptContent)
			if len(match) > 1 {
				jsonStr := match[1]
				type MapperZoneData struct {
					UIMapName string      `json:"uiMapName"`
					Coords    [][]float64 `json:"coords"`
				}
				var mapperData map[string][]MapperZoneData
				if err := json.Unmarshal([]byte(jsonStr), &mapperData); err == nil {
					for zoneID, zones := range mapperData {
						for _, zone := range zones {
							// If we haven't set zone name or coords yet, use this
							if data.ZoneName == "" {
								data.ZoneName = zone.UIMapName
							}
							if len(zone.Coords) > 0 && data.X == 0 && data.Y == 0 {
								data.X = zone.Coords[0][0]
								data.Y = zone.Coords[0][1]
							}
							// Try to set map URL from zone ID
							if data.MapURL == "" && zoneID != "" {
								// TurtleCraft usually uses standard IDs, assume classic Zamimg map
								data.MapURL = "https://wow.zamimg.com/images/wow/classic/maps/enus/original/" + zoneID + ".jpg"
							}
							return
						}
					}
				}
			}
		}
	})

	// Try to find model viewer image
	doc.Find(".model-container img, .model-viewer img, img[src*='model']").Each(func(i int, s *goquery.Selection) {
		if src, exists := s.Attr("src"); exists && data.ModelImageURL == "" {
			data.ModelImageURL = src
		}
	})

	// Resolve the map URL / zone name. octowow only exposes the numeric zone
	// ID, so reverse-map it to a name when needed and build the URL from it.
	if data.MapURL == "" {
		zoneMapIDs := map[string]string{
			"Western Plaguelands":  "1422",
			"Eastern Plaguelands":  "1423",
			"Tirisfal Glades":      "85",
			"Silverpine Forest":    "130",
			"Hillsbrad Foothills":  "267",
			"Alterac Mountains":    "36",
			"Stranglethorn Vale":   "33",
			"Duskwood":             "10",
			"Westfall":             "40",
			"Elwynn Forest":        "12",
			"Redridge Mountains":   "44",
			"Burning Steppes":      "46",
			"Searing Gorge":        "51",
			"Badlands":             "3",
			"Swamp of Sorrows":     "8",
			"Blasted Lands":        "4",
			"Deadwind Pass":        "41",
			"Dun Morogh":           "1",
			"Loch Modan":           "38",
			"Wetlands":             "11",
			"Arathi Highlands":     "45",
			"The Hinterlands":      "47",
			"Durotar":              "14",
			"The Barrens":          "17",
			"Mulgore":              "215",
			"Stonetalon Mountains": "406",
			"Ashenvale":            "331",
			"Thousand Needles":     "400",
			"Desolace":             "405",
			"Feralas":              "357",
			"Dustwallow Marsh":     "15",
			"Tanaris":              "440",
			"Un'Goro Crater":       "490",
			"Silithus":             "1377",
			"Felwood":              "361",
			"Winterspring":         "618",
			"Moonglade":            "493",
			"Azshara":              "16",
			"Teldrassil":           "141",
			"Darkshore":            "148",
		}
		// octowow gives only the zone ID -> recover the name from the table
		if data.ZoneName == "" && mapZoneID != "" {
			for name, id := range zoneMapIDs {
				if id == mapZoneID {
					data.ZoneName = name
					break
				}
			}
		}
		// Fall back to the name->ID lookup when only the name is known
		if mapZoneID == "" && data.ZoneName != "" {
			mapZoneID = zoneMapIDs[data.ZoneName]
		}
		if mapZoneID != "" {
			data.MapURL = fmt.Sprintf("https://wow.zamimg.com/images/wow/classic/maps/enus/original/%s.jpg", mapZoneID)
		}
	}

	return data, nil
}
