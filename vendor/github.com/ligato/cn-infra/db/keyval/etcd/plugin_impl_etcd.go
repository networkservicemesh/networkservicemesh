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

package etcd

import (
	"fmt"
	"time"

	"github.com/ligato/cn-infra/core"
	"github.com/ligato/cn-infra/datasync/resync"
	"github.com/ligato/cn-infra/db/keyval"
	"github.com/ligato/cn-infra/db/keyval/kvproto"
	"github.com/ligato/cn-infra/flavors/local"
	"github.com/ligato/cn-infra/health/statuscheck"
	"github.com/ligato/cn-infra/utils/safeclose"
)

const (
	// healthCheckProbeKey is a key used to probe Etcd state
	healthCheckProbeKey = "/probe-etcd-connection"
)

// Plugin implements etcd plugin.
type Plugin struct {
	Deps
	// Plugin is disabled if there is no config file available
	disabled bool
	// ETCD connection encapsulation
	connection *BytesConnectionEtcd
	// Read/Write proto modelled data
	protoWrapper *kvproto.ProtoWrapper

	autoCompactDone chan struct{}
	reconnectResync bool
	lastConnErr     error
}

// Deps lists dependencies of the etcd plugin.
// If injected, etcd plugin will use StatusCheck to signal the connection status.
type Deps struct {
	local.PluginInfraDeps
	Resync *resync.Plugin
}

// Init retrieves ETCD configuration and establishes a new connection
// with the etcd data store.
// If the configuration file doesn't exist or cannot be read, the returned error
// will be of os.PathError type. An untyped error is returned in case the file
// doesn't contain a valid YAML configuration.
// The function may also return error if TLS connection is selected and the
// CA or client certificate is not accessible(os.PathError)/valid(untyped).
// Check clientv3.New from coreos/etcd for possible errors returned in case
// the connection cannot be established.
func (plugin *Plugin) Init() (err error) {
	// Read ETCD configuration file. Returns error if does not exists.
	etcdCfg, err := plugin.getEtcdConfig()
	if err != nil || plugin.disabled {
		return err
	}
	// Transforms .yaml config to ETCD client configuration
	etcdClientCfg, err := ConfigToClient(&etcdCfg)
	if err != nil {
		return err
	}
	// Uses config file to establish connection with the database
	plugin.connection, err = NewEtcdConnectionWithBytes(*etcdClientCfg, plugin.Log)
	if err != nil {
		plugin.Log.Errorf("Err: %v", err)
		return err
	}
	plugin.reconnectResync = etcdCfg.ReconnectResync
	if etcdCfg.AutoCompact > 0 {
		if etcdCfg.AutoCompact < time.Duration(time.Minute*60) {
			plugin.Log.Warnf("Auto compact option for ETCD is set to less than 60 minutes!")
		}
		plugin.startPeriodicAutoCompact(etcdCfg.AutoCompact)
	}
	plugin.protoWrapper = kvproto.NewProtoWrapperWithSerializer(plugin.connection, &keyval.SerializerJSON{})

	// Register for providing status reports (polling mode).
	if plugin.StatusCheck != nil {
		plugin.StatusCheck.Register(core.PluginName(plugin.PluginName), func() (statuscheck.PluginState, error) {
			_, _, _, err := plugin.connection.GetValue(healthCheckProbeKey)
			if err == nil {
				if plugin.reconnectResync && plugin.lastConnErr != nil {
					plugin.Log.Info("Starting resync after ETCD reconnect")
					if plugin.Resync != nil {
						plugin.Resync.DoResync()
						plugin.lastConnErr = nil
					} else {
						plugin.Log.Warn("Expected resync after ETCD reconnect could not start beacuse of missing Resync plugin")
					}
				}
				return statuscheck.OK, nil
			}
			plugin.lastConnErr = err
			return statuscheck.Error, err
		})
	} else {
		plugin.Log.Warnf("Unable to start status check for etcd")
	}

	return nil
}

// Close shutdowns the connection.
func (plugin *Plugin) Close() error {
	err := safeclose.Close(plugin.autoCompactDone)
	return err
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
// etcd configuration.
func (plugin *Plugin) Disabled() (disabled bool) {
	return plugin.disabled
}

// PutIfNotExists puts given key-value pair into etcd if there is no value set for the key. If the put was successful
// succeeded is true. If the key already exists succeeded is false and the value for the key is untouched.
func (plugin *Plugin) PutIfNotExists(key string, value []byte) (succeeded bool, err error) {
	if plugin.connection != nil {
		return plugin.connection.PutIfNotExists(key, value)
	}
	return false, fmt.Errorf("connection is not established")
}

// Compact compatcs the ETCD database to the specific revision
func (plugin *Plugin) Compact(rev ...int64) (toRev int64, err error) {
	if plugin.connection != nil {
		return plugin.connection.Compact(rev...)
	}
	return 0, fmt.Errorf("connection is not established")
}

func (plugin *Plugin) getEtcdConfig() (Config, error) {
	var etcdCfg Config
	found, err := plugin.PluginConfig.GetValue(&etcdCfg)
	if err != nil {
		return etcdCfg, err
	}
	if !found {
		plugin.Log.Info("ETCD config not found, skip loading this plugin")
		plugin.disabled = true
		return etcdCfg, nil
	}
	return etcdCfg, nil
}

func (plugin *Plugin) startPeriodicAutoCompact(period time.Duration) {
	plugin.autoCompactDone = make(chan struct{})
	go func() {
		plugin.Log.Infof("Starting periodic auto compacting every %v", period)
		for {
			select {
			case <-time.After(period):
				plugin.Log.Debugf("Executing auto compact")
				if toRev, err := plugin.connection.Compact(); err != nil {
					plugin.Log.Errorf("Periodic auto compacting failed: %v", err)
				} else {
					plugin.Log.Infof("Auto compacting finished (to revision %v)", toRev)
				}
			case <-plugin.autoCompactDone:
				return
			}
		}
	}()
}
