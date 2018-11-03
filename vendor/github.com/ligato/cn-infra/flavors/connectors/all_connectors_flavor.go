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

package connectors

import (
	"github.com/ligato/cn-infra/core"
	"github.com/ligato/cn-infra/datasync"
	"github.com/ligato/cn-infra/datasync/kvdbsync"
	"github.com/ligato/cn-infra/datasync/resync"
	"github.com/ligato/cn-infra/db/keyval/consul"
	"github.com/ligato/cn-infra/db/keyval/etcd"
	"github.com/ligato/cn-infra/db/keyval/redis"
	"github.com/ligato/cn-infra/db/sql/cassandra"
	"github.com/ligato/cn-infra/flavors/local"
	"github.com/ligato/cn-infra/messaging/kafka"
)

// NewAgent returns a new instance of the Agent with plugins.
// It is an alias for core.NewAgent() to implicit use of the FlavorLocal.
func NewAgent(opts ...core.Option) *core.Agent {
	return core.NewAgent(&AllConnectorsFlavor{}, opts...)
}

// WithPlugins for adding custom plugins to SFC Controller
// <listPlugins> is a callback that uses flavor input to
// inject dependencies for custom plugins that are in output
//
// Example:
//
//    NewAgent(connectors.WithPlugins(func(flavor) {
// 	       return []*core.NamedPlugin{{"my-plugin", &MyPlugin{DependencyXY: &flavor.ETCD}}}
//    }))
func WithPlugins(listPlugins func(local *AllConnectorsFlavor) []*core.NamedPlugin) core.WithPluginsOpt {
	return &withPluginsOpt{listPlugins}
}

// AllConnectorsFlavor is a combination of all plugins that allow
// connectivity to external database/messaging...
// Effectively it is combination of ETCD, Kafka, Redis, Cassandra
// plugins.
//
// User/admin can enable those plugins/connectors by providing
// configs (at least endpoints) for them.
type AllConnectorsFlavor struct {
	*local.FlavorLocal

	ETCD         etcd.Plugin
	ETCDDataSync kvdbsync.Plugin

	Consul         consul.Plugin
	ConsulDataSync kvdbsync.Plugin

	Kafka kafka.Plugin

	Redis         redis.Plugin
	RedisDataSync kvdbsync.Plugin

	Cassandra cassandra.Plugin

	ResyncOrch resync.Plugin // the order is important because of AfterInit()

	injected bool
}

// Inject initializes flavor references/dependencies.
func (f *AllConnectorsFlavor) Inject() bool {
	if f.injected {
		return false
	}
	f.injected = true

	if f.FlavorLocal == nil {
		f.FlavorLocal = &local.FlavorLocal{}
	}
	f.FlavorLocal.Inject()

	f.Consul.Deps.PluginInfraDeps = *f.InfraDeps("consul", local.WithConf())
	f.Consul.Deps.Resync = &f.ResyncOrch
	InjectKVDBSync(&f.ConsulDataSync, &f.Consul, f.Consul.PluginName, f.FlavorLocal, &f.ResyncOrch)

	f.ETCD.Deps.PluginInfraDeps = *f.InfraDeps("etcd", local.WithConf())
	f.ETCD.Deps.Resync = &f.ResyncOrch
	InjectKVDBSync(&f.ETCDDataSync, &f.ETCD, f.ETCD.PluginName, f.FlavorLocal, &f.ResyncOrch)

	f.FlavorLocal.StatusCheck.Transport = &datasync.CompositeKVProtoWriter{Adapters: []datasync.KeyProtoValWriter{
		&f.ETCDDataSync,
		&f.ConsulDataSync,
	}}

	f.Redis.Deps.PluginInfraDeps = *f.InfraDeps("redis", local.WithConf())
	InjectKVDBSync(&f.RedisDataSync, &f.Redis, f.Redis.PluginName, f.FlavorLocal, &f.ResyncOrch)

	f.Kafka.Deps.PluginInfraDeps = *f.InfraDeps("kafka", local.WithConf())

	f.Cassandra.Deps.PluginInfraDeps = *f.InfraDeps("cassandra", local.WithConf())

	f.ResyncOrch.PluginLogDeps = *f.LogDeps("resync-orch")

	return true
}

// Plugins combines all Plugins in flavor to the list
func (f *AllConnectorsFlavor) Plugins() []*core.NamedPlugin {
	f.Inject()
	return core.ListPluginsInFlavor(f)
}

// withPluginsOpt is return value of connectors.WithPlugins() utility
// to easily define new plugins for the agent based on LocalFlavor.
type withPluginsOpt struct {
	callback func(local *AllConnectorsFlavor) []*core.NamedPlugin
}

// OptionMarkerCore is just for marking implementation that it implements this interface
func (opt *withPluginsOpt) OptionMarkerCore() {}

// Plugins methods is here to implement core.WithPluginsOpt go interface
// <flavor> is a callback that uses flavor input for dependency injection
// for custom plugins (returned as NamedPlugin)
func (opt *withPluginsOpt) Plugins(flavors ...core.Flavor) []*core.NamedPlugin {
	for _, flavor := range flavors {
		if f, ok := flavor.(*AllConnectorsFlavor); ok {
			return opt.callback(f)
		}
	}

	panic("wrong usage of connectors.WithPlugin() for other than AllConnectorsFlavor")
}
