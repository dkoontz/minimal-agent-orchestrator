package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/fsnotify/fsnotify"
	"github.com/muesli/termenv"
)

func main() {
	var goalsDir string
	flag.StringVar(&goalsDir, "goals", "./goals", "path to the goals/ directory")
	flag.StringVar(&goalsDir, "g", "./goals", "path to the goals/ directory (shorthand)")

	var dump bool
	flag.BoolVar(&dump, "dump", false, "print the rendered status + entries to stdout and exit (no TUI)")

	flag.Parse()
	if flag.NArg() >= 1 {
		goalsDir = flag.Arg(0)
	}
	if abs, err := filepath.Abs(goalsDir); err == nil {
		goalsDir = abs
	}

	store := &Store{Dir: goalsDir}

	if dump {
		if err := runDump(store); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
		return
	}

	// Detect terminal background ONCE, before the tea program takes over the
	// TTY. glamour.WithAutoStyle() does this detection on demand via a blocking
	// OSC query that races with Bubble Tea's input loop and stalls startup, so
	// we resolve it up front and hand the result to the browser.
	dark := termenv.HasDarkBackground()

	m := newBrowser(store, dark)
	p := tea.NewProgram(m, tea.WithAltScreen())

	go watch(store, p)

	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

// watch reloads the model when goals/*.json changes, with a periodic poll as a
// fallback (and to pick up goals/ and goals/completed/ dirs that did not exist
// at startup). fsnotify is not recursive, so both dirs are watched explicitly.
func watch(store *Store, p *tea.Program) {
	var w *fsnotify.Watcher
	if fw, err := fsnotify.NewWatcher(); err == nil {
		w = fw
		defer w.Close()
	}

	watchDirs := []string{store.Dir, store.CompletedDir()}
	added := map[string]bool{}
	addExisting := func() {
		if w == nil {
			return
		}
		for _, d := range watchDirs {
			if added[d] {
				continue
			}
			if fi, err := os.Stat(d); err == nil && fi.IsDir() {
				if w.Add(d) == nil {
					added[d] = true
				}
			}
		}
	}
	addExisting()

	ticker := time.NewTicker(1500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case ev, ok := <-events(w):
			if !ok {
				return
			}
			if strings.HasSuffix(ev.Name, ".json") {
				p.Send(reloadMsg{})
			}
		case _, ok := <-errors(w):
			if !ok {
				return
			}
		case <-ticker.C:
			addExisting() // pick up goals/ or completed/ created after start
			p.Send(reloadMsg{})
		}
	}
}

func events(w *fsnotify.Watcher) <-chan fsnotify.Event {
	if w == nil {
		return nil
	}
	return w.Events
}

func errors(w *fsnotify.Watcher) <-chan error {
	if w == nil {
		return nil
	}
	return w.Errors
}

// runDump renders a goal's status and entries to stdout. Used for piping or
// verifying the data layer without launching the TUI.
func runDump(store *Store) error {
	goals, err := store.DiscoverGoals()
	if err != nil {
		return err
	}
	if len(goals) == 0 {
		return fmt.Errorf("no goals found in %s", store.Dir)
	}
	for _, g := range goals {
		items, plan, err := store.BuildItems(g)
		if err != nil {
			return err
		}
		st := ParseStatus(plan.Plan)
		name := g.Name
		if g.Completed {
			name += " (completed)"
		}
		fmt.Printf("=== %s ===\n", name)
		fmt.Printf("cycle: %s   phase: %s   next: %s\n", st.Cycle, st.Phase, st.Next)
		fmt.Printf("%d items\n\n", len(items))
		for _, it := range items {
			fmt.Printf("--- %s", it.Label)
			if it.Cycle > 0 {
				fmt.Printf("  [id %s]", it.ID)
			}
			if it.Timestamp != "" {
				fmt.Printf("  (%s)", it.Timestamp)
			}
			fmt.Println()
			if it.Empty {
				fmt.Println("(empty)")
			} else {
				fmt.Println(strings.TrimRight(it.Body, "\n"))
			}
			fmt.Println()
		}
	}
	return nil
}
