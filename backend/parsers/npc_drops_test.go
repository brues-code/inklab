package parsers

import "testing"

// Verbatim shape of octowow's NPC "drop" Listview, taken from Edwin VanCleef
// (npc=639): a plain drop, a stacking drop, and a quest drop at percent: -100.
// The neighbouring 'pick-pocketing' listview must not bleed into the result.
const vanCleefPage = `<html><body>
<script>
new Listview({template: 'item', id: 'drop', name: 'Drops', data: [
{name: '5Dalaran Sharp',description: '',level: 15,reqlevel: 5,classs: 0,subclass: 0,percent: 4,id: 414},
{name: '5Linen Cloth',description: '',level: 5,classs: 7,subclass: 0,stack:[1,5],percent: 13,id: 2589},
{name: '5Wool Cloth',description: '',level: 15,classs: 7,subclass: 0,stack:[1,4],percent: 17,id: 2592},
{name: '5An Unsent Letter',description: 'A letter.',level: 16,reqlevel: 16,classs: 12,subclass: 0,percent: 100,id: 2874},
{name: '5Head of VanCleef',description: '',level: 1,classs: 12,subclass: 0,percent: -100,id: 3637}
]})
new Listview({template: 'item', id: 'pick-pocketing', name: 'Pickpocketing', data: [
{name: '5Silver Coin',description: '',level: 1,classs: 12,subclass: 0,percent: 50,id: 99999}
]})
</script>
</body></html>`

func TestParseNpcDrops(t *testing.T) {
	drops := ParseNpcDrops(vanCleefPage)
	if len(drops) != 5 {
		t.Fatalf("parsed %d drops, want 5: %+v", len(drops), drops)
	}

	byID := map[int]ItemDrop{}
	for _, d := range drops {
		byID[d.ItemEntry] = d
	}

	// Plain drop: chance kept as a percentage, stack defaults to 1-1.
	if got := byID[414]; got.Chance != 4 || got.MinCount != 1 || got.MaxCount != 1 {
		t.Errorf("Dalaran Sharp = %+v; want chance 4, stack 1-1", got)
	}
	// stack:[1,5] becomes the min/max count.
	if got := byID[2589]; got.Chance != 13 || got.MinCount != 1 || got.MaxCount != 5 {
		t.Errorf("Linen Cloth = %+v; want chance 13, stack 1-5", got)
	}
	// A quest drop keeps its negative chance — the world-DB convention the loot
	// column already uses. Dropping the sign would turn it into a 100% drop.
	if got := byID[3637]; got.Chance != -100 {
		t.Errorf("Head of VanCleef chance = %v; want -100", got.Chance)
	}
	// The pickpocketing listview is a different table.
	if _, ok := byID[99999]; ok {
		t.Errorf("pick-pocketing item leaked into drops: %+v", drops)
	}
}

// Fractional percentages must survive intact — they are the whole point of
// scraping drops rather than inferring them (the item-sync fallback writes 0).
func TestParseNpcDropsKeepsPrecision(t *testing.T) {
	page := `new Listview({template: 'item', id: 'drop', data: [
{name: '5Tough Jerky',description: '',level: 5,classs: 0,subclass: 0,percent: 6.0935,id: 117}
]})`
	drops := ParseNpcDrops(page)
	if len(drops) != 1 || drops[0].Chance != 6.0935 {
		t.Fatalf("got %+v; want one drop at 6.0935", drops)
	}
}

func TestParseNpcDropsNoBlock(t *testing.T) {
	if got := ParseNpcDrops(`<html><body>An NPC with no drop tab</body></html>`); got != nil {
		t.Fatalf("got %+v; want nil for a page with no drop listview", got)
	}
	// An interstitial or truncated page must yield nothing rather than a
	// partial table — the write path treats empty as "keep what we have".
	if got := ParseNpcDrops(`<title>Just a moment please...</title>`); got != nil {
		t.Fatalf("got %+v; want nil for a challenge page", got)
	}
}
