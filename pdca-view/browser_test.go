package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// ansiRE strips ANSI escape codes so substring assertions are reliable.
var ansiRE = regexp.MustCompile("\x1b\\[[0-9;]*[A-Za-z]")

func stripANSI(s string) string { return ansiRE.ReplaceAllString(s, "") }

// writeSample creates a goals/ directory with two goals and returns its path.
func writeSample(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	must := func(name, content string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	must("fix-login-bug.plan.json", `{
  "goal": "fix-login-bug",
  "cycle": 2,
  "plan": "## Goal\n\nUsers can log in successfully.\n\n## Status\n\n- Cycle: 2\n- Phase: act\n- Next: plan cycle 3\n\n## Known Facts\n\n- Login returns 401 after refresh (cycle 1)\n"
}`)
	must("fix-login-bug-1.json", `{
  "cycle": 1,
  "entries": {
    "cycle-1-plan": { "type": "Plan", "timestamp": "2026-06-30T10:00:00Z", "body": "**Intent**: investigate" },
    "cycle-1-investigator": { "type": "Investigator", "timestamp": "2026-06-30T10:05:00Z", "body": "**Answer**: in `+"`auth/refresh.ts:42`"+` the session is cleared." },
    "cycle-1-act": { "type": "Act", "timestamp": "2026-06-30T10:20:00Z", "body": "**Decision**: continue to cycle 2" }
  }
}`)
	must("fix-login-bug-2.json", `{
  "cycle": 2,
  "entries": {
    "cycle-2-plan": { "type": "Plan", "timestamp": "2026-06-30T11:00:00Z", "body": "**Intent**: implement" }
  }
}`)
	must("second-goal.plan.json", `{
  "goal": "second-goal",
  "cycle": 1,
  "plan": "## Goal\n\nA second goal.\n\n## Status\n\n- Cycle: 1\n- Phase: plan\n- Next: begin\n"
}`)

	// an archived (completed) goal lives under goals/completed/, exactly as
	// the pdca "complete" command writes it.
	completed := filepath.Join(dir, "completed")
	if err := os.MkdirAll(completed, 0o755); err != nil {
		t.Fatalf("mkdir completed: %v", err)
	}
	mustAt(t, completed, "archived-feature.plan.json", `{
  "goal": "archived-feature",
  "cycle": 3,
  "plan": "## Goal\n\nA finished goal.\n\n## Status\n\n- Cycle: 3\n- Phase: act\n- Next: none\n"
}`)
	mustAt(t, completed, "archived-feature-1.json", `{
  "cycle": 1,
  "entries": {
    "cycle-1-act": { "type": "Act", "timestamp": "2026-06-01T09:00:00Z", "body": "**Decision**: goal met" }
  }
}`)
	return dir
}

// mustAt writes content to name inside subdir.
func mustAt(t *testing.T, subdir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(subdir, name), []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}

// writeFile writes content to name inside dir.
func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}

// send is a tiny helper to run the model's Update and keep the concrete type.
func send(m *browser, msg tea.Msg) *browser {
	mm, _ := m.Update(msg)
	return mm.(*browser)
}

func TestParseStatus(t *testing.T) {
	body := "## Goal\n\nX\n\n## Status\n\n- Cycle: 7\n- Phase: reflect\n- Next: reframe\n\n## Known Facts\n"
	st := ParseStatus(body)
	if st.Cycle != "7" || st.Phase != "reflect" || st.Next != "reframe" {
		t.Fatalf("got %+v", st)
	}
}

