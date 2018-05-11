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

package local

import (
	"github.com/ligato/cn-infra/config"
	"github.com/ligato/cn-infra/core"
	"github.com/ligato/cn-infra/health/statuscheck"
	"github.com/ligato/cn-infra/logging"
	"github.com/ligato/cn-infra/logging/logmanager"
	"github.com/ligato/cn-infra/logging/logrus"
	"github.com/ligato/cn-infra/servicelabel"
	"github.com/namsral/flag"
)

// LogsFlagDefault - default file name
const LogsFlagDefault = "logs.conf"

// LogsFlagUsage used as flag usage (see implementation in declareFlags())
const LogsFlagUsage = "Location of the configuration files; also set via 'LOGS_CONFIG' env variable."

// NewAgent returns a new instance of the Agent with plugins.
// It is an alias for core.NewAgent() to implicit use of the FlavorLocal.
//
// Example:
//
//    local.NewAgent(local.WithPlugins(func(flavor *FlavorLocal) {
// 	       return []*core.NamedPlugin{{"my-plugin", &MyPlugin{DependencyXY: &flavor.StatusCheck}}}
//    }))
func NewAgent(opts ...core.Option) *core.Agent {
	return core.NewAgent(&FlavorLocal{}, opts...)
}

// WithPlugins for adding custom plugins to SFC Controller
// <listPlugins> is a callback that uses flavor input to
// inject dependencies for custom plugins that are in output.
//
// Use this option either for core.NewAgent() or local.NewAgent()
//
// Example:
//
//    NewAgent(local.WithPlugins(func(flavor) {
// 	       return []*core.NamedPlugin{{"my-plugin", &MyPlugin{DependencyXY: &flavor.StatusCheck}}}
//    }))
func WithPlugins(listPlugins func(local *FlavorLocal) []*core.NamedPlugin) core.WithPluginsOpt {
	return &withPluginsOpt{listPlugins}
}

// FlavorLocal glues together very minimal subset of cn-infra plugins
// that can be embedded inside different projects without running
// any agent specific server.
type FlavorLocal struct {
	logRegistry  logging.Registry
	Logs         logmanager.Plugin //needs to be first plugin (it updates log level from config)
	ServiceLabel servicelabel.Plugin
	StatusCheck  statuscheck.Plugin

	injected bool
}

// Inject injects logger into StatusCheck.
// Composite flavors embedding local flavor are supposed to call this
// method.
// Method returns <false> in case the injection has been already executed.
func (f *FlavorLocal) Inject() bool {
	if f.injected {
		return false
	}
	f.injected = true

	declareFlags()

	f.Logs.Deps.LogRegistry = f.LogRegistry()
	f.Logs.Deps.Log = f.LoggerFor("logs")
	f.Logs.Deps.PluginName = core.PluginName("logs")
	f.Logs.Deps.PluginConfig = config.ForPlugin("logs", LogsFlagDefault, LogsFlagUsage)

	f.StatusCheck.Deps.Log = f.LoggerFor("status-check")
	f.StatusCheck.Deps.PluginName = core.PluginName("status-check")

	return true
}

// Plugins combines all Plugins in flavor to the list
func (f *FlavorLocal) Plugins() []*core.NamedPlugin {
	f.Inject()
	return core.ListPluginsInFlavor(f)
}

// LogRegistry for getting Logging Registry instance
// (not thread safe)
func (f *FlavorLocal) LogRegistry() logging.Registry {
	if f.logRegistry == nil {
		f.logRegistry = logrus.NewLogRegistry()
	}

	return f.logRegistry
}

// LoggerFor for getting PlugginLogger instance:
// - logger name is pre-initialized (see logging.ForPlugin)
// This method is just convenient shortcut for Flavor.Inject()
func (f *FlavorLocal) LoggerFor(pluginName string) logging.PluginLogger {
	return logging.ForPlugin(pluginName, f.LogRegistry())
}

// LogDeps is a helper method for injecting PluginLogDeps dependencies with
// plugins from the Local flavor.
// <pluginName> argument value is injected as the plugin name.
// Injected logger uses the same name as the plugin (see logging.ForPlugin)
// This method is just a convenient shortcut to be used in Flavor.Inject()
// by flavors that embed the LocalFlavor.
func (f *FlavorLocal) LogDeps(pluginName string) *PluginLogDeps {
	return &PluginLogDeps{
		logging.ForPlugin(pluginName, f.LogRegistry()),
		core.PluginName(pluginName)}

}

// InfraDeps is a helper method for injecting PluginInfraDeps dependencies with
// plugins from the Local flavor.
// <pluginName> argument value is injected as the plugin name.
// Logging dependencies are resolved using the LogDeps() method.
// Plugin configuration file name is derived from the plugin name,
// see PluginConfig.GetConfigName().
// This method is just a convenient shortcut to be used in Flavor.Inject()
// by flavors that embed the LocalFlavor..
func (f *FlavorLocal) InfraDeps(pluginName string, opts ...InfraDepsOpts) *PluginInfraDeps {
	if len(opts) == 1 {
		if confOpt, ok := opts[0].(*ConfOpts); ok {
			return &PluginInfraDeps{
				*f.LogDeps(pluginName),
				config.ForPlugin(pluginName, confOpt.confDefault, confOpt.confUsage),
				&f.StatusCheck,
				&f.ServiceLabel}
		}
	}

	return &PluginInfraDeps{
		*f.LogDeps(pluginName),
		config.ForPlugin(pluginName),
		&f.StatusCheck,
		&f.ServiceLabel}
}

// InfraDepsOpts is to make typesafe the InfraDeps varargs
type InfraDepsOpts interface {
	// InfraDepsOpts method is maker to declare implementation of InfraDepsOpts interface
	InfraDepsOpts()
}

// WithConf is a function to create option for InfraDeps()
// no need to pass opts (used for defining flag if it was not already defined), if so in this order:
// - default value
// - usage
func WithConf(deafultUsageOpts ...string) *ConfOpts {
	if len(deafultUsageOpts) > 1 {
		return &ConfOpts{deafultUsageOpts[0], deafultUsageOpts[1]}
	} else if len(deafultUsageOpts) > 0 {
		return &ConfOpts{deafultUsageOpts[0], ""}
	}

	return &ConfOpts{}
}

// ConfOpts is a structure that holds default value & usage for configuration flag
type ConfOpts struct {
	confDefault, confUsage string
}

// InfraDepsOpts method is maker to declare implementation of InfraDepsOpts interface
func (*ConfOpts) InfraDepsOpts() {}

func declareFlags() {
	if flag.Lookup(config.DirFlag) == nil {
		flag.String(config.DirFlag, config.DirDefault, config.DirUsage)
	}
}

// withPluginsOpt is return value of local.WithPlugins() utility
// to easily define new plugins for the agent based on LocalFlavor.
type withPluginsOpt struct {
	callback func(local *FlavorLocal) []*core.NamedPlugin
}

// OptionMarkerCore is just for marking implementation that it implements this interface
func (opt *withPluginsOpt) OptionMarkerCore() {}

// Plugins methods is here to implement core.WithPluginsOpt go interface
// <flavor> is a callback that uses flavor input for dependency injection
// for custom plugins (returned as NamedPlugin)
func (opt *withPluginsOpt) Plugins(flavors ...core.Flavor) []*core.NamedPlugin {
	for _, flavor := range flavors {
		if f, ok := flavor.(*FlavorLocal); ok {
			return opt.callback(f)
		}
	}

	panic("wrong usage of local.WithPlugin() for other than FlavorLocal")
}
