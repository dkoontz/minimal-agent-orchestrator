package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
)

// messages
type reloadMsg struct{}

type mode int

const (
	modeGoals mode = iota
	modeBrowse
)

var (
	titleStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))
	mutedStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	detailStyle = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1)
	helpStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
)

// goalItem adapts a GoalEntry into a list item.
type goalItem struct{ entry GoalEntry }

func (g goalItem) FilterValue() string { return g.entry.Name }
func (g goalItem) Title() string       { return g.entry.Name }
func (g goalItem) Description() string {
	if g.entry.Completed {
		return "completed"
	}
	return ""
}

// browser is the tea.Model: a goal picker that drills into a master-detail
// view of a single goal's plan and cycle history.
type browser struct {
	store *Store

	mode      mode
	goals     []GoalEntry
	goalIndex int

	items   []docItem
	plan    *Plan
	status  Status
	loadErr string

	list     list.Model
	viewport viewport.Model
	renderer *glamour.TermRenderer

	width, height int
	dark          bool
}

func newBrowser(store *Store, dark bool) *browser {
	delegate := list.NewDefaultDelegate()
	delegate.SetSpacing(0)
	l := list.New(nil, delegate, 80, 24)
	l.Title = "PDCA"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowHelp(false)
	l.SetShowTitle(true)
	vp := viewport.New(80, 24)
	vp.SetContent("")
	return &browser{store: store, list: l, viewport: vp, mode: modeGoals, dark: dark}
}

func (b *browser) Init() tea.Cmd { return nil }

// ── layout dims ───────────────────────────────────────────────

func (b *browser) headerHeight() int { return 2 }
func (b *browser) footerHeight() int { return 1 }
func (b *browser) bodyHeight() int {
	h := b.height - b.headerHeight() - b.footerHeight()
	if h < 2 {
		return 2
	}
	return h
}

func (b *browser) listWidth() int {
	w := b.width * 3 / 10
	if w < 24 {
		w = 24
	}
	if w > b.width/2 {
		w = b.width / 2
	}
	return w
}

func (b *browser) detailWidth() int {
	w := b.width - b.listWidth()
	if w < 10 {
		w = 10
	}
	return w
}

// detailInnerWidth is the usable text width inside the bordered detail box.
func (b *browser) detailInnerWidth() int {
	w := b.detailWidth() - 4 // border (2) + padding (2)
	if w < 8 {
		w = 8
	}
	return w
}

func (b *browser) ensureRenderer() {
	// Use an explicit style rather than WithAutoStyle(): WithAutoStyle resolves
	// via termenv.HasDarkBackground(), a synchronous OSC terminal query that
	// races with Bubble Tea's TTY reader and can stall the update loop for
	// several seconds. The dark/light choice is detected once at startup
	// (before the tea program owns the TTY) and passed in as b.dark.
	style := "light"
	if b.dark {
		style = "dark"
	}
	r, err := glamour.NewTermRenderer(glamour.WithStandardStyle(style), glamour.WithWordWrap(b.detailInnerWidth()))
	if err == nil {
		b.renderer = r
	}
}

func (b *browser) resize() {
	bh := b.bodyHeight()
	b.list.SetSize(b.listWidth(), bh)
	b.viewport.Width = b.detailInnerWidth()
	b.viewport.Height = bh
	b.renderer = nil
	b.ensureRenderer()
	b.refreshViewport(false)
}

// ── data reload ───────────────────────────────────────────────

func (b *browser) reload() {
	goals, _ := b.store.DiscoverGoals()
	b.goals = goals
	if len(goals) == 0 {
		b.loadErr = fmt.Sprintf("no goals in %s", b.store.Dir)
		b.plan = nil
		b.items = nil
		b.list.SetItems(nil)
		return
	}
	b.loadErr = ""
	if b.goalIndex >= len(goals) {
		b.goalIndex = len(goals) - 1
	}
	if b.goalIndex < 0 {
		b.goalIndex = 0
	}
	if b.mode == modeGoals {
		b.setGoalsList()
		return
	}
	// Browse mode: this is a background refresh (ticker / fs watcher), so
	// re-resolve the current goal by name — it may have moved into
	// goals/completed/ since we last loaded it — and refresh without
	// resetting the scroll position.
	name := ""
	if b.goalIndex < len(b.goals) {
		name = b.goals[b.goalIndex].Name
	}
	if entry, idx, ok := findGoalEntry(b.goals, name); ok {
		b.goalIndex = idx
		b.applyGoal(entry, false)
	} else {
		// current goal vanished → fall back to the goal list
		b.mode = modeGoals
		b.setGoalsList()
	}
}

