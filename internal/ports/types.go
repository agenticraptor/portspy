// Package ports enumerates the sockets listening on the local machine and
// enriches each with the process, command, start time, lineage, and project
// behind it. It is the engine that powers both the portspy TUI and CLI.
package ports

import (
	"fmt"
	"net"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Proto is a transport protocol.
type Proto string

// Supported protocols.
const (
	ProtoTCP Proto = "tcp"
	ProtoUDP Proto = "udp"
)

// ProcRef is a lightweight reference to a process in a lineage chain.
type ProcRef struct {
	PID  int    `json:"pid"`
	Name string `json:"name"`
}

// Project describes the project a listening process belongs to, derived from
// the process working directory and command line.
type Project struct {
	Name string `json:"name,omitempty"` // e.g. "my-api"
	Type string `json:"type,omitempty"` // e.g. "node", "go", "python", "rust"
	Root string `json:"root,omitempty"` // absolute path to the project root
}

// Empty reports whether no project information was resolved.
func (p Project) Empty() bool { return p.Name == "" && p.Type == "" && p.Root == "" }

// Listener is a single listening socket enriched with everything portspy knows
// about the process and project behind it.
type Listener struct {
	Proto      Proto     `json:"proto"`
	Addr       string    `json:"addr"` // bind address as reported by the OS
	Port       int       `json:"port"`
	PID        int       `json:"pid"`
	Process    string    `json:"process"`           // short process name, e.g. "node"
	Command    string    `json:"command,omitempty"` // full command line
	Exe        string    `json:"exe,omitempty"`     // executable path
	CreateTime time.Time `json:"started,omitempty"` // process start time (zero if unknown)
	Parents    []ProcRef `json:"parents,omitempty"` // lineage, immediate parent first
	Project    Project   `json:"project,omitempty"`
	Service    string    `json:"service,omitempty"` // friendly guess, e.g. "Vite", "Postgres"
	Exposed    bool      `json:"exposed"`           // bound to a non-loopback interface
	Self       bool      `json:"self,omitempty"`    // this is the portspy process itself
}

// Uptime returns how long the process has been running, or 0 if unknown.
func (l Listener) Uptime() time.Duration {
	if l.CreateTime.IsZero() {
		return 0
	}
	return time.Since(l.CreateTime)
}

// DisplayAddr collapses common bind addresses into friendly labels:
//
//	0.0.0.0, ::, ""  -> "*"          (all interfaces)
//	127.0.0.1, ::1   -> "localhost"  (loopback only)
//	anything else    -> the literal IP
func (l Listener) DisplayAddr() string { return displayAddr(l.Addr) }

func displayAddr(addr string) string {
	switch addr {
	case "", "0.0.0.0", "::", "[::]", "*":
		return "*"
	case "127.0.0.1", "::1", "[::1]":
		return "localhost"
	default:
		return addr
	}
}

// isLoopback reports whether an address is loopback-only.
func isLoopback(addr string) bool {
	switch addr {
	case "127.0.0.1", "::1", "[::1]":
		return true
	}
	ip := net.ParseIP(strings.Trim(addr, "[]"))
	return ip != nil && ip.IsLoopback()
}

// isAllInterfaces reports whether an address binds every interface.
func isAllInterfaces(addr string) bool {
	switch addr {
	case "", "0.0.0.0", "::", "[::]", "*":
		return true
	}
	ip := net.ParseIP(strings.Trim(addr, "[]"))
	return ip != nil && ip.IsUnspecified()
}

// Label is a one-line, human-friendly description of what is behind the port,
// preferring (in order) the service guess, the project name, then the process.
func (l Listener) Label() string {
	switch {
	case l.Service != "" && !l.Project.Empty():
		return fmt.Sprintf("%s · %s", l.Service, l.Project.Name)
	case l.Service != "":
		return l.Service
	case !l.Project.Empty() && l.Project.Name != "":
		return l.Project.Name
	default:
		return l.Process
	}
}

// SortBy enumerates the orderings portspy can present.
type SortBy int

// Sort orderings.
const (
	SortPort SortBy = iota
	SortPID
	SortProcess
	SortUptime
)

// String implements fmt.Stringer for status displays.
func (s SortBy) String() string {
	switch s {
	case SortPID:
		return "pid"
	case SortProcess:
		return "process"
	case SortUptime:
		return "uptime"
	default:
		return "port"
	}
}

// Next cycles to the next sort ordering.
func (s SortBy) Next() SortBy { return (s + 1) % 4 }

// Sort orders listeners in place by the requested key. Port is always the
// tie-breaker so the output is stable.
func Sort(ls []Listener, by SortBy) {
	sort.SliceStable(ls, func(i, j int) bool {
		a, b := ls[i], ls[j]
		switch by {
		case SortPID:
			if a.PID != b.PID {
				return a.PID < b.PID
			}
		case SortProcess:
			if !strings.EqualFold(a.Process, b.Process) {
				return strings.ToLower(a.Process) < strings.ToLower(b.Process)
			}
		case SortUptime:
			// Longest-running first; unknown start times sink to the bottom.
			au, bu := a.Uptime(), b.Uptime()
			if au != bu {
				return au > bu
			}
		}
		if a.Port != b.Port {
			return a.Port < b.Port
		}
		return a.Proto < b.Proto
	})
}

// Filter keeps the listeners whose searchable text contains the (lower-cased)
// query. An empty query returns the input unchanged.
func Filter(ls []Listener, query string) []Listener {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return ls
	}
	out := make([]Listener, 0, len(ls))
	for _, l := range ls {
		if strings.Contains(l.searchText(), q) {
			out = append(out, l)
		}
	}
	return out
}

func (l Listener) searchText() string {
	parts := []string{
		string(l.Proto),
		strconv.Itoa(l.Port),
		strconv.Itoa(l.PID),
		l.Process,
		l.Command,
		l.Service,
		l.Project.Name,
		l.Project.Type,
	}
	return strings.ToLower(strings.Join(parts, " "))
}
