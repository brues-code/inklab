package parsers

import (
	"io"
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
	Side            string // "Alliance", "Horde", "Both"
	PrevQuestID     int
	NextQuestID     int
	Starters        []int // NPC IDs
	Enders          []int // NPC IDs
}

func ParseQuestDataTurtlecraft(r io.Reader, entry int) (*ScrapedQuestData, error) {
	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		return nil, err
	}

	data := &ScrapedQuestData{Entry: entry}

	// Title
	data.Title = strings.TrimSpace(doc.Find("h1.heading-size-1").First().Text())
	// Remove " - Quests - Turtle WoW Database" suffix if present (handled usually by scraping title element)
	if idx := strings.Index(data.Title, " - Quests"); idx > 0 {
		data.Title = data.Title[:idx]
	}

	// Quick Facts (li elements)
	doc.Find("div.infobox li").Each(func(i int, s *goquery.Selection) {
		text := strings.TrimSpace(s.Text())
		if strings.HasPrefix(text, "Level: ") {
			data.QuestLevel, _ = strconv.Atoi(strings.TrimPrefix(text, "Level: "))
		} else if strings.HasPrefix(text, "Requires level: ") {
			data.MinLevel, _ = strconv.Atoi(strings.TrimPrefix(text, "Requires level: "))
		} else if strings.HasPrefix(text, "Side: ") {
			data.Side = strings.TrimPrefix(text, "Side: ")
		} else if strings.HasPrefix(text, "ZoneOrSort: ") {
			data.ZoneOrSort, _ = strconv.Atoi(strings.TrimPrefix(text, "ZoneOrSort: "))
		}
	})

	// Starters/Enders (found in infobox links or specific sections)
	// Turtlecraft format: "Start: [NPC Name]"
	doc.Find("table.series-list").Each(func(i int, s *goquery.Selection) {
		// Series list often contains the chain
		// But first let's find Starters/Enders text
	})

	// Parse Text Sections
	// Description, Progress (Objectives?), Completion
	doc.Find("h3").Each(func(i int, s *goquery.Selection) {
		header := strings.TrimSpace(s.Text())
		content := strings.TrimSpace(s.Next().Text())

		if header == "Description" {
			data.Details = content
		} else if header == "Progress" {
			// This is usually OfferRewardText (when talking to NPC before completion)
			data.OfferRewardText = content
		} else if header == "Completion" {
			data.EndText = content
		}
	})

	// Objectives are often just before Description or in a summary
	// In TurtleCraft, Objectives text might be separate?
	// Based on read_url_content, it seems "Description" is the main text.
	// Objectives logic might need refinement. For now, map Description -> Details.

	// Parse Series / Chain
	// Look for lists containing quest links
	// This is tricky without seeing exact HTML structure for the chain.
	// But we saw links like `[Frix's Folly](...quest=55008)`

	// Let's assume links in a specific container or just parsing all quest links in the infobox area
	// For now, extraction of Prev/Next is hard without precise selectors.

	// Extract Starters/Enders from IDs in links if 'Start' / 'End' text is found
	// Searching in the whole body for "Start" followed by NPC link
	// Find IDs... simplified approach:

	doc.Find("a").Each(func(i int, s *goquery.Selection) {
		href, _ := s.Attr("href")
		if strings.Contains(href, "npc=") {
			// Check if previous text node says "Start" or "End"
			// This requires traversing nodes, goquery makes this slightly hard.
			// skipping accurate start/end scraping for now, rely on existing DB relations.
		}
	})

	return data, nil
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
