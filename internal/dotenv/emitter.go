package dotenv

import (
	"fmt"
	"io"
	"strings"
)

// Emit writes lines to w. Non-KeyVal lines and unresolved KeyVal lines are
// written verbatim (Raw+Ending). Resolved KeyVal lines are rewritten as
// Key=quote(Value)+Ending. A resolved value containing a raw newline is
// rejected.
func Emit(w io.Writer, lines []Line) error {
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

		if _, err := io.WriteString(w, l.Key+"="+quote(l.Value)+l.Ending); err != nil {
			return err
		}
	}
	return nil
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
