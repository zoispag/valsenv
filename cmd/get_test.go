package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const multilineKey = "-----BEGIN OPENSSH PRIVATE KEY-----\nabc\ndef\n-----END OPENSSH PRIVATE KEY-----\n"

type stubResolver struct{ val string }

func (s stubResolver) Resolve(refs map[string]string) (map[string]string, error) {
	out := make(map[string]string, len(refs))
	for k := range refs {
		out[k] = s.val
	}
	return out, nil
}

func TestGetRefArg(t *testing.T) {
	var out, errOut bytes.Buffer

	code := executeWithCode([]string{"get", "ref+echo://hello"}, strings.NewReader(""), &out, &errOut)

	if code != 0 {
		t.Fatalf("exit = %d, want 0; stderr=%q", code, errOut.String())
	}
	if got := out.String(); got != "hello" {
		t.Errorf("stdout = %q, want %q (raw value, no added newline)", got, "hello")
	}
}

func TestGetFromStdin(t *testing.T) {
	var out, errOut bytes.Buffer

	code := executeWithCode([]string{"get"}, strings.NewReader("ref+echo://hi\n"), &out, &errOut)

	if code != 0 {
		t.Fatalf("exit = %d, want 0; stderr=%q", code, errOut.String())
	}
	if got := out.String(); got != "hi" {
		t.Errorf("stdout = %q, want %q", got, "hi")
	}
}

func TestGetFromFileToFile(t *testing.T) {
	var out, errOut bytes.Buffer
	dir := t.TempDir()
	in := filepath.Join(dir, "deploy_ssh_key")
	if err := os.WriteFile(in, []byte("ref+echo://key\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(dir, "resolved")

	code := executeWithCode([]string{"get", "-f", in, "-o", target}, strings.NewReader(""), &out, &errOut)

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
	if string(data) != "key" {
		t.Errorf("target = %q, want %q", data, "key")
	}
}

func TestGetResolveInPlace(t *testing.T) {
	var out, errOut bytes.Buffer
	path := filepath.Join(t.TempDir(), "secretfile")
	if err := os.WriteFile(path, []byte("ref+echo://inplace\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	code := executeWithCode([]string{"get", "-f", path, "-o", path}, strings.NewReader(""), &out, &errOut)

	if code != 0 {
		t.Fatalf("exit = %d, want 0; stderr=%q", code, errOut.String())
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "inplace" {
		t.Errorf("file = %q, want %q", data, "inplace")
	}
}

func TestGetMultilineValueAllowed(t *testing.T) {
	withResolver(t, stubResolver{val: multilineKey})
	var out, errOut bytes.Buffer

	code := executeWithCode([]string{"get", "ref+doppler://p/c/KEY"}, strings.NewReader(""), &out, &errOut)

	if code != 0 {
		t.Fatalf("exit = %d, want 0; stderr=%q", code, errOut.String())
	}
	if got := out.String(); got != multilineKey {
		t.Errorf("stdout = %q, want exact multiline value %q", got, multilineKey)
	}
}

func TestGetNonRefInput(t *testing.T) {
	var out, errOut bytes.Buffer

	code := executeWithCode([]string{"get"}, strings.NewReader("PLAIN=value\n"), &out, &errOut)

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

func TestGetMultiRefInputRejected(t *testing.T) {
	var out, errOut bytes.Buffer

	code := executeWithCode([]string{"get"}, strings.NewReader("ref+echo://a\nref+echo://b\n"), &out, &errOut)

	if code != 1 {
		t.Fatalf("exit = %d, want 1", code)
	}
	if out.Len() != 0 {
		t.Errorf("stdout = %q, want empty", out.String())
	}
}

func TestGetNonAsciiWhitespaceRejected(t *testing.T) {
	var out, errOut bytes.Buffer

	code := executeWithCode([]string{"get", "ref+echo://a\vref+echo://b"}, strings.NewReader(""), &out, &errOut)

	if code != 1 {
		t.Fatalf("exit = %d, want 1", code)
	}
	if out.Len() != 0 {
		t.Errorf("stdout = %q, want empty", out.String())
	}
}

func TestGetEmptyInput(t *testing.T) {
	var out, errOut bytes.Buffer

	code := executeWithCode([]string{"get"}, strings.NewReader(""), &out, &errOut)

	if code != 1 {
		t.Fatalf("exit = %d, want 1", code)
	}
}

func TestGetArgAndFileMutuallyExclusive(t *testing.T) {
	var out, errOut bytes.Buffer

	code := executeWithCode([]string{"get", "ref+echo://x", "-f", "whatever"}, strings.NewReader(""), &out, &errOut)

	if code != 2 {
		t.Fatalf("exit = %d, want 2", code)
	}
	if out.Len() != 0 {
		t.Errorf("stdout = %q, want empty", out.String())
	}
}

func TestGetInputFileNotFound(t *testing.T) {
	var out, errOut bytes.Buffer

	code := executeWithCode([]string{"get", "-f", "/no/such/file"}, strings.NewReader(""), &out, &errOut)

	if code != 1 {
		t.Fatalf("exit = %d, want 1", code)
	}
}

func TestGetResolverFailureLeavesTargetUntouched(t *testing.T) {
	withResolver(t, failResolver{})
	var out, errOut bytes.Buffer
	target := filepath.Join(t.TempDir(), "out")

	code := executeWithCode([]string{"get", "ref+echo://x", "-o", target}, strings.NewReader(""), &out, &errOut)

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
