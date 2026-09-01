package render

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/zoispag/valsenv/internal/dotenv"
)

type fakeRenderResolver struct {
	values   map[string]string
	err      error
	calls    int
	lastN    int
	received map[string]struct{}
}

func (f *fakeRenderResolver) Resolve(refs map[string]string) (map[string]string, error) {
	f.calls++
	f.lastN = len(refs)
	if f.received == nil {
		f.received = make(map[string]struct{})
	}
	for _, expr := range refs {
		f.received[expr] = struct{}{}
	}
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
	if err := Render(strings.NewReader(input), &out, res, dotenv.QuoteMinimal); err != nil {
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
	if err := Render(strings.NewReader(input), &out, res, dotenv.QuoteMinimal); err != nil {
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
	if err := Render(strings.NewReader(input), &out, res, dotenv.QuoteMinimal); err != nil {
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
	err := Render(strings.NewReader(input), &out, res, dotenv.QuoteMinimal)
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
	if err := Render(strings.NewReader(input), &out, res, dotenv.QuoteMinimal); err != nil {
		t.Fatalf("Render returned error: %v", err)
	}

	want := "DUP=one\nDUP=two\n"
	if got := out.String(); got != want {
		t.Fatalf("duplicate-key mismatch:\n got: %q\nwant: %q", got, want)
	}
}

func TestRejectMultiline(t *testing.T) {
	input := "SECRET=ref+echo://value\n"
	res := &fakeRenderResolver{values: map[string]string{
		"ref+echo://value": "line1\nline2",
	}}

	var out bytes.Buffer
	err := Render(strings.NewReader(input), &out, res, dotenv.QuoteMinimal)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "SECRET") {
		t.Fatalf("expected error mentioning key SECRET, got %v", err)
	}
	if out.Len() != 0 {
		t.Fatalf("expected empty output on multiline reject, got %q", out.String())
	}
}

func TestFailClosedNoPartial(t *testing.T) {
	input := "SECRET=ref+echo://value\n"
	res := &fakeRenderResolver{err: errors.New("resolver failed")}

	var out bytes.Buffer
	err := Render(strings.NewReader(input), &out, res, dotenv.QuoteMinimal)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if out.Len() != 0 {
		t.Fatalf("expected no partial output on resolver error, got %q", out.String())
	}
}

func TestResidualRefGuard(t *testing.T) {
	input := "SECRET=ref+echo://still-a-ref\n"
	res := &fakeRenderResolver{values: map[string]string{
		"ref+echo://still-a-ref": "ref+echo://still-a-ref",
	}}

	var out bytes.Buffer
	err := Render(strings.NewReader(input), &out, res, dotenv.QuoteMinimal)
	if err == nil {
		t.Fatal("expected error on residual ref, got nil")
	}
	if out.Len() != 0 {
		t.Fatalf("expected empty output when residual ref detected, got %q", out.String())
	}
}

func TestRenderQuotedRefs(t *testing.T) {
	cases := []struct {
		name          string
		input         string
		values        map[string]string
		want          string
		wantExprs     []string
		wantNoResolve bool
	}{
		{
			name:      "unquoted ref",
			input:     "A=ref+echo://x\n",
			values:    map[string]string{"ref+echo://x": "aval"},
			want:      "A=aval\n",
			wantExprs: []string{"ref+echo://x"},
		},
		{
			name:      "double-quoted ref",
			input:     "B=\"ref+echo://x\"\n",
			values:    map[string]string{"ref+echo://x": "bval"},
			want:      "B=bval\n",
			wantExprs: []string{"ref+echo://x"},
		},
		{
			name:      "single-quoted ref",
			input:     "C='ref+echo://x'\n",
			values:    map[string]string{"ref+echo://x": "cval"},
			want:      "C=cval\n",
			wantExprs: []string{"ref+echo://x"},
		},
		{
			name:          "non-ref double-quoted verbatim",
			input:         "D=\"plain\"\n",
			values:        map[string]string{},
			want:          "D=\"plain\"\n",
			wantNoResolve: true,
		},
		{
			name:          "half-quoted literal",
			input:         "E=\"ref+echo://x\n",
			values:        map[string]string{},
			want:          "E=\"ref+echo://x\n",
			wantNoResolve: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res := &fakeRenderResolver{values: tc.values}
			var out bytes.Buffer
			if err := Render(strings.NewReader(tc.input), &out, res, dotenv.QuoteMinimal); err != nil {
				t.Fatalf("Render returned error: %v", err)
			}
			if got := out.String(); got != tc.want {
				t.Fatalf("output mismatch:\n got: %q\nwant: %q", got, tc.want)
			}
			if tc.wantNoResolve {
				if res.calls != 0 {
					t.Fatalf("expected resolver not invoked, got %d calls", res.calls)
				}
				return
			}
			for _, want := range tc.wantExprs {
				if _, ok := res.received[want]; !ok {
					t.Fatalf("resolver did not receive expr %q; received %v", want, res.received)
				}
			}
		})
	}
}

func TestRenderShellQuotesResolvedValue(t *testing.T) {
	input := "SECRET=ref+echo://x\nPLAIN=literal\n"
	res := &fakeRenderResolver{values: map[string]string{
		"ref+echo://x": "a;b$c",
	}}

	var out bytes.Buffer
	if err := Render(strings.NewReader(input), &out, res, dotenv.QuoteShell); err != nil {
		t.Fatalf("Render returned error: %v", err)
	}

	want := "SECRET='a;b$c'\nPLAIN=literal\n"
	if got := out.String(); got != want {
		t.Fatalf("shell output mismatch:\n got: %q\nwant: %q", got, want)
	}
}

// TestRenderMinimalRegression guards the default: minimal output for a
// representative file must stay byte-identical to the pre-change behavior.
func TestRenderMinimalRegression(t *testing.T) {
	input := "# comment\nA=ref+echo://plain\nB=ref+echo://spacey\nC=ref+echo://hashy\nD=literal\n"
	res := &fakeRenderResolver{values: map[string]string{
		"ref+echo://plain":  "plainValue123",
		"ref+echo://spacey": "a b",
		"ref+echo://hashy":  "a#b",
	}}

	var out bytes.Buffer
	if err := Render(strings.NewReader(input), &out, res, dotenv.QuoteMinimal); err != nil {
		t.Fatalf("Render returned error: %v", err)
	}

	want := "# comment\nA=plainValue123\nB=\"a b\"\nC=\"a#b\"\nD=literal\n"
	if got := out.String(); got != want {
		t.Fatalf("minimal regression mismatch:\n got: %q\nwant: %q", got, want)
	}
}

func TestRenderShellRejectsMultiline(t *testing.T) {
	input := "SECRET=ref+echo://value\n"
	res := &fakeRenderResolver{values: map[string]string{
		"ref+echo://value": "line1\nline2",
	}}

	var out bytes.Buffer
	err := Render(strings.NewReader(input), &out, res, dotenv.QuoteShell)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "SECRET") {
		t.Fatalf("expected error mentioning key SECRET, got %v", err)
	}
	if out.Len() != 0 {
		t.Fatalf("expected empty output on multiline reject, got %q", out.String())
	}
}

// TestInlineCommentAfterRef pins the DEFINED behavior for an inline comment
// after a ref: the whole trailing text becomes the value, so it does NOT
// silently succeed with a stripped comment. Here the resolver echoes the value
// unchanged, so the emitted value still starts with "ref+" and the residual
// scan catches it (in production vals would instead error on the malformed
// expression). Either way the render fails closed.
func TestInlineCommentAfterRef(t *testing.T) {
	input := "SECRET=ref+echo://x # note\n"
	res := &fakeRenderResolver{values: map[string]string{
		"ref+echo://x # note": "ref+echo://x # note",
	}}

	var out bytes.Buffer
	err := Render(strings.NewReader(input), &out, res, dotenv.QuoteMinimal)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if out.Len() != 0 {
		t.Fatalf("expected empty output, got %q", out.String())
	}
}
