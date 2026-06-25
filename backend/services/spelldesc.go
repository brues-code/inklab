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
	basePoints  [3]int
	dieSides    [3]int
	amplitude   [3]int     // ms
	chainTarget [3]int
	radiusYd    [3]float64 // pre-resolved from spell_radius
	durationMs  int        // pre-resolved from spell_durations
	procChance  int
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
// in arithmetic ($/N;). Returns ok=false for tokens that aren't plain numbers.
func (v spellVars) numToken(letter byte, idx int) (float64, bool) {
	switch letter {
	case 's':
		return float64(absInt(v.effMax(idx))), true
	case 'm':
		return float64(absInt(v.effMin(idx))), true
	case 'M':
		return float64(absInt(v.effMax(idx))), true
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
	case 'd':
		return float64(v.durationMs) / 1000.0, true
	}
	return 0, false
}

// displayToken returns the human-readable substitution for a token.
func (v spellVars) displayToken(letter byte, idx int) (string, bool) {
	switch letter {
	case 'd':
		return formatDuration(v.durationMs), true
	case 's':
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
	reRef     = regexp.MustCompile(`\$(\d+)([a-zA-Z])(\d?)`)        // $6788d, $1234s2
	reDiv     = regexp.MustCompile(`\$/(\d+);([a-zA-Z])(\d?)`)      // $/1000;s1
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

	// 2. Division: $/N;TOK -> (TOK value)/N.
	out = reDiv.ReplaceAllStringFunc(out, func(m string) string {
		g := reDiv.FindStringSubmatch(m)
		div, _ := strconv.Atoi(g[1])
		idx := 1
		if g[3] != "" {
			idx, _ = strconv.Atoi(g[3])
		}
		if n, ok := self.numToken(g[2][0], idx); ok && div != 0 {
			return formatNum(n / float64(div))
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