// findGoalEntry returns the entry matching name plus its index.
func findGoalEntry(goals []GoalEntry, name string) (GoalEntry, int, bool) {
	for i, e := range goals {
		if e.Name == name {
			return e, i, true
		}
	}
	return GoalEntry{}, -1, false
}

func (b *browser) setGoalsList() {
	items := make([]list.Item, len(b.goals))
	for i, e := range b.goals {
		items[i] = goalItem{entry: e}
	}
	b.list.SetItems(items)
	b.list.Title = "Goals"
	if b.goalIndex < len(b.goals) {
		b.list.Select(b.goalIndex)
	}
}

// loadGoal (re)loads a goal after an explicit action (entering the goal list,
// switching goals). The detail pane is reset to the top of the entry.
func (b *browser) loadGoal(g GoalEntry) { b.applyGoal(g, true) }

// applyGoal rebuilds the goal's items and detail. When reset is true the detail
// scrolls to the top (explicit selection); when false the scroll offset is
// preserved (background refresh from the watcher).
func (b *browser) applyGoal(g GoalEntry, reset bool) {
	prevID := ""
	if it, ok := b.list.SelectedItem().(docItem); ok {
		prevID = it.ID
	}
	items, plan, err := b.store.BuildItems(g)
	if err != nil {
		b.loadErr = err.Error()
		b.plan = nil
		b.items = nil
		b.list.SetItems(nil)
		return
	}
	b.loadErr = ""
	b.items = items
	b.plan = plan
	b.status = ParseStatus(plan.Plan)

	listItems := make([]list.Item, len(items))
	for i, it := range items {
		listItems[i] = it
	}
	b.list.SetItems(listItems)
	b.list.Title = g.Name

	idx := 0
	if prevID != "" {
		for i, it := range items {
			if it.ID == prevID {
				idx = i
				break
			}
		}
	}
	if idx >= len(items) {
		idx = len(items) - 1
	}
	if idx < 0 {
		idx = 0
	}
	b.list.Select(idx)
	b.refreshViewport(reset)
}

// ── viewport rendering ────────────────────────────────────────

func (b *browser) refreshViewport(reset bool) {
	if b.renderer == nil {
		b.ensureRenderer()
	}
	item, ok := b.list.SelectedItem().(docItem)
	if !ok {
		b.viewport.SetContent("(no item selected)")
		return
	}
	var meta string
	meta = fmt.Sprintf("id: %s   type: %s", item.ID, item.Kind)
	if item.Cycle > 0 {
		meta += fmt.Sprintf("   cycle: %d", item.Cycle)
	}
	if item.Timestamp != "" {
		meta += "   " + item.Timestamp
	}
	var content string
	if item.Empty {
		content = mutedStyle.Render(meta) + "\n\n(empty entry)"
	} else if b.renderer != nil {
		rendered, err := b.renderer.Render(item.Body)
		if err != nil {
			rendered = item.Body
		}
		content = mutedStyle.Render(meta) + "\n\n" + strings.TrimRight(rendered, "\n")
	} else {
		content = mutedStyle.Render(meta) + "\n\n" + item.Body
	}
	b.viewport.SetContent(content)
	// SetContent preserves the scroll offset; only jump to the top on an
	// explicit selection change, never on a background refresh.
	if reset {
		b.viewport.GotoTop()
	}
}

// ── update ────────────────────────────────────────────────────

func (b *browser) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case reloadMsg:
		b.reload()
		return b, nil

	case tea.WindowSizeMsg:
		b.width, b.height = msg.Width, msg.Height
		b.resize()
		if len(b.goals) == 0 {
			b.reload()
			if len(b.goals) == 1 {
				b.mode = modeBrowse
				b.loadGoal(b.goals[0])
			}
		}
		return b, nil

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return b, tea.Quit
		}

		if b.mode == modeGoals {
			return b.updateGoals(msg)
		}
		return b.updateBrowse(msg)
	}
	return b, nil
}

