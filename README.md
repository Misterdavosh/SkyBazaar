# SkyBazaar

A terminal UI (TUI) for browsing the [Hypixel SkyBlock Bazaar](https://wiki.hypixel.net/Bazaar) — live market prices, search, sorting, and item rarity colors, built with [Charm](https://charm.sh) libraries (Bubble Tea, Bubbles, Lipgloss).

> **Note:** This is a hobby project built for learning and experimentation. It is **not** intended for actual day-to-day use — a full price tracker or in-game flipping tool would be far more convenient in practice.

## Features

- Live bazaar prices, auto-refreshed every 60 seconds (manual refresh with `ctrl+r`)
- Instant sell/buy prices plus top-of-book sell and buy order prices
- Fuzzy-ish search that matches both item IDs and real in-game names (spaces, underscores, and punctuation are normalized)
- Item names colored by their in-game rarity (common through supreme)
- Vim-style navigation: `j`/`k`, `g`/`G`, `ctrl+u`/`ctrl+d`
- Sort popup (`s`) with five sort modes, and a keybinding reference popup (`?`)
- Clean, single-palette TUI built with Bubble Tea and Lipgloss

## Usage

```bash
go build -o skybazaar .
./skybazaar
```

| Key | Action |
| --- | --- |
| `/` or `enter` | edit search |
| `esc` | done editing |
| `j` / `k` | move cursor |
| `g` / `G` | jump to top / bottom |
| `ctrl+u` / `ctrl+d` | half-page scroll |
| `s` | open sort menu |
| `?` | keybinding help |
| `ctrl+r` | refresh prices |
| `q` | quit |

## How it works

Data comes from the public [Hypixel API](https://api.hypixel.net/):

- `skyblock/bazaar` — live order books and quick prices (no API key required)
- `resources/skyblock/items` — item names and rarity tiers, used for display names and rarity colors

## Status

Experiment — expect rough edges. Data refresh behavior is limited to whole 60-second polls, and the fixed-width layout targets terminals of roughly 100 columns or wider.