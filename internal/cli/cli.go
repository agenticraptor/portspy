// Package cli wires the portspy command-line interface together.
package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/agenticraptor/portspy/internal/buildinfo"
	"github.com/agenticraptor/portspy/internal/ports"
	"github.com/agenticraptor/portspy/internal/tui"
)

// Execute runs the root command and returns a process exit code.
func Execute() int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := newRootCmd().ExecuteContext(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 1
	}
	return 0
}

func newRootCmd() *cobra.Command {
	var proto string

	cmd := &cobra.Command{
		Use:   "portspy",
		Short: "See and kill whatever's hogging your local ports",
		Long: `portspy shows everything listening on your local ports — and the
project, command, start time, and process lineage behind each one — then lets
you kill it with one key.

Run with no arguments to open the interactive TUI. Use the subcommands for
scripting:

  portspy list            print listeners as a table (add --json for scripts)
  portspy kill 3000       free a port by terminating whatever holds it
  portspy doctor          check platform support and permissions`,
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       buildinfo.Version,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return tui.Run(cmd.Context(), ports.Options{Proto: proto})
		},
	}
	cmd.Flags().StringVar(&proto, "proto", "all", "protocol filter: tcp, udp, or all")
	cmd.SetVersionTemplate("{{.Name}} {{.Version}}\n")

	cmd.AddCommand(
		newListCmd(),
		newKillCmd(),
		newDoctorCmd(),
		newVersionCmd(),
	)
	return cmd
}

// isTTY reports whether f is an interactive terminal (stdlib-only check).
func isTTY(f *os.File) bool {
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}