func (b *browser) updateGoals(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "esc":
		return b, tea.Quit
	case "enter":
		if gi, ok := b.list.SelectedItem().(goalItem); ok {
			b.goalIndex = b.list.Index()
			b.mode = modeBrowse
			b.loadGoal(gi.entry)
		}
		return b, nil
	}
	var cmd tea.Cmd
	b.list, cmd = b.list.Update(msg)
	return b, cmd
}

func (b *browser) updateBrowse(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	switch key {
	case "q":
		return b, tea.Quit
	case "esc", "backspace":
		b.mode = modeGoals
		b.reload()
		return b, nil
	case "r":
		b.reload()
		return b, nil
	case "[":
		if b.goalIndex > 0 {
			b.goalIndex--
			b.loadGoal(b.goals[b.goalIndex])
		}
		return b, nil
	case "]":
		if b.goalIndex < len(b.goals)-1 {
			b.goalIndex++
			b.loadGoal(b.goals[b.goalIndex])
		}
		return b, nil
	case "left":
		b.moveSelection(-1)
		return b, nil
	case "right":
		b.moveSelection(1)
		return b, nil
	}

	// up/down scroll the detail one line; page/half/top/bottom are aliases
	switch key {
	case "up", "k":
		b.viewport.LineUp(1)
		return b, nil
	case "down", "j":
		b.viewport.LineDown(1)
		return b, nil
	case "pgup":
		b.viewport.PageUp()
		return b, nil
	case "pgdown":
		b.viewport.PageDown()
		return b, nil
	case "u":
		b.viewport.HalfPageUp()
		return b, nil
	case "d":
		b.viewport.HalfPageDown()
		return b, nil
	case "g":
		b.viewport.GotoTop()
		return b, nil
	case "G":
		b.viewport.GotoBottom()
		return b, nil
	}
	return b, nil
}

// moveSelection changes the active list item by delta (clamped) and refreshes
// the detail pane to the top of the newly selected entry.
func (b *browser) moveSelection(delta int) {
	n := len(b.items)
	if n == 0 {
		return
	}
	idx := b.list.Index() + delta
	if idx < 0 {
		idx = 0
	}
	if idx >= n {
		idx = n - 1
	}
	b.list.Select(idx)
	b.refreshViewport(true)
}

// ── view ──────────────────────────────────────────────────────

func (b *browser) View() string {
	if b.width == 0 {
		return "Initializing…"
	}
	header := b.renderHeader()
	var body string
	if b.mode == modeGoals {
		body = b.list.View()
	} else {
		detail := detailStyle.Width(b.detailWidth()).Render(b.viewport.View())
		body = lipgloss.JoinHorizontal(lipgloss.Top, b.list.View(), detail)
	}
	footer := helpStyle.Render("[/] goal  ←/→ entry  ↑/↓ scroll line  pgup/pgdn scroll page  r reload  esc back  q quit")
	return lipgloss.JoinVertical(lipgloss.Left, header, body, footer)
}

func (b *browser) renderHeader() string {
	if b.mode == modeGoals {
		line1 := titleStyle.Render("PDCA View")
		line2 := "Select a goal"
		if b.loadErr != "" {
			line2 = b.loadErr
		}
		line2 = mutedStyle.MaxWidth(b.width).Render(line2)
		return lipgloss.JoinVertical(lipgloss.Left, line1, line2)
	}

	var cur GoalEntry
	if b.goalIndex < len(b.goals) {
		cur = b.goals[b.goalIndex]
	}
	left := titleStyle.Render("PDCA View — " + cur.Name)
	rightParts := []string{fmt.Sprintf("cycle %s · phase %s", b.status.Cycle, b.status.Phase)}
	if cur.Completed {
		rightParts = append(rightParts, "completed")
	}
	right := mutedStyle.Render(strings.Join(rightParts, "   "))
	line1 := lipgloss.JoinHorizontal(lipgloss.Top, left, "   ", right)

	line2 := "Next: " + b.status.Next
	if b.status.Next == "" {
		line2 = "Next: —"
	}
	if b.loadErr != "" {
		line2 = b.loadErr
	}
	line2 = mutedStyle.MaxWidth(b.width).Render(line2)
	return lipgloss.JoinVertical(lipgloss.Left, line1, line2)
}
