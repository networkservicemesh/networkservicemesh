// Copyright (c) 2019 Cisco and/or its affiliates.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at:
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package common

import (
	"context"
	"github.com/networkservicemesh/networkservicemesh/pkg/security"

	"github.com/opentracing/opentracing-go"

	"github.com/networkservicemesh/networkservicemesh/sdk/monitor"

	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/registry"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/model"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/local/connection"
	unified_connection "github.com/networkservicemesh/networkservicemesh/controlplane/api/nsm/connection"
)

// ContextKeyType - a type object for context values.
type ContextKeyType string

const (
	clientConnectionKey   ContextKeyType = "ClientConnection"
	modelConnectionKey    ContextKeyType = "ModelConnection"
	monitorServerKey      ContextKeyType = "MonitorServer"
	logKey                ContextKeyType = "Log"
	forwarderKey          ContextKeyType = "Forwarder"
	endpointKey           ContextKeyType = "Endpoint"
	endpointConnectionKey ContextKeyType = "EndpointConnection"
	originalSpan          ContextKeyType = "OriginalSpan"
	ignoredEndpoints      ContextKeyType = "IgnoredEndpoints"
	workspaceName         ContextKeyType = "WorkspaceName"
	securityContextKey    ContextKeyType = "SecurityContext"
)

// WithClientConnection -
//   Wraps 'parent' in a new Context that has the ClientConnection
//   provided in:
//       connection *connection.Connection
//   using Context.Value(...) and returns the result.
//   Note: any previously existing ClientConnection will be overwritten.
//
//   Recommended use: in any Request or Close call that creates a ClientConnection, call:
//      ctx = WithClientConnection(ctx)
//   to ensure that the ctx has a ClientConnection
//   In any Request or Close call that consumes a ClientConnection, call:
//	    connectionMap := ClientConnection(ctx)
//   to retrieve the ClientConnection from the context.Context
func WithClientConnection(parent context.Context, connection *connection.Connection) context.Context {
	if parent == nil {
		parent = context.Background()
	}
	return context.WithValue(parent, clientConnectionKey, connection)
}

// ClientConnection -
//    Returns a ClientConnection from:
//      ctx context.Context
//    If any is present, otherwise nil
func ClientConnection(ctx context.Context) *connection.Connection {
	conn := ctx.Value(clientConnectionKey)
	if conn == nil {
		return nil
	}
	return conn.(*connection.Connection)
}

// WithLog -
//   Provides a FieldLogger in context
func WithLog(parent context.Context, log logrus.FieldLogger) context.Context {
	if parent == nil {
		parent = context.TODO()
	}
	return context.WithValue(parent, logKey, log)
}

// Log - return FieldLogger from context
func Log(ctx context.Context) logrus.FieldLogger {
	if rv, ok := ctx.Value(logKey).(logrus.FieldLogger); ok {
		return rv
	}
	return logrus.New()
}

// WithMonitorServer -
//   Wraps 'parent' in a new Context that has the local connection Monitor
//   using Context.Value(...) and returns the result.
//   Note: any previously existing MonitorServer will be overwritten.
//
func WithMonitorServer(parent context.Context, monitorServer monitor.Server) context.Context {
	if parent == nil {
		parent = context.Background()
	}
	return context.WithValue(parent, monitorServerKey, monitorServer)
}

// MonitorServer -
//    Returns a MonitorServer from:
//      ctx context.Context
//    If any is present, otherwise nil
func MonitorServer(ctx context.Context) monitor.Server {
	value := ctx.Value(monitorServerKey)
	if value == nil {
		return nil
	}
	return value.(monitor.Server)
}

// WithModelConnection -
//   Wraps 'parent' in a new Context that has the model connection
//   using Context.Value(...) and returns the result.
//   Note: any previously existing value will be overwritten.
//
func WithModelConnection(parent context.Context, connection *model.ClientConnection) context.Context {
	if parent == nil {
		parent = context.Background()
	}
	return context.WithValue(parent, modelConnectionKey, connection)
}

// ModelConnection -
//    Returns a ModelConnection from:
//      ctx context.Context
//    If any is present, otherwise nil
func ModelConnection(ctx context.Context) *model.ClientConnection {
	conn := ctx.Value(modelConnectionKey)
	if conn == nil {
		return nil
	}
	return conn.(*model.ClientConnection)
}

