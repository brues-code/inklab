package parsers

import (
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

// Live integration tests against octowow.st (the AoWoW database the sync
// service scrapes). These fetch real pages on every run so they catch
// page-structure changes that would silently break syncing — title suffix,
// mapper/coords format, infobox element, missing heading-size-1 class,
// relative model image paths, etc.
//
// A network/transport failure skips (octowow unreachable shouldn't fail an
// unrelated run); a non-200 status or a field mismatch is a real failure.
// Run just these with:  go test -run Octowow -v ./backend/parsers/

const octowowBase = "https://octowow.st/db"

var octowowClient = &http.Client{Timeout: 20 * time.Second}

func liveGet(t *testing.T, url string) string {
	t.Helper()
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0")
	resp, err := octowowClient.Do(req)
	if err != nil {
		t.Skipf("octowow unreachable (%s): %v", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET %s: status %d", url, resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read %s: %v", url, err)
	}
	return string(body)
}

func TestOctowowItemLive(t *testing.T) {
	content := liveGet(t, octowowBase+"/?item=19019") // Thunderfury

	ok, name := ParseItemTitle(content)
	if !ok || name != "Thunderfury, Blessed Blade of the Windseeker" {
		t.Fatalf("ParseItemTitle = (%v, %q); want (true, Thunderfury...)", ok, name)
	}
	item, _, err := ParseItem(content, 19019)
	if err != nil {
		t.Fatalf("ParseItem error: %v", err)
	}
	if item.Name != "Thunderfury, Blessed Blade of the Windseeker" {
		t.Errorf("Name = %q", item.Name)
	}
	if item.Quality != 5 {
		t.Errorf("Quality = %d; want 5 (Legendary)", item.Quality)
	}
	if item.DisplayId != 30606 {
		t.Errorf("DisplayId = %d; want 30606", item.DisplayId)
	}
}

func TestOctowowSpellLive(t *testing.T) {
	content := liveGet(t, octowowBase+"/?spell=133") // Fireball
	name, desc := ParseSpell(content)
	if name != "Fireball" {
		t.Errorf("spell name = %q; want Fireball", name)
	}
	if len(desc) == 0 {
		t.Errorf("spell description is empty")
	}
}

func TestOctowowNpcLive(t *testing.T) {
	content := liveGet(t, octowowBase+"/?npc=448") // Hogger
	data, err := ParseNpcDataTurtlecraft(strings.NewReader(content))
	if err != nil {
		t.Fatalf("ParseNpcDataTurtlecraft error: %v", err)
	}

	// Coordinates from the myMapper.update({coords: [[x,y,...]]}) onclick.
	if data.X != 26.58 || data.Y != 93.94 {
		t.Errorf("coords = (%.2f, %.2f); want (26.58, 93.94)", data.X, data.Y)
	}
	if data.ZoneName == "" {
		t.Errorf("ZoneName is empty")
	}
	// Map URL derived from zone id 12 (Elwynn).
	if !strings.Contains(data.MapURL, "/12.jpg") {
		t.Errorf("MapURL = %q; want it to reference zone 12", data.MapURL)
	}
	// Model image is a relative path here; the scraper resolves it to absolute.
	if !strings.Contains(data.ModelImageURL, "models/384.png") {
		t.Errorf("ModelImageURL = %q; want models/384.png", data.ModelImageURL)
	}
	// Infobox key/values still parse from the <table class=infobox> <li>s.
	if got := data.Infobox["Display ID"]; got != "384" {
		t.Errorf("Infobox[Display ID] = %q; want 384", got)
	}
	if got := data.Infobox["Level"]; got != "11" {
		t.Errorf("Infobox[Level] = %q; want 11", got)
	}
}

func TestOctowowQuestLive(t *testing.T) {
	content := liveGet(t, octowowBase+"/?quest=83") // Red Linen Goods
	data, err := ParseQuestDataTurtlecraft(strings.NewReader(content), 83)
	if err != nil {
		t.Fatalf("ParseQuestDataTurtlecraft error: %v", err)
	}
	if data.Title != "Red Linen Goods" {
		t.Errorf("Title = %q; want Red Linen Goods", data.Title)
	}
	if data.QuestLevel != 9 {
		t.Errorf("QuestLevel = %d; want 9", data.QuestLevel)
	}
	if data.MinLevel != 4 {
		t.Errorf("MinLevel = %d; want 4", data.MinLevel)
	}
	if !strings.Contains(data.Side, "Human") {
		t.Errorf("Side = %q; want it to contain Human", data.Side)
	}
	// Description text (bare text + <br> after the <h3>Description</h3> header).
	if !strings.Contains(data.Details, "Defias") {
		t.Errorf("Details = %q; want it to contain the quest description", data.Details)
	}
	if !strings.Contains(data.EndText, "bandanas") {
		t.Errorf("EndText = %q; want the completion text", data.EndText)
	}
}
