package cli

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/agenticraptor/portspy/internal/killer"
	"github.com/agenticraptor/portspy/internal/ports"
)

func newKillCmd() *cobra.Command {
	var (
		proto string
		force bool
		yes   bool
	)

	cmd := &cobra.Command{
		Use:   "kill <port> [port...]",
		Short: "Free one or more ports by terminating whatever is listening",
		Long: `Find whatever is listening on the given port(s) and terminate it.

By default a graceful signal is sent first and escalated to a forceful kill only
if the process refuses to exit. Use --force to skip straight to the forceful
kill, and --yes to skip the confirmation prompt (required when not attached to
a terminal).`,
		Args:          cobra.MinimumNArgs(1),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(_ *cobra.Command, args []string) error {
			return runKill(args, proto, force, yes)
		},
	}
	cmd.Flags().StringVar(&proto, "proto", "all", "protocol filter: tcp, udp, or all")
	cmd.Flags().BoolVarP(&force, "force", "f", false, "skip the graceful signal and force-kill immediately")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "do not prompt for confirmation")
	return cmd
}

func runKill(args []string, proto string, force, yes bool) error {
	wantPorts, err := parsePorts(args)
	if err != nil {
		return err
	}

	all, err := ports.Scan(ports.Options{Proto: proto})
	if err != nil {
		return err
	}

	// Collect the matching listeners, de-duplicated by PID.
	var targets []ports.Listener
	seenPID := map[int]bool{}
	for _, p := range wantPorts {
		matched := false
		for _, l := range all {
			if l.Port != p {
				continue
			}
			matched = true
			if l.Self {
				fmt.Fprintf(os.Stderr, "skipping :%d — that's portspy itself\n", p)
				continue
			}
			if !seenPID[l.PID] {
				seenPID[l.PID] = true
				targets = append(targets, l)
			}
		}
		if !matched {
			fmt.Printf("nothing is listening on :%d\n", p)
		}
	}
	if len(targets) == 0 {
		return nil
	}

	fmt.Println("About to kill:")
	for _, l := range targets {
		fmt.Printf("  :%-5d  %s (pid %d) — %s\n", l.Port, l.Process, l.PID, l.Label())
	}

	if !yes {
		if !isTTY(os.Stdin) {
			return fmt.Errorf("refusing to kill without confirmation; re-run with --yes")
		}
		if !confirm(os.Stdin, fmt.Sprintf("Kill %s? [y/N] ", plural(len(targets), "process", "processes"))) {
			fmt.Println("Aborted.")
			return nil
		}
	}

	k := killer.New()
	var failures int
	for _, l := range targets {
		res, err := k.Kill(l.PID, force)
		if err != nil {
			failures++
			fmt.Fprintf(os.Stderr, "  ✗ :%d (pid %d): %v\n", l.Port, l.PID, err)
			continue
		}
		fmt.Printf("  ✓ :%d (pid %d) %s\n", l.Port, l.PID, res.Outcome)
	}
	if failures > 0 {
		return fmt.Errorf("%s could not be killed (try --force, or sudo for processes owned by another user)", plural(failures, "process", "processes"))
	}
	return nil
}

func parsePorts(args []string) ([]int, error) {
	out := make([]int, 0, len(args))
	for _, a := range args {
		a = strings.TrimPrefix(strings.TrimSpace(a), ":")
		n, err := strconv.Atoi(a)
		if err != nil || n < 1 || n > 65535 {
			return nil, fmt.Errorf("invalid port %q (want 1–65535)", a)
		}
		out = append(out, n)
	}
	return out, nil
}

func confirm(in *os.File, prompt string) bool {
	fmt.Print(prompt)
	scanner := bufio.NewScanner(in)
	if !scanner.Scan() {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(scanner.Text())) {
	case "y", "yes":
		return true
	default:
		return false
	}
}

func plural(n int, singular, pluralForm string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, singular)
	}
	return fmt.Sprintf("%d %s", n, pluralForm)
}
