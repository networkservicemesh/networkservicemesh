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

package config

import (
	"os"
	"path/filepath"

	"github.com/ligato/networkservicemesh/utils/command"
	"github.com/ligato/networkservicemesh/utils/registry"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	// DefaultName for config - used if no Deps.Name is provided
	DefaultName = "config"
	// DefaultConfigFileSuffix is appended to the config.Plugin.Deps.Name
	// to form the config filename.  Note: Viper requires you to use suffixes
	// matching file contents like .yaml, .json, .toml, .properties etc
	DefaultConfigFileSuffix = ".yaml"

	// ConfigFileUsagePrefix is prepended to the config.Plugin.Deps.Cmd.Name()
	// to form the flag usage
	ConfigFileUsagePrefix = "Config file for "

	// ConfigPathUsagePrefix is prepended to the config.Plugin.Deps.Cmd.Name()
	// to form the flag usage
	ConfigPathUsagePrefix = "Config path for "
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
	addFlags(p)
	return p
}

// SharedPlugin provides a single shared Plugin that has the same Deps/Config as would result
// from the application of opts
func SharedPlugin(opts ...Option) *Plugin {
	p := NewPlugin(opts...) // Use Options to construct Deps/Config
	return registry.Shared().LoadOrStore(p).(*Plugin)
}

// UseDeps creates an Option to set the Deps for a Plugin
func UseDeps(deps *Deps) Option {
	return func(p *Plugin) {
		d := &p.Deps
		d.Name = deps.Name
		d.DefaultConfig = deps.DefaultConfig
		d.Cmd = deps.Cmd
	}
}

// DefaultDeps creates an Option to set any unset Dependencies to Default Values
// NewPlugin()/SharedPlugin() apply DefaultDeps() after all supplied opts
func DefaultDeps() Option {
	return func(p *Plugin) {
		d := &p.Deps
		if d.Name == "" {
			d.Name = DefaultName
		}
		if d.Cmd == nil {
			d.Cmd = command.RootCmd()
		}
		if d.DefaultConfig == nil {
			// If you don't provide a DefaultConfig, we will use an empty
			// one, but life will be very very boring
			type Config struct {
			}
			d.DefaultConfig = &Config{}
		}
		// DefaultConfig intentionally left as nil
	}
}

func addFlags(p *Plugin) {
	d := &p.Deps
	// Setup Viper instance
	p.Viper = viper.New()

	// Bind to global config file if one is offered
	flagName := GlobalConfigFileFlag
	flagDefault := d.Cmd.Name() + DefaultConfigFileSuffix
	flagUsage := ConfigFileUsagePrefix + d.Cmd.Name()
	addFlag(d.Cmd, p.Viper, flagName, flagDefault, flagUsage)

	// Bind to global config dir if one is offered
	flagName = GlobalConfigPathFlag
	// TODO put more thought into default config path
	flagDefault = "/etc/" + d.Cmd.Name()
	executable, _ := os.Executable()
	executableDir := filepath.Dir(executable)
	flagDefault = flagDefault + ":" + executableDir
	flagDefault = flagDefault + ":."
	flagUsage = ConfigPathUsagePrefix + d.Cmd.Name()
	addFlag(d.Cmd, p.Viper, flagName, flagDefault, flagUsage)
	p.Viper.SetConfigFile(p.Viper.GetString(flagName))

	// TODO add support for getting more specific per 'p.Name' config
	// files
}

func addFlag(cmd *cobra.Command, viper *viper.Viper, flagName string, flagDefault string, flagUsage string) {
	flag := cmd.Flags().Lookup(flagName)
	if flag == nil {
		cmd.Flags().String(flagName, flagDefault, flagUsage)
	}
	viper.BindPFlag(flagName, cmd.Flags().Lookup(flagName))
}
