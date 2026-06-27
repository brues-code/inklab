package parsers

import (
	"io"
	"regexp"
	"strconv"
)

// ObjectLootEntry is one item a gameobject contains, scraped from an octowow
// (aowow) object page's id:'contains' Listview.
type ObjectLootEntry struct {
	ItemID  int     `json:"itemId"`
	Percent float64 `json:"percent"`
}

// aowow embeds an object's loot on its page as:
//
//	new Listview({template:'item',id:'contains',...,data: [{...,percent:20,...,id:18806}, ...]})
//
// We anchor on id:'contains', then capture the data array up to its real
// terminator "]})" — entry values like group:'1 [100%]' contain a bare "]", so
// stopping at the first "]" would truncate mid-array. Then pull each entry's
// id + percent.
var (
	containsAnchorRe = regexp.MustCompile(`id:\s*'contains'`)
	lootDataRe       = regexp.MustCompile(`(?s)data:\s*\[(.*?)\]\}\)`)
	lootObjRe        = regexp.MustCompile(`\{[^{}]*\}`)
	lootIDRe         = regexp.MustCompile(`\bid:\s*(\d+)`)
	lootPercentRe    = regexp.MustCompile(`\bpercent:\s*(-?\d+(?:\.\d+)?)`)
)

// ParseObjectLoot extracts the items a gameobject contains from an octowow
// object page body. Returns nil (no error) when the page has no contains list.
func ParseObjectLoot(r io.Reader) ([]ObjectLootEntry, error) {
	body, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	html := string(body)

	anchor := containsAnchorRe.FindStringIndex(html)
	if anchor == nil {
		return nil, nil
	}
	m := lootDataRe.FindStringSubmatch(html[anchor[1]:])
	if m == nil {
		return nil, nil
	}

	var out []ObjectLootEntry
	seen := map[int]bool{}
	for _, obj := range lootObjRe.FindAllString(m[1], -1) {
		idm := lootIDRe.FindStringSubmatch(obj)
		if idm == nil {
			continue
		}
		id, err := strconv.Atoi(idm[1])
		if err != nil || id <= 0 || seen[id] {
			continue
		}
		seen[id] = true
		e := ObjectLootEntry{ItemID: id}
		if pm := lootPercentRe.FindStringSubmatch(obj); pm != nil {
			e.Percent, _ = strconv.ParseFloat(pm[1], 64)
		}
		out = append(out, e)
	}
	return out, nil
}