// WithForwarder -
//   Wraps 'parent' in a new Context that has the forwarder selected
//   using Context.Value(...) and returns the result.
//   Note: any previously existing value will be overwritten.
//
func WithForwarder(parent context.Context, dp *model.Forwarder) context.Context {
	if parent == nil {
		parent = context.Background()
	}
	return context.WithValue(parent, forwarderKey, dp)
}

// Forwarder - Return forwarder
//
func Forwarder(ctx context.Context) *model.Forwarder {
	value := ctx.Value(forwarderKey)
	if value == nil {
		return nil
	}
	return value.(*model.Forwarder)
}

// WithEndpoint -
//   Wraps 'parent' in a new Context that has the endpoint selected
//   using Context.Value(...) and returns the result.
//   Note: any previously existing value will be overwritten.
//
func WithEndpoint(parent context.Context, endpoint *registry.NSERegistration) context.Context {
	if parent == nil {
		parent = context.Background()
	}
	return context.WithValue(parent, endpointKey, endpoint)
}

// Endpoint - Return selected endpoint object
func Endpoint(ctx context.Context) *registry.NSERegistration {
	value := ctx.Value(endpointKey)
	if value == nil {
		return nil
	}
	return value.(*registry.NSERegistration)
}

// WithEndpointConnection -
//   Wraps 'parent' in a new Context that has endpoint connection object
//   using Context.Value(...) and returns the result.
//   Note: any previously existing value will be overwritten.
//
func WithEndpointConnection(parent context.Context, connection unified_connection.Connection) context.Context {
	if parent == nil {
		parent = context.Background()
	}
	return context.WithValue(parent, endpointConnectionKey, connection)
}

// EndpointConnection - Return endpoint connection object
func EndpointConnection(ctx context.Context) unified_connection.Connection {
	value := ctx.Value(endpointConnectionKey)
	if value == nil {
		return nil
	}
	return value.(unified_connection.Connection)
}

// WithOriginalSpan -
//   Wraps 'parent' in a new Context that has the original opentracing Span.
//   using Context.Value(...) and returns the result.
//   Note: any previously existing value will be overwritten.
//
func WithOriginalSpan(parent context.Context, span opentracing.Span) context.Context {
	if parent == nil {
		parent = context.Background()
	}
	return context.WithValue(parent, originalSpan, span)
}

// OriginalSpan - Return forwarder
func OriginalSpan(ctx context.Context) opentracing.Span {
	value := ctx.Value(originalSpan)
	if value == nil {
		return nil
	}
	return value.(opentracing.Span)
}

// WithIgnoredEndpoints -
//   Wraps 'parent' in a new Context that has the map of ignored endpoints
//   using Context.Value(...) and returns the result.
//   Note: any previously existing value will be overwritten.
//
func WithIgnoredEndpoints(parent context.Context, endpoints map[registry.EndpointNSMName]*registry.NSERegistration) context.Context {
	if parent == nil {
		parent = context.Background()
	}
	return context.WithValue(parent, ignoredEndpoints, endpoints)
}

// IgnoredEndpoints - Return a map of ignored endpoints or empty map.
// key == endpointname + ":" + NetworkServiceManager.Url
func IgnoredEndpoints(ctx context.Context) map[registry.EndpointNSMName]*registry.NSERegistration {
	value := ctx.Value(ignoredEndpoints)
	if value == nil {
		return map[registry.EndpointNSMName]*registry.NSERegistration{}
	}
	return value.(map[registry.EndpointNSMName]*registry.NSERegistration)
}

// WithWorkspaceName -
//   Wraps 'parent' in a new Context that has the workspace name;
//   using Context.Value(...) and returns the result.
//   Note: any previously existing value will be overwritten.
//
func WithWorkspaceName(parent context.Context, name string) context.Context {
	if parent == nil {
		parent = context.Background()
	}
	return context.WithValue(parent, workspaceName, name)
}

// WorkspaceName - Return a workspace name
func WorkspaceName(ctx context.Context) string {
	value := ctx.Value(workspaceName)
	if value == nil {
		return ""
	}
	return value.(string)
}

func WithSecurityContext(parent context.Context, sc security.Context) context.Context {
	if parent == nil {
		parent = context.Background()
	}
	return context.WithValue(parent, securityContextKey, sc)
}

func SecurityContext(ctx context.Context) security.Context {
	value := ctx.Value(securityContextKey)
	if value == nil {
		logrus.Info("IT IS NIL")
		return nil
	}
	logrus.Info("NOT NIL")
	return value.(security.Context)
}
