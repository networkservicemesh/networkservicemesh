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

package crd

import (
	"reflect"
	"sync"

	"github.com/ligato/networkservicemesh/plugins/objectstore"

	"github.com/ligato/networkservicemesh/plugins/handler"

	"github.com/ligato/networkservicemesh/plugins/logger"
	"github.com/ligato/networkservicemesh/utils/command"
)

const (
	// DefaultName of the netmeshplugincrd.Plugin
	DefaultName = "netmeshplugincrd"
	// KubeConfigFlagName - Cmd line flag for specifying kubeconfig filename
	KubeConfigFlagName = "kube-config"
	// KubeConfigFlagDefault - default value of KubeConfig
	KubeConfigFlagDefault = ""
	// KubeConfigFlagUsage - usage for flag for specifying kubeconfig filename
	KubeConfigFlagUsage = "Path to the kubeconfig file to use for the client connection to K8s cluster"
)

// Option acts on a Plugin in order to set its Deps or Config
type Option func(*Plugin)

var sharedPlugins []*Plugin
var sharedPluginLock sync.Mutex

// NewPlugin creates a new Plugin with Deps/Config set by the supplied opts
func NewPlugin(opts ...Option) *Plugin {
	p := newPlugin(opts...)
	sharedPlugins = append(sharedPlugins, p)
	return p
}

func newPlugin(opts ...Option) *Plugin {
	p := &Plugin{}
	for _, o := range opts {
		o(p)
	}
	DefaultDeps()(p)
	return p
}

// SharedPlugin provides a single shared Plugin that has the same Deps/Config as would result
// from the application of opts
func SharedPlugin(opts ...Option) *Plugin {
	p := newPlugin(opts...)
	sharedPluginLock.Lock()
	defer sharedPluginLock.Unlock()
	_, plug := p.findSharedPlugin()
	if plug != nil {
		return plug
	}
	sharedPlugins = append(sharedPlugins, p)
	return p
}

func (p *Plugin) findSharedPlugin() (int, *Plugin) {
	for i, value := range sharedPlugins {
		if reflect.DeepEqual(p.Deps, value.Deps) {
			return i, value
		}
	}
	return -1, nil
}

// UseDeps creates an Option to set the Deps for a Plugin
func UseDeps(deps *Deps) Option {
	return func(p *Plugin) {
		d := &p.Deps
		d.Name = deps.Name
		d.Log = deps.Log
		d.Handler = deps.Handler
		d.ObjectStore = deps.ObjectStore
		d.KubeConfig = deps.KubeConfig
	}
}

// DefaultDeps creates an Option to set any unset Dependencies to Default Values
// DefaultDeps() is always applied by NewPlugin/SharedPlugin after all other Options
func DefaultDeps() Option {
	return func(p *Plugin) {
		d := &p.Deps
		if d.Name == "" {
			d.Name = DefaultName
		}
		if d.Log == nil {
			d.Log = logger.ByName(d.Name)
		}
		if d.Handler == nil {
			d.Handler = handler.SharedPlugin()
		}
		if d.ObjectStore == nil {
			d.ObjectStore = objectstore.SharedPlugin()
		}
		if d.KubeConfig == "" {
			cmd := command.RootCmd()
			flag := cmd.Flags().Lookup(KubeConfigFlagName)
			if flag == nil {
				cmd.Flags().String(KubeConfigFlagName, KubeConfigFlagDefault, KubeConfigFlagUsage)
				flag = cmd.Flags().Lookup(KubeConfigFlagName)
			}
		}
	}
}
