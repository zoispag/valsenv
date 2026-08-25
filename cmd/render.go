package cmd

import (
	"bytes"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/zoispag/valsenv/internal/render"
)

// newResolver is a seam so tests can inject a failing resolver; production uses
// the real vals-backed resolver.
var newResolver = func() (render.Resolver, error) { return render.NewValsResolver() }

func newRenderCmd() *cobra.Command {
	var inPath, outPath string
	cmd := &cobra.Command{
		Use:   "render",
		Short: "Render a dotenv stream, resolving vals ref+ references",
		Args:  cobra.NoArgs,
		RunE: func(c *cobra.Command, _ []string) error {
			in := c.InOrStdin()
			if inPath != "" {
				f, err := os.Open(inPath)
				if err != nil {
					return &exitError{1, err}
				}
				defer f.Close()
				in = f
			}

			res, err := newResolver()
			if err != nil {
				return &exitError{1, err}
			}

			// Buffer the full render before writing anything, so failures leave
			// stdout empty and any -o target untouched (fail-closed, atomic).
			var buf bytes.Buffer
			if err := render.Render(in, &buf, res); err != nil {
				return &exitError{1, err}
			}

			if outPath == "" {
				if _, err := c.OutOrStdout().Write(buf.Bytes()); err != nil {
					return &exitError{1, err}
				}
				return nil
			}
			if err := writeAtomic(outPath, buf.Bytes()); err != nil {
				return &exitError{1, err}
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&inPath, "file", "f", "", "input dotenv file (default: stdin)")
	cmd.Flags().StringVarP(&outPath, "output", "o", "", "output file (default: stdout)")
	return cmd
}

// writeAtomic writes data to a temp file in target's directory, then renames it
// over target. On any failure the temp is removed and target is left untouched.
func writeAtomic(target string, data []byte) error {
	dir := filepath.Dir(target)
	tmp, err := os.CreateTemp(dir, ".valsenv-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}
	if err := os.Rename(tmpName, target); err != nil {
		os.Remove(tmpName)
		return err
	}
	return nil
}
