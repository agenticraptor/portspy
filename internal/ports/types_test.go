package ports

import (
	"testing"
	"time"
)

func TestDisplayAddr(t *testing.T) {
	cases := map[string]string{
		"":            "*",
		"0.0.0.0":     "*",
		"::":          "*",
		"[::]":        "*",
		"127.0.0.1":   "localhost",
		"::1":         "localhost",
		"192.168.1.5": "192.168.1.5",
	}
	for in, want := range cases {
		if got := displayAddr(in); got != want {
			t.Errorf("displayAddr(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestAddrClassification(t *testing.T) {
	if !isLoopback("127.0.0.1") || !isLoopback("::1") {
		t.Error("expected loopback addresses to be classified as loopback")
	}
	if isLoopback("0.0.0.0") {
		t.Error("0.0.0.0 is not loopback")
	}
	if !isAllInterfaces("0.0.0.0") || !isAllInterfaces("::") || !isAllInterfaces("") {
		t.Error("expected unspecified addresses to be all-interfaces")
	}
	if isAllInterfaces("127.0.0.1") {
		t.Error("127.0.0.1 is not all-interfaces")
	}
}

func TestLabel(t *testing.T) {
	cases := []struct {
		name string
		l    Listener
		want string
	}{
		{"service+project", Listener{Service: "Vite", Project: Project{Name: "web"}}, "Vite · web"},
		{"service only", Listener{Service: "PostgreSQL", Process: "postgres"}, "PostgreSQL"},
		{"project only", Listener{Project: Project{Name: "api"}, Process: "node"}, "api"},
		{"process fallback", Listener{Process: "redis-server"}, "redis-server"},
	}
	for _, c := range cases {
		if got := c.l.Label(); got != c.want {
			t.Errorf("%s: Label() = %q, want %q", c.name, got, c.want)
		}
	}
}

func TestSort(t *testing.T) {
	now := time.Now()
	base := []Listener{
		{Port: 8080, PID: 30, Process: "redis", CreateTime: now.Add(-time.Hour)},
		{Port: 3000, PID: 10, Process: "node", CreateTime: now.Add(-time.Minute)},
		{Port: 5432, PID: 20, Process: "Apache", CreateTime: now.Add(-24 * time.Hour)},
	}

	byPort := append([]Listener(nil), base...)
	Sort(byPort, SortPort)
	if byPort[0].Port != 3000 || byPort[2].Port != 8080 {
		t.Errorf("SortPort order wrong: %d, %d, %d", byPort[0].Port, byPort[1].Port, byPort[2].Port)
	}

	byPID := append([]Listener(nil), base...)
	Sort(byPID, SortPID)
	if byPID[0].PID != 10 || byPID[2].PID != 30 {
		t.Errorf("SortPID order wrong")
	}

	byProc := append([]Listener(nil), base...)
	Sort(byProc, SortProcess)
	if byProc[0].Process != "Apache" { // case-insensitive
		t.Errorf("SortProcess should be case-insensitive, got %q first", byProc[0].Process)
	}

	byUp := append([]Listener(nil), base...)
	Sort(byUp, SortUptime)
	if byUp[0].Port != 5432 { // longest-running first
		t.Errorf("SortUptime should put longest-running first, got port %d", byUp[0].Port)
	}
}

func TestFilter(t *testing.T) {
	ls := []Listener{
		{Port: 3000, Process: "node", Project: Project{Name: "web"}},
		{Port: 5432, Process: "postgres", Service: "PostgreSQL"},
	}
	if got := Filter(ls, ""); len(got) != 2 {
		t.Errorf("empty filter should return all, got %d", len(got))
	}
	if got := Filter(ls, "postgre"); len(got) != 1 || got[0].Port != 5432 {
		t.Errorf("filter by service failed: %+v", got)
	}
	if got := Filter(ls, "web"); len(got) != 1 || got[0].Port != 3000 {
		t.Errorf("filter by project failed: %+v", got)
	}
	if got := Filter(ls, "3000"); len(got) != 1 {
		t.Errorf("filter by port number failed")
	}
	if got := Filter(ls, "nope"); len(got) != 0 {
		t.Errorf("non-matching filter should return none, got %d", len(got))
	}
}

func TestSortByCycle(t *testing.T) {
	got := []string{}
	s := SortPort
	for i := 0; i < 4; i++ {
		got = append(got, s.String())
		s = s.Next()
	}
	if s != SortPort {
		t.Error("Next() should cycle back to SortPort after 4 steps")
	}
	want := []string{"port", "pid", "process", "uptime"}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("cycle[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestHumanDuration(t *testing.T) {
	cases := []struct {
		d    time.Duration
		want string
	}{
		{0, "—"},
		{-5 * time.Second, "—"},
		{5 * time.Second, "5s"},
		{90 * time.Second, "1m"},
		{3 * time.Hour, "3h"},
		{50 * time.Hour, "2d"},
	}
	for _, c := range cases {
		if got := HumanDuration(c.d); got != c.want {
			t.Errorf("HumanDuration(%v) = %q, want %q", c.d, got, c.want)
		}
	}
}
