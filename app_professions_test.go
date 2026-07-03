package main

import (
	"os"
	"testing"

	"inklab/backend/database"
)

// TestGetProfessionRecipes exercises the professions API against the real dev
// database (skipped when data/inklab.db isn't present, e.g. in CI).
func TestGetProfessionRecipes(t *testing.T) {
	if _, err := os.Stat("data/inklab.db"); err != nil {
		t.Skip("data/inklab.db not present")
	}
	db, err := database.NewSQLiteDB("data/inklab.db")
	if err != nil {
		t.Fatal(err)
	}
	a := &App{db: db}

	profs := a.GetProfessions()
	if len(profs) == 0 {
		t.Skip("no professions imported (run a client data import first)")
	}
	t.Logf("%d professions", len(profs))

	// Mining: small, well-known line — Smelt Bronze thresholds are documented.
	recipes := a.GetProfessionRecipes(186)
	if len(recipes) == 0 {
		t.Fatal("no mining recipes")
	}
	for _, r := range recipes {
		t.Logf("[%d] %-26s learn=%-3d y=%-3d g=%-3d grey=%-3d trainer=%v teach=%v quest=%v crafts=%v reagents=%d",
			r.SpellID, r.Name, r.Learn, r.Yellow, r.Green, r.Grey, r.Trainer, r.TeachItem != nil, r.Quest, r.Crafts != nil, len(r.Reagents))
	}

	var bronze *ProfessionRecipe
	for i := range recipes {
		if recipes[i].SpellID == 2659 {
			bronze = &recipes[i]
		}
	}
	if bronze == nil {
		t.Fatal("Smelt Bronze (2659) missing")
	}
	if bronze.Yellow != 65 || bronze.Grey != 115 {
		t.Errorf("Smelt Bronze thresholds wrong: %+v", bronze)
	}
	if bronze.Crafts == nil || bronze.Crafts.Entry != 2841 {
		t.Errorf("Smelt Bronze should craft Bronze Bar (2841): %+v", bronze.Crafts)
	}
	if len(bronze.Reagents) == 0 {
		t.Errorf("Smelt Bronze has no reagents")
	}
	if !bronze.Trainer {
		t.Errorf("Smelt Bronze should be trainer-taught")
	}
}
