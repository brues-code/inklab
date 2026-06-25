package services

// Local spell-description resolver: turns raw Spell.dbc description templates
// (with $s1/$d/$o1/... placeholders) into readable text using the spell's own
// DBC fields, with no network access. This replaces the old web scrape — the
// values are already in spell_template + the spell_durations/spell_radius aux
// tables, so the substitution is fully local.
//
// Supported tokens (case-sensitive, optional trailing effect index 1-3):
//   $sN  effect value (or "min to max" range)   $mN/$MN  effect min / max
//   $oN  periodic total over the duration        $tN      tick interval (sec)
//   $aN  effect radius (yards)                    $xN      chain/jump targets
//   $h   proc chance (%)                          $d       duration ("8 sec")
//   $<id>TOK   the same token resolved against another spell id
//   $/N;TOK    TOK divided by N
//   $lsing:plur;  plural form chosen by the preceding number
//   $gmale:female; gender form (we pick the first)
// Unknown tokens are left untouched rather than mangled.

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
)

// spellVars holds the per-spell DBC fields needed to resolve description tokens.
// durationMs and radiusYd are pre-resolved from the aux tables.
type spellVars struct {
	basePoints     [3]int
	dieSides       [3]int
	amplitude      [3]int     // ms
	chainTarget    [3]int
	radiusYd       [3]float64 // pre-resolved from spell_radius
	durationMs     int        // pre-resolved from spell_durations
	rangeYd        float64    // pre-resolved from spell_range (range_max)
	procChance     int
	procCharges    int
	maxTargets     int
	maxTargetLevel int
}

// lowerByte folds an ASCII letter to lowercase (tokens are case-insensitive,
// except $m=min vs $M=max which the caller distinguishes before folding).
func lowerByte(b byte) byte {
	if b >= 'A' && b <= 'Z' {
		return b + 32
	}
	return b
}

func clampIdx(idx int) int {
	if idx < 1 || idx > 3 {
		return 1
	}
	return idx
}

// effMin/effMax are the effect's value range. EffectBasePoints is stored as
// (value - 1) in the DBC, so the value is BasePoints+1 .. BasePoints+DieSides.
func (v spellVars) effMin(idx int) int { i := clampIdx(idx) - 1; return v.basePoints[i] + 1 }
func (v spellVars) effMax(idx int) int {
	i := clampIdx(idx) - 1
	d := v.dieSides[i]
	if d < 1 {
		d = 1
	}
	return v.basePoints[i] + d
}

