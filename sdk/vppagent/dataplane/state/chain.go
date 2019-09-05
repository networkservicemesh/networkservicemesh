package state

import (
	"context"
	"sync"

	"github.com/golang/protobuf/ptypes/empty"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/crossconnect"
	"github.com/networkservicemesh/networkservicemesh/dataplane/pkg/apis/dataplane"
)

type chainKey string

const (
	chainState chainKey = "chainState"
	current    chainKey = "current"
)

type chain struct {
	source sync.Map
}

func (c *chain) SetCurrent(server dataplane.DataplaneServer) {
	c.source.Store(current, server)
}

func (c *chain) Current() dataplane.DataplaneServer {
	if v, ok := c.source.Load(current); ok {
		if casted, ok := v.(dataplane.DataplaneServer); ok {
			return casted
		}
	}
	return nil
}

func (c *chain) Next(current dataplane.DataplaneServer) dataplane.DataplaneServer {
	if stored, ok := c.source.Load(current); ok {
		if v, ok := stored.(dataplane.DataplaneServer); ok {
			return v
		}
	}
	return nil
}

func WithChain(ctx context.Context, first dataplane.DataplaneServer, next []dataplane.DataplaneServer) context.Context {
	result := &chain{}
	for i := len(next) - 2; i >= 0; i-- {
		result.source.Store(next[i], next[i+1])
	}
	if len(next) > 0 {
		result.source.Store(current, first)
		result.source.Store(first, next[0])
	}
	return context.WithValue(ctx, chainState, result)
}

func NextDataplaneClose(ctx context.Context, crossConnect *crossconnect.CrossConnect) (*empty.Empty, error) {
	next := NextDataplaneServer(ctx)
	if next != nil {
		return next.Close(ctx, crossConnect)
	}
	return new(empty.Empty), nil
}

func NextDataplaneRequest(ctx context.Context, crossConnect *crossconnect.CrossConnect) (*crossconnect.CrossConnect, error) {
	next := NextDataplaneServer(ctx)
	if next != nil {
		return next.Request(ctx, crossConnect)
	}
	return crossConnect, nil
}

func NextDataplaneServer(ctx context.Context) dataplane.DataplaneServer {
	state := getChainState(ctx)
	if state == nil {
		return nil
	}
	next := state.Next(state.Current())
	state.SetCurrent(next)
	return next
}

func getChainState(ctx context.Context) *chain {
	obj := ctx.Value(chainState)
	if obj == nil {
		return nil
	}
	if v, ok := obj.(*chain); ok {
		return v
	}
	return nil
}
