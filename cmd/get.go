package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

func newGetCmd() *cobra.Command {
	var inPath, outPath string
	cmd := &cobra.Command{
		Use:   "get [REF]",
		Short: "Resolve a single vals ref+ reference and output its raw value",
		Long: `Resolve a single ref+/secretref+ reference and output the raw resolved value.

The reference is taken from the REF argument, or from --file/stdin where the
entire (whitespace-trimmed) input must be one reference. Unlike render, the
output is the resolved bytes only: no dotenv semantics, no quoting, and
multiline values (e.g. SSH private keys, PEM certificates) are allowed.

The value is written exactly as the backend returns it; no trailing newline is
added or removed.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			if len(args) == 1 && inPath != "" {
				return &exitError{2, errors.New("REF argument and --file are mutually exclusive")}
			}

			var expr string
			if len(args) == 1 {
				expr = args[0]
			} else {
				in := c.InOrStdin()
				if inPath != "" {
					f, err := os.Open(inPath)
					if err != nil {
						return &exitError{1, err}
					}
					defer f.Close()
					in = f
				}
				data, err := io.ReadAll(in)
				if err != nil {
					return &exitError{1, err}
				}
				expr = strings.TrimSpace(string(data))
			}

			// The input must be exactly one full-value reference. Internal
			// whitespace means the caller pointed us at something else (e.g. a
			// dotenv file); fail rather than send garbage to the resolver.
			if !strings.HasPrefix(expr, "ref+") && !strings.HasPrefix(expr, "secretref+") {
				return &exitError{1, fmt.Errorf("input is not a ref+/secretref+ reference")}
			}
			if strings.ContainsAny(expr, " \t\r\n") {
				return &exitError{1, fmt.Errorf("input must be a single reference")}
			}

			res, err := newResolver()
			if err != nil {
				return &exitError{1, err}
			}
			resolved, err := res.Resolve(map[string]string{"0": expr})
			if err != nil {
				return &exitError{1, err}
			}
			val := resolved["0"]

			if outPath == "" {
				if _, err := io.WriteString(c.OutOrStdout(), val); err != nil {
					return &exitError{1, err}
				}
				return nil
			}
			if err := writeAtomic(outPath, []byte(val)); err != nil {
				return &exitError{1, err}
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&inPath, "file", "f", "", "file whose entire content is the reference (default: REF arg or stdin)")
	cmd.Flags().StringVarP(&outPath, "output", "o", "", "output file (default: stdout)")
	return cmd
}
