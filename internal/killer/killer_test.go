package killer

import (
	"errors"
	"testing"
	"time"
)

type fakeProc struct {
	terminate func() error
	kill      func() error
	termCalls int
	killCalls int
}

func (f *fakeProc) Terminate() error {
	f.termCalls++
	if f.terminate != nil {
		return f.terminate()
	}
	return nil
}

func (f *fakeProc) Kill() error {
	f.killCalls++
	if f.kill != nil {
		return f.kill()
	}
	return nil
}

// newTestKiller wires a Killer with deterministic, no-op sleeping.
func newTestKiller(p *fakeProc, alive func(int) bool) *Killer {
	return &Killer{
		Lookup: func(int) (Process, error) { return p, nil },
		Alive:  alive,
		Grace:  3 * time.Millisecond,
		Poll:   1 * time.Millisecond,
		sleep:  func(time.Duration) {},
	}
}

func TestKillGraceful(t *testing.T) {
	alive := true
	p := &fakeProc{terminate: func() error { alive = false; return nil }}
	k := newTestKiller(p, func(int) bool { return alive })

	res, err := k.Kill(1234, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Outcome != OutcomeTerminated {
		t.Errorf("outcome = %v, want terminated", res.Outcome)
	}
	if p.termCalls != 1 || p.killCalls != 0 {
		t.Errorf("expected one terminate, no kill; got term=%d kill=%d", p.termCalls, p.killCalls)
	}
}

func TestKillEscalates(t *testing.T) {
	alive := true
	p := &fakeProc{
		terminate: func() error { return nil }, // ignores the graceful signal
		kill:      func() error { alive = false; return nil },
	}
	k := newTestKiller(p, func(int) bool { return alive })

	res, err := k.Kill(1234, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Outcome != OutcomeKilled {
		t.Errorf("outcome = %v, want killed", res.Outcome)
	}
	if p.termCalls != 1 || p.killCalls != 1 {
		t.Errorf("expected terminate then kill; got term=%d kill=%d", p.termCalls, p.killCalls)
	}
}

func TestKillForce(t *testing.T) {
	p := &fakeProc{kill: func() error { return nil }}
	k := newTestKiller(p, func(int) bool { return true })

	res, err := k.Kill(1234, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Outcome != OutcomeKilled {
		t.Errorf("outcome = %v, want killed", res.Outcome)
	}
	if p.termCalls != 0 || p.killCalls != 1 {
		t.Errorf("force should skip terminate; got term=%d kill=%d", p.termCalls, p.killCalls)
	}
}

func TestKillAlreadyGone(t *testing.T) {
	p := &fakeProc{}
	k := newTestKiller(p, func(int) bool { return false })

	res, err := k.Kill(1234, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Outcome != OutcomeGone {
		t.Errorf("outcome = %v, want gone", res.Outcome)
	}
	if p.termCalls != 0 || p.killCalls != 0 {
		t.Errorf("gone process should not be signaled")
	}
}

func TestKillInvalidPID(t *testing.T) {
	k := New()
	if _, err := k.Kill(0, false); err == nil {
		t.Error("expected error for pid 0")
	}
	if _, err := k.Kill(-1, true); err == nil {
		t.Error("expected error for negative pid")
	}
}

func TestKillTerminateError(t *testing.T) {
	p := &fakeProc{terminate: func() error { return errors.New("nope") }}
	k := newTestKiller(p, func(int) bool { return true })

	if _, err := k.Kill(1234, false); err == nil {
		t.Error("expected terminate error to propagate")
	}
}

func TestKillLookupError(t *testing.T) {
	k := &Killer{
		Lookup: func(int) (Process, error) { return nil, errors.New("no such process") },
		Alive:  func(int) bool { return true },
	}
	if _, err := k.Kill(1234, true); err == nil {
		t.Error("expected lookup error to propagate")
	}
}

func TestOutcomeString(t *testing.T) {
	cases := map[Outcome]string{
		OutcomeTerminated: "terminated",
		OutcomeKilled:     "force-killed",
		OutcomeGone:       "already gone",
	}
	for o, want := range cases {
		if got := o.String(); got != want {
			t.Errorf("Outcome(%d).String() = %q, want %q", o, got, want)
		}
	}
}
