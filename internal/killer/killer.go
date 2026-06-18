// Package killer terminates processes by PID, escalating from a graceful signal
// to a forceful one when a process refuses to exit. The escalation policy is
// isolated behind injectable hooks so it can be unit-tested without spawning
// real processes.
package killer

import (
	"fmt"
	"time"

	"github.com/shirou/gopsutil/v3/process"
)

// Outcome describes how a kill resolved.
type Outcome int

// Kill outcomes.
const (
	OutcomeTerminated Outcome = iota // exited after a graceful signal
	OutcomeKilled                    // required a forceful kill
	OutcomeGone                      // was not running to begin with
)

// String implements fmt.Stringer for user-facing messages.
func (o Outcome) String() string {
	switch o {
	case OutcomeKilled:
		return "force-killed"
	case OutcomeGone:
		return "already gone"
	default:
		return "terminated"
	}
}

// Process is the subset of process control the killer needs. gopsutil satisfies
// it in production; tests provide fakes.
type Process interface {
	Terminate() error // SIGTERM on Unix; TerminateProcess on Windows
	Kill() error      // SIGKILL on Unix; TerminateProcess on Windows
}

// Result reports how a single kill resolved.
type Result struct {
	PID     int
	Outcome Outcome
}

// Killer terminates processes with a graceful-then-forceful policy.
type Killer struct {
	// Lookup resolves a PID to a controllable process.
	Lookup func(pid int) (Process, error)
	// Alive reports whether a PID is currently running.
	Alive func(pid int) bool
	// Grace is how long to wait for a graceful exit before escalating.
	Grace time.Duration
	// Poll is how often liveness is checked while waiting.
	Poll time.Duration

	sleep func(time.Duration)
}

// New returns a Killer wired to the host OS via gopsutil, with sensible
// graceful-shutdown timing.
func New() *Killer {
	return &Killer{
		Lookup: gopsutilLookup,
		Alive:  gopsutilAlive,
		Grace:  3 * time.Second,
		Poll:   100 * time.Millisecond,
		sleep:  time.Sleep,
	}
}

// Kill terminates pid. With force=false it sends a graceful signal and waits up
// to Grace for the process to exit, escalating to a forceful kill only if it is
// still alive. With force=true it goes straight to the forceful kill.
func (k *Killer) Kill(pid int, force bool) (Result, error) {
	if pid <= 0 {
		return Result{PID: pid}, fmt.Errorf("invalid pid %d", pid)
	}
	if k.Alive != nil && !k.Alive(pid) {
		return Result{PID: pid, Outcome: OutcomeGone}, nil
	}
	p, err := k.Lookup(pid)
	if err != nil {
		return Result{PID: pid}, fmt.Errorf("find process %d: %w", pid, err)
	}

	if force {
		if err := p.Kill(); err != nil {
			return Result{PID: pid}, fmt.Errorf("force kill %d: %w", pid, err)
		}
		return Result{PID: pid, Outcome: OutcomeKilled}, nil
	}

	if err := p.Terminate(); err != nil {
		return Result{PID: pid}, fmt.Errorf("terminate %d: %w", pid, err)
	}
	if k.waitGone(pid) {
		return Result{PID: pid, Outcome: OutcomeTerminated}, nil
	}
	if err := p.Kill(); err != nil {
		return Result{PID: pid}, fmt.Errorf("force kill %d after graceful timeout: %w", pid, err)
	}
	return Result{PID: pid, Outcome: OutcomeKilled}, nil
}

// waitGone polls until the process exits or the grace period elapses.
func (k *Killer) waitGone(pid int) bool {
	if k.Alive == nil {
		return true
	}
	poll := k.Poll
	if poll <= 0 {
		poll = 50 * time.Millisecond
	}
	sleep := k.sleep
	if sleep == nil {
		sleep = time.Sleep
	}
	for waited := time.Duration(0); waited < k.Grace; waited += poll {
		sleep(poll)
		if !k.Alive(pid) {
			return true
		}
	}
	return !k.Alive(pid)
}

// gopsutilLookup adapts a gopsutil process to the Process interface.
func gopsutilLookup(pid int) (Process, error) {
	p, err := process.NewProcess(int32(pid))
	if err != nil {
		return nil, err
	}
	return gopsutilProcess{p}, nil
}

type gopsutilProcess struct{ p *process.Process }

func (g gopsutilProcess) Terminate() error { return g.p.Terminate() }
func (g gopsutilProcess) Kill() error      { return g.p.Kill() }

func gopsutilAlive(pid int) bool {
	ok, err := process.PidExists(int32(pid))
	return err == nil && ok
}
