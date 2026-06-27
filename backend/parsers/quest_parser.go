package parsers

import (
	"html"
	"io"
	"regexp"
	"strconv"
	"strings"

	"inklab/backend/database/models"

	"github.com/PuerkitoBio/goquery"
)

type ScrapedQuestData struct {
	Entry           int
	Title           string
	QuestLevel      int
	MinLevel        int
	Details         string
	Objectives      string
	OfferRewardText string
	EndText         string
	ZoneOrSort      int
	RewXP           int    // base reward XP (infobox RewXP, not the level-adjusted "Gains")
	Side            string // "Alliance", "Horde", "Both"
	RaceMask        int    // RequiredRaces
	ClassMask       int    // RequiredClasses
	PrevQuestID     int
	NextQuestID     int
	Starters        []int            // start-NPC ids (creature_questrelation)
	Enders          []int            // end-NPC ids (creature_involvedrelation)
	RepRewards      []QuestRepReward // reputation gains (faction name + value)
}

// QuestRepReward is a reputation reward scraped from the "Gains" section. The
// faction is a link (faction=<id>), so we capture the id directly.
type QuestRepReward struct {
	FactionID int
	Value     int
}

var (
	questRefRe = map[string]*regexp.Regexp{
		"npc":   regexp.MustCompile(`npc=(\d+)`),
		"quest": regexp.MustCompile(`quest=(\d+)`),
	}
	// e.g. "75 Reputation with <a href="?faction=72">Stormwind</a>"
	questRepRe = regexp.MustCompile(`(-?\d+)\s+Reputation with\s*<a[^>]*faction=(\d+)`)
)

// firstRefID returns the first id of the given kind (npc/quest) linked inside a
// selection, or 0.
func firstRefID(s *goquery.Selection, kind string) int {
	id := 0
	s.Find("a").EachWithBreak(func(_ int, a *goquery.Selection) bool {
		href, _ := a.Attr("href")
		if m := questRefRe[kind].FindStringSubmatch(href); m != nil {
			id, _ = strconv.Atoi(m[1])
			return false
		}
		return true
	})
	return id
}

func ParseQuestDataTurtlecraft(r io.Reader, entry int) (*ScrapedQuestData, error) {
	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		return nil, err
	}

	data := &ScrapedQuestData{Entry: entry}

	// Title
	data.Title = strings.TrimSpace(doc.Find("h1").First().Text())
	// Remove " - Quests - Turtle WoW Database" suffix if present (handled usually by scraping title element)
	if idx := strings.Index(data.Title, " - Quests"); idx > 0 {
		data.Title = data.Title[:idx]
	}

	// Quick Facts (li elements). octowow exposes the server-side fields the WDB
	// quest cache lacks — Race/Class masks, Start/End NPCs — as labeled items.
	doc.Find(".infobox li").Each(func(i int, s *goquery.Selection) {
		text := strings.TrimSpace(s.Text())
		switch {
		case strings.HasPrefix(text, "Level: "):
			data.QuestLevel, _ = strconv.Atoi(strings.TrimPrefix(text, "Level: "))
		case strings.HasPrefix(text, "Requires level: "):
			data.MinLevel, _ = strconv.Atoi(strings.TrimPrefix(text, "Requires level: "))
		case strings.HasPrefix(text, "Side: "):
			data.Side = strings.TrimPrefix(text, "Side: ")
		case strings.HasPrefix(text, "ZoneOrSort: "):
			data.ZoneOrSort, _ = strconv.Atoi(strings.TrimPrefix(text, "ZoneOrSort: "))
		case strings.HasPrefix(text, "RewXP: "):
			data.RewXP, _ = strconv.Atoi(strings.TrimPrefix(text, "RewXP: "))
		case strings.HasPrefix(text, "Race Mask: "):
			data.RaceMask, _ = strconv.Atoi(strings.TrimPrefix(text, "Race Mask: "))
		case strings.HasPrefix(text, "Class Mask: "):
			data.ClassMask, _ = strconv.Atoi(strings.TrimPrefix(text, "Class Mask: "))
		}
		// Start/End NPCs are labeled li items that carry an npc link ("Start :
		// Name", "End : Name"). The npc link distinguishes them from "Start
		// Script: 0" etc.
		if npcID := firstRefID(s, "npc"); npcID > 0 {
			if strings.HasPrefix(text, "Start") {
				data.Starters = append(data.Starters, npcID)
			} else if strings.HasPrefix(text, "End") {
				data.Enders = append(data.Enders, npcID)
			}
		}
	})

	// Quest chain: the Series table lists the chain in order; the current quest is
	// bold (no link). Prev/Next are its neighbors.
	var chain []int
	currentIdx := -1
	doc.Find("table.series tr").Each(func(_ int, tr *goquery.Selection) {
		if qid := firstRefID(tr, "quest"); qid > 0 {
			chain = append(chain, qid)
		} else if tr.Find("b").Length() > 0 {
			chain = append(chain, entry)
			currentIdx = len(chain) - 1
		}
	})
	if currentIdx > 0 {
		data.PrevQuestID = chain[currentIdx-1]
	}
	if currentIdx >= 0 && currentIdx < len(chain)-1 {
		data.NextQuestID = chain[currentIdx+1]
	}

	// Parse text sections (Description / Progress / Completion). On octowow the
	// body is bare text with <br> tags after an <h3> header, running up to the
	// next header — so extract the HTML between headers instead of relying on a
	// single sibling element (which .Next() cannot reach across text nodes).
	if fullHTML, herr := doc.Html(); herr == nil {
		data.Details = extractQuestSection(fullHTML, "Description")
		data.OfferRewardText = extractQuestSection(fullHTML, "Progress")
		data.EndText = extractQuestSection(fullHTML, "Completion")

		// Reputation rewards live in the "Gains" list as
		// "N Reputation with <a href=?faction=ID>Name</a>".
		for _, m := range questRepRe.FindAllStringSubmatch(fullHTML, -1) {
			val, _ := strconv.Atoi(m[1])
			fid, _ := strconv.Atoi(m[2])
			if val != 0 && fid != 0 {
				data.RepRewards = append(data.RepRewards, QuestRepReward{FactionID: fid, Value: val})
			}
		}
	}

	return data, nil
}

