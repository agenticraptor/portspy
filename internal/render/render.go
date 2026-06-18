// Package render formats listeners for non-interactive output: an aligned text
// table for humans and stable JSON for scripts.
package render

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/agenticraptor/portspy/internal/ports"
)

var headerStyle = lipgloss.NewStyle().Bold(true)

// Table writes an aligned, human-readable table of listeners to w. When color
// is true the header row is emphasized.
//
// The table is laid out in plain text first and only the header line is colored
// afterwards: tabwriter measures column widths by counting runes, so embedding
// ANSI escape codes before alignment would corrupt the column widths.
func Table(w io.Writer, ls []ports.Listener, color bool) error {
	if len(ls) == 0 {
		_, err := fmt.Fprintln(w, "No listening ports found.")
		return err
	}

	var buf bytes.Buffer
	tw := tabwriter.NewWriter(&buf, 0, 2, 2, ' ', 0)
	fmt.Fprintln(tw, "PORT\tPROTO\tPID\tPROCESS\tUPTIME\tADDRESS\tWHAT")
	for _, l := range ls {
		addr := l.DisplayAddr()
		if addr == "*" {
			addr = "* (exposed)"
		} else if l.Exposed {
			addr += " (exposed)"
		}
		fmt.Fprintf(tw, "%d\t%s\t%d\t%s\t%s\t%s\t%s\n",
			l.Port,
			l.Proto,
			l.PID,
			truncate(l.Process, 22),
			l.HumanUptime(),
			addr,
			truncate(l.Label(), 42),
		)
	}
	if err := tw.Flush(); err != nil {
		return err
	}

	if !color {
		_, err := w.Write(buf.Bytes())
		return err
	}

	// Colorize only the (already-aligned) header line.
	header, rest, _ := strings.Cut(buf.String(), "\n")
	if _, err := fmt.Fprintln(w, headerStyle.Render(strings.TrimRight(header, " "))); err != nil {
		return err
	}
	_, err := io.WriteString(w, rest)
	return err
}

// jsonProject is the wire representation of a project.
type jsonProject struct {
	Name string `json:"name,omitempty"`
	Type string `json:"type,omitempty"`
	Root string `json:"root,omitempty"`
}

// jsonListener is the stable, documented JSON shape emitted by `portspy list
// --json`. It is intentionally decoupled from the internal model.
type jsonListener struct {
	Proto         string          `json:"proto"`
	Port          int             `json:"port"`
	Addr          string          `json:"addr"`
	DisplayAddr   string          `json:"display_addr"`
	Exposed       bool            `json:"exposed"`
	PID           int             `json:"pid"`
	Process       string          `json:"process"`
	Command       string          `json:"command,omitempty"`
	Exe           string          `json:"exe,omitempty"`
	Service       string          `json:"service,omitempty"`
	Project       *jsonProject    `json:"project,omitempty"`
	Started       string          `json:"started,omitempty"`
	UptimeSeconds int64           `json:"uptime_seconds,omitempty"`
	Parents       []ports.ProcRef `json:"parents,omitempty"`
	Label         string          `json:"label"`
	Self          bool            `json:"self,omitempty"`
}

// JSON writes listeners as an indented JSON array to w.
func JSON(w io.Writer, ls []ports.Listener) error {
	out := make([]jsonListener, 0, len(ls))
	for _, l := range ls {
		jl := jsonListener{
			Proto:       string(l.Proto),
			Port:        l.Port,
			Addr:        l.Addr,
			DisplayAddr: l.DisplayAddr(),
			Exposed:     l.Exposed,
			PID:         l.PID,
			Process:     l.Process,
			Command:     l.Command,
			Exe:         l.Exe,
			Service:     l.Service,
			Parents:     l.Parents,
			Label:       l.Label(),
			Self:        l.Self,
		}
		if !l.Project.Empty() {
			jl.Project = &jsonProject{Name: l.Project.Name, Type: l.Project.Type, Root: l.Project.Root}
		}
		if !l.CreateTime.IsZero() {
			jl.Started = l.CreateTime.UTC().Format(time.RFC3339)
			jl.UptimeSeconds = int64(l.Uptime().Seconds())
		}
		out = append(out, jl)
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

// truncate shortens s to max runes, appending an ellipsis when cut.
func truncate(s string, max int) string {
	if max <= 1 {
		return s
	}
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max-1]) + "…"
}
