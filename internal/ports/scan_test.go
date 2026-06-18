package ports

import (
	"testing"
)

type fakeProcs map[int]procInfo

func (f fakeProcs) Lookup(pid int) (procInfo, bool) {
	pi, ok := f[pid]
	return pi, ok
}

type fakeFinder map[string]Project

func (f fakeFinder) Find(dir string) Project { return f[dir] }

func TestBuildMergesAndEnriches(t *testing.T) {
	raw := []rawListener{
		// Same TCP listener reported on IPv4 and IPv6 — must merge into one row.
		{Proto: ProtoTCP, Addr: "127.0.0.1", Port: 3000, PID: 10},
		{Proto: ProtoTCP, Addr: "::1", Port: 3000, PID: 10},
		// Exposed on all interfaces.
		{Proto: ProtoTCP, Addr: "0.0.0.0", Port: 8080, PID: 20},
		// A Vite dev server with a project.
		{Proto: ProtoTCP, Addr: "127.0.0.1", Port: 5173, PID: 30},
		// Unknown process (not in the proc source).
		{Proto: ProtoUDP, Addr: "192.168.1.9", Port: 5353, PID: 40},
	}
	procs := fakeProcs{
		10: {PID: 10, Name: "node", Command: "node server.js", Cwd: "/work/api"},
		20: {PID: 20, Name: "python3", Command: "python -m http.server 8080"},
		30: {PID: 30, Name: "node", Command: "node node_modules/.bin/vite", Cwd: "/work/web"},
	}
	finder := fakeFinder{
		"/work/api": {Name: "api", Type: "go", Root: "/work/api"},
		"/work/web": {Name: "web", Type: "node", Root: "/work/web"},
	}

	got := build(raw, procs, finder, 20 /* selfPID */)

	if len(got) != 4 {
		t.Fatalf("expected 4 merged listeners, got %d", len(got))
	}
	// Sorted by port: 3000, 5173, 5353, 8080.
	if got[0].Port != 3000 || got[1].Port != 5173 || got[2].Port != 5353 || got[3].Port != 8080 {
		t.Fatalf("unexpected sort order: %d %d %d %d", got[0].Port, got[1].Port, got[2].Port, got[3].Port)
	}

	// :3000 — merged loopback, project attached, not exposed, not self.
	l3000 := got[0]
	if l3000.Exposed {
		t.Error(":3000 should not be exposed (loopback only)")
	}
	if l3000.DisplayAddr() != "localhost" {
		t.Errorf(":3000 DisplayAddr = %q, want localhost", l3000.DisplayAddr())
	}
	if l3000.Project.Name != "api" {
		t.Errorf(":3000 project = %+v", l3000.Project)
	}
	if l3000.Self {
		t.Error(":3000 should not be flagged self")
	}

	// :5173 — service detected and combined with project in the label.
	l5173 := got[1]
	if l5173.Service != "Vite" {
		t.Errorf(":5173 service = %q, want Vite", l5173.Service)
	}
	if l5173.Label() != "Vite · web" {
		t.Errorf(":5173 label = %q", l5173.Label())
	}

	// :5353 — unknown process.
	l5353 := got[2]
	if l5353.Process != "unknown" {
		t.Errorf(":5353 process = %q, want unknown", l5353.Process)
	}
	if !l5353.Exposed {
		t.Error(":5353 on 192.168.x should be exposed")
	}

	// :8080 — exposed, self, normalized display.
	l8080 := got[3]
	if !l8080.Exposed || l8080.DisplayAddr() != "*" {
		t.Errorf(":8080 exposure/addr wrong: exposed=%v addr=%q", l8080.Exposed, l8080.DisplayAddr())
	}
	if !l8080.Self {
		t.Error(":8080 should be flagged self (selfPID=20)")
	}
}

func TestBuildExeDirFallback(t *testing.T) {
	raw := []rawListener{{Proto: ProtoTCP, Addr: "127.0.0.1", Port: 9000, PID: 50}}
	procs := fakeProcs{
		50: {PID: 50, Name: "server", Exe: "/opt/cool/bin/server"}, // no Cwd
	}
	finder := fakeFinder{"/opt/cool/bin": {Name: "cool", Type: "go", Root: "/opt/cool/bin"}}

	got := build(raw, procs, finder, 0)
	if len(got) != 1 || got[0].Project.Name != "cool" {
		t.Errorf("expected exe-dir project fallback, got %+v", got)
	}
}

func TestCheckProto(t *testing.T) {
	for _, ok := range []string{"", "all", "tcp", "udp", "TCP", "  Udp "} {
		if err := CheckProto(ok); err != nil {
			t.Errorf("CheckProto(%q) unexpected error: %v", ok, err)
		}
	}
	for _, bad := range []string{"sctp", "icmp", "nonsense"} {
		if err := CheckProto(bad); err == nil {
			t.Errorf("CheckProto(%q) should have errored", bad)
		}
	}
}
