// Package dotenv provides a line-preserving scanner and emitter for dotenv
// files. Unlike map-based parsers, it retains original bytes, ordering,
// comments, blank lines, and line terminators so that unmodified lines pass
// through byte-for-byte and only resolved values are rewritten.
package dotenv

import (
	"bufio"
	"io"
	"strings"
)

// LineKind classifies a physical line of a dotenv file.
type LineKind int

const (
	// KindBlank is an empty line or one containing only whitespace.
	KindBlank LineKind = iota
	// KindComment is a line whose first non-whitespace byte is '#'.
	KindComment
	// KindKeyVal is a well-formed KEY=VALUE line.
	KindKeyVal
	// KindOther is any non-blank, non-comment line without an '='.
	KindOther
)

// Line is one physical line of a dotenv file.
type Line struct {
	Raw      string // exact original bytes of the line, WITHOUT the line terminator
	Ending   string // the terminator that followed Raw: "\n", "\r\n", or "" (last line without a trailing newline)
	Kind     LineKind
	Key      string // set only for KindKeyVal
	Value    string // raw value substring for KindKeyVal (may contain '='), as it appeared in the source
	Resolved bool   // false from Scan; the render pipeline sets true after replacing Value with a resolved secret
}

// Scan reads r fully and splits it into ordered, byte-faithful Lines. Each
// line's terminator is preserved in Ending ("\n", "\r\n", or "" for a final
// line lacking a newline). No trimming, quote-stripping, or resolution occurs.
func Scan(r io.Reader) ([]Line, error) {
	br := bufio.NewReader(r)
	var lines []Line

	for {
		chunk, err := br.ReadString('\n')
		if len(chunk) == 0 && err != nil {
			if err == io.EOF {
				return lines, nil
			}
			return nil, err
		}

		raw := chunk
		ending := ""
		if strings.HasSuffix(raw, "\n") {
			raw = raw[:len(raw)-1]
			if strings.HasSuffix(raw, "\r") {
				raw = raw[:len(raw)-1]
				ending = "\r\n"
			} else {
				ending = "\n"
			}
		}

		lines = append(lines, classify(raw, ending))

		if err != nil {
			if err == io.EOF {
				return lines, nil
			}
			return nil, err
		}
	}
}

func classify(raw, ending string) Line {
	l := Line{Raw: raw, Ending: ending}

	trimmed := strings.TrimLeft(raw, " \t")
	switch {
	case trimmed == "":
		l.Kind = KindBlank
	case trimmed[0] == '#':
		l.Kind = KindComment
	default:
		if key, val, found := strings.Cut(raw, "="); found {
			l.Kind = KindKeyVal
			l.Key = key
			l.Value = val
		} else {
			l.Kind = KindOther
		}
	}
	return l
}
