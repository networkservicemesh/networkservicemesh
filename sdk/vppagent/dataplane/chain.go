package dataplane

import (
	"context"

	"github.com/networkservicemesh/networkservicemesh/dataplane/pkg/apis/dataplane"
)

type chainKeyType string

const (
	chainKey chainKeyType = "chain"
)

type chain struct {
	handlers []dataplane.DataplaneServer
	index    int
}

func (c *chain) Next() dataplane.DataplaneServer {
	if c.index+1 >= len(c.handlers) {
		return nil
	}
	c.index++
	return c.handlers[c.index]
}

// WithChain retruns context with chain of dataplane server handlers
func WithChain(ctx context.Context, next []dataplane.DataplaneServer) context.Context {
	return context.WithValue(ctx, chainKey, &chain{handlers: next, index: 0})
}

// Next returns next dataplane server of current chain state. Returns nil if context has not chain.
func Next(ctx context.Context) dataplane.DataplaneServer {
	if chain, ok := ctx.Value(chainKey).(*chain); ok {
		return chain.Next()
	}
	return nil
}
