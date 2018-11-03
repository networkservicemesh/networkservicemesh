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

package redis

import (
	"github.com/ligato/cn-infra/db/keyval"
	"github.com/ligato/cn-infra/db/keyval/kvproto"
	"github.com/ligato/cn-infra/flavors/local"
	"github.com/ligato/cn-infra/health/statuscheck"
)

const (
	// healthCheckProbeKey is a key used to probe Redis state.
	healthCheckProbeKey = "probe-redis-connection"
)

// Plugin implements redis plugin.
type Plugin struct {
	Deps
	// Plugin is disabled if there is no config file available
	disabled bool
	// Redis connection encapsulation
	connection *BytesConnectionRedis
	// Read/Write proto modelled data
	protoWrapper *kvproto.ProtoWrapper
}

// Deps lists dependencies of the redis plugin.
type Deps struct {
	local.PluginInfraDeps //inject
}

// Init retrieves redis configuration and establishes a new connection
// with the redis data store.
// If the configuration file doesn't exist or cannot be read, the returned error
// will be of os.PathError type. An untyped error is returned in case the file
// doesn't contain a valid YAML configuration.
func (plugin *Plugin) Init() (err error) {
	redisCfg, err := plugin.getRedisConfig()
	if err != nil || plugin.disabled {
		return err
	}
	// Create client according to config
	client, err := ConfigToClient(redisCfg)
	if err != nil {
		return err
	}
	// Uses config file to establish connection with the database
	plugin.connection, err = NewBytesConnection(client, plugin.Log)
	if err != nil {
		return err
	}
	plugin.protoWrapper = kvproto.NewProtoWrapperWithSerializer(plugin.connection, &keyval.SerializerJSON{})

	// Register for providing status reports (polling mode)
	if plugin.StatusCheck != nil {
		plugin.StatusCheck.Register(plugin.PluginName, func() (statuscheck.PluginState, error) {
			_, _, err := plugin.NewBroker("/").GetValue(healthCheckProbeKey, nil)
			if err == nil {
				return statuscheck.OK, nil
			}
			return statuscheck.Error, err
		})
	} else {
		plugin.Log.Warnf("Unable to start status check for redis")
	}

	return nil
}

// Close does nothing for redis plugin.
func (plugin *Plugin) Close() error {
	return nil
}

// NewBroker creates new instance of prefixed broker that provides API with arguments of type proto.Message.
func (plugin *Plugin) NewBroker(keyPrefix string) keyval.ProtoBroker {
	return plugin.protoWrapper.NewBroker(keyPrefix)
}

// NewWatcher creates new instance of prefixed broker that provides API with arguments of type proto.Message.
func (plugin *Plugin) NewWatcher(keyPrefix string) keyval.ProtoWatcher {
	return plugin.protoWrapper.NewWatcher(keyPrefix)
}

// Disabled returns *true* if the plugin is not in use due to missing
// redis configuration.
func (plugin *Plugin) Disabled() (disabled bool) {
	return plugin.disabled
}

func (plugin *Plugin) getRedisConfig() (cfg interface{}, err error) {
	found, _ := plugin.PluginConfig.GetValue(&struct{}{})
	if !found {
		plugin.Log.Info("Redis config not found, skip loading this plugin")
		plugin.disabled = true
		return
	}
	configFile := plugin.PluginConfig.GetConfigName()
	if configFile != "" {
		cfg, err = LoadConfig(configFile)
		if err != nil {
			return
		}
	}
	return
}
