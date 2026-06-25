package services

import "testing"

func TestResolveSpellDescription(t *testing.T) {
	refs := map[int]spellVars{
		6788: {durationMs: 15000},                          // Weakened Soul
		8026: {basePoints: [3]int{76}, dieSides: [3]int{1}}, // ref for division (m1 = 77)
	}
	cases := []struct {
		name string
		raw  string
		self spellVars
		want string
	}{
		{
			name: "blizzard periodic + duration",
			raw:  "Ice shards pelt the target area doing $o1 Frost damage over $d.",
			self: spellVars{basePoints: [3]int{24}, dieSides: [3]int{1}, amplitude: [3]int{1000}, durationMs: 8000},
			want: "Ice shards pelt the target area doing 200 Frost damage over 8 sec.",
		},
		{
			name: "shield value + duration + ref",
			raw:  "absorbing $s1 damage.  Lasts $d.  Cannot be shielded again for $6788d.",
			self: spellVars{basePoints: [3]int{43}, dieSides: [3]int{1}, durationMs: 30000},
			want: "absorbing 44 damage.  Lasts 30 sec.  Cannot be shielded again for 15 sec.",
		},
		{
			name: "minutes duration",
			raw:  "Stuns target for $d.",
			self: spellVars{durationMs: 60000},
			want: "Stuns target for 1 min.",
		},
		{
			name: "range value",
			raw:  "Hits for $s1 to enemies.",
			self: spellVars{basePoints: [3]int{9}, dieSides: [3]int{5}}, // 10..14
			want: "Hits for 10 to 14 to enemies.",
		},
		{
			name: "division",
			raw:  "Restores $/1000;s1 power.",
			self: spellVars{basePoints: [3]int{4999}, dieSides: [3]int{1}}, // 5000 -> /1000 = 5
			want: "Restores 5 power.",
		},
		{
			name: "plural by preceding number",
			raw:  "Lasts $d$lsecond:seconds;.",
			self: spellVars{durationMs: 3000}, // "3 sec" -> last number 3 -> plural
			want: "Lasts 3 secseconds.",
		},
		{
			name: "gender picks first",
			raw:  "Restores $ghis:her; mana.",
			self: spellVars{},
			want: "Restores his mana.",
		},
		{
			name: "charges / targets / level",
			raw:  "$n balls hit up to $i enemies of level $v.",
			self: spellVars{procCharges: 3, maxTargets: 4, maxTargetLevel: 40},
			want: "3 balls hit up to 4 enemies of level 40.",
		},
		{
			name: "uppercase token",
			raw:  "Reduces cast time by $/1000;S1 sec.",
			self: spellVars{basePoints: [3]int{499}, dieSides: [3]int{1}}, // 500/1000 = 0.5
			want: "Reduces cast time by 0.5 sec.",
		},
		{
			name: "division of referenced spell token",
			raw:  "Deals $/77;8026m1 extra damage.",
			self: spellVars{},
			want: "Deals 1 extra damage.", // ref 8026 m1 = 77; 77/77 = 1
		},
	}
	for _, c := range cases {
		got := ResolveSpellDescription(c.raw, c.self, refs)
		if got != c.want {
			t.Errorf("%s:\n  got  %q\n  want %q", c.name, got, c.want)
		}
	}
}
