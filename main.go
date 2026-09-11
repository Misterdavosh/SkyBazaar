package main

import (
	"fmt"
	"net/http"
	"os/exec"
	"runtime"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type tickMsg time.Time
type fetchedMsg struct {
	data *BazaarResponse
	err  error
}
type itemsMsg struct {
	items map[string]ItemResource
	err   error
}

// ---------------------------------------------------------------------------
// Theme
// ---------------------------------------------------------------------------

var (
	themeBorder = lipgloss.AdaptiveColor{Light: "#C678DD", Dark: "#6B5BC7"}
	themeTitle  = lipgloss.AdaptiveColor{Light: "#C678DD", Dark: "#C678DD"}
	themeAccent = lipgloss.AdaptiveColor{Light: "#61AFEF", Dark: "#61AFEF"}
	themeSell   = lipgloss.AdaptiveColor{Light: "#7CFF6B", Dark: "#7CFF6B"}
	themeBuy    = lipgloss.AdaptiveColor{Light: "#FFD75F", Dark: "#FFD75F"}
	themeDim    = lipgloss.AdaptiveColor{Light: "#7A7F8A", Dark: "#8B909D"}
	themeText   = lipgloss.AdaptiveColor{Light: "#D7DAE0", Dark: "#E8EAF0"}
	themeErr    = lipgloss.AdaptiveColor{Light: "#FF6C6B", Dark: "#FF6C6B"}

	appStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(themeBorder).
			Padding(1, 2)

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(themeTitle)

	searchStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(themeAccent).
			Padding(0, 1).
			Width(42)

	searchingStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#8FD7FF")).
			Padding(0, 1).
			Width(42)

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(themeTitle)

	errStyle = lipgloss.NewStyle().Foreground(themeErr)

	dimStyle = lipgloss.NewStyle().Foreground(themeDim)

	sellStyle = lipgloss.NewStyle().Foreground(themeSell)

	buyStyle = lipgloss.NewStyle().Foreground(themeBuy)

	cursorStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.AdaptiveColor{Light: "#99AAB5", Dark: "#3E4451"})

	itemStyle = lipgloss.NewStyle().Bold(true)

	popupStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#B490FF")).
			Foreground(themeText).
			Padding(1, 3)

	popupTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(themeTitle).
			PaddingBottom(1)

	popupSelectedStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#FFFFFF")).
				Background(lipgloss.Color("#7A3FF2"))

	popupKeyStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(themeAccent)
)

// Minecraft-style rarity colors.
var rarityColors = map[string]lipgloss.Color{
	"COMMON":       "#FFFFFF",
	"UNCOMMON":     "#55FF55",
	"RARE":         "#5555FF",
	"EPIC":         "#AA00AA",
	"LEGENDARY":    "#FFAA00",
	"MYTHIC":       "#FF55FF",
	"SPECIAL":      "#FF5555",
	"VERY_SPECIAL": "#FF5555",
	"SUPREME":      "#55FFFF",
}

// ---------------------------------------------------------------------------
// Key bindings
// ---------------------------------------------------------------------------

type keymap struct {
	up       key.Binding
	down     key.Binding
	top      key.Binding
	bottom   key.Binding
	halfUp   key.Binding
	halfDown key.Binding
	search   key.Binding
	open     key.Binding
	done     key.Binding
	sort     key.Binding
	help     key.Binding
	refresh  key.Binding
	quit     key.Binding
}

