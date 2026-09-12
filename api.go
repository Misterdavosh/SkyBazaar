package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

const bazaarURL = "https://api.hypixel.net/v2/skyblock/bazaar"

type Order struct {
	Price      float64 `json:"pricePerUnit"`
	Volume     int     `json:"amount"`
	Orders     int     `json:"orders"`
	MovingWeek int     `json:"movingWeek"`
}

type Product struct {
	ProductID   string  `json:"product_id"`
	SellSummary []Order `json:"sell_summary"`
	BuySummary  []Order `json:"buy_summary"`
	QuickStatus struct {
		SellPrice  float64 `json:"sellPrice"`
		BuyPrice   float64 `json:"buyPrice"`
		SellVolume int     `json:"sellVolume"`
		BuyVolume  int     `json:"buyVolume"`
	} `json:"quick_status"`
}

type BazaarResponse struct {
	Success     bool               `json:"success"`
	LastUpdated int64              `json:"lastUpdated"`
	Products    map[string]Product `json:"products"`
}

func fetchBazaar(client *http.Client) (*BazaarResponse, error) {
	resp, err := client.Get(bazaarURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}
	var data BazaarResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}
	if !data.Success {
		return nil, fmt.Errorf("API reported failure")
	}
	return &data, nil
}

// normalize makes text comparable: lowercase, drop spaces/underscores/hyphens
// and other punctuation, so "enchanted bread" matches "ENCHANTED_BREAD".
func normalize(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		}
	}
	return b.String()
}

// Items returns a sorted slice of products filtered by the query. The query is
// matched against both the product id and the in-game item name (via nameOf).
func Items(data *BazaarResponse, query string, nameOf func(string) string) []Product {
	if data == nil {
		return nil
	}
	q := normalize(query)
	items := make([]Product, 0, len(data.Products))
	for _, p := range data.Products {
		if q != "" &&
			!strings.Contains(normalize(p.ProductID), q) &&
			!strings.Contains(normalize(nameOf(p.ProductID)), q) {
			continue
		}
		items = append(items, p)
	}
	return items
}

func prettyID(id string) string {
	return strings.Title(strings.ReplaceAll(strings.ReplaceAll(id, "_", " "), "-", " "))
}

func formatCoins(v float64) string {
	switch {
	case v >= 1_000_000:
		return fmt.Sprintf("%.2fM", v/1_000_000)
	case v >= 10_000:
		return fmt.Sprintf("%.1fk", v/1_000)
	default:
		return fmt.Sprintf("%.1f", v)
	}
}

func lastUpdatedString(ts int64) string {
	if ts == 0 {
		return "never"
	}
	return time.Unix(ts/1000, 0).Format("15:04:05")
}

const itemsURL = "https://api.hypixel.net/v2/resources/skyblock/items"

type ItemResource struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Tier string `json:"tier"`
}

type ItemsResponse struct {
	Success bool           `json:"success"`
	Items   []ItemResource `json:"items"`
}

// fetchItems returns a map of item id -> item resource (name + rarity tier).
func fetchItems(client *http.Client) (map[string]ItemResource, error) {
	resp, err := client.Get(itemsURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("items API returned status %d", resp.StatusCode)
	}
	var data ItemsResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}
	items := make(map[string]ItemResource, len(data.Items))
	for _, it := range data.Items {
		items[it.ID] = it
	}
	return items, nil
}

// topSellOrder returns the highest order in sell_summary: the best price you
// can instantly sell into (top bid), or 0 if the book is empty.
func topSellOrder(p Product) float64 {
	if len(p.SellSummary) == 0 {
		return 0
	}
	best := p.SellSummary[0].Price
	for _, o := range p.SellSummary {
		if o.Price > best {
			best = o.Price
		}
	}
	return best
}

// topBuyOrder returns the lowest order in buy_summary: the best price you can
// instantly buy from (top ask), or 0 if the book is empty.
func topBuyOrder(p Product) float64 {
	if len(p.BuySummary) == 0 {
		return 0
	}
	best := p.BuySummary[0].Price
	for _, o := range p.BuySummary {
		if o.Price < best {
			best = o.Price
		}
	}
	return best
}

func decodeJSON(body io.Reader, v any) error {
	return json.NewDecoder(body).Decode(v)
}

const shardsURL = "https://raw.githubusercontent.com/NotEnoughUpdates/NotEnoughUpdates-REPO/master/constants/attribute_shards.json"

type shardInfo struct {
	BazaarName  string `json:"bazaarName"`
	DisplayName string `json:"displayName"`
	Rarity      string `json:"rarity"`
}

// fetchShards returns a map of bazaar shard id -> shard name and rarity from
// the community-maintained NotEnoughUpdates dataset.
func fetchShards(client *http.Client) (map[string]shardInfo, error) {
	resp, err := client.Get(shardsURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("shards dataset returned status %d", resp.StatusCode)
	}
	var raw struct {
		Attributes []shardInfo `json:"attributes"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}
	out := make(map[string]shardInfo, len(raw.Attributes))
	for _, s := range raw.Attributes {
		out[s.BazaarName] = s
	}
	return out, nil
}

const bestiaryURL = "https://raw.githubusercontent.com/NotEnoughUpdates/NotEnoughUpdates-REPO/master/constants/bestiary.json"

// MobEntry is a bestiary entry: kill cap, difficulty bracket (1-8), and
// bracket type. CRITTERS-type entries are caught rather than fought, so
// their cap/bracket do not reflect farming difficulty.
type MobEntry struct {
	Cap     int
	Bracket int
	Type    string
}

// fetchBestiary returns a map of cleaned mob name -> bestiary kill cap and
// difficulty bracket. Names have Minecraft color codes stripped.
func fetchBestiary(client *http.Client) (map[string]MobEntry, error) {
	resp, err := client.Get(bestiaryURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bestiary returned status %d", resp.StatusCode)
	}
	var doc any
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return nil, err
	}
	mobs := make(map[string]MobEntry)
	var walk func(v any)
	walk = func(v any) {
		switch t := v.(type) {
		case map[string]any:
			if name, _ := t["name"].(string); name != "" {
				cap, cok := t["cap"].(float64)
				br, bok := t["bracket"].(float64)
				bt, _ := t["bracketType"].(string)
				if cok && bok && cap > 0 {
					n := strings.TrimSpace(bestiaryColorRe.ReplaceAllString(name, ""))
					mobs[n] = MobEntry{Cap: int(cap), Bracket: int(br), Type: bt}
				}
			}
			for _, sub := range t {
				walk(sub)
			}
		case []any:
			for _, sub := range t {
				walk(sub)
			}
		}
	}
	walk(doc)
	return mobs, nil
}

var bestiaryColorRe = regexp.MustCompile(`§.`)
