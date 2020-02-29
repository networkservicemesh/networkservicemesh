package vppagent

import (
	"context"

	"go.ligato.io/vpp-agent/v3/proto/ligato/configurator"
	interfaces "go.ligato.io/vpp-agent/v3/proto/ligato/vpp/interfaces"
)

type contextKeyType string

const (
	vppAgentConfigKey contextKeyType = "VppAgentConfig"
	connectionMapKey  contextKeyType = "ConnectionMap"
)

// WithConfig -
//   If 'parent' already has a VppAgentConfig value, returns 'parent'
//   Else wraps 'parent' in a new Context that has an empty VppAgentConfig
//   using Context.Value(...) and returns the result.
//
//   Recommended use: in any Request or Close call, start with:
//      ctx = WithConfig(ctx)
//   to ensure that the ctx has a VppAgentConfig
//   followed by:
//	    vppAgentConfig := VppAgentConfig(ctx)
//   to retrieve the VppAgentConfig from the context.Context
//   feel free to *edit* the VppAgentConfig, but you cannot *replace* it for the
//   Context of a given call.
func WithConfig(parent context.Context) context.Context {
	if parent == nil {
		parent = context.Background()
	}
	value := parent.Value(vppAgentConfigKey)
	if value == nil {
		vppAgentConfig := &configurator.Config{}
		return context.WithValue(parent, vppAgentConfigKey, vppAgentConfig)
	}
	// Note on why this type assertion is safe:
	// Because the vppContextKey is package private, the only way to get an entry of
	// this key type is *here*, so if the value isn't nil, then this function put it there
	// And its of the *VppAgentContextKey type
	return context.WithValue(parent, vppAgentConfigKey, value.(*configurator.Config))
}

// Config -
//  Returns the Config from the Context (if any is present)
//
//  Recommended use: in any Request or Close call, start with:
//      ctx = WithVppAgentConfig(ctx)
//   to ensure that the ctx has a Config
//   followed by:
//	    vppAgentConfig := Config(ctx)
//   to retrieve the Config from the context.Context
//   feel free to *edit* the Config, but you cannot *replace* it for the
//   Context of a given call.
func Config(ctx context.Context) *configurator.Config {
	return ctx.Value(vppAgentConfigKey).(*configurator.Config)
}

// WithConnectionMap -
//   If 'parent' already has a ConnectionMap value, returns 'parent'
//   Else wraps 'parent' in a new Context that has an empty ConnectionMap
//   using Context.Value(...) and returns the result.
//
//   A ConnectionMap is simply a map[*connection.Connection]*interfaces.Interface)
//   mapping Connections to the VppAgent Interface objects they as associated with
//
//   Recommended use: in any Request or Close call, start with:
//      ctx = WithConnectionMap(ctx)
//   to ensure that the ctx has a ConnectionMap
//   followed by:
//	    connectionMap := ConnectionMap(ctx)
//   to retrieve the ConnectionMap from the context.Context
//   feel free to *edit* the ConnectionMap, but you cannot *replace* it for the
//   Context of a given call.
func WithConnectionMap(parent context.Context) context.Context {
	if parent == nil {
		parent = context.Background()
	}
	value := parent.Value(connectionMapKey)
	if value == nil {
		connectionMap := make(map[string]*interfaces.Interface)
		return context.WithValue(parent, connectionMapKey, connectionMap)
	}
	// Note on why this type assertion is safe:
	// Because the vppContextKey is package private, the only way to get an entry of
	// this key type is *here*, so if the value isn't nil, then this function put it there
	// And its of the *VppAgentContextKey type
	return context.WithValue(parent, connectionMapKey, value.(map[string]*interfaces.Interface))
}

// ConnectionMap -
//  Returns the ConnectionMap from the Context (if any is present)
//
//  Recommended use: in any Request or Close call, start with:
//      ctx = WithConnectionMap(ctx)
//   to ensure that the ctx has a ConnectionMap
//   followed by:
//	    connectionMap := ConnectionMap(ctx)
//   to retrieve the ConnectionMap from the context.Context
//   feel free to *edit* the ConnectionMap, but you cannot *replace* it for the
//   Context of a given call.
func ConnectionMap(ctx context.Context) map[string]*interfaces.Interface {
	connectionMap := ctx.Value(connectionMapKey)
	if connectionMap != nil {
		return connectionMap.(map[string]*interfaces.Interface)
	}
	return nil
}
