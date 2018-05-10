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

package etcdv3

import (
	"fmt"
	"time"

	"github.com/ligato/cn-infra/core"
	"github.com/ligato/cn-infra/db/keyval/plugin"
	"github.com/ligato/cn-infra/flavors/local"
	"github.com/ligato/cn-infra/health/statuscheck"
	"github.com/ligato/cn-infra/servicelabel"
	"github.com/ligato/cn-infra/utils/safeclose"
)

const (
	// healthCheckProbeKey is a key used to probe Etcd state
	healthCheckProbeKey = "/probe-etcd-connection"
)

// Plugin implements etcdv3 plugin.
type Plugin struct {
	Deps // inject
	*plugin.Skeleton
	disabled        bool
	connection      *BytesConnectionEtcd
	autoCompactDone chan struct{}
}

// Deps lists dependencies of the etcdv3 plugin.
// If injected, etcd plugin will use StatusCheck to signal the connection status.
type Deps struct {
	local.PluginInfraDeps // inject
}

// Init retrieves etcd configuration and establishes a new connection
// with the etcd data store.
// If the configuration file doesn't exist or cannot be read, the returned error
// will be of os.PathError type. An untyped error is returned in case the file
// doesn't contain a valid YAML configuration.
// The function may also return error if TLS connection is selected and the
// CA or client certificate is not accessible(os.PathError)/valid(untyped).
// Check clientv3.New from coreos/etcd for possible errors returned in case
// the connection cannot be established.
func (p *Plugin) Init() (err error) {
	// Init connection
	if p.Skeleton == nil {
		// Retrieve config
		var cfg Config
		if found, err := p.PluginConfig.GetValue(&cfg); err != nil {
			return err
		} else if !found {
			p.Log.Info("etcd config not found ", p.PluginConfig.GetConfigName(), " - skip loading this plugin")
			p.disabled = true
			return nil
		}

		etcdConfig, err := ConfigToClientv3(&cfg)
		if err != nil {
			return err
		}

		p.connection, err = NewEtcdConnectionWithBytes(*etcdConfig, p.Log)
		if err != nil {
			return err
		}

		if cfg.AutoCompact > 0 {
			if cfg.AutoCompact < time.Duration(time.Minute*60) {
				p.Log.Warnf("auto compact option for ETCD is set to less than 60 minutes!")
			}
			p.startPeriodicAutoCompact(cfg.AutoCompact)
		}

		p.Skeleton = plugin.NewSkeleton(p.String(),
			p.ServiceLabel,
			p.connection,
		)
	}

	if err := p.Skeleton.Init(); err != nil {
		return err
	}

	// Register for providing status reports (polling mode).
	if p.StatusCheck != nil {
		p.StatusCheck.Register(core.PluginName(p.String()), func() (statuscheck.PluginState, error) {
			_, _, _, err := p.connection.GetValue(healthCheckProbeKey)
			if err == nil {
				return statuscheck.OK, nil
			}
			return statuscheck.Error, err
		})
	} else {
		p.Log.Warnf("Unable to start status check for etcd")
	}

	return nil
}

// AfterInit registers status polling function with StatusCheck plugin
// (if injected).
func (p *Plugin) AfterInit() error {
	return nil
}

func (p *Plugin) startPeriodicAutoCompact(period time.Duration) {
	p.autoCompactDone = make(chan struct{})
	go func() {
		p.Log.Infof("Starting periodic auto compacting every %v", period)
		for {
			select {
			case <-time.After(period):
				p.Log.Debugf("Executing auto compact")
				if toRev, err := p.connection.Compact(); err != nil {
					p.Log.Errorf("Periodic auto compacting failed: %v", err)
				} else {
					p.Log.Infof("Auto compacting finished (to revision %v)", toRev)
				}
			case <-p.autoCompactDone:
				return
			}
		}
	}()
}

// FromExistingConnection is used mainly for testing of existing connection
// injection into the plugin.
// Note, need to set Deps for returned value!
func FromExistingConnection(connection *BytesConnectionEtcd, sl servicelabel.ReaderAPI) *Plugin {
	skel := plugin.NewSkeleton("testing", sl, connection)
	return &Plugin{Skeleton: skel, connection: connection}
}

// Close shutdowns the connection.
func (p *Plugin) Close() error {
	_, err := safeclose.CloseAll(p.Skeleton, p.autoCompactDone)
	return err
}

// String returns the plugin name from dependencies if injected,
// "kvdbsync" otherwise.
func (p *Plugin) String() string {
	if len(p.Deps.PluginName) == 0 {
		return "kvdbsync"
	}
	return string(p.Deps.PluginName)
}

// Disabled returns *true* if the plugin is not in use due to missing
// etcd configuration.
func (p *Plugin) Disabled() (disabled bool) {
	return p.disabled
}

// PutIfNotExists puts given key-value pair into etcd if there is no value set for the key. If the put was successful
// succeeded is true. If the key already exists succeeded is false and the value for the key is untouched.
func (p *Plugin) PutIfNotExists(key string, value []byte) (succeeded bool, err error) {
	if p.connection != nil {
		return p.connection.PutIfNotExists(key, value)
	}
	return false, fmt.Errorf("connection is not established")
}

// Compact compatcs the ETCD database to the specific revision
func (p *Plugin) Compact(rev ...int64) (toRev int64, err error) {
	if p.connection != nil {
		return p.connection.Compact(rev...)
	}
	return 0, fmt.Errorf("connection is not established")
}
