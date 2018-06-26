// Copyright (c) 2018 Cisco and/or its affiliates.
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

package grpc

import (
	"github.com/ligato/cn-infra/config"
	"github.com/ligato/cn-infra/logging"
	"github.com/ligato/cn-infra/logging/logrus"
	"github.com/ligato/cn-infra/rpc/grpc"
	"sync"
)

type sharedKey struct {
	grpc.Deps
	grpc.Config
}

var sharedPlugins sync.Map

func SharedPlugin(opts ...Option) *Plugin {
	plugin := NewPlugin(opts...)
	key := &sharedKey{
		plugin.Deps,
		*plugin.Config,
	}
	p, _ := sharedPlugins.LoadOrStore(key, plugin)
	return p.(*Plugin)
}

func NewPlugin(opts ...Option) *Plugin {
	p := &Plugin{}

	for _, o := range opts {
		o(p)
	}

	DefaultDeps()(p)

	return p
}

type Option func(*Plugin)

func UseDeps(deps grpc.Deps) Option {
	return func(p *Plugin) {
		d := &p.Deps
		d.PluginName = deps.PluginName
		d.Log = deps.Log
		d.PluginConfig = deps.PluginConfig
		d.HTTP = deps.HTTP
	}
}

func DefaultDeps() Option {
	return func(p *Plugin) {
		d := &p.Deps
		if d.PluginName == "" {
			d.PluginName = "GRPC"
		}
		if d.Log == nil {
			d.Log = logging.ForPlugin(string(d.PluginName), logrus.NewLogRegistry())
		}
		if d.PluginConfig == nil {
			d.PluginConfig = config.ForPlugin(string(p.Plugin.Deps.PluginName))
		}
	}
}

func UseConf(conf grpc.Config) Option {
	return func(p *Plugin) {
		p.Plugin.Config = &conf
	}
}
