package render

import (
	"os"
	"strings"
	"testing"
)

func TestResolverOptions(t *testing.T) {
	r, err := NewValsResolver()
	if err != nil {
		t.Fatalf("NewValsResolver() error = %v", err)
	}
	if r == nil {
		t.Fatal("NewValsResolver() returned nil resolver")
	}

	src, err := os.ReadFile("resolver.go")
	if err != nil {
		t.Fatalf("read resolver.go: %v", err)
	}
	if !strings.Contains(string(src), "FailOnMissingKeyInMap: true") {
		t.Error("resolver.go must construct vals with FailOnMissingKeyInMap: true")
	}
}

type fakeResolver struct {
	out   map[string]string
	calls int
}

func (f *fakeResolver) Resolve(refs map[string]string) (map[string]string, error) {
	f.calls++
	return f.out, nil
}

func TestFakeResolver(t *testing.T) {
	var r Resolver = &fakeResolver{out: map[string]string{"API_KEY": "secret"}}

	got, err := r.Resolve(map[string]string{"API_KEY": "ref+doppler://p/c#/API_KEY"})
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if got["API_KEY"] != "secret" {
		t.Errorf("Resolve()[API_KEY] = %q, want %q", got["API_KEY"], "secret")
	}
}

func TestCoerce(t *testing.T) {
	t.Run("all strings", func(t *testing.T) {
		got, err := coerce(map[string]interface{}{"A": "one", "B": "two"})
		if err != nil {
			t.Fatalf("coerce() error = %v", err)
		}
		if got["A"] != "one" || got["B"] != "two" {
			t.Errorf("coerce() = %v", got)
		}
	})

	t.Run("nil value fails closed", func(t *testing.T) {
		_, err := coerce(map[string]interface{}{"A": nil})
		if err == nil {
			t.Fatal("coerce() with nil value should error")
		}
	})

	t.Run("non-string value fails closed", func(t *testing.T) {
		_, err := coerce(map[string]interface{}{"A": 42})
		if err == nil {
			t.Fatal("coerce() with non-string value should error")
		}
	})
}
