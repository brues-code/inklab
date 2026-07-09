package services

import "testing"

func TestResolveSpellDescription(t *testing.T) {
	refs := map[int]spellVars{
		6788:  {durationMs: 15000},                                           // Weakened Soul
		8026:  {basePoints: [3]int{76}, dieSides: [3]int{1}},                 // ref for division (m1 = 77)
		29203: {stackAmount: 3},                                              // Healing Way aura, stacks 3x
		51461: {basePoints: [3]int{-1}, realPointsPerLevel: [3]float64{1.0}}, // Wand Spec mana proc
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
		{
			name: "max stacks via referenced spell",
			raw:  "This effect stacks up to $29203u times.",
			self: spellVars{},
			want: "This effect stacks up to 3 times.",
		},
		{
			name: "max stacks on self",
			raw:  "Stacks up to $u times.",
			self: spellVars{stackAmount: 5},
			want: "Stacks up to 5 times.",
		},
		{
			// Smite R1: base 12, die 5, rppl 0.5, capped at maxLevel 6 → effLevel 5,
			// scaledBase 14.5 → floor 14 (+1=15) .. round 15 (+5=20). Matches in game.
			name: "level-scaled damage (Smite R1)",
			raw:  "Smite an enemy for $s1 Holy damage.",
			self: spellVars{basePoints: [3]int{12}, dieSides: [3]int{5}, realPointsPerLevel: [3]float64{0.5}, spellLevel: 1, baseLevel: 1, maxLevel: 6},
			want: "Smite an enemy for 15 to 20 Holy damage.",
		},
		{
			// Smite R2: base 24, die 7, rppl 0.6 (float32 → 3.0000001 at level 5).
			// round must absorb the float noise (→27), not ceil (→28). In game 28-34.
			name: "level-scaled, float32 rounding (Smite R2)",
			raw:  "Smite an enemy for $s1 Holy damage.",
			self: spellVars{basePoints: [3]int{24}, dieSides: [3]int{7}, realPointsPerLevel: [3]float64{0.6}, spellLevel: 1, baseLevel: 1, maxLevel: 6},
			want: "Smite an enemy for 28 to 34 Holy damage.",
		},
		{
			// Lesser Heal R1: base 45, die 11, rppl 0.9, maxLevel 3 → effLevel 2,
			// scaledBase 46.8 → 47 .. 58. In game 47-58.
			name: "level-scaled heal (Lesser Heal R1)",
			raw:  "Heal your target for $s1.",
			self: spellVars{basePoints: [3]int{45}, dieSides: [3]int{11}, realPointsPerLevel: [3]float64{0.9}, spellLevel: 1, baseLevel: 1, maxLevel: 3},
			want: "Heal your target for 47 to 58.",
		},
		{
			// Unscaled effect (rppl 0): must reduce to the original basePoints+1..+die.
			name: "unscaled unchanged",
			raw:  "Deals $s1 damage.",
			self: spellVars{basePoints: [3]int{9}, dieSides: [3]int{5}},
			want: "Deals 10 to 14 damage.",
		},
		{
			// Wand Specialization's mana proc (spell 51461): base -1, rppl 1.0,
			// uncapped → effLevel 60 → floor(-1+60)+1 = 60. Matches the in-game
			// tooltip at max (300) wand skill, since 300/5 == the level-60 anchor.
			name: "weapon-skill proc at max skill (spell 51461)",
			raw:  "Restores $51461m1 mana.",
			self: spellVars{},
			want: "Restores 60 mana.",
		},
	}
	for _, c := range cases {
		got := ResolveSpellDescription(c.raw, c.self, refs)
		if got != c.want {
			t.Errorf("%s:\n  got  %q\n  want %q", c.name, got, c.want)
		}
	}
}
