package dotenv

import (
	"bytes"
	"strings"
	"testing"
)

func TestScanRoundTrip(t *testing.T) {
	cases := map[string]string{
		"lf":   "# leading comment\n\nKEY=val\n  # indented comment\nKEY2=a=b=c\nNONL=last",
		"crlf": "# leading comment\r\n\r\nKEY=val\r\n  # indented comment\r\nKEY2=a=b=c\r\nNONL=last",
	}
	for name, in := range cases {
		t.Run(name, func(t *testing.T) {
			lines, err := Scan(strings.NewReader(in))
			if err != nil {
				t.Fatalf("Scan: %v", err)
			}
			var b strings.Builder
			for _, l := range lines {
				b.WriteString(l.Raw)
				b.WriteString(l.Ending)
			}
			if got := b.String(); got != in {
				t.Fatalf("round-trip mismatch\n got: %q\nwant: %q", got, in)
			}
		})
	}
}

func TestScanEqualsInValue(t *testing.T) {
	lines, err := Scan(strings.NewReader("DSN=a=b=c\n"))
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(lines) != 1 {
		t.Fatalf("want 1 line, got %d", len(lines))
	}
	l := lines[0]
	if l.Kind != KindKeyVal {
		t.Fatalf("want KindKeyVal, got %v", l.Kind)
	}
	if l.Key != "DSN" {
		t.Errorf("Key = %q, want DSN", l.Key)
	}
	if l.Value != "a=b=c" {
		t.Errorf("Value = %q, want a=b=c", l.Value)
	}
}

func TestEmitVerbatim(t *testing.T) {
	in := "# comment\n\nKEY=val\n  # indented\nOTHER line no equals\nKEY2=a=b=c\nNONL=last"
	lines, err := Scan(strings.NewReader(in))
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	var b bytes.Buffer
	if err := Emit(&b, lines, QuoteMinimal); err != nil {
		t.Fatalf("Emit: %v", err)
	}
	if got := b.String(); got != in {
		t.Fatalf("verbatim mismatch\n got: %q\nwant: %q", got, in)
	}
}

// naiveParse mimics a downstream consumer: split on the first '=', strip
// surrounding double quotes, then reverse the emitter's escaping.
func naiveParse(line string) (string, string) {
	k, v, _ := strings.Cut(line, "=")
	v = strings.Trim(v, `"`)
	v = strings.NewReplacer(`\"`, `"`, `\\`, `\`).Replace(v)
	return k, v
}

func TestEmitQuotingRoundTrip(t *testing.T) {
	t.Run("specials", func(t *testing.T) {
		lines := []Line{{Kind: KindKeyVal, Key: "KEY", Value: `a b"c`, Resolved: true, Ending: "\n"}}
		var b bytes.Buffer
		if err := Emit(&b, lines, QuoteMinimal); err != nil {
			t.Fatalf("Emit: %v", err)
		}
		_, v := naiveParse(strings.TrimRight(b.String(), "\n"))
		if v != `a b"c` {
			t.Fatalf("recovered %q, want %q", v, `a b"c`)
		}
	})

	t.Run("empty", func(t *testing.T) {
		lines := []Line{{Kind: KindKeyVal, Key: "KEY", Value: "", Resolved: true, Ending: "\n"}}
		var b bytes.Buffer
		if err := Emit(&b, lines, QuoteMinimal); err != nil {
			t.Fatalf("Emit: %v", err)
		}
		if got := b.String(); got != "KEY=\n" {
			t.Fatalf("empty value emitted %q, want %q", got, "KEY=\n")
		}
	})

	t.Run("plain", func(t *testing.T) {
		lines := []Line{{Kind: KindKeyVal, Key: "KEY", Value: "hello", Resolved: true, Ending: "\n"}}
		var b bytes.Buffer
		if err := Emit(&b, lines, QuoteMinimal); err != nil {
			t.Fatalf("Emit: %v", err)
		}
		if got := b.String(); got != "KEY=hello\n" {
			t.Fatalf("plain value emitted %q, want %q", got, "KEY=hello\n")
		}
	})
}

func TestEmitRejectsMultiline(t *testing.T) {
	lines := []Line{{Kind: KindKeyVal, Key: "KEY", Value: "a\nb", Resolved: true, Ending: "\n"}}
	var b bytes.Buffer
	if err := Emit(&b, lines, QuoteMinimal); err == nil {
		t.Fatal("Emit: want error for multiline resolved value, got nil")
	}
}
