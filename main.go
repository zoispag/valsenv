// Command valsenv renders a dotenv stream, resolving vals ref+ references.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/zoispag/valsenv/internal/render"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

const usage = "usage: valsenv render [-f file] [-o file]"

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintf(stderr, "valsenv: %s\n", usage)
		return 2
	}
	switch args[0] {
	case "render":
		res, err := render.NewValsResolver()
		if err != nil {
			fmt.Fprintf(stderr, "valsenv: %v\n", err)
			return 1
		}
		return runWith(args, stdin, stdout, stderr, res)
	default:
		fmt.Fprintf(stderr, "valsenv: unknown command %q\n%s\n", args[0], usage)
		return 2
	}
}

// runWith executes the render subcommand with an injected resolver. It buffers
// the full render before writing anything, so failures leave stdout empty and
// any -o target untouched (fail-closed, atomic).
func runWith(args []string, stdin io.Reader, stdout, stderr io.Writer, res render.Resolver) int {
	fs := flag.NewFlagSet("render", flag.ContinueOnError)
	fs.SetOutput(stderr)
	inPath := fs.String("f", "", "input dotenv file (default: stdin)")
	outPath := fs.String("o", "", "output file (default: stdout)")
	if err := fs.Parse(args[1:]); err != nil {
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintf(stderr, "valsenv: unexpected argument %q\n%s\n", fs.Arg(0), usage)
		return 2
	}

	in := stdin
	if *inPath != "" {
		f, err := os.Open(*inPath)
		if err != nil {
			fmt.Fprintf(stderr, "valsenv: %v\n", err)
			return 1
		}
		defer f.Close()
		in = f
	}

	var buf bytes.Buffer
	if err := render.Render(in, &buf, res); err != nil {
		fmt.Fprintf(stderr, "valsenv: %v\n", err)
		return 1
	}

	if *outPath == "" {
		if _, err := stdout.Write(buf.Bytes()); err != nil {
			fmt.Fprintf(stderr, "valsenv: %v\n", err)
			return 1
		}
		return 0
	}

	if err := writeAtomic(*outPath, buf.Bytes()); err != nil {
		fmt.Fprintf(stderr, "valsenv: %v\n", err)
		return 1
	}
	return 0
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
