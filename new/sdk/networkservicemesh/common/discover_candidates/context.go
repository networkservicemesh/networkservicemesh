package discover_candidates

import (
	"context"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/registry"
)

const (
	candidatesKey contextKeyType = "Candidates"
)

type contextKeyType string

// WithCandidates -
//    Wraps 'parent' in a new Context that has the Candidates
func WithCandidates(parent context.Context, candidates *registry.FindNetworkServiceResponse) context.Context {
	if parent == nil {
		parent = context.TODO()
	}
	return context.WithValue(parent, candidatesKey, candidates)
}

// Candidates -
//   Returns the Candidates
func Candidates(ctx context.Context) *registry.FindNetworkServiceResponse {
	if rv, ok := ctx.Value(candidatesKey).(*registry.FindNetworkServiceResponse); ok {
		return rv
	}
	return nil
}
