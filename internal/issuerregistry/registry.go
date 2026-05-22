package issuerregistry

import "github.com/example/prrject-fatbaby/internal/identity"

type IssuerRegistry struct {
	ByCIK map[string][]identity.SecurityRef
}

func New(byCIK map[string][]identity.SecurityRef) *IssuerRegistry {
	cp := make(map[string][]identity.SecurityRef, len(byCIK))
	for cik, refs := range byCIK {
		vals := make([]identity.SecurityRef, len(refs))
		copy(vals, refs)
		cp[cik] = vals
	}
	return &IssuerRegistry{ByCIK: cp}
}

func (r *IssuerRegistry) ResolveByCIK(cik string) []identity.SecurityRef {
	if r == nil {
		return nil
	}
	refs := r.ByCIK[cik]
	out := make([]identity.SecurityRef, len(refs))
	copy(out, refs)
	return out
}
