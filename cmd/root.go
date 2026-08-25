// Package cmd implements the valsenv command-line interface using cobra.
package cmd

import (
	"errors"
	"fmt"
	"io"

	"github.com/spf13/cobra"
)

// exitError wraps a runtime failure with the process exit code it should map
// to. render's RunE returns *exitError{1, err} for input/resolver/render/write
// failures so Execute can distinguish them from cobra-native usage errors.
type exitError struct {
	code int
	err  error
}

func (e *exitError) Error() string { return e.err.Error() }
func (e *exitError) Unwrap() error { return e.err }

// newRootCmd builds a fresh root command tree. A factory keeps tests isolated
// from shared cobra state (flag values, parsed args) between runs.
func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           "valsenv",
		Short:         "Resolve ref+ references in dotenv files via helmfile/vals",
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          cobra.ArbitraryArgs,
		// No bare invocation: a missing subcommand is a usage error (exit 2),
		// matching the pre-cobra contract.
		RunE: func(c *cobra.Command, args []string) error {
			if len(args) == 0 {
				return &exitError{2, errors.New("a subcommand is required")}
			}
			return &exitError{2, fmt.Errorf("unknown command %q", args[0])}
		},
	}
	// A bad flag is a usage error (exit 2), not a runtime error. Tag cobra's
	// flag-parse errors so Execute maps them to 2.
	root.SetFlagErrorFunc(func(c *cobra.Command, err error) error {
		return &exitError{2, err}
	})
	// Only render and version are supported; drop cobra's default completion cmd.
	root.CompletionOptions.DisableDefaultCmd = true
	root.AddCommand(newRenderCmd())
	root.AddCommand(newVersionCmd())
	return root
}

// Execute runs the root command and maps the resulting error to a process exit
// code, preserving the historical contract:
//
//	0 -> success
//	1 -> runtime/render failure (fail-closed; nothing written on error)
//	2 -> usage error (missing/unknown command, unknown flag, unexpected arg)
//
// SilenceErrors is set, so Execute is responsible for printing errors to stderr.
func Execute() int {
	root := newRootCmd()
	return mapError(root.ErrOrStderr(), root.Execute())
}

// mapError prints err (if any) to w and returns the process exit code: 0 for
// nil, an *exitError's own code, or 2 for any other (cobra usage) error.
func mapError(w io.Writer, err error) int {
	if err == nil {
		return 0
	}
	fmt.Fprintf(w, "valsenv: %v\n", err)
	var ee *exitError
	if errors.As(err, &ee) {
		return ee.code
	}
	return 2
}
