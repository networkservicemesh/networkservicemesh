// Copyright (c) 2017 Cisco and/or its affiliates.
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

package rpc

import (
	"github.com/ligato/cn-infra/core"
	"github.com/ligato/cn-infra/flavors/local"
	"github.com/ligato/cn-infra/health/probe"
	"github.com/ligato/cn-infra/rpc/grpc"
	"github.com/ligato/cn-infra/rpc/prometheus"
	"github.com/ligato/cn-infra/rpc/rest"
)

// NewAgent returns a new instance of the Agent with plugins.
// It is an alias for core.NewAgent() to implicit use of the FlavorRPC
func NewAgent(opts ...core.Option) *core.Agent {
	return core.NewAgent(&FlavorRPC{}, opts...)
}

// WithPlugins for adding custom plugins to SFC Controller
// <listPlugins> is a callback that uses flavor input to
// inject dependencies for custom plugins that are in output
//
// Example:
//
//    NewAgent(rpc.WithPlugins(func(flavor) {
// 	       return []*core.NamedPlugin{{"my-plugin", &MyPlugin{DependencyXY: &flavor.GRPC}}}
//    }))
func WithPlugins(listPlugins func(local *FlavorRPC) []*core.NamedPlugin) core.WithPluginsOpt {
	return &withPluginsOpt{listPlugins}
}

// FlavorRPC glues together multiple plugins that provide RPC-like access.
// They are typically used to enable remote management for other plugins.
type FlavorRPC struct {
	*local.FlavorLocal

	HTTP       rest.Plugin
	HTTPProbe  rest.ForkPlugin
	Prometheus prometheus.Plugin

	HealthRPC     probe.Plugin
	PrometheusRPC probe.PrometheusPlugin

	GRPC grpc.Plugin

	injected bool
}

// Inject initializes flavor references/dependencies.
func (f *FlavorRPC) Inject() bool {
	if f.injected {
		return false
	}
	f.injected = true

	if f.FlavorLocal == nil {
		f.FlavorLocal = &local.FlavorLocal{}
	}
	f.FlavorLocal.Inject()

	rest.DeclareHTTPPortFlag("http")
	httpPlugDeps := *f.InfraDeps("http", local.WithConf())
	f.HTTP.Deps.Log = httpPlugDeps.Log
	f.HTTP.Deps.PluginConfig = httpPlugDeps.PluginConfig
	f.HTTP.Deps.PluginName = httpPlugDeps.PluginName

	f.Logs.HTTP = &f.HTTP

	f.Prometheus.Deps.PluginInfraDeps = *f.InfraDeps("prometheus")
	f.Prometheus.HTTP = &f.HTTPProbe

	grpc.DeclareGRPCPortFlag("grpc")
	grpcPlugDeps := *f.InfraDeps("grpc", local.WithConf())
	f.GRPC.Deps.Log = grpcPlugDeps.Log
	f.GRPC.Deps.PluginConfig = grpcPlugDeps.PluginConfig
	f.GRPC.Deps.PluginName = grpcPlugDeps.PluginName
	f.GRPC.Deps.HTTP = &f.HTTP

	rest.DeclareHTTPPortFlag("http-probe")
	httpProbeDeps := *f.InfraDeps("http-probe", local.WithConf())
	f.HTTPProbe.Deps.Log = httpProbeDeps.Log
	f.HTTPProbe.Deps.PluginConfig = httpProbeDeps.PluginConfig
	f.HTTPProbe.Deps.PluginName = httpProbeDeps.PluginName
	f.HTTPProbe.Deps.DefaultHTTP = &f.HTTP

	f.HealthRPC.Deps.PluginInfraDeps = *f.InfraDeps("health-rpc")
	f.HealthRPC.Deps.HTTP = &f.HTTPProbe
	f.HealthRPC.Deps.StatusCheck = &f.StatusCheck
	//TODO f.HealthRPC.Transport inject restsync

	f.PrometheusRPC.PluginInfraDeps = *f.InfraDeps("health-prometheus-rpc")
	f.PrometheusRPC.Prometheus = &f.Prometheus
	f.PrometheusRPC.StatusCheck = &f.StatusCheck

	return true
}

// Plugins combines all Plugins in flavor to the list.
func (f *FlavorRPC) Plugins() []*core.NamedPlugin {
	f.Inject()
	return core.ListPluginsInFlavor(f)
}

// withPluginsOpt is return value of rpc.WithPlugins() utility
// to easily define new plugins for the agent based on LocalFlavor.
type withPluginsOpt struct {
	callback func(local *FlavorRPC) []*core.NamedPlugin
}

// OptionMarkerCore is just for marking implementation that it implements this interface
func (opt *withPluginsOpt) OptionMarkerCore() {}

// Plugins methods is here to implement core.WithPluginsOpt go interface
// <flavor> is a callback that uses flavor input for dependency injection
// for custom plugins (returned as NamedPlugin)
func (opt *withPluginsOpt) Plugins(flavors ...core.Flavor) []*core.NamedPlugin {
	for _, flavor := range flavors {
		if f, ok := flavor.(*FlavorRPC); ok {
			return opt.callback(f)
		}
	}

	panic("wrong usage of rpc.WithPlugin() for other than FlavorRPC")
}
