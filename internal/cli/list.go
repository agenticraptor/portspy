package cli

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/agenticraptor/portspy/internal/ports"
	"github.com/agenticraptor/portspy/internal/render"
)

func newListCmd() *cobra.Command {
	var (
		proto   string
		asJSON  bool
		noColor bool
	)

	cmd := &cobra.Command{
		Use:           "list",
		Short:         "List listening ports as a table or JSON",
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(_ *cobra.Command, _ []string) error {
			ls, err := ports.Scan(ports.Options{Proto: proto})
			if err != nil {
				return err
			}
			ports.Sort(ls, ports.SortPort)

			if asJSON {
				return render.JSON(os.Stdout, ls)
			}
			return render.Table(os.Stdout, ls, !noColor && isTTY(os.Stdout))
		},
	}
	cmd.Flags().StringVar(&proto, "proto", "all", "protocol filter: tcp, udp, or all")
	cmd.Flags().BoolVar(&asJSON, "json", false, "output JSON instead of a table")
	cmd.Flags().BoolVar(&noColor, "no-color", false, "disable colored output")
	return cmd
}