var (
	questSectionRe = map[string]*regexp.Regexp{}
	brTagRe        = regexp.MustCompile(`(?i)<br\s*/?>`)
	anyTagRe       = regexp.MustCompile(`<[^>]+>`)
	horizSpaceRe   = regexp.MustCompile(`[ \t]+`)
	spacedNewateRe = regexp.MustCompile(` *\n *`)
)

// extractQuestSection returns the cleaned text that follows an <h3>label</h3>
// header up to the next header (or closing container) in the given HTML.
func extractQuestSection(content, label string) string {
	re, ok := questSectionRe[label]
	if !ok {
		re = regexp.MustCompile(`(?is)<h3[^>]*>\s*` + regexp.QuoteMeta(label) + `\s*</h3>(.*?)(?:<h[1-4][\s>]|</div>)`)
		questSectionRe[label] = re
	}
	m := re.FindStringSubmatch(content)
	if len(m) < 2 {
		return ""
	}
	return cleanQuestText(m[1])
}

// cleanQuestText turns the inner HTML of a quest text section into plain text:
// <br> becomes a newline, other tags are stripped, entities are decoded, and
// the indentation whitespace from the source markup is collapsed away.
func cleanQuestText(s string) string {
	s = brTagRe.ReplaceAllString(s, "\n")
	s = anyTagRe.ReplaceAllString(s, "")
	s = html.UnescapeString(s)
	s = horizSpaceRe.ReplaceAllString(s, " ")
	s = spacedNewateRe.ReplaceAllString(s, "\n")
	return strings.TrimSpace(s)
}

// ParseQuestTitle checks if content contains a valid quest title
func ParseQuestTitle(content string) (bool, string) {
	data, err := ParseQuestDataTurtlecraft(strings.NewReader(content), 0)
	if err != nil || data.Title == "" {
		return false, ""
	}
	return true, data.Title
}

// ParseQuest parses content into a QuestDetail model
func ParseQuest(content string, entry int) (*models.QuestDetail, error) {
	data, err := ParseQuestDataTurtlecraft(strings.NewReader(content), entry)
	if err != nil {
		return nil, err
	}

	// Map to QuestDetail
	q := &models.QuestDetail{
		Entry:           data.Entry,
		Title:           data.Title,
		Details:         data.Details,
		Objectives:      data.Objectives,
		OfferRewardText: data.OfferRewardText,
		EndText:         data.EndText,
		QuestLevel:      data.QuestLevel,
		MinLevel:        data.MinLevel,
		ZoneOrSort:      data.ZoneOrSort,
	}
	return q, nil
}
