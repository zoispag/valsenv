package render

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

type fakeRenderResolver struct {
	values map[string]string
	err    error
	calls  int
	lastN  int
}

func (f *fakeRenderResolver) Resolve(refs map[string]string) (map[string]string, error) {
	f.calls++
	f.lastN = len(refs)
	if f.err != nil {
		return nil, f.err
	}
	out := make(map[string]string, len(refs))
	for id, expr := range refs {
		out[id] = f.values[expr]
	}
	return out, nil
}

func TestRenderMixedFile(t *testing.T) {
	input := "# comment\n\nPLAIN=literal\nSECRET=ref+echo://resolved-value\n  # comment\nLAST=ref+echo://last"
	res := &fakeRenderResolver{values: map[string]string{
		"ref+echo://resolved-value": "resolved-value",
		"ref+echo://last":           "last",
	}}

	var out bytes.Buffer
	if err := Render(strings.NewReader(input), &out, res); err != nil {
		t.Fatalf("Render returned error: %v", err)
	}

	want := "# comment\n\nPLAIN=literal\nSECRET=resolved-value\n  # comment\nLAST=last"
	if got := out.String(); got != want {
		t.Fatalf("output mismatch:\n got: %q\nwant: %q", got, want)
	}
}

func TestRenderBatched(t *testing.T) {
	input := "A=ref+echo://a\nB=ref+echo://b\nC=ref+echo://c\n"
	res := &fakeRenderResolver{values: map[string]string{
		"ref+echo://a": "a",
		"ref+echo://b": "b",
		"ref+echo://c": "c",
	}}

	var out bytes.Buffer
	if err := Render(strings.NewReader(input), &out, res); err != nil {
		t.Fatalf("Render returned error: %v", err)
	}

	if res.calls != 1 {
		t.Fatalf("expected exactly 1 resolver call, got %d", res.calls)
	}
	if res.lastN != 3 {
		t.Fatalf("expected 3 refs in the single call, got %d", res.lastN)
	}
}

func TestRenderNoRefs(t *testing.T) {
	input := "# comment\n\nPLAIN=literal\nOTHER=value=with=equals\n"
	res := &fakeRenderResolver{values: map[string]string{}}

	var out bytes.Buffer
	if err := Render(strings.NewReader(input), &out, res); err != nil {
		t.Fatalf("Render returned error: %v", err)
	}

	if got := out.String(); got != input {
		t.Fatalf("output mismatch:\n got: %q\nwant: %q", got, input)
	}
	if res.calls != 0 {
		t.Fatalf("expected resolver not invoked, got %d calls", res.calls)
	}
}

func TestRenderResolverErrorNoOutput(t *testing.T) {
	input := "SECRET=ref+echo://value\n"
	res := &fakeRenderResolver{err: errors.New("boom")}

	var out bytes.Buffer
	err := Render(strings.NewReader(input), &out, res)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if out.Len() != 0 {
		t.Fatalf("expected empty output on error, got %q", out.String())
	}
}

func TestRenderDuplicateKeys(t *testing.T) {
	input := "DUP=ref+echo://one\nDUP=ref+echo://two\n"
	res := &fakeRenderResolver{values: map[string]string{
		"ref+echo://one": "one",
		"ref+echo://two": "two",
	}}

	var out bytes.Buffer
	if err := Render(strings.NewReader(input), &out, res); err != nil {
		t.Fatalf("Render returned error: %v", err)
	}

	want := "DUP=one\nDUP=two\n"
	if got := out.String(); got != want {
		t.Fatalf("duplicate-key mismatch:\n got: %q\nwant: %q", got, want)
	}
}
