package parsers

import (
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
func ParseNpcDataTurtlecraft(r io.Reader) (*ScrapedNpcData, error) {
	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		return nil, err
	}

	data := &ScrapedNpcData{
		Infobox: make(map[string]string),
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
