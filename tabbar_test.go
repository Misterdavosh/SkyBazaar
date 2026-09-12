package main

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestTabBarHorizontal(t *testing.T) {
	m := newModel()
	out := ansi.Strip(m.tabBarView())
	if strings.Contains(out, "\n") {
		t.Fatalf("tab bar wrapped: %q", out)
	}
	t.Log(out)
}
