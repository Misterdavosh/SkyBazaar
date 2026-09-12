package main

import (
	"net/http"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func TestTabSwitch(t *testing.T) {
	m := newModel()
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	mm := m2.(model)
	if mm.tab != tabShards {
		t.Fatalf("expected shards tab, got %d", mm.tab)
	}
	m3, _ := mm.Update(tea.KeyMsg{Type: tea.KeyTab})
	if m3.(model).tab != tabBazaar {
		t.Fatalf("expected wrap to bazaar, got %d", m3.(model).tab)
	}
}

func TestShardRarity(t *testing.T) {
	c := &http.Client{Timeout: 15 * time.Second}
	shards, err := fetchShards(c)
	if err != nil {
		t.Skipf("shards dataset unreachable: %v", err)
	}
	if s, ok := shards["SHARD_DIVE_GHAST"]; !ok || s.Rarity == "" {
		t.Fatalf("expected rarity for SHARD_DIVE_GHAST, got %+v", s)
	}
}
