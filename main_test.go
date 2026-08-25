package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zoispag/valsenv/internal/render"
)

type failResolver struct{}

func (failResolver) Resolve(map[string]string) (map[string]string, error) {
	return nil, errors.New("boom")
}

func TestRunStdinToStdout(t *testing.T) {
	var out, errOut bytes.Buffer
	stdin := strings.NewReader("A=1\n# c\nB=ref+echo://hi\n")

	code := run([]string{"render"}, stdin, &out, &errOut)

	if code != 0 {
		t.Fatalf("exit = %d, want 0; stderr=%q", code, errOut.String())
	}
	got := out.String()
	for _, want := range []string{"A=1", "# c", "B=hi"} {
		if !strings.Contains(got, want) {
			t.Errorf("stdout missing %q; got %q", want, got)
		}
	}
	if errOut.Len() != 0 {
		t.Errorf("stderr = %q, want empty", errOut.String())
	}
}

func TestRunOutputFile(t *testing.T) {
	var out, errOut bytes.Buffer
	target := filepath.Join(t.TempDir(), "sub.env")
	stdin := strings.NewReader("X=ref+echo://yy\n")

	code := run([]string{"render", "-o", target}, stdin, &out, &errOut)

	if code != 0 {
		t.Fatalf("exit = %d, want 0; stderr=%q", code, errOut.String())
	}
	if out.Len() != 0 {
		t.Errorf("stdout = %q, want empty", out.String())
	}
	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read target: %v", err)
	}
	if !strings.Contains(string(data), "X=yy") {
		t.Errorf("target = %q, want X=yy", data)
	}
}

func TestRunMissingSubcommand(t *testing.T) {
	var out, errOut bytes.Buffer
	code := run(nil, strings.NewReader(""), &out, &errOut)
	if code != 2 {
		t.Fatalf("exit = %d, want 2", code)
	}
	if out.Len() != 0 {
		t.Errorf("stdout = %q, want empty", out.String())
	}
	if errOut.Len() == 0 {
		t.Error("stderr empty, want usage")
	}
}

func TestRunUnknownSubcommand(t *testing.T) {
	var out, errOut bytes.Buffer
	code := run([]string{"frobnicate"}, strings.NewReader(""), &out, &errOut)
	if code != 2 {
		t.Fatalf("exit = %d, want 2", code)
	}
	if out.Len() != 0 {
		t.Errorf("stdout = %q, want empty", out.String())
	}
}

func TestRunBadFlag(t *testing.T) {
	var out, errOut bytes.Buffer
	code := run([]string{"render", "-zzz"}, strings.NewReader(""), &out, &errOut)
	if code != 2 {
		t.Fatalf("exit = %d, want 2", code)
	}
	if out.Len() != 0 {
		t.Errorf("stdout = %q, want empty", out.String())
	}
}

func TestRunInputFileNotFound(t *testing.T) {
	var out, errOut bytes.Buffer
	code := run([]string{"render", "-f", "/no/such/file"}, strings.NewReader(""), &out, &errOut)
	if code != 1 {
		t.Fatalf("exit = %d, want 1", code)
	}
	if out.Len() != 0 {
		t.Errorf("stdout = %q, want empty", out.String())
	}
	if errOut.Len() == 0 {
		t.Error("stderr empty, want error")
	}
}

func TestRunWithResolverFailure(t *testing.T) {
	var out, errOut bytes.Buffer
	stdin := strings.NewReader("A=ref+echo://x\n")

	code := runWith([]string{"render"}, stdin, &out, &errOut, failResolver{})

	if code != 1 {
		t.Fatalf("exit = %d, want 1", code)
	}
	if out.Len() != 0 {
		t.Errorf("stdout = %q, want empty on failure", out.String())
	}
	if errOut.Len() == 0 {
		t.Error("stderr empty, want error")
	}
}

func TestRunWithOutputUntouchedOnFailure(t *testing.T) {
	var out, errOut bytes.Buffer
	target := filepath.Join(t.TempDir(), "out.env")
	stdin := strings.NewReader("A=ref+echo://x\n")

	code := runWith([]string{"render", "-o", target}, stdin, &out, &errOut, failResolver{})

	if code != 1 {
		t.Fatalf("exit = %d, want 1", code)
	}
	if out.Len() != 0 {
		t.Errorf("stdout = %q, want empty", out.String())
	}
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Errorf("target exists after failure (err=%v), want absent", err)
	}
}

var _ render.Resolver = failResolver{}