var keys = keymap{
	up: key.NewBinding(
		key.WithKeys("k", "up"),
		key.WithHelp("k/↑", "up"),
	),
	down: key.NewBinding(
		key.WithKeys("j", "down"),
		key.WithHelp("j/↓", "down"),
	),
	top: key.NewBinding(
		key.WithKeys("g"),
		key.WithHelp("g", "top"),
	),
	bottom: key.NewBinding(
		key.WithKeys("G"),
		key.WithHelp("G", "bottom"),
	),
	halfUp: key.NewBinding(
		key.WithKeys("ctrl+u"),
		key.WithHelp("ctrl+u", "half page up"),
	),
	halfDown: key.NewBinding(
		key.WithKeys("ctrl+d"),
		key.WithHelp("ctrl+d", "half page down"),
	),
	search: key.NewBinding(
		key.WithKeys("/"),
		key.WithHelp("/", "search"),
	),
	open: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "open on coflnet"),
	),
	done: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "done"),
	),
	sort: key.NewBinding(
		key.WithKeys("s"),
		key.WithHelp("s", "sort"),
	),
	help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "more help"),
	),
	refresh: key.NewBinding(
		key.WithKeys("ctrl+r"),
		key.WithHelp("ctrl+r", "refresh"),
	),
	quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
}

func (k keymap) ShortHelp() []key.Binding {
	return []key.Binding{k.up, k.down, k.search, k.sort, k.help, k.quit}
}

func (k keymap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.up, k.down, k.top, k.bottom, k.halfUp, k.halfDown},
		{k.search, k.open, k.done, k.sort},
		{k.refresh, k.help, k.quit},
	}
}

// ---------------------------------------------------------------------------
// Sort modes
// ---------------------------------------------------------------------------

type sortMode int

const (
	sortSellDesc sortMode = iota
	sortSellAsc
	sortBuyDesc
	sortBuyAsc
	sortAlpha
	sortModeCount
)

var sortLabels = map[sortMode]string{
	sortSellDesc: "highest sell order price",
	sortSellAsc:  "lowest sell order price",
	sortBuyDesc:  "highest buy order price",
	sortBuyAsc:   "lowest buy order price",
	sortAlpha:    "item name (A to Z)",
}

// ---------------------------------------------------------------------------
// Model
// ---------------------------------------------------------------------------

const listHeight = 18

type mode int

const (
	modeNav mode = iota
	modeSearch
	modeSort
	modeHelp
)

type model struct {
	data     *BazaarResponse
	items    []Product
	itemDB   map[string]ItemResource
	search   textinput.Model
	client   *http.Client
	err      error
	fetching bool
	cursor   int
	offset   int
	sort     sortMode
	sortCur  int // cursor inside the sort popup
	mode     mode
	help     help.Model
	width    int
	height   int
}

func newModel() model {
	ti := textinput.New()
	ti.Placeholder = "search item (e.g. enchanted bread)..."
	ti.CharLimit = 100
	ti.Width = 38
	return model{
		search: ti,
		client: &http.Client{Timeout: 15 * time.Second},
		help:   help.New(),
	}
}

func rarityStyle(tier string) lipgloss.Style {
	c, ok := rarityColors[tier]
	if !ok {
		return itemStyle
	}
	return itemStyle.Foreground(c)
}

// displayName returns the real in-game item name when known.
func (m model) displayName(id string) string {
	if it, ok := m.itemDB[id]; ok && it.Name != "" {
		return it.Name
	}
	return prettyID(id)
}

func (m model) tier(id string) string {
	if it, ok := m.itemDB[id]; ok {
		return it.Tier
	}
	return ""
}

func (m model) fetch() tea.Msg {
	data, err := fetchBazaar(m.client)
	return fetchedMsg{data: data, err: err}
}

func (m model) fetchItems() tea.Msg {
	items, err := fetchItems(m.client)
	return itemsMsg{items: items, err: err}
}

func (m model) openCoflnet(id string) tea.Cmd {
	url := fmt.Sprintf("https://sky.coflnet.com/item/%s", id)
	cmd := execOpen(url)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Start(); err != nil {
		m.err = fmt.Errorf("could not open browser: %w", err)
	}
	return nil
}

func execOpen(url string) *exec.Cmd {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", url)
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return exec.Command("xdg-open", url)
	}
}

