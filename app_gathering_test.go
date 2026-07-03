package main

import (
	"os"
	"testing"

	"inklab/backend/database"
)

// TestGatheringNodes exercises the gathering-map API against the real dev
// database (skipped when data/inklab.db isn't present, e.g. in CI).
func TestGatheringNodes(t *testing.T) {
	if _, err := os.Stat("data/inklab.db"); err != nil {
		t.Skip("data/inklab.db not present")
	}
	db, err := database.NewSQLiteDB("data/inklab.db")
	if err != nil {
		t.Fatal(err)
	}
	a := &App{db: db}

	profs := a.GetGatheringProfessions()
	if len(profs) == 0 {
		t.Skip("no gathering data (run a client data import first)")
	}
	for _, p := range profs {
		t.Logf("%-14s id=%-3d nodes=%-3d spawns=%d", p.Name, p.ID, p.Nodes, p.Spawns)
	}

	// Mining (lock type 3): Copper Vein must exist with plenty of spawns.
	nodes := a.GetGatheringNodes(3)
	if len(nodes) == 0 {
		t.Fatal("no mining nodes")
	}
	var copper, richThorium *GatheringNode
	for i := range nodes {
		if nodes[i].Name == "Copper Vein" {
			copper = &nodes[i]
		}
		if nodes[i].Name == "Rich Thorium Vein" {
			richThorium = &nodes[i]
		}
	}
	if copper == nil || len(copper.Spawns) < 100 {
		t.Fatalf("Copper Vein missing or sparse: %+v", copper)
	}
	if richThorium == nil || richThorium.ReqSkill != 275 {
		t.Errorf("Rich Thorium Vein should require 275: %+v", richThorium)
	}
	zones := map[string]bool{}
	for _, s := range copper.Spawns {
		zones[s.Zone] = true
	}
	t.Logf("Copper Vein: %d spawns across %d zones (req %d)", len(copper.Spawns), len(zones), copper.ReqSkill)
	if !zones["Elwynn"] && !zones["Elwynn Forrest"] && !zones["Durotar"] {
		t.Errorf("Copper Vein spawns missing expected starter zones: %v", zones)
	}
}
