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
	"reflect"
	"strings"

	"github.com/ligato/networkservicemesh/utils/helper/errtools"
	"github.com/ligato/networkservicemesh/utils/idempotent"
	"github.com/ligato/networkservicemesh/utils/registry"
	"github.com/spf13/viper"
)

const (
	// GlobalConfigFileFlag flag for global config file
	GlobalConfigFileFlag = "configfile"
	// GlobalConfigPathFlag flag for global config dir
	GlobalConfigPathFlag = "configpath"
)

// Plugin provides a ConfigLoader
//      Deps: Plugin Dependencies
//      Viper: An instance of viper.Viper
type Plugin struct {
	idempotent.Impl
	Deps
	Viper *viper.Viper
}

// Init initializes the Plugin
func (p *Plugin) Init() error {
	return errtools.Wrap(p.Impl.IdempotentInit(p.init))
}

// TODO: Add WatchConfig
func (p *Plugin) init() error {
	return nil
}

// Close closes the plugin
func (p *Plugin) Close() error {
	return errtools.Wrap(p.Impl.IdempotentClose(p.close))
}

// TODO: Add WatchConfig
func (p *Plugin) close() error {
	registry.Shared().Delete(p)
	return nil
}

// LoadConfig - Load config from config files or other sources and return them
func (p *Plugin) LoadConfig() (config Config) {
	// TODO with a bit of reflect magic it should be possible
	// to merge loadedconfig and DefaultConfig
	// such for every field in Config it gets:
	// if returnedConfig.field == nil {
	//      returnedConfig.field = loadedConfig.field
	// }
	// if returnedConfig.field == nil {
	//      returnedConfig.field = DefaultConfig.field
	// }

	// TODO also allow for remote config from etcd or CRD
	// TODO also add support for Env Variables
	path := p.Viper.GetString(GlobalConfigPathFlag)
	for _, dir := range strings.Split(path, ":") {
		p.Viper.AddConfigPath(dir)
	}
	p.Viper.SetConfigFile(p.Viper.GetString(GlobalConfigFileFlag))
	err := p.Viper.ReadInConfig()
	if err != nil {
		return p.DefaultConfig
	}
	configType := reflect.TypeOf(p.DefaultConfig)
	if configType.Kind() == reflect.Ptr {
		configType = configType.Elem()
	}
	if !p.Viper.IsSet(p.Name) {
		return p.DefaultConfig
	}
	loadedConfig := reflect.New(configType).Interface()
	err = p.Viper.UnmarshalKey(p.Name, loadedConfig)
	// Note: err here literally can't be anything but nil today given current viper implementation
	// but we check it anyway... but code coverage for testing can't reach it
	if err != nil {
		return p.DefaultConfig
	}
	return loadedConfig
}