func tick() tea.Cmd {
	return tea.Tick(60*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m model) Init() tea.Cmd {
	return tea.Batch(m.fetch, m.fetchItems, tick())
}

func (m *model) refreshList() {
	m.items = Items(m.data, m.search.Value(), m.displayName)
	m.sortItems()
	if m.cursor >= len(m.items) {
		m.cursor = max(len(m.items)-1, 0)
	}
	m.clampOffset()
}

func (m *model) sortItems() {
	items := m.items
	switch m.sort {
	case sortSellDesc:
		sort.SliceStable(items, func(i, j int) bool { return items[i].QuickStatus.BuyPrice > items[j].QuickStatus.BuyPrice })
	case sortSellAsc:
		sort.SliceStable(items, func(i, j int) bool { return items[i].QuickStatus.BuyPrice < items[j].QuickStatus.BuyPrice })
	case sortBuyDesc:
		sort.SliceStable(items, func(i, j int) bool { return items[i].QuickStatus.SellPrice > items[j].QuickStatus.SellPrice })
	case sortBuyAsc:
		sort.SliceStable(items, func(i, j int) bool { return items[i].QuickStatus.SellPrice < items[j].QuickStatus.SellPrice })
	case sortAlpha:
		sort.SliceStable(items, func(i, j int) bool { return m.displayName(items[i].ProductID) < m.displayName(items[j].ProductID) })
	}
}

func (m *model) clampOffset() {
	maxOffset := max(len(m.items)-listHeight, 0)
	if m.offset > maxOffset {
		m.offset = maxOffset
	}
	if m.offset < 0 {
		m.offset = 0
	}
}

// keepCursorVisible scrolls the offset so the cursor row stays in view.
func (m *model) keepCursorVisible() {
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	if m.cursor >= m.offset+listHeight {
		m.offset = m.cursor - listHeight + 1
	}
	m.clampOffset()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch m.mode {
		case modeSearch:
			switch {
			case key.Matches(msg, keys.done), key.Matches(msg, keys.open):
				m.mode = modeNav
				m.search.Blur()
				return m, nil
			}
		case modeSort:
			switch {
			case key.Matches(msg, keys.quit), key.Matches(msg, keys.done):
				m.mode = modeNav
				return m, nil
			case key.Matches(msg, keys.up):
				if m.sortCur > 0 {
					m.sortCur--
				}
				return m, nil
			case key.Matches(msg, keys.down):
				if m.sortCur < int(sortModeCount)-1 {
					m.sortCur++
				}
				return m, nil
			case key.Matches(msg, keys.open):
				m.sort = sortMode(m.sortCur)
				m.cursor = 0
				m.offset = 0
				m.sortItems()
				m.mode = modeNav
				return m, nil
			default:
				if len(msg.Runes) == 1 && msg.Runes[0] >= '1' && msg.Runes[0] <= '5' {
					m.sort = sortMode(msg.Runes[0] - '1')
					m.sortCur = int(m.sort)
					m.cursor = 0
					m.offset = 0
					m.sortItems()
					m.mode = modeNav
				}
				return m, nil
			}
		case modeHelp:
			m.mode = modeNav
			return m, nil
		case modeNav:
			switch {
			case key.Matches(msg, keys.quit):
				return m, tea.Quit
			case key.Matches(msg, keys.open):
				if len(m.items) > 0 {
					return m, m.openCoflnet(m.items[m.cursor].ProductID)
				}
			case key.Matches(msg, keys.refresh):
				if !m.fetching {
					m.fetching = true
					return m, m.fetch
				}
			case key.Matches(msg, keys.search):
				m.mode = modeSearch
				return m, m.search.Focus()
			case key.Matches(msg, keys.sort):
				m.sortCur = int(m.sort)
				m.mode = modeSort
				return m, nil
			case key.Matches(msg, keys.help):
				m.mode = modeHelp
				return m, nil
			case key.Matches(msg, keys.up):
				if m.cursor > 0 {
					m.cursor--
					m.keepCursorVisible()
				}
			case key.Matches(msg, keys.down):
				if m.cursor < len(m.items)-1 {
					m.cursor++
					m.keepCursorVisible()
				}
			case key.Matches(msg, keys.top):
				m.cursor = 0
				m.keepCursorVisible()
			case key.Matches(msg, keys.bottom):
				m.cursor = max(len(m.items)-1, 0)
				m.keepCursorVisible()
			case key.Matches(msg, keys.halfDown):
				m.cursor = min(m.cursor+listHeight/2, max(len(m.items)-1, 0))
				m.keepCursorVisible()
			case key.Matches(msg, keys.halfUp):
				m.cursor = max(m.cursor-listHeight/2, 0)
				m.keepCursorVisible()
			}
			return m, nil
		}

	case tickMsg:
		if !m.fetching {
			m.fetching = true
			return m, tea.Batch(m.fetch, tick())
		}
		return m, tick()

	case fetchedMsg:
		m.fetching = false
		m.err = msg.err
		m.data = msg.data
		m.refreshList()
		return m, nil

	case itemsMsg:
		if msg.err == nil {
			m.itemDB = msg.items
			m.refreshList()
		}
		return m, nil
	}

	var cmd tea.Cmd
	if m.mode == modeSearch {
		prev := m.search.Value()
		m.search, cmd = m.search.Update(msg)
		if m.search.Value() != prev {
			m.cursor = 0
			m.offset = 0
			m.refreshList()
		}
	}
	return m, cmd
}

