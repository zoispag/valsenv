package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestVersion(t *testing.T) {
	var out, errOut bytes.Buffer

	code := executeWithCode([]string{"version"}, strings.NewReader(""), &out, &errOut)

	if code != 0 {
		t.Fatalf("exit = %d, want 0; stderr=%q", code, errOut.String())
	}
	got := out.String()
	for _, want := range []string{"valsenv", "dev"} {
		if !strings.Contains(got, want) {
			t.Errorf("version output missing %q; got %q", want, got)
		}
	}
}
