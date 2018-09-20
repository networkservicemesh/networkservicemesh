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

package nsmcommand

import (
	"github.com/ligato/networkservicemesh/plugins/nsmserver"
	"github.com/ligato/networkservicemesh/plugins/objectstore"
	"github.com/ligato/networkservicemesh/utils/command"
	"github.com/ligato/networkservicemesh/utils/registry"

	"github.com/ligato/networkservicemesh/plugins/crd"
	"github.com/ligato/networkservicemesh/plugins/finalizer"
	"github.com/ligato/networkservicemesh/plugins/logger"
)

const (
	// DefaultName of the nsmcommand.Plugin
	DefaultName = "netmesh"
)

// Option acts on a Plugin in order to set its Deps or Config
type Option func(*Plugin)

// NewPlugin creates a new Plugin with Deps/Config set by the supplied opts
func NewPlugin(opts ...Option) *Plugin {
	p := newPlugin(opts...)
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
	return registry.Shared().LoadOrStore(p).(*Plugin)
}

// UseDeps creates an Option to set the Deps for a Plugin
func UseDeps(deps *Deps) Option {
	return func(p *Plugin) {
		d := &p.Deps
		d.Name = deps.Name
		d.Cmd = deps.Cmd
		d.Log = deps.Log
		d.CRD = deps.CRD
		d.NSMServer = deps.NSMServer
		d.ObjectStore = deps.ObjectStore
		// d.Finalizer = deps.Finalizer
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
		if d.Cmd == nil {
			d.Cmd = command.RootCmd()
		}
		if d.Log == nil {
			d.Log = logger.ByName(d.Name)
		}
		if d.CRD == nil {
			d.CRD = crd.SharedPlugin()
		}
		if d.NSMServer == nil {
			d.NSMServer = nsmserver.SharedPlugin()
		}
		if d.ObjectStore == nil {
			d.ObjectStore = objectstore.SharedPlugin()
		}
		if d.Finalizer == nil {
			d.Finalizer = finalizer.SharedPlugin()
		}
	}
}