func absInt(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// numToken returns the numeric value of a token against the given vars, for use
// in arithmetic ($/N;). Tokens are case-insensitive except $M (max) vs $m (min).
// Returns ok=false for tokens that aren't plain numbers.
func (v spellVars) numToken(letter byte, idx int) (float64, bool) {
	if letter == 'M' { // explicit max
		return float64(absInt(v.effMax(idx))), true
	}
	switch lowerByte(letter) {
	case 's', 'q': // effect value (q: e.g. Resurrection restored mana)
		return float64(absInt(v.effMax(idx))), true
	case 'm': // min
		return float64(absInt(v.effMin(idx))), true
	case 'o':
		val := absInt(v.effMax(idx))
		i := clampIdx(idx) - 1
		if v.amplitude[i] > 0 && v.durationMs > 0 {
			return float64(val * (v.durationMs / v.amplitude[i])), true
		}
		return float64(val), true
	case 't':
		return float64(v.amplitude[clampIdx(idx)-1]) / 1000.0, true
	case 'a':
		return v.radiusYd[clampIdx(idx)-1], true
	case 'x':
		return float64(v.chainTarget[clampIdx(idx)-1]), true
	case 'h':
		return float64(v.procChance), true
	case 'n': // number of charges (e.g. Lightning Shield "$n balls")
		return float64(v.procCharges), true
	case 'i': // max affected targets (e.g. Whirlwind "$i enemies")
		return float64(v.maxTargets), true
	case 'v': // max target level (e.g. Mind Soothe "level $v")
		return float64(v.maxTargetLevel), true
	case 'r': // spell range in yards (e.g. totem "$<id>r1 yards")
		return v.rangeYd, true
	case 'd':
		return float64(v.durationMs) / 1000.0, true
	}
	return 0, false
}

// displayToken returns the human-readable substitution for a token.
func (v spellVars) displayToken(letter byte, idx int) (string, bool) {
	switch lowerByte(letter) {
	case 'd':
		return formatDuration(v.durationMs), true
	case 'z': // home/bind location — not a DBC number
		return "your home", true
	case 's', 'q': // value, shown as a range when the dice spread it
		mn, mx := absInt(v.effMin(idx)), absInt(v.effMax(idx))
		if mn > mx {
			mn, mx = mx, mn
		}
		if mn == mx {
			return strconv.Itoa(mn), true
		}
		return fmt.Sprintf("%d to %d", mn, mx), true
	}
	if n, ok := v.numToken(letter, idx); ok {
		return formatNum(n), true
	}
	return "", false
}

func formatNum(f float64) string {
	if f == math.Trunc(f) {
		return strconv.FormatInt(int64(f), 10)
	}
	return strconv.FormatFloat(f, 'f', -1, 64)
}

// formatDuration renders milliseconds as WoW shows it: whole minutes as "N min",
// otherwise seconds ("8 sec", "1.5 sec").
func formatDuration(ms int) string {
	if ms <= 0 {
		return "0 sec"
	}
	if ms >= 60000 && ms%60000 == 0 {
		return fmt.Sprintf("%d min", ms/60000)
	}
	if ms%1000 == 0 {
		return fmt.Sprintf("%d sec", ms/1000)
	}
	return fmt.Sprintf("%s sec", formatNum(float64(ms)/1000.0))
}

var (
	reRef     = regexp.MustCompile(`\$(\d+)([a-zA-Z])(\d?)`)         // $6788d, $1234s2
	reDiv     = regexp.MustCompile(`\$/(\d+);(\d*)([a-zA-Z])(\d?)`)  // $/1000;s1, $/77;8026m1
	reMul     = regexp.MustCompile(`\$\*(\d+);(\d*)([a-zA-Z])(\d?)`) // $*15;s1, $*100;F1
	reToken   = regexp.MustCompile(`\$([a-zA-Z])(\d?)`)             // $s1, $d, $o1
	reCond    = regexp.MustCompile(`\$([lg])([^:;]*):([^;]*);`)     // $lsing:plur; $gm:f;
	reLastNum = regexp.MustCompile(`(\d+)(?:\.\d+)?\D*$`)           // trailing number
)

// ResolveSpellDescription substitutes the DBC placeholders in raw using self (the
// spell's own fields) and refs (entry -> vars) for cross-spell references.
func ResolveSpellDescription(raw string, self spellVars, refs map[int]spellVars) string {
	if !strings.Contains(raw, "$") {
		return raw
	}

	// 1. Cross-spell references: $<id>TOK.
	out := reRef.ReplaceAllStringFunc(raw, func(m string) string {
		g := reRef.FindStringSubmatch(m)
		id, _ := strconv.Atoi(g[1])
		rv, ok := refs[id]
		if !ok {
			return m
		}
		idx := 1
		if g[3] != "" {
			idx, _ = strconv.Atoi(g[3])
		}
		if s, ok := rv.displayToken(g[2][0], idx); ok {
			return s
		}
		return m
	})

	// 2. Division: $/N;TOK or $/N;<id>TOK -> (TOK value)/N. The token may target
	//    another spell (e.g. $/77;8026m1 = spell 8026's $m1 ÷ 77).
	out = reDiv.ReplaceAllStringFunc(out, func(m string) string {
		g := reDiv.FindStringSubmatch(m)
		div, _ := strconv.Atoi(g[1])
		if div == 0 {
			return m
		}
		vars := self
		if g[2] != "" { // referenced spell id
			id, _ := strconv.Atoi(g[2])
			rv, ok := refs[id]
			if !ok {
				return m
			}
			vars = rv
		}
		idx := 1
		if g[4] != "" {
			idx, _ = strconv.Atoi(g[4])
		}
		if n, ok := vars.numToken(g[3][0], idx); ok {
			return formatNum(n / float64(div))
		}
		return m
	})

	// 2b. Multiplication: $*N;TOK or $*N;<id>TOK -> (TOK value)*N.
	out = reMul.ReplaceAllStringFunc(out, func(m string) string {
		g := reMul.FindStringSubmatch(m)
		mul, _ := strconv.Atoi(g[1])
		vars := self
		if g[2] != "" {
			id, _ := strconv.Atoi(g[2])
			rv, ok := refs[id]
			if !ok {
				return m
			}
			vars = rv
		}
		idx := 1
		if g[4] != "" {
			idx, _ = strconv.Atoi(g[4])
		}
		if n, ok := vars.numToken(g[3][0], idx); ok {
			return formatNum(n * float64(mul))
		}
		return m
	})

	// 3. Plain self tokens: $sN, $d, $o1, ... (skip l/g — handled as conditionals).
	out = reToken.ReplaceAllStringFunc(out, func(m string) string {
		g := reToken.FindStringSubmatch(m)
		letter := g[1][0]
		if letter == 'l' || letter == 'g' {
			return m
		}
		idx := 1
		if g[2] != "" {
			idx, _ = strconv.Atoi(g[2])
		}
		if s, ok := self.displayToken(letter, idx); ok {
			return s
		}
		return m // leave unknown tokens untouched
	})

	// 4. Conditionals: $gA:B; (pick A) and $lA:B; (plural by the preceding number).
	out = reCond.ReplaceAllStringFunc(out, func(m string) string {
		g := reCond.FindStringSubmatch(m)
		if g[1] == "g" {
			return g[2] // gender: take the first (male) form
		}
		// plural: singular when the number immediately before this token is 1.
		prefix := out[:strings.Index(out, m)]
		singular := false
		if nm := reLastNum.FindStringSubmatch(prefix); nm != nil {
			singular = nm[1] == "1"
		}
		if singular {
			return g[2]
		}
		return g[3]
	})

	return out
}
