package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Build metadata, injected at release time via -ldflags -X.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Args:  cobra.NoArgs,
		Run: func(c *cobra.Command, _ []string) {
			fmt.Fprintf(c.OutOrStdout(), "valsenv %s (commit %s, built %s)\n", version, commit, date)
		},
	}
}
