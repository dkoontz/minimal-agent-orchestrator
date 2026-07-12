package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Plan is the on-disk shape of <goal>.plan.json (see opencode/tools/pdca.ts).
type Plan struct {
	Goal  string `json:"goal"`
	Cycle int    `json:"cycle"`
	Plan  string `json:"plan"`
}

// Entry is one per-cycle history entry value.
type Entry struct {
	Type      string `json:"type"`
	Timestamp string `json:"timestamp"`
	Body      string `json:"body"`
}

// CycleFile is the on-disk shape of <goal>-<N>.json.
type CycleFile struct {
	Cycle   int              `json:"cycle"`
	Entries map[string]Entry `json:"entries"`
}

// Store reads the PDCA log files written by the pdca tool.
type Store struct {
	Dir string
}

func (s *Store) Exists() bool {
	fi, err := os.Stat(s.Dir)
	return err == nil && fi.IsDir()
}

// GoalEntry locates a goal's files. Completed goals live under goals/completed/
// (see the pdca "complete" command); their file layout is identical to active
// goals — only Dir differs.
type GoalEntry struct {
	Name      string
	Completed bool
	Dir       string // directory holding <name>.plan.json and <name>-<N>.json
}

// CompletedDir is the subdir the pdca tool archives finished goals into.
func (s *Store) CompletedDir() string { return filepath.Join(s.Dir, "completed") }

// DiscoverGoals lists active goals then completed goals (each set sorted by
// name). A goal present in both locations is reported only as active.
func (s *Store) DiscoverGoals() ([]GoalEntry, error) {
	var entries []GoalEntry
	seen := map[string]bool{}

	add := func(dir string, completed bool) error {
		matches, err := filepath.Glob(filepath.Join(dir, "*.plan.json"))
		if err != nil {
			return err
		}
		for _, m := range matches {
			name := strings.TrimSuffix(filepath.Base(m), ".plan.json")
			if seen[name] {
				continue
			}
			seen[name] = true
			entries = append(entries, GoalEntry{Name: name, Completed: completed, Dir: dir})
		}
		return nil
	}
	if err := add(s.Dir, false); err != nil {
		return nil, err
	}
	if err := add(s.CompletedDir(), true); err != nil {
		return nil, err
	}
	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].Completed != entries[j].Completed {
			return !entries[i].Completed // active first
		}
		return entries[i].Name < entries[j].Name
	})
	return entries, nil
}

func (s *Store) LoadPlan(g GoalEntry) (*Plan, error) {
	data, err := os.ReadFile(filepath.Join(g.Dir, g.Name+".plan.json"))
	if err != nil {
		return nil, err
	}
	var plan Plan
	if err := json.Unmarshal(data, &plan); err != nil {
		return nil, fmt.Errorf("parse plan for %s: %w", g.Name, err)
	}
	return &plan, nil
}

// LoadCycles returns cycle files for g, sorted ascending by cycle number.
func (s *Store) LoadCycles(g GoalEntry) ([]CycleFile, error) {
	prefix := g.Name + "-"
	matches, err := filepath.Glob(filepath.Join(g.Dir, g.Name+"-*.json"))
	if err != nil {
		return nil, err
	}
	var cycles []CycleFile
	for _, m := range matches {
		base := filepath.Base(m)
		if !strings.HasPrefix(base, prefix) {
			continue
		}
		num := strings.TrimSuffix(strings.TrimPrefix(base, prefix), ".json")
		n, err := strconv.Atoi(num)
		if err != nil {
			continue // not a "<goal>-<N>.json" cycle file
		}
		data, err := os.ReadFile(m)
		if err != nil {
			continue
		}
		var cf CycleFile
		if err := json.Unmarshal(data, &cf); err != nil {
			continue
		}
		cf.Cycle = n
		cycles = append(cycles, cf)
	}
	sort.Slice(cycles, func(i, j int) bool { return cycles[i].Cycle < cycles[j].Cycle })
	return cycles, nil
}

// docItem is one row in the browser list: either the living plan or a single
// cycle entry.
type docItem struct {
	ID        string // "plan" or "cycle-<N>-<type>"
	Kind      string // "plan" or the entry type (lowercase)
	Cycle     int
	Label     string
	Timestamp string
	Body      string
	Empty     bool
}

