package render

import (
	"bytes"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/zoispag/valsenv/internal/dotenv"
)

// Render scans a dotenv stream from r, resolves every full-value ref+/secretref+
// reference through res in a single batch, splices the resolved values back into
// their exact source lines, and writes the byte-faithful result to w.
//
// It is fail-closed: any failure (multiline reject, resolver error, emit
// failure, or residual-reference scan) returns an error and writes NOTHING to
// w. Nothing is ever written to w until the full output is buffered and every
// guard has passed. Per-backend credentials are the resolver's concern: vals
// surfaces its own auth errors uniformly, which Render propagates unchanged.
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

	// Guard B (multiline reject, POST-resolve): a resolved value containing a
	// raw newline/carriage-return cannot be represented on a single dotenv line.
	// Both downstream consumers split naively on '\n', so we reject rather than
	// escape. This fires with a clear per-key message BEFORE any write to w
	// (the dotenv emitter has its own multiline guard as a second layer).
	for _, i := range refIdx {
		if strings.ContainsAny(resolved[strconv.Itoa(i)], "\r\n") {
			return fmt.Errorf("multiline value unsupported for key %q", lines[i].Key)
		}
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

	// Guard C (residual-ref scan, POST-emit): defense-in-depth for our own
	// splicing bugs. vals never emits an unresolved ref on success, so this is a
	// SECONDARY guard, never the only one.
	//
	// Scan is line-based, not a naive substring search: we flag only a KeyVal
	// line whose emitted value STARTS WITH "ref+"/"secretref+", i.e. a reference
	// that survived resolution. A secret whose value merely CONTAINS "ref+"
	// mid-value (e.g. "myref+token") is legitimate and must not trip the guard.
	if err := scanResidualRefs(buf.Bytes()); err != nil {
		return err
	}

	_, err = w.Write(buf.Bytes())
	return err
}

// scanResidualRefs reports an error if any emitted KeyVal line still carries an
// unresolved reference, defined precisely as a value that STARTS WITH "ref+" or
// "secretref+". Comment and non-KeyVal lines are ignored, and refs embedded
// mid-value are not flagged.
func scanResidualRefs(out []byte) error {
	for _, raw := range strings.Split(string(out), "\n") {
		line := strings.TrimSuffix(raw, "\r")
		trimmed := strings.TrimLeft(line, " \t")
		if trimmed == "" || trimmed[0] == '#' {
			continue
		}
		_, val, found := strings.Cut(line, "=")
		if !found {
			continue
		}
		// The emitter may wrap a value in double quotes; strip one leading quote
		// so the prefix check sees the value's real first bytes.
		val = strings.TrimPrefix(val, `"`)
		if strings.HasPrefix(val, "ref+") || strings.HasPrefix(val, "secretref+") {
			return fmt.Errorf("residual unresolved reference in output")
		}
	}
	return nil
}
