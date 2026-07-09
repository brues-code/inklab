package services

// Spell descriptions are resolved entirely locally from the DBC fields already
// in the database (spell_template + spell_durations/spell_radius) — see
// spelldesc.go. There is no spell web scraping: the data is identical to what
// the site served, and the placeholders ($s1/$d/$o1/...) resolve from our own
// columns.

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// SyncSpellResult represents the result of resolving a single spell.
type SyncSpellResult struct {
	Success     bool   `json:"success"`
	SpellID     int    `json:"spellId"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	Error       string `json:"error,omitempty"`
}

const spellVarCols = `durationIndex,
	effectBasePoints1, effectBasePoints2, effectBasePoints3,
	effectDieSides1, effectDieSides2, effectDieSides3,
	effectAmplitude1, effectAmplitude2, effectAmplitude3,
	effectChainTarget1, effectChainTarget2, effectChainTarget3,
	effectRadiusIndex1, effectRadiusIndex2, effectRadiusIndex3,
	procChance, procCharges, maxAffectedTargets, maxTargetLevel, rangeIndex, stackAmount`

func (s *SyncService) loadDurationMap() map[int]int {
	m := map[int]int{}
	rows, err := s.db.Query("SELECT id, duration_base FROM spell_durations")
	if err != nil {
		return m
	}
	defer rows.Close()
	for rows.Next() {
		var id, base int
		if rows.Scan(&id, &base) == nil {
			m[id] = base
		}
	}
	return m
}

func (s *SyncService) loadRadiusMap() map[int]float64 {
	m := map[int]float64{}
	rows, err := s.db.Query("SELECT id, radius_base FROM spell_radius")
	if err != nil {
		return m
	}
	defer rows.Close()
	for rows.Next() {
		var id int
		var base float64
		if rows.Scan(&id, &base) == nil {
			m[id] = base
		}
	}
	return m
}

func (s *SyncService) loadRangeMap() map[int]float64 {
	m := map[int]float64{}
	rows, err := s.db.Query("SELECT id, range_max FROM spell_range")
	if err != nil {
		return m
	}
	defer rows.Close()
	for rows.Next() {
		var id int
		var rmax float64
		if rows.Scan(&id, &rmax) == nil {
			m[id] = rmax
		}
	}
	return m
}

// scanSpellVars reads the spellVarCols (in order) plus a leading entry id into a
// spellVars, resolving the duration/radius indices via the aux maps.
func scanSpellVars(scan func(...interface{}) error, durMap map[int]int, radMap, rangeMap map[int]float64) (int, spellVars, error) {
	var entry, durIdx, proc, charges, maxTgts, maxTgtLvl, rangeIdx, stackAmt int
	var bp, die, amp, chain, radIdx [3]int
	err := scan(&entry, &durIdx,
		&bp[0], &bp[1], &bp[2],
		&die[0], &die[1], &die[2],
		&amp[0], &amp[1], &amp[2],
		&chain[0], &chain[1], &chain[2],
		&radIdx[0], &radIdx[1], &radIdx[2],
		&proc, &charges, &maxTgts, &maxTgtLvl, &rangeIdx, &stackAmt)
	if err != nil {
		return 0, spellVars{}, err
	}
	v := spellVars{basePoints: bp, dieSides: die, amplitude: amp, chainTarget: chain,
		procChance: proc, procCharges: charges, maxTargets: maxTgts, maxTargetLevel: maxTgtLvl,
		stackAmount: stackAmt}
	v.durationMs = durMap[durIdx]
	v.rangeYd = rangeMap[rangeIdx]
	for i := 0; i < 3; i++ {
		v.radiusYd[i] = radMap[radIdx[i]]
	}
	return entry, v, nil
}

// loadAllSpellVars loads every spell's resolver fields into a map (for the batch
// pass, where any spell may be referenced by any other via $<id>TOK).
func (s *SyncService) loadAllSpellVars(durMap map[int]int, radMap, rangeMap map[int]float64) map[int]spellVars {
	out := map[int]spellVars{}
	rows, err := s.db.Query("SELECT entry, " + spellVarCols + " FROM spell_template")
	if err != nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		entry, v, err := scanSpellVars(rows.Scan, durMap, radMap, rangeMap)
		if err == nil {
			out[entry] = v
		}
	}
	return out
}

// loadSpellVarsByIDs loads the resolver fields for a specific set of spell ids.
func (s *SyncService) loadSpellVarsByIDs(ids []int, durMap map[int]int, radMap, rangeMap map[int]float64) map[int]spellVars {
	out := map[int]spellVars{}
	for _, id := range ids {
		row := s.db.QueryRow("SELECT entry, "+spellVarCols+" FROM spell_template WHERE entry = ?", id)
		entry, v, err := scanSpellVars(row.Scan, durMap, radMap, rangeMap)
		if err == nil {
			out[entry] = v
		}
	}
	return out
}

// reRefIDs captures spell ids referenced by $<id>TOK and by division targets
// like $/77;8026m1 (so the single-spell path loads them for resolution).
var reRefIDs = regexp.MustCompile(`\$(?:/\d+;)?(\d+)[a-zA-Z]`)

// refIDsIn returns the spell ids referenced by $<id>TOK tokens in a description.
func refIDsIn(desc string) []int {
	var ids []int
	for _, m := range reRefIDs.FindAllStringSubmatch(desc, -1) {
		if id, err := strconv.Atoi(m[1]); err == nil {
			ids = append(ids, id)
		}
	}
	return ids
}

// ResolveSpellByID resolves one spell's description in place from local DBC data.
// fallbackDesc (e.g. a resolved description scraped from an item page) is used
// only when the spell has no usable description of its own.
func (s *SyncService) ResolveSpellByID(spellID int, fallbackDesc string) *SyncSpellResult {
	if spellID == 0 {
		return &SyncSpellResult{Success: false, SpellID: spellID, Error: "invalid spell id"}
	}
	var name, desc string
	err := s.db.QueryRow("SELECT COALESCE(name,''), COALESCE(description,'') FROM spell_template WHERE entry = ?", spellID).Scan(&name, &desc)
	if err != nil {
		return &SyncSpellResult{Success: false, SpellID: spellID, Error: "spell not in database"}
	}

	// Back-fill an empty description from the item-page fallback if provided.
	if len(desc) < 5 && fallbackDesc != "" {
		desc = fallbackDesc
		s.db.Exec("UPDATE spell_template SET description = ? WHERE entry = ?", desc, spellID)
	}

	if strings.Contains(desc, "$") {
		durMap, radMap, rangeMap := s.loadDurationMap(), s.loadRadiusMap(), s.loadRangeMap()
		ids := append(refIDsIn(desc), spellID)
		vars := s.loadSpellVarsByIDs(ids, durMap, radMap, rangeMap)
		resolved := ResolveSpellDescription(desc, vars[spellID], vars)
		if resolved != desc {
			if _, err := s.db.Exec("UPDATE spell_template SET description = ? WHERE entry = ?", resolved, spellID); err == nil {
				desc = resolved
			}
		}
	}
	return &SyncSpellResult{Success: true, SpellID: spellID, Name: name, Description: desc}
}

// FullSyncSpells resolves the $-placeholder descriptions of every spell locally
// from DBC data. The delayMs/fixIcons/iconDir args are retained for API
// compatibility but unused — there is no network access. startFrom resumes from
// a spell id. Honors the stop flag.
func (s *SyncService) FullSyncSpells(delayMs int, fixIcons bool, iconDir string, startFrom int, progressCb ProgressCallback) *FullSyncResult {
	result := &FullSyncResult{Errors: []string{}, StartFromID: startFrom}

	durMap, radMap, rangeMap := s.loadDurationMap(), s.loadRadiusMap(), s.loadRangeMap()
	allVars := s.loadAllSpellVars(durMap, radMap, rangeMap)

	rows, err := s.db.Query(
		"SELECT entry, description FROM spell_template WHERE description LIKE '%$%' AND entry >= ? ORDER BY entry",
		startFrom)
	if err != nil {
		result.Message = fmt.Sprintf("Error querying spells: %v", err)
		result.Errors = append(result.Errors, err.Error())
		return result
	}
	type job struct {
		id   int
		desc string
	}
	var todo []job
	for rows.Next() {
		var j job
		if rows.Scan(&j.id, &j.desc) == nil {
			todo = append(todo, j)
		}
	}
	rows.Close()

	result.TotalItems = len(todo)
	fmt.Printf("[ResolveSpells] Resolving %d spell descriptions locally...\n", len(todo))

	for i, j := range todo {
		if s.IsStopped() {
			result.Message = "Resolve stopped by user"
			return result
		}
		resolved := ResolveSpellDescription(j.desc, allVars[j.id], allVars)
		if resolved != j.desc {
			if _, err := s.db.Exec("UPDATE spell_template SET description = ? WHERE entry = ?", resolved, j.id); err == nil {
				result.Updated++
			}
		}
		result.LastSyncedID = j.id
		if progressCb != nil {
			progressCb(i+1, len(todo), j.id, fmt.Sprintf("Spell %d", j.id))
		}
	}

	result.Message = fmt.Sprintf("Resolved %d spell descriptions locally", result.Updated)
	return result
}
