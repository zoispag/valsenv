package render

import (
	"bytes"
	"io"
	"strconv"
	"strings"

	"github.com/zoispag/valsenv/internal/dotenv"
)

// Render scans a dotenv stream from r, resolves every full-value ref+/secretref+
// reference through res in a single batch, splices the resolved values back into
// their exact source lines, and writes the byte-faithful result to w.
//
// It is fail-closed: if the resolver errors (or the emit fails), nothing is
// written to w and the error is returned.
func Render(r io.Reader, w io.Writer, res Resolver) error {
	// Stage 1: scan into ordered, byte-faithful lines.
	lines, err := dotenv.Scan(r)
	if err != nil {
		return err
	}

	// Stage 2: collect full-value refs. Only lines whose Value STARTS WITH
	// "ref+" or "secretref+" are references; embedded refs (e.g. "x-ref+...")
	// are out of scope and pass through as literals.
	//
	// The Resolver map is keyed by a synthetic per-line id (the line index as a
	// string), never by the dotenv Key. Keys can repeat within a file, so keying
	// by Key would merge/drop duplicate-key ref lines; per-line ids avoid that.
	var refIdx []int
	refs := make(map[string]string)
	for i, l := range lines {
		if l.Kind != dotenv.KindKeyVal {
			continue
		}
		if strings.HasPrefix(l.Value, "ref+") || strings.HasPrefix(l.Value, "secretref+") {
			refIdx = append(refIdx, i)
			refs[strconv.Itoa(i)] = l.Value
		}
	}

	// Stage 3: no refs -> emit unchanged and return without invoking the resolver.
	if len(refs) == 0 {
		return dotenv.Emit(w, lines)
	}

	// Stage 4: resolve in one batch. Fail-closed: on error, write nothing.
	resolved, err := res.Resolve(refs)
	if err != nil {
		return err
	}

	// Stage 5: splice resolved values back into their exact lines.
	for _, i := range refIdx {
		lines[i].Value = resolved[strconv.Itoa(i)]
		lines[i].Resolved = true
	}

	// Stage 6: buffer the emit; only write to w once the full output is ready,
	// so an emit error never leaves partial output on w.
	var buf bytes.Buffer
	if err := dotenv.Emit(&buf, lines); err != nil {
		return err
	}
	_, err = w.Write(buf.Bytes())
	return err
}
