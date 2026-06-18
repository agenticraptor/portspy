package ports

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// rawListener is a transport-level listening socket with its owning PID, before
// any process/project enrichment.
type rawListener struct {
	Proto Proto
	Addr  string
	Port  int
	PID   int
}

// procInfo is everything we resolve about a single process.
type procInfo struct {
	PID        int
	Name       string
	Command    string
	Exe        string
	Cwd        string
	CreateTime time.Time
	Parents    []ProcRef
}

// procSource resolves a PID to process details. It is an interface so the
// assembly logic can be unit-tested with fakes instead of a live system.
type procSource interface {
	Lookup(pid int) (procInfo, bool)
}

// projectFinder resolves a directory to the project that owns it.
type projectFinder interface {
	Find(dir string) Project
}

// Options configure a Scan.
type Options struct {
	// Proto filters by protocol: "tcp", "udp", or "" / "all".
	Proto string
}

// Scan returns the current set of listening sockets on the local machine,
// enriched with the process, command, start time, lineage, and project behind
// each one.
func Scan(opts Options) ([]Listener, error) {
	if _, err := normalizeProto(opts.Proto); err != nil {
		return nil, err
	}
	raw, err := enumerate(opts.Proto)
	if err != nil {
		return nil, fmt.Errorf("enumerate listening sockets: %w", err)
	}
	return build(raw, newProcCache(), fsProjectFinder{}, os.Getpid()), nil
}

// build assembles enriched listeners from raw sockets and the given sources.
// It merges IPv4/IPv6 duplicates of the same (proto, port, pid) into one row.
// It is pure and the primary unit-tested seam of the package.
func build(raw []rawListener, procs procSource, finder projectFinder, selfPID int) []Listener {
	type key struct {
		proto Proto
		port  int
		pid   int
	}
	addrs := map[key][]string{}
	order := make([]key, 0, len(raw))
	for _, r := range raw {
		k := key{r.Proto, r.Port, r.PID}
		if _, seen := addrs[k]; !seen {
			order = append(order, k)
		}
		addrs[k] = append(addrs[k], r.Addr)
	}

	out := make([]Listener, 0, len(order))
	for _, k := range order {
		l := Listener{Proto: k.proto, Port: k.port, PID: k.pid}
		l.Addr, l.Exposed = pickAddr(addrs[k])

		if pi, ok := procs.Lookup(k.pid); ok {
			l.Process = pi.Name
			l.Command = pi.Command
			l.Exe = pi.Exe
			l.CreateTime = pi.CreateTime
			l.Parents = pi.Parents
			if pi.Cwd != "" {
				l.Project = finder.Find(pi.Cwd)
			}
			if l.Project.Empty() && pi.Exe != "" {
				l.Project = finder.Find(filepath.Dir(pi.Exe))
			}
		}
		if l.Process == "" {
			l.Process = "unknown"
		}
		l.Service = detectService(l.Process, l.Command)
		l.Self = selfPID != 0 && k.pid == selfPID
		out = append(out, l)
	}
	Sort(out, SortPort)
	return out
}

// pickAddr chooses the most informative bind address to display for a merged
// row and reports whether any bind exposes the port beyond loopback.
func pickAddr(addrs []string) (addr string, exposed bool) {
	rank := -1
	for _, a := range addrs {
		r := addrRank(a)
		if r >= 1 {
			exposed = true
		}
		if r > rank {
			rank = r
			addr = a
		}
	}
	if rank == 2 { // normalize "all interfaces" for consistent display
		addr = "0.0.0.0"
	}
	return addr, exposed
}

func addrRank(a string) int {
	switch {
	case isAllInterfaces(a):
		return 2
	case isLoopback(a):
		return 0
	default:
		return 1
	}
}

// CheckProto validates a protocol filter string ("", "all", "tcp", or "udp").
func CheckProto(p string) error {
	_, err := normalizeProto(p)
	return err
}

func normalizeProto(p string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(p)) {
	case "", "all":
		return "", nil
	case "tcp":
		return "tcp", nil
	case "udp":
		return "udp", nil
	default:
		return "", fmt.Errorf("invalid protocol %q (want tcp, udp, or all)", p)
	}
}
