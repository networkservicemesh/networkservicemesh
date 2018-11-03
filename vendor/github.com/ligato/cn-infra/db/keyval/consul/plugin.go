//  Copyright (c) 2018 Cisco and/or its affiliates.
//
//  Licensed under the Apache License, Version 2.0 (the "License");
//  you may not use this file except in compliance with the License.
//  You may obtain a copy of the License at:
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
//  Unless required by applicable law or agreed to in writing, software
//  distributed under the License is distributed on an "AS IS" BASIS,
//  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//  See the License for the specific language governing permissions and
//  limitations under the License.

package consul

import (
	"github.com/hashicorp/consul/api"
	"github.com/ligato/cn-infra/core"
	"github.com/ligato/cn-infra/datasync/resync"
	"github.com/ligato/cn-infra/db/keyval"
	"github.com/ligato/cn-infra/db/keyval/kvproto"
	"github.com/ligato/cn-infra/flavors/local"
	"github.com/ligato/cn-infra/health/statuscheck"
)

const (
	// healthCheckProbeKey is a key used to probe connection state
	healthCheckProbeKey = "/probe-consul-connection"
)

// Config represents configuration for Consul plugin.
type Config struct {
	Address         string `json:"address"`
	ReconnectResync bool   `json:"resync-after-reconnect"`
}

// Plugin implements Consul as plugin.
type Plugin struct {
	Deps

	// Plugin is disabled if there is no config file available
	disabled bool
	// Consul client encapsulation
	client *Client
	// Read/Write proto modelled data
	protoWrapper *kvproto.ProtoWrapper

	reconnectResync bool
	lastConnErr     error
}

// Deps lists dependencies of the Consul plugin.
// If injected, Consul plugin will use StatusCheck to signal the connection status.
type Deps struct {
	local.PluginInfraDeps
	Resync *resync.Plugin
}

// Disabled returns *true* if the plugin is not in use due to missing configuration.
func (plugin *Plugin) Disabled() bool {
	return plugin.disabled
}

func (plugin *Plugin) getConfig() (*Config, error) {
	var cfg Config
	found, err := plugin.PluginConfig.GetValue(&cfg)
	if err != nil {
		return nil, err
	}
	if !found {
		plugin.Log.Info("Consul config not found, skip loading this plugin")
		plugin.disabled = true
		return nil, nil
	}
	return &cfg, nil
}

// ConfigToClient transforms Config into api.Config,
// which is ready for use with underlying consul package.
func ConfigToClient(cfg *Config) (*api.Config, error) {
	clientCfg := api.DefaultConfig()
	if cfg.Address != "" {
		clientCfg.Address = cfg.Address
	}
	return clientCfg, nil
}

// Init initializes Consul plugin.
func (plugin *Plugin) Init() (err error) {
	cfg, err := plugin.getConfig()
	if err != nil || plugin.disabled {
		return err
	}
	clientCfg, err := ConfigToClient(cfg)
	if err != nil {
		return err
	}
	plugin.client, err = NewClient(clientCfg)
	if err != nil {
		plugin.Log.Errorf("Err: %v", err)
		return err
	}
	plugin.reconnectResync = cfg.ReconnectResync
	plugin.protoWrapper = kvproto.NewProtoWrapperWithSerializer(plugin.client, &keyval.SerializerJSON{})

	// Register for providing status reports (polling mode).
	if plugin.StatusCheck != nil {
		plugin.StatusCheck.Register(core.PluginName(plugin.PluginName), func() (statuscheck.PluginState, error) {
			_, _, _, err := plugin.client.GetValue(healthCheckProbeKey)
			if err == nil {
				if plugin.reconnectResync && plugin.lastConnErr != nil {
					plugin.Log.Info("Starting resync after Consul reconnect")
					if plugin.Resync != nil {
						plugin.Resync.DoResync()
						plugin.lastConnErr = nil
					} else {
						plugin.Log.Warn("Expected resync after Consul reconnect could not start beacuse of missing Resync plugin")
					}
				}
				return statuscheck.OK, nil
			}
			plugin.lastConnErr = err
			return statuscheck.Error, err
		})
	} else {
		plugin.Log.Warnf("Unable to start status check for consul")
	}

	return nil
}

// Close closes Consul plugin.
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