// ---------------------------------------------------------------------------
// View
// ---------------------------------------------------------------------------

func (m model) headerView() string {
	return fmt.Sprintf("%s %s %s %s %s",
		headerStyle.Render(pad("ITEM", 26)),
		headerStyle.Render(padRight("INSTASELL", 11)),
		headerStyle.Render(padRight("INSTABUY", 11)),
		headerStyle.Render(padRight("SELL ORDER", 12)),
		headerStyle.Render(padRight("BUY ORDER", 11)),
	)
}

func (m model) rowView(i int) string {
	p := m.items[i]
	name := rarityStyle(m.tier(p.ProductID)).Render(pad(truncate(m.displayName(p.ProductID), 26), 26))
	sell := sellStyle.Render(padRight(formatCoins(p.QuickStatus.SellPrice), 11))
	buy := buyStyle.Render(padRight(formatCoins(p.QuickStatus.BuyPrice), 11))
	sellOrder := padRight(formatCoins(topSellOrder(p)), 12)
	buyOrder := padRight(formatCoins(topBuyOrder(p)), 11)
	row := fmt.Sprintf("%s %s %s %s %s", name, sell, buy, sellOrder, buyOrder)
	if i == m.cursor {
		row = cursorStyle.Render(pad(row, 77))
	}
	return row
}

func (m model) listView() string {
	var b strings.Builder
	b.WriteString(m.headerView() + "\n")
	b.WriteString(dimStyle.Render(strings.Repeat("─", 78)) + "\n")

	if m.err != nil {
		b.WriteString(errStyle.Render("error: "+m.err.Error()) + "\n")
	} else if m.data == nil {
		b.WriteString(dimStyle.Render("loading bazaar data...") + "\n")
	} else if len(m.items) == 0 {
		b.WriteString(dimStyle.Render("no items match your search") + "\n")
	} else {
		end := min(m.offset+listHeight, len(m.items))
		for i := m.offset; i < end; i++ {
			b.WriteString(m.rowView(i) + "\n")
		}
		if len(m.items) > listHeight {
			pos := fmt.Sprintf(" %d/%d ", m.cursor+1, len(m.items))
			b.WriteString(dimStyle.Render(strings.Repeat("─", 78-len(pos))+pos) + "\n")
		}
	}
	return b.String()
}

