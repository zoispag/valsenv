// Package render resolves dotenv "ref+..." references into secret values.
package render

import (
	"fmt"

	"github.com/helmfile/vals"
)

// Resolver resolves a set of ref+ references to their secret values.
// Keyed by the dotenv KEY; each value is a full "ref+..." expression.
type Resolver interface {
	Resolve(refs map[string]string) (map[string]string, error)
}

// ValsResolver resolves references through the helmfile/vals runtime.
type ValsResolver struct {
	rt *vals.Runtime
}

// NewValsResolver constructs a fail-closed vals runtime. FailOnMissingKeyInMap
// ensures a missing fragment key (e.g. ref+doppler://p/c#/KEY) errors instead
// of silently resolving to nil.
func NewValsResolver() (*ValsResolver, error) {
	rt, err := vals.New(vals.Options{FailOnMissingKeyInMap: true})
	if err != nil {
		return nil, err
	}
	return &ValsResolver{rt: rt}, nil
}

// Resolve evaluates every ref in one batch Eval and returns KEY->value.
// Any Eval error is returned unchanged (fail-closed; caller aborts).
func (r *ValsResolver) Resolve(refs map[string]string) (map[string]string, error) {
	in := make(map[string]interface{}, len(refs))
	for k, v := range refs {
		in[k] = v
	}
	out, err := r.rt.Eval(in)
	if err != nil {
		return nil, err
	}
	return coerce(out)
}

func coerce(out map[string]interface{}) (map[string]string, error) {
	result := make(map[string]string, len(out))
	for k, v := range out {
		s, ok := v.(string)
		if !ok || v == nil {
			return nil, fmt.Errorf("resolver: non-string or nil value for key %q", k)
		}
		result[k] = s
	}
	return result, nil
}
