package dotenv

import (
	"fmt"
	"io"
	"strings"
)

// QuoteStyle selects how resolved values are quoted on emit.
type QuoteStyle int

const (
	// QuoteMinimal is the default, byte-faithful quoting: values are emitted
	// bare unless they contain a shell-significant byte, in which case they are
	// double-quoted with '\' and '"' escaped.
	QuoteMinimal QuoteStyle = iota
	// QuoteShell produces output safe for a POSIX shell that consumes the file
	// via `source`/`.`: values are single-quoted (everything inside single
	// quotes is literal) unless they are a "boring" token.
	QuoteShell
)

// Emit writes lines to w. Non-KeyVal lines and unresolved KeyVal lines are
// written verbatim (Raw+Ending). Resolved KeyVal lines are rewritten as
// Key=quote(Value, style)+Ending. A resolved value containing a raw newline is
// rejected regardless of style.
func Emit(w io.Writer, lines []Line, style QuoteStyle) error {
	for _, l := range lines {
		if l.Kind != KindKeyVal || !l.Resolved {
			if _, err := io.WriteString(w, l.Raw+l.Ending); err != nil {
				return err
			}
			continue
		}

		if strings.ContainsAny(l.Value, "\r\n") {
			return fmt.Errorf("dotenv: refusing to emit multiline value for key %q", l.Key)
		}

		if _, err := io.WriteString(w, l.Key+"="+quoteValue(l.Value, style)+l.Ending); err != nil {
			return err
		}
	}
	return nil
}

// quoteValue dispatches to the quoter for the requested style.
func quoteValue(v string, style QuoteStyle) string {
	if style == QuoteShell {
		return quoteShell(v)
	}
	return quote(v)
}

// quote renders a resolved value so a naive consumer (strings.Cut on '=', then
// strings.Trim(v, `"`), then reverse-unescape) recovers the original bytes.
// Values with no shell-significant bytes are emitted bare; otherwise they are
// wrapped in double quotes with '\' and '"' escaped.
func quote(v string) string {
	if v == "" {
		return ""
	}
	if !strings.ContainsAny(v, " \t\"'#\\\r\n") {
		return v
	}
	r := strings.NewReplacer(`\`, `\\`, `"`, `\"`)
	return `"` + r.Replace(v) + `"`
}

// quoteShell renders a resolved value so that a POSIX shell recovers the exact
// bytes when the file is consumed via `source`/`.`. A value is emitted bare
// only when every rune is safe unquoted; otherwise it is single-quoted, since
// everything inside single quotes is literal in POSIX shell, with the sole
// escape of ' becoming '\”.
func quoteShell(v string) string {
	if v == "" {
		return ""
	}
	safe := func(r rune) bool {
		return (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') ||
			strings.ContainsRune("_@%+=:,./-", r)
	}
	bare := true
	for _, r := range v {
		if !safe(r) {
			bare = false
			break
		}
	}
	if bare {
		return v
	}
	return "'" + strings.ReplaceAll(v, "'", `'\''`) + "'"
}
