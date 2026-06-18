package tui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/agenticraptor/portspy/internal/ports"
)

func sample() []ports.Listener {
	now := time.Now()
	return []ports.Listener{
		{
			Proto: ports.ProtoTCP, Addr: "127.0.0.1", Port: 3000, PID: 111,
			Process: "node", Command: "node node_modules/.bin/vite", Exe: "/usr/local/bin/node",
			Service: "Vite", Project: ports.Project{Name: "web", Type: "node", Root: "/code/web"},
			CreateTime: now.Add(-6 * time.Minute),
			Parents:    []ports.ProcRef{{PID: 90, Name: "npm"}, {PID: 50, Name: "zsh"}},
		},
		{
			Proto: ports.ProtoTCP, Addr: "0.0.0.0", Port: 8080, PID: 222,
			Process: "python3", Command: "python -m http.server 8080", Exposed: true,
		},
	}
}

// update applies a message and returns the concrete model.
func update(t *testing.T, m model, msg tea.Msg) (model, tea.Cmd) {
	t.Helper()
	nm, cmd := m.Update(msg)
	mm, ok := nm.(model)
	if !ok {
		t.Fatalf("Update returned %T, want model", nm)
	}
	return mm, cmd
}

func keyRunes(s string) tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)} }

func ready(t *testing.T) model {
	m := newModel(ports.Options{Proto: "all"})
	m, _ = update(t, m, tea.WindowSizeMsg{Width: 100, Height: 30})
	m, _ = update(t, m, scannedMsg{listeners: sample()})
	return m
}

func TestTableRendersListeners(t *testing.T) {
	m := ready(t)
	out := m.View()
	for _, want := range []string{"portspy", "3000", "8080", "Vite · web", "PORT", "WHAT"} {
		if !strings.Contains(out, want) {
			t.Errorf("View() missing %q\n%s", want, out)
		}
	}
}

func TestKillConfirmFlow(t *testing.T) {
	m := ready(t)

	// Press `x` on the first row → confirmation dialog appears.
	m, _ = update(t, m, keyRunes("x"))
	if !m.confirming {
		t.Fatal("expected confirming=true after pressing x")
	}
	out := m.View()
	if !strings.Contains(out, "Terminate this process?") {
		t.Errorf("confirm view missing prompt:\n%s", out)
	}
	if !strings.Contains(out, ":3000/tcp") {
		t.Errorf("confirm view should name the target port:\n%s", out)
	}

	// Confirm with `y` → a kill command is issued and the dialog closes.
	m, cmd := update(t, m, keyRunes("y"))
	if m.confirming {
		t.Error("expected confirming=false after confirming")
	}
	if cmd == nil {
		t.Error("expected a kill command to be returned on confirm")
	}
}

func TestKillConfirmCancel(t *testing.T) {
	m := ready(t)
	m, _ = update(t, m, keyRunes("x"))
	m, _ = update(t, m, keyRunes("n"))
	if m.confirming {
		t.Error("expected confirming=false after cancel")
	}
}

func TestInspectShowsLineage(t *testing.T) {
	m := ready(t)
	// `enter` opens the detail view for the selected row.
	m, _ = update(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if !m.inspecting {
		t.Fatal("expected inspecting=true after enter")
	}
	out := m.View()
	for _, want := range []string{":3000/tcp", "npm ← zsh", "Started", "/code/web"} {
		if !strings.Contains(out, want) {
			t.Errorf("inspect view missing %q\n%s", want, out)
		}
	}
	// `x` from the detail view goes straight to the kill confirmation.
	m, _ = update(t, m, keyRunes("x"))
	if m.inspecting {
		t.Error("inspecting should close when killing")
	}
	if !m.confirming {
		t.Error("expected confirmation after pressing x in detail view")
	}
}

func TestFilterNarrowsView(t *testing.T) {
	m := ready(t)
	m, _ = update(t, m, keyRunes("/"))
	if !m.filtering {
		t.Fatal("expected filtering=true after /")
	}
	for _, r := range "8080" {
		m, _ = update(t, m, keyRunes(string(r)))
	}
	if len(m.view) != 1 || m.view[0].Port != 8080 {
		t.Fatalf("filter should narrow to :8080, got %d rows", len(m.view))
	}
	// Esc clears the filter and restores all rows.
	m, _ = update(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.filtering || len(m.view) != 2 {
		t.Errorf("esc should clear filter and restore rows, got filtering=%v rows=%d", m.filtering, len(m.view))
	}
}

func TestSelectionFollowsSocketAcrossRefresh(t *testing.T) {
	m := ready(t)
	// Move to the second row (:8080).
	m, _ = update(t, m, keyRunes("j"))
	sel, ok := m.selected()
	if !ok || sel.Port != 8080 {
		t.Fatalf("expected :8080 selected, got %+v ok=%v", sel, ok)
	}
	// A refresh that reorders the list (now :8080 first) must keep :8080 selected.
	reordered := []ports.Listener{sample()[1], sample()[0]}
	m, _ = update(t, m, scannedMsg{listeners: reordered})
	sel, ok = m.selected()
	if !ok || sel.Port != 8080 {
		t.Errorf("selection should follow :8080 across refresh, got %+v", sel)
	}
}

func TestLineageString(t *testing.T) {
	got := lineageString([]ports.ProcRef{{PID: 1, Name: "npm"}, {PID: 2, Name: ""}})
	if got != "npm ← pid 2" {
		t.Errorf("lineageString = %q", got)
	}
	if lineageString(nil) != "" {
		t.Error("empty lineage should be empty string")
	}
}

func TestQuit(t *testing.T) {
	m := ready(t)
	_, cmd := update(t, m, keyRunes("q"))
	if cmd == nil {
		t.Error("q should return a quit command")
	}
}
