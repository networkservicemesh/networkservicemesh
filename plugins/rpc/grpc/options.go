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

var sharedPlugin *Plugin
var once sync.Once

func SharedPlugin() *Plugin {
	once.Do(func() {
		sharedPlugin = NewPlugin()
	})
	return sharedPlugin
}

func NewPlugin(opts ...Option) *Plugin {
	p := &Plugin{}

	for _, o := range opts {
		o(p)
	}

	deps := &p.Deps
	if deps.PluginName == "" {
		deps.PluginName = "GRPC"
	}
	if deps.Log == nil {
		deps.Log = logging.ForPlugin(string(deps.PluginName), logrus.NewLogRegistry())
	}
	if deps.PluginConfig == nil {
		deps.PluginConfig = config.ForPlugin(string(deps.PluginName))
	}

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

func UseConf(conf grpc.Config) Option {
	return func(p *Plugin) {
		p.Plugin.Config = &conf
	}
}
