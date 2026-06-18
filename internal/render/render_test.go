package render

import (
	"bytes"
	"encoding/json"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"github.com/agenticraptor/portspy/internal/ports"
)

func sampleListeners() []ports.Listener {
	return []ports.Listener{
		{
			Proto:      ports.ProtoTCP,
			Addr:       "127.0.0.1",
			Port:       3000,
			PID:        111,
			Process:    "node",
			Command:    "node node_modules/.bin/vite",
			Service:    "Vite",
			Project:    ports.Project{Name: "web", Type: "node", Root: "/work/web"},
			CreateTime: time.Now().Add(-90 * time.Second),
		},
		{
			Proto:   ports.ProtoTCP,
			Addr:    "0.0.0.0",
			Port:    8080,
			PID:     222,
			Process: "python3",
			Exposed: true,
		},
	}
}

func TestTable(t *testing.T) {
	var buf bytes.Buffer
	if err := Table(&buf, sampleListeners(), false); err != nil {
		t.Fatalf("Table error: %v", err)
	}
	out := buf.String()

	for _, want := range []string{"PORT", "PROTO", "PID", "PROCESS", "UPTIME", "ADDRESS", "WHAT"} {
		if !strings.Contains(out, want) {
			t.Errorf("table missing header %q\n%s", want, out)
		}
	}
	for _, want := range []string{"3000", "node", "Vite · web", "localhost", "8080", "exposed"} {
		if !strings.Contains(out, want) {
			t.Errorf("table missing value %q\n%s", want, out)
		}
	}
}

var ansiRE = regexp.MustCompile("\x1b\\[[0-9;]*m")

// TestTableColorAlignment guards the tabwriter/ANSI bug: coloring the header
// must not change column alignment. Stripping the ANSI codes from the colored
// output must yield exactly the plain output.
func TestTableColorAlignment(t *testing.T) {
	// Force a color profile so headerStyle.Render actually emits ANSI codes;
	// lipgloss disables color under `go test` (no TTY) by default.
	old := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.ANSI)
	defer lipgloss.SetColorProfile(old)

	ls := sampleListeners()

	var plain, colored bytes.Buffer
	if err := Table(&plain, ls, false); err != nil {
		t.Fatalf("plain Table error: %v", err)
	}
	if err := Table(&colored, ls, true); err != nil {
		t.Fatalf("colored Table error: %v", err)
	}

	stripped := ansiRE.ReplaceAllString(colored.String(), "")
	if stripped != plain.String() {
		t.Errorf("colored output (ANSI-stripped) differs from plain output:\n--- plain ---\n%q\n--- stripped ---\n%q", plain.String(), stripped)
	}
	if !strings.Contains(colored.String(), "\x1b[") {
		t.Error("expected ANSI escape codes in colored output")
	}
}

func TestTableEmpty(t *testing.T) {
	var buf bytes.Buffer
	if err := Table(&buf, nil, false); err != nil {
		t.Fatalf("Table error: %v", err)
	}
	if !strings.Contains(buf.String(), "No listening ports") {
		t.Errorf("expected empty-state message, got %q", buf.String())
	}
}

func TestJSON(t *testing.T) {
	var buf bytes.Buffer
	if err := JSON(&buf, sampleListeners()); err != nil {
		t.Fatalf("JSON error: %v", err)
	}

	var decoded []map[string]any
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	if len(decoded) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(decoded))
	}

	first := decoded[0]
	if first["proto"] != "tcp" || first["port"].(float64) != 3000 {
		t.Errorf("unexpected first entry: %+v", first)
	}
	if first["label"] != "Vite · web" {
		t.Errorf("label = %v, want 'Vite · web'", first["label"])
	}
	if first["display_addr"] != "localhost" {
		t.Errorf("display_addr = %v, want localhost", first["display_addr"])
	}
	proj, ok := first["project"].(map[string]any)
	if !ok || proj["name"] != "web" {
		t.Errorf("project not rendered correctly: %+v", first["project"])
	}
	if _, ok := first["started"]; !ok {
		t.Error("expected a 'started' timestamp for a process with a known start time")
	}

	// The second entry has no project/start time — those keys should be omitted.
	second := decoded[1]
	if _, ok := second["project"]; ok {
		t.Error("expected project to be omitted when empty")
	}
	if _, ok := second["started"]; ok {
		t.Error("expected started to be omitted when unknown")
	}
	if second["exposed"] != true {
		t.Error("expected second entry to be exposed")
	}
}
