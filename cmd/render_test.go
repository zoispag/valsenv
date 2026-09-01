package cmd

import (
	"bytes"
	"errors"
	"io"
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

var _ render.Resolver = failResolver{}

// executeWithCode runs a fresh root command with the given args and streams,
// returning the mapped exit code exactly as the real Execute would.
func executeWithCode(args []string, in io.Reader, out, errOut io.Writer) int {
	root := newRootCmd()
	root.SetArgs(args)
	root.SetIn(in)
	root.SetOut(out)
	root.SetErr(errOut)

	return mapError(root.ErrOrStderr(), root.Execute())
}

// withResolver swaps the package resolver seam for the duration of a test.
func withResolver(t *testing.T, r render.Resolver) {
	t.Helper()
	prev := newResolver
	newResolver = func() (render.Resolver, error) { return r, nil }
	t.Cleanup(func() { newResolver = prev })
}

func TestRenderStdinToStdout(t *testing.T) {
	var out, errOut bytes.Buffer
	in := strings.NewReader("A=1\n# c\nB=ref+echo://hi\n")

	code := executeWithCode([]string{"render"}, in, &out, &errOut)

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

func TestRenderOutputFile(t *testing.T) {
	var out, errOut bytes.Buffer
	target := filepath.Join(t.TempDir(), "sub.env")
	in := strings.NewReader("X=ref+echo://yy\n")

	code := executeWithCode([]string{"render", "-o", target}, in, &out, &errOut)

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

func TestRenderMissingSubcommand(t *testing.T) {
	var out, errOut bytes.Buffer
	code := executeWithCode(nil, strings.NewReader(""), &out, &errOut)
	if code != 2 {
		t.Fatalf("exit = %d, want 2", code)
	}
	if out.Len() != 0 {
		t.Errorf("stdout = %q, want empty", out.String())
	}
}

func TestRenderUnknownSubcommand(t *testing.T) {
	var out, errOut bytes.Buffer
	code := executeWithCode([]string{"frobnicate"}, strings.NewReader(""), &out, &errOut)
	if code != 2 {
		t.Fatalf("exit = %d, want 2", code)
	}
	if out.Len() != 0 {
		t.Errorf("stdout = %q, want empty", out.String())
	}
}

func TestRenderBadFlag(t *testing.T) {
	var out, errOut bytes.Buffer
	code := executeWithCode([]string{"render", "-zzz"}, strings.NewReader(""), &out, &errOut)
	if code != 2 {
		t.Fatalf("exit = %d, want 2", code)
	}
	if out.Len() != 0 {
		t.Errorf("stdout = %q, want empty", out.String())
	}
}

func TestRenderUnexpectedArg(t *testing.T) {
	var out, errOut bytes.Buffer
	code := executeWithCode([]string{"render", "extra"}, strings.NewReader(""), &out, &errOut)
	if code != 2 {
		t.Fatalf("exit = %d, want 2", code)
	}
	if out.Len() != 0 {
		t.Errorf("stdout = %q, want empty", out.String())
	}
}

func TestRenderInputFileNotFound(t *testing.T) {
	var out, errOut bytes.Buffer
	code := executeWithCode([]string{"render", "-f", "/no/such/file"}, strings.NewReader(""), &out, &errOut)
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

func TestRenderResolverFailure(t *testing.T) {
	withResolver(t, failResolver{})
	var out, errOut bytes.Buffer
	in := strings.NewReader("A=ref+echo://x\n")

	code := executeWithCode([]string{"render"}, in, &out, &errOut)

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

func TestRenderQuoteShell(t *testing.T) {
	var out, errOut bytes.Buffer
	in := strings.NewReader("S=ref+echo://a;b\n")

	code := executeWithCode([]string{"render", "--quote=shell"}, in, &out, &errOut)

	if code != 0 {
		t.Fatalf("exit = %d, want 0; stderr=%q", code, errOut.String())
	}
	if got := out.String(); got != "S='a;b'\n" {
		t.Errorf("stdout = %q, want %q", got, "S='a;b'\n")
	}
}

func TestRenderQuoteMinimalDefault(t *testing.T) {
	var out, errOut bytes.Buffer
	in := strings.NewReader("S=ref+echo://a;b\n")

	code := executeWithCode([]string{"render"}, in, &out, &errOut)

	if code != 0 {
		t.Fatalf("exit = %d, want 0; stderr=%q", code, errOut.String())
	}
	if got := out.String(); got != "S=a;b\n" {
		t.Errorf("stdout = %q, want %q (default must be minimal)", got, "S=a;b\n")
	}
}

func TestRenderQuoteInvalid(t *testing.T) {
	var out, errOut bytes.Buffer
	in := strings.NewReader("S=ref+echo://x\n")

	code := executeWithCode([]string{"render", "--quote=bogus"}, in, &out, &errOut)

	if code != 2 {
		t.Fatalf("exit = %d, want 2", code)
	}
	if out.Len() != 0 {
		t.Errorf("stdout = %q, want empty", out.String())
	}
	if errOut.Len() == 0 {
		t.Error("stderr empty, want error")
	}
}

func TestRenderOutputUntouchedOnFailure(t *testing.T) {
	withResolver(t, failResolver{})
	var out, errOut bytes.Buffer
	target := filepath.Join(t.TempDir(), "out.env")
	in := strings.NewReader("A=ref+echo://x\n")

	code := executeWithCode([]string{"render", "-o", target}, in, &out, &errOut)

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