func (d docItem) FilterValue() string { return d.Label }
func (d docItem) Title() string       { return d.Label }
func (d docItem) Description() string {
	if d.Empty {
		return "empty"
	}
	if d.Timestamp != "" {
		return formatTS(d.Timestamp)
	}
	return ""
}

// formatTS renders an RFC3339 timestamp as "2006-01-02 15:04"; falls back to
// the raw string when it cannot be parsed.
func formatTS(ts string) string {
	if t, err := time.Parse(time.RFC3339, ts); err == nil {
		return t.Format("2006-01-02 15:04")
	}
	return ts
}

// entryTypeFromID extracts the lowercase type from "cycle-<N>-<type>".
func entryTypeFromID(id string) string {
	parts := strings.SplitN(id, "-", 3)
	if len(parts) == 3 {
		return parts[2]
	}
	return id
}

// entryOrder defines the canonical display order of entries within a cycle.
var entryOrder = map[string]int{
	"plan": 0, "investigator": 1, "planner": 2, "revision": 3,
	"critic": 4, "synthesizer": 5, "developer": 6, "reviewer": 7,
	"qa": 8, "debugger": 9, "reflector": 10, "act": 11,
}

func entryRank(id string) int {
	if r, ok := entryOrder[entryTypeFromID(id)]; ok {
		return r
	}
	return 99
}

// BuildItems returns the living-plan item followed by every entry of every
// cycle (asc), with entries within a cycle in canonical PDCA order.
func (s *Store) BuildItems(g GoalEntry) ([]docItem, *Plan, error) {
	plan, err := s.LoadPlan(g)
	if err != nil {
		return nil, nil, err
	}
	planLabel := "Plan (in progress)"
	if g.Completed {
		planLabel = "Plan (completed)"
	}
	items := []docItem{{
		ID:    "plan",
		Kind:  "plan",
		Label: planLabel,
		Body:  plan.Plan,
		Empty: strings.TrimSpace(plan.Plan) == "",
	}}

	cycles, err := s.LoadCycles(g)
	if err != nil {
		return nil, nil, err
	}
	for _, cf := range cycles {
		ids := make([]string, 0, len(cf.Entries))
		for id := range cf.Entries {
			ids = append(ids, id)
		}
		sort.SliceStable(ids, func(i, j int) bool {
			ri, rj := entryRank(ids[i]), entryRank(ids[j])
			if ri != rj {
				return ri < rj
			}
			return ids[i] < ids[j]
		})
		for _, id := range ids {
			e := cf.Entries[id]
			items = append(items, docItem{
				ID:        id,
				Kind:      entryTypeFromID(id),
				Cycle:     cf.Cycle,
				Label:     fmt.Sprintf("C%d · %s", cf.Cycle, e.Type),
				Timestamp: e.Timestamp,
				Body:      e.Body,
				Empty:     strings.TrimSpace(e.Body) == "",
			})
		}
	}
	return items, plan, nil
}

// Status is parsed from the "## Status" block of the living plan.
type Status struct {
	Cycle string
	Phase string
	Next  string
}

// ParseStatus scans the plan markdown for a "## Status" section and pulls out
// the Cycle / Phase / Next bullet values.
func ParseStatus(planBody string) Status {
	var st Status
	inStatus := false
	for _, raw := range strings.Split(planBody, "\n") {
		t := strings.TrimSpace(raw)
		if strings.HasPrefix(t, "## ") {
			inStatus = strings.EqualFold(t, "## Status")
			continue
		}
		if !inStatus {
			continue
		}
		s := strings.TrimPrefix(t, "- ")
		switch {
		case strings.HasPrefix(s, "Cycle:"):
			st.Cycle = strings.TrimSpace(strings.TrimPrefix(s, "Cycle:"))
		case strings.HasPrefix(s, "Phase:"):
			st.Phase = strings.TrimSpace(strings.TrimPrefix(s, "Phase:"))
		case strings.HasPrefix(s, "Next:"):
			st.Next = strings.TrimSpace(strings.TrimPrefix(s, "Next:"))
		}
	}
	return st
}
