package main

import (
	"bytes"
	"os"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// TestProgram_RunHeadless runs the real tea.Program against virtual input
// (a pipe that blocks forever) and an in-memory output buffer, driving it with
// queued messages. This exercises Init/Update/View through bubbletea's actual
// render loop — the same bytes a live terminal receives — without needing a PTY.
func TestProgram_RunHeadless(t *testing.T) {
	dir := writeSample(t)
	m := newBrowser(&Store{Dir: dir}, true)

	var buf bytes.Buffer
	rIn, _, _ := os.Pipe() // reader that blocks forever (no writer)
	defer rIn.Close()

	p := tea.NewProgram(m, tea.WithInput(rIn), tea.WithOutput(&buf), tea.WithoutSignalHandler())

	go func() {
		time.Sleep(100 * time.Millisecond)
		p.Send(tea.WindowSizeMsg{Width: 100, Height: 30}) // initial size → reload
		time.Sleep(150 * time.Millisecond)
		p.Send(tea.KeyMsg{Type: tea.KeyEnter}) // enter first goal
		time.Sleep(150 * time.Millisecond)
		p.Send(tea.KeyMsg{Type: tea.KeyDown}) // step to a cycle entry
		time.Sleep(150 * time.Millisecond)
		p.Send(tea.KeyMsg{Type: tea.KeyCtrlC}) // quit (handled as tea.Quit)
	}()

	if _, err := p.Run(); err != nil {
		t.Fatalf("program.Run: %v", err)
	}

	// These substrings prove the real program loop rendered the goal-select
	// frame and then the browse frame (including the markdown body of the
	// living plan). Header corner cases like "cycle 2" are asserted directly
	// against View() in TestView_RendersBothModes; here we avoid the inline
	// renderer's differential line-overwrite artifacts in the byte buffer.
	out := stripANSI(buf.String())
	for _, want := range []string{"PDCA View", "fix-login-bug", "Users can log in successfully."} {
		if !strings.Contains(out, want) {
			t.Fatalf("rendered output missing %q:\n%s", want, out)
		}
	}
}
