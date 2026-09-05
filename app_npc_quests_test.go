package main

import (
	"os"
	"testing"

	"inklab/backend/database"
	"inklab/backend/services"
)

// TestNpcObjectiveOfQuests checks the "objective of" relation against the real
// dev database (skipped when data/inklab.db isn't present, e.g. in CI).
//
// Dark Iron Saboteur (1052) is one of four creatures "The Dark Iron War" (303)
// asks you to kill, and octowow lists exactly that quest under the NPC's
// objective-of tab. The relation has no table of its own — it lives in the
// quest's ReqCreatureOrGOId columns — so this guards the derivation, not a join.
func TestNpcObjectiveOfQuests(t *testing.T) {
	if _, err := os.Stat("data/inklab.db"); err != nil {
		t.Skip("data/inklab.db not present")
	}
	db, err := database.NewSQLiteDB("data/inklab.db")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	svc := services.NewNpcService(db.DB(), nil, nil,
		database.NewItemRepository(db), database.NewCreatureRepository(db), "data")
	details, err := svc.GetNpcDetails(1052)
	if err != nil {
		t.Fatalf("GetNpcDetails(1052): %v", err)
	}

	var objectives []services.NpcQuest
	for _, q := range details.Quests {
		if q.Type == "objective" {
			objectives = append(objectives, q)
		}
	}
	if len(objectives) == 0 {
		t.Fatalf("no objective-of quests for 1052; got %+v", details.Quests)
	}
	found := false
	for _, q := range objectives {
		if q.QuestID == 303 {
			found = true
			if q.Title == "" {
				t.Errorf("quest 303 has no title: %+v", q)
			}
		}
	}
	if !found {
		t.Errorf("quest 303 (The Dark Iron War) missing from %+v", objectives)
	}

	// A creature nothing asks you to kill must not pick one up: the columns hold
	// gameobject objectives as negative ids, and a stray sign error would match
	// every creature against them.
	quartermaster, err := svc.GetNpcDetails(1461) // Quartermaster Hurthkul, a vendor
	if err == nil {
		for _, q := range quartermaster.Quests {
			if q.Type == "objective" {
				t.Errorf("vendor 1461 reported as a quest objective: %+v", q)
			}
		}
	}
}
