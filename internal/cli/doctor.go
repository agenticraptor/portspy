package cli

import (
	"fmt"
	"os"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/agenticraptor/portspy/internal/buildinfo"
	"github.com/agenticraptor/portspy/internal/ports"
)

func newDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:           "doctor",
		Short:         "Check platform support and permissions",
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(_ *cobra.Command, _ []string) error {
			return runDoctor()
		},
	}
}

func runDoctor() error {
	fmt.Println(buildinfo.String())
	fmt.Printf("platform: %s/%s\n", runtime.GOOS, runtime.GOARCH)
	fmt.Printf("privileges: %s\n", privilegeNote())
	fmt.Println()

	fmt.Println("Running a test scan…")
	ls, err := ports.Scan(ports.Options{Proto: "all"})
	if err != nil {
		fmt.Printf("  ✗ scan failed: %v\n", err)
		return err
	}

	var named, withProject, unknown, exposed int
	for _, l := range ls {
		switch {
		case l.Process == "" || l.Process == "unknown":
			unknown++
		default:
			named++
		}
		if !l.Project.Empty() {
			withProject++
		}
		if l.Exposed {
			exposed++
		}
	}

	fmt.Printf("  ✓ found %s\n", plural(len(ls), "listening socket", "listening sockets"))
	fmt.Printf("    • %d attributed to a process\n", named)
	fmt.Printf("    • %d linked to a project\n", withProject)
	fmt.Printf("    • %d bound beyond loopback (exposed)\n", exposed)
	if unknown > 0 {
		fmt.Printf("    • %d could not be attributed", unknown)
		if !isElevated() {
			fmt.Print(" — some may belong to other users; try sudo for full detail")
		}
		fmt.Println()
	}

	fmt.Println()
	fmt.Println("All good. Run `portspy` for the interactive view.")
	return nil
}

func privilegeNote() string {
	if isElevated() {
		return "elevated (full process detail available)"
	}
	return "standard user (process detail for other users may be limited)"
}

// isElevated reports whether the process has administrative privileges. On
// Windows this is approximated as unknown and reported as standard.
func isElevated() bool {
	if runtime.GOOS == "windows" {
		return false
	}
	return os.Geteuid() == 0
}