func TestBuildItems_OrderAndShape(t *testing.T) {
	dir := writeSample(t)
	items, plan, err := (&Store{Dir: dir}).BuildItems(GoalEntry{Name: "fix-login-bug", Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	if plan.Cycle != 2 {
		t.Fatalf("plan cycle = %d, want 2", plan.Cycle)
	}
	if items[0].Label != "Plan (in progress)" {
		t.Fatalf("plan label = %q, want %q", items[0].Label, "Plan (in progress)")
	}
	wantIDs := []string{"plan", "cycle-1-plan", "cycle-1-investigator", "cycle-1-act", "cycle-2-plan"}
	if len(items) != len(wantIDs) {
		t.Fatalf("got %d items, want %d (%+v)", len(items), len(wantIDs), items)
	}
	for i, want := range wantIDs {
		if items[i].ID != want {
			t.Fatalf("item %d = %s, want %s (order: %v)", i, items[i].ID, want, itemIDs(items))
		}
	}
}

func itemIDs(items []docItem) []string {
	ids := make([]string, len(items))
	for i, it := range items {
		ids[i] = it.ID
	}
	return ids
}

// TestDiscoverGoals_ActiveThenCompleted asserts active goals come first
// (sorted), then completed goals (sorted), each pointing at the right dir.
func TestDiscoverGoals_ActiveThenCompleted(t *testing.T) {
	dir := writeSample(t)
	s := &Store{Dir: dir}
	goals, err := s.DiscoverGoals()
	if err != nil {
		t.Fatal(err)
	}
	type want struct {
		name      string
		completed bool
	}
	wants := []want{
		{"fix-login-bug", false},
		{"second-goal", false},
		{"archived-feature", true},
	}
	if len(goals) != len(wants) {
		t.Fatalf("got %d goals: %+v", len(goals), goals)
	}
	for i, w := range wants {
		g := goals[i]
		if g.Name != w.name || g.Completed != w.completed {
			t.Fatalf("goal %d = %+v, want %+v", i, g, w)
		}
		wantDir := dir
		if w.completed {
			wantDir = s.CompletedDir()
		}
		if g.Dir != wantDir {
			t.Fatalf("goal %s Dir = %s, want %s", g.Name, g.Dir, wantDir)
		}
	}
}

// TestBuildItems_CompletedGoal reads an archived goal from goals/completed/.
func TestBuildItems_CompletedGoal(t *testing.T) {
	dir := writeSample(t)
	s := &Store{Dir: dir}
	g := GoalEntry{Name: "archived-feature", Completed: true, Dir: s.CompletedDir()}
	items, plan, err := s.BuildItems(g)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Cycle != 3 {
		t.Fatalf("plan cycle = %d, want 3", plan.Cycle)
	}
	if items[0].Label != "Plan (completed)" {
		t.Fatalf("plan label = %q, want %q", items[0].Label, "Plan (completed)")
	}
	// Plan (in progress) + the single cycle-1-act entry
	wantIDs := []string{"plan", "cycle-1-act"}
	if len(items) != len(wantIDs) {
		t.Fatalf("got %d items (%v), want %d", len(items), itemIDs(items), len(wantIDs))
	}
	for i, want := range wantIDs {
		if items[i].ID != want {
			t.Fatalf("item %d = %s, want %s", i, items[i].ID, want)
		}
	}
}

// TestView_RendersBothModes drives the browser through the goal list into a
// goal detail view and asserts the rendered output contains the expected text
// without panicking.
func TestView_RendersBothModes(t *testing.T) {
	dir := writeSample(t)
	m := newBrowser(&Store{Dir: dir}, true)

	// initial size → reload, stays in modeGoals (3 goals: 2 active, 1 completed)
	m = send(m, tea.WindowSizeMsg{Width: 100, Height: 30})
	if m.mode != modeGoals {
		t.Fatalf("expected modeGoals, got %v", m.mode)
	}
	goalsView := stripANSI(m.View())
	if !strings.Contains(goalsView, "Select a goal") {
		t.Fatalf("goals header missing:\n%s", goalsView)
	}
	for _, want := range []string{"fix-login-bug", "second-goal", "archived-feature", "completed"} {
		if !strings.Contains(goalsView, want) {
			t.Fatalf("goal list missing %q:\n%s", want, goalsView)
		}
	}

	// enter the first goal
	m = send(m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.mode != modeBrowse {
		t.Fatalf("expected modeBrowse, got %v", m.mode)
	}
	browseView := stripANSI(m.View())
	for _, want := range []string{"PDCA View", "fix-login-bug", "cycle 2", "phase act", "Users can log in successfully."} {
		if !strings.Contains(browseView, want) {
			t.Fatalf("browse view missing %q:\n%s", want, browseView)
		}
	}

	// move selection right to the first cycle entry (plan item is index 0;
	// one right lands on cycle-1-plan) and confirm the detail pane updates.
	m = send(m, tea.KeyMsg{Type: tea.KeyRight})
	rightView := stripANSI(m.View())
	if !strings.Contains(rightView, "Intent") {
		t.Fatalf("detail did not update on selection change:\n%s", rightView)
	}

	// up/down now scroll the detail (no selection change); the plan body must
	// remain visible because selection stayed on cycle-1-plan.
	m = send(m, tea.KeyMsg{Type: tea.KeyDown})
	if m.list.Index() != 1 {
		t.Fatalf("down arrow changed selection to %d, want it to scroll only", m.list.Index())
	}

	// ']' moves to the next goal and reloads its detail
	m = send(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("]")})
	nextView := stripANSI(m.View())
	if !strings.Contains(nextView, "second-goal") {
		t.Fatalf("did not switch to second goal:\n%s", nextView)
	}
}

// TestBackgroundReload_PreservesScroll is a regression test for the two
// symptoms: (a) a background refresh (the watcher's reloadMsg) must NOT reset
// the detail pane to the top, and (b) an explicit selection change must.
func TestBackgroundReload_PreservesScroll(t *testing.T) {
	dir := t.TempDir()
	// Build a goal whose only entry is a tall code block so the viewport can
	// actually scroll.
	var bodyBuilder strings.Builder
	bodyBuilder.WriteString("```\n")
	for i := 1; i <= 80; i++ {
		fmt.Fprintf(&bodyBuilder, "Line %d\n", i)
	}
	bodyBuilder.WriteString("```")
	body := bodyBuilder.String()

	writeFile(t, dir, "tall.plan.json", `{"goal":"tall","cycle":1,"plan":"## Goal\n\nTall.\n\n## Status\n\n- Cycle: 1\n- Phase: act\n- Next: x\n"}`)
	writeFile(t, dir, "tall-1.json", fmt.Sprintf(`{"cycle":1,"entries":{"cycle-1-act":{"type":"Act","timestamp":"2026-06-30T10:00:00Z","body":%q}}}`, body))

	m := newBrowser(&Store{Dir: dir}, true)
	m = send(m, tea.WindowSizeMsg{Width: 100, Height: 30})
	if m.mode != modeBrowse {
		t.Fatalf("expected single-goal auto-enter to browse, got %v", m.mode)
	}

	// plan (idx 0) is selected; move right to the tall entry
	m = send(m, tea.KeyMsg{Type: tea.KeyRight})
	if m.list.Index() != 1 {
		t.Fatalf("expected entry at index 1, got %d", m.list.Index())
	}
	if m.viewport.YOffset != 0 {
		t.Fatalf("explicit selection should start at top, YOffset=%d", m.viewport.YOffset)
	}

	// scroll down 10 lines
	for i := 0; i < 10; i++ {
		m = send(m, tea.KeyMsg{Type: tea.KeyDown})
	}
	if m.viewport.YOffset != 10 {
		t.Fatalf("after scrolling, YOffset=%d want 10", m.viewport.YOffset)
	}

	// background reload must preserve the scroll position
	m = send(m, reloadMsg{})
	if m.viewport.YOffset != 10 {
		t.Fatalf("background reload reset scroll: YOffset=%d want 10", m.viewport.YOffset)
	}

	// an explicit selection change (back to plan) must reset to the top
	m = send(m, tea.KeyMsg{Type: tea.KeyLeft})
	if m.viewport.YOffset != 0 {
		t.Fatalf("explicit selection did not reset scroll: YOffset=%d want 0", m.viewport.YOffset)
	}
}
