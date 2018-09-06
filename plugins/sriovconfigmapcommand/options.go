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

package sriovconfigmapcommand

import (
	"github.com/ligato/networkservicemesh/plugins/k8sclient"
	"github.com/ligato/networkservicemesh/plugins/logger"
	"github.com/ligato/networkservicemesh/plugins/logger/hooks/pid"
	"github.com/ligato/networkservicemesh/utils/helper/utilities"
	"github.com/ligato/networkservicemesh/utils/registry"
	"github.com/onrik/logrus/filename"
	"github.com/sirupsen/logrus"
)

const (
	// DefaultName of the command.
	// In the event a logger.Plugin is created without a
	// logger.Plugin.Name, this is the value that is used.
	DefaultName = "nsm-generate-sriov-configmap"
)

// Option acts on a Plugin in order to set its Deps or Config
type Option func(*Plugin)

// NewPlugin creates a new Plugin with Deps/Config set by the supplied opts
func NewPlugin(opts ...Option) *Plugin {
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
	p := NewPlugin(opts...)
	return registry.Shared().LoadOrStore(p).(*Plugin)
}

// UseDeps creates an Option to set the Deps for a Plugin
func UseDeps(deps *Deps) Option {
	return func(p *Plugin) {
		d := &p.Deps
		d.Name = deps.Name
		d.Log = deps.Log
		d.Utilities = deps.Utilities
		d.K8sClient = deps.K8sClient
	}
}

var defaultHooks = []logrus.Hook{filename.NewHook(), pid.NewHook()}

// DefaultDeps creates an Option to set any unset Dependencies to Default Values
// DefaultDeps() is always applied by NewPlugin/SharedPlugin after all other Options
func DefaultDeps() Option {
	return func(p *Plugin) {
		d := &p.Deps
		if p.Name == "" {
			d.Name = DefaultName
		}
		if d.Log == nil {
			d.Log = logger.SharedPlugin(logger.DefaultDeps())
		}
		if d.Utilities == nil {
			d.Utilities = utilities.SharedPlugin(utilities.DefaultDeps())
		}
		if d.K8sClient == nil {
			d.K8sClient = k8sclient.SharedPlugin(k8sclient.DefaultDeps())
		}
	}
}
