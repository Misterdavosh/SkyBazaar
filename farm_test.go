package main

import (
	"net/http"
	"testing"
	"time"
)

func TestFarmEfficiency(t *testing.T) {
	c := &http.Client{Timeout: 20 * time.Second}
	b, err := fetchBazaar(c)
	if err != nil {
		t.Skipf("bazaar unreachable: %v", err)
	}
	mobs, err := fetchBestiary(c)
	if err != nil {
		t.Skipf("bestiary unreachable: %v", err)
	}
	if _, ok := mobs["Crypt Ghoul"]; !ok {
		t.Fatal("expected Crypt Ghoul in bestiary")
	}
	shards, err := fetchShards(c)
	if err != nil {
		t.Skipf("shards dataset unreachable: %v", err)
	}
	m := newModel()
	m.bestiary = mobs
	m.shardDB = shards
	// Crypt Ghoul-like mob: high cap, low bracket -> low difficulty
	if d := m.farmDifficulty("SHARD_NONEXISTENT"); d != 100 {
		t.Logf("fallback difficulty: %v (expect 100)", d)
	}
	p := b.Products["SHARD_TAURUS"]
	if p.ProductID == "" {
		t.Fatal("SHARD_TAURUS missing from bazaar")
	}
	eff := m.farmEfficiency(p)
	t.Logf("Taurus: difficulty=%.3f efficiency=%.1f", m.farmDifficulty("SHARD_TAURUS"), eff)
	if d := m.farmDifficulty("SHARD_TAURUS"); d != 100 {
		t.Logf("Taurus matched bestiary: difficulty=%.3f", d)
	} else {
		t.Log("Taurus did not match bestiary (fallback rarity difficulty)")
	}
	if eff <= 0 {
		t.Fatalf("expected positive efficiency, got %f", eff)
	}
}

func TestCrittersAreEasy(t *testing.T) {
	c := &http.Client{Timeout: 20 * time.Second}
	mobs, err := fetchBestiary(c)
	if err != nil {
		t.Skipf("bestiary unreachable: %v", err)
	}
	shards, err := fetchShards(c)
	if err != nil {
		t.Skipf("shards unreachable: %v", err)
	}
	m := newModel()
	m.bestiary = mobs
	m.shardDB = shards
	// any shard whose source is a CRITTERS-type mob must have difficulty 1
	critters := 0
	for id, s := range shards {
		if mob, ok := mobs[s.DisplayName]; ok && mob.Type == "CRITTERS" {
			critters++
			if d := m.farmDifficulty(id); d != 1 {
				t.Fatalf("%s is a critter but difficulty=%v", id, d)
			}
		}
	}
	if critters == 0 {
		t.Fatal("expected some critter-type shard sources")
	}
	t.Logf("critter-typed shard sources: %d", critters)
}
