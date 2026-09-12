package main

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestLinkTargetToggle(t *testing.T) {
	m := newModel()
	if got := m.linkURL("ENCHANTED_BREAD"); !strings.Contains(got, "sky.coflnet.com/item/ENCHANTED_BREAD") {
		t.Fatalf("default link should be Coflnet, got %s", got)
	}
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})
	mm := m2.(model)
	if mm.link != linkWiki {
		t.Fatalf("o should switch to wiki, got %d", mm.link)
	}
	if got := mm.linkURL("ENCHANTED_BREAD"); !strings.HasPrefix(got, "https://hypixelskyblock.minecraft.wiki/w/") {
		t.Fatalf("wiki URL mismatch: %s", got)
	}
	// itemDB names are used for wiki links once loaded
	mm.itemDB = map[string]ItemResource{"ENCHANTED_BREAD": {Name: "Enchanted Bread"}}
	if got := mm.linkURL("ENCHANTED_BREAD"); got != "https://hypixelskyblock.minecraft.wiki/w/Enchanted_Bread" {
		t.Fatalf("wiki URL should use display name, got %s", got)
	}
}
