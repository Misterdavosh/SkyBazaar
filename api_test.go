package main

import (
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestFetchBazaar(t *testing.T) {
	c := &http.Client{Timeout: 15 * time.Second}
	data, err := fetchBazaar(c)
	if err != nil {
		t.Fatalf("fetch failed: %v", err)
	}
	if len(data.Products) < 100 {
		t.Fatalf("expected many products, got %d", len(data.Products))
	}
	items := Items(data, "enchanted bread", func(string) string { return "" })
	if len(items) == 0 {
		t.Fatal("no matches for 'enchanted bread'")
	}
	t.Logf("enchanted matches: %d, first: %s sell=%.1f buy=%.1f", len(items), items[0].ProductID, items[0].QuickStatus.SellPrice, items[0].QuickStatus.BuyPrice)
	_ = time.Now
}

func TestNormalizeAndSearch(t *testing.T) {
	c := &http.Client{Timeout: 15 * time.Second}
	data, err := fetchBazaar(c)
	if err != nil {
		t.Fatal(err)
	}
	if got := normalize("Enchanted Bread!!"); got != "enchantedbread" {
		t.Errorf("normalize = %q", got)
	}
	nameOf := func(id string) string { return strings.ReplaceAll(id, "_", " ") }
	items := Items(data, "enchanted bread", nameOf)
	if len(items) == 0 {
		t.Fatal("space query 'enchanted bread' found nothing")
	}
	for _, p := range items {
		if !strings.Contains(normalize(p.ProductID), "enchantedbread") {
			t.Fatalf("bad match: %s", p.ProductID)
		}
	}
}
