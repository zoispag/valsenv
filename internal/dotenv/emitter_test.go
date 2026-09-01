package dotenv

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestQuoteShell(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"", ""},
		{"plainValue123", "plainValue123"},
		{"_@%+=:,./-", "_@%+=:,./-"},
		{"a;b", `'a;b'`},
		{"a b", `'a b'`},
		{"a$b", `'a$b'`},
		{"a`b", "'a`b'"},
		{"a'b", `'a'\''b'`},
		{"a#b", `'a#b'`},
		{"a&b", `'a&b'`},
		{"a|b", `'a|b'`},
		{"a*b", `'a*b'`},
		{`a\b`, `'a\b'`},
	}
	for _, tc := range cases {
		if got := quoteShell(tc.in); got != tc.want {
			t.Errorf("quoteShell(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// TestQuoteShellRoundTrip is the strongest guarantee: emit each nasty value in
// shell mode, then have a real POSIX shell `source` the file and echo the var
// back, asserting the recovered bytes equal the original.
func TestQuoteShellRoundTrip(t *testing.T) {
	sh, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("sh not available")
	}

	values := []string{"a;b", "a$b", "a`b", "a b", "it's", "a#b", "$(echo hi)", `a"b`}
	for _, v := range values {
		t.Run(v, func(t *testing.T) {
			file := filepath.Join(t.TempDir(), ".env")
			if err := os.WriteFile(file, []byte("MYKEY="+quoteShell(v)+"\n"), 0o600); err != nil {
				t.Fatalf("write env: %v", err)
			}
			out, err := exec.Command(sh, "-c", `. "$1"; printf %s "$MYKEY"`, "_", file).Output()
			if err != nil {
				t.Fatalf("sourcing failed: %v", err)
			}
			if string(out) != v {
				t.Fatalf("round-trip mismatch: got %q, want %q", out, v)
			}
		})
	}
}
