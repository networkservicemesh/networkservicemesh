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

package core

import (
	"time"

	"github.com/ligato/cn-infra/logging"
)

// Option defines the maximum time for which the notification delivery is attempted.
type Option interface {
	//OptionMarkerCore is just for marking implementation that implements this interface.
	OptionMarkerCore()
}

// WithTimeoutOpt defines the maximum time for which the notification delivery is attempted.
type WithTimeoutOpt struct {
	Timeout time.Duration
}

// WithTimeout creates an option for ToChan function that defines a notification delivery timeout.
func WithTimeout(timeout time.Duration) *WithTimeoutOpt {
	return &WithTimeoutOpt{Timeout: timeout}
}

// WithLoggerOpt defines a logger that logs if notification delivery is unsuccessful.
type WithLoggerOpt struct {
	Logger logging.Logger
}

// WithLogger creates an option for ToChan function that specifies a logger to be used.
func WithLogger(logger logging.Logger) *WithLoggerOpt {
	return &WithLoggerOpt{Logger: logger}
}

// OptionMarkerCore is only for marking implementation that implements this interface.
func (marker *WithTimeoutOpt) OptionMarkerCore() {}

// OptionMarkerCore is just for marking implementation that implements this interface.
func (marker *WithLoggerOpt) OptionMarkerCore() {}

// WithPluginsOpt is used in NewAgent()
type WithPluginsOpt interface {
	Option

	// return list named plugins with injected dependencies
	// the order in list impacts the order of Init(), AfterInit(), Close() sequence
	Plugins(...Flavor) []*NamedPlugin
}

// WithPlugin for adding a custom plugins to the Agent
//
// Example:
//
//    flavor := &MyFlavor{}
//	  flavor.Inject()
//    NewAgent(myFlavor, WithPlugin("my-plugin", &MyPlugin{DependencyXY: &flavor.ETCD}))
//    }))
func WithPlugin(pluginName string, plugin Plugin) WithPluginsOpt {
	return &withPluginOpt{&NamedPlugin{PluginName(pluginName), plugin}}
}

// WithPlugin
type withPluginOpt struct {
	plugin *NamedPlugin
}

// OptionMarkerCore is just for marking implementation that it implements this interface
func (opt *withPluginOpt) OptionMarkerCore() {}

// Plugins is just for marking implementation that it implements this interface
func (opt *withPluginOpt) Plugins(...Flavor) []*NamedPlugin {
	return []*NamedPlugin{opt.plugin}
}
