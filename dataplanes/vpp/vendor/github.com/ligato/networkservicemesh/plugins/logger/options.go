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

package logger

import (
	"github.com/ligato/networkservicemesh/plugins/logger/hooks/pid"
	"github.com/ligato/networkservicemesh/utils/registry"
	"github.com/onrik/logrus/filename"
	"github.com/sirupsen/logrus"

	"github.com/ligato/networkservicemesh/plugins/config"
)

const (
	// DefaultName of the logger
	//      In the event a logger.Plugin is created without a
	//      logger.Plugin.Name, this is the value that is used
	DefaultName = "logger"
	// LogNameFieldName - name of the field that is added to each
	// Log entry to indictae which logger is logging
	LogNameFieldName = "logname"
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
	// Set a defensive p.FieldLogger so its *never* nil
	// This should be overwritten by something proper in Init()
	p.Log = logrus.New()
	p.FieldLogger = p.Log
	return p
}

// SharedPlugin provides a single shared Plugin that has the same Deps/Config as would result
// from the application of opts
func SharedPlugin(opts ...Option) *Plugin {
	p := NewPlugin(opts...)
	return registry.Shared().LoadOrStore(p).(*Plugin)
}

// ByName - If you just want a logger with a custom name and the rest of options
// defaulted, this small convenience method will do that for you
func ByName(name string) *Plugin {
	return SharedPlugin(UseDeps(&Deps{Name: name}))
}

// UseDeps creates an Option to set the Deps for a Plugin
func UseDeps(deps *Deps) Option {
	return func(p *Plugin) {
		d := &p.Deps
		d.Name = deps.Name
		d.Fields = deps.Fields
		d.Hooks = deps.Hooks
		d.Formatter = deps.Formatter
		d.Out = deps.Out
		d.ConfigLoader = deps.ConfigLoader
	}
}

var defaultHooks = []logrus.Hook{filename.NewHook(), pid.NewHook()}

// DefaultDeps creates an Option to set any unset Dependencies to Default Values
// DefaultDeps() is always applied by NewPlugin/SharedPlugin after all other Options
func DefaultDeps() Option {
	return func(p *Plugin) {
		d := &p.Deps
		if d.Name == "" {
			d.Name = DefaultName
		}
		// Make sure we have a logname field
		if d.Fields == nil {
			d.Fields = map[string]interface{}{LogNameFieldName: p.Name}
		} else {
			_, found := d.Fields[LogNameFieldName]
			if !found {
				d.Fields[LogNameFieldName] = p.Name
			}
		}

		// Make sure we have filename and pid hooks
		if d.Hooks == nil {
			d.Hooks = defaultHooks
		} else {
			d.Hooks = append(d.Hooks, defaultHooks...)
		}
		if d.Formatter == nil {
			d.Formatter = DefaultFormatter()
		}

		if d.Out == nil {
			d.Out = DefaultOut()
		}

		if d.ConfigLoader == nil {
			d.ConfigLoader = config.SharedPlugin(config.UseDeps(&config.Deps{Name: d.Name,
				DefaultConfig: &defaultConfig}))
		}
	}
}

// UseConfig creates an Option to set values for the Config
func UseConfig(config *Config) Option {
	return func(p *Plugin) {
		p.Config = config
	}
}

func UseLevel(level logrus.Level) Option {
	ce := ConfigEntry{Level: level}
	ces := []ConfigEntry{ce}
	config := &Config{ConfigEntries: ces}
	return UseConfig(config)
}