func (m model) sortPopupView() string {
	var b strings.Builder
	b.WriteString(popupTitleStyle.Render("Sort by") + "\n")
	width := 0
	for i := 0; i < int(sortModeCount); i++ {
		if w := lipgloss.Width(fmt.Sprintf(" %d.  %s", i+1, sortLabels[sortMode(i)])); w > width {
			width = w
		}
	}
	for i := 0; i < int(sortModeCount); i++ {
		plain := pad(fmt.Sprintf(" %d.  %s", i+1, sortLabels[sortMode(i)]), width)
		if i == m.sortCur {
			plain = "▸" + plain[1:]
			b.WriteString(popupSelectedStyle.Render(plain) + "\n")
		} else if i == int(m.sort) {
			b.WriteString(popupKeyStyle.Render(plain) + "\n")
		} else {
			b.WriteString(lipgloss.NewStyle().Foreground(themeText).Render(plain) + "\n")
		}
	}
	b.WriteString("\n" + dimStyle.Render(" j/k move  enter select  esc cancel"))
	return popupStyle.Render(b.String())
}

func (m model) helpPopupView() string {
	h := m.help
	h.Styles = helpStyles()
	var b strings.Builder
	b.WriteString(popupTitleStyle.Render("Keybindings") + "\n\n")
	for _, group := range keys.FullHelp() {
		b.WriteString(h.ShortHelpView(group) + "\n")
	}
	b.WriteString("\n" + dimStyle.Render(" esc close"))
	return popupStyle.Render(b.String())
}

func helpStyles() help.Styles {
	s := help.New().Styles
	s.ShortKey = lipgloss.NewStyle().Bold(true).Foreground(themeAccent)
	s.ShortDesc = lipgloss.NewStyle().Foreground(themeText)
	s.ShortSeparator = lipgloss.NewStyle().Foreground(themeDim)
	return s
}

func (m model) statusBarView() string {
	updated := "never"
	if m.data != nil {
		updated = lastUpdatedString(m.data.LastUpdated)
	}
	left := "updated: " + updated
	if m.fetching {
		left = "refreshing..."
	}
	sortLabel := "sort: " + sortLabels[m.sort]
	gap := 78 - len(left) - len(sortLabel)
	if gap < 1 {
		gap = 1
	}
	return dimStyle.Render(left + strings.Repeat(" ", gap) + sortLabel)
}

func (m model) View() string {
	var b strings.Builder

	title := titleStyle.Render("⛏ Hypixel SkyBlock Bazaar")
	if m.mode == modeSearch {
		title += dimStyle.Render("  (editing search, esc to finish)")
	}
	b.WriteString(title + "\n\n")

	box := searchStyle
	if m.mode == modeSearch {
		box = searchingStyle
	}
	b.WriteString(box.Render(m.search.View()) + "\n\n")

	if m.mode == modeSort {
		b.WriteString(lipgloss.PlaceHorizontal(78, lipgloss.Center, m.sortPopupView()))
	} else if m.mode == modeHelp {
		b.WriteString(lipgloss.PlaceHorizontal(78, lipgloss.Center, m.helpPopupView()))
	} else {
		b.WriteString(m.listView())
	}

	b.WriteString("\n" + m.statusBarView() + "\n")

	h := m.help
	h.Styles = helpStyles()
	h.ShowAll = false
	b.WriteString(h.View(keys))

	return appStyle.Render(b.String())
}

// ---------------------------------------------------------------------------
// Small helpers
// ---------------------------------------------------------------------------

func pad(s string, n int) string {
	if len(s) >= n {
		return s
	}
	return s + strings.Repeat(" ", n-len(s))
}

func padRight(s string, n int) string {
	return pad(s, n)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}

func main() {
	if _, err := tea.NewProgram(newModel(), tea.WithAltScreen()).Run(); err != nil {
		fmt.Println("error running program:", err)
	}
}
