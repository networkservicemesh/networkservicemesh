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

// //go:generate protoc -I ./model/pod --go_out=plugins=grpc:./model/pod ./model/pod/pod.proto

package nsmserver

import (
	"fmt"
	"time"

	"github.com/ligato/networkservicemesh/plugins/logger"
	"github.com/ligato/networkservicemesh/plugins/objectstore"
	"github.com/ligato/networkservicemesh/utils/idempotent"
)

// Plugin watches K8s resources and causes all changes to be reflected in the ETCD
// data store.
type Plugin struct {
	Deps
	nsmClientEndpoints nsmClientEndpoints
	pluginStopCh       chan bool
	idempotent.Impl
}

// Deps defines dependencies of netmesh plugin.
type Deps struct {
	Name string
	Log  logger.FieldLoggerPlugin
}

// Init initializes ObjectStore plugin
func (p *Plugin) Init() error {
	return p.IdempotentInit(p.init)
}

func (p *Plugin) init() error {
	// p.Log.SetLevel(logging.DebugLevel)
	p.pluginStopCh = make(chan bool)
	err := p.Log.Init()
	if err != nil {
		return err
	}
	return p.afterInit()
}

// afterInit is called after all plugins are initialized
func (p *Plugin) afterInit() error {
	var os objectstore.Interface
	// Wait for ObjectStore to be ready
	ticker := time.NewTicker(objectstore.ObjectStoreReadyInterval)
	timeout := time.After(objectstore.ObjectStoreReadyTimeout)
	defer ticker.Stop()
	// Wait for objectstore to initialize
	ready := false
	for !ready {
		select {
		case <-timeout:
			return fmt.Errorf("timeout waiting for ObjectStore")
		case <-ticker.C:
			if os = objectstore.SharedPlugin(); os != nil {
				ticker.Stop()
				ready = true
				p.Log.Info("ObjectStore is ready, starting Consumer")
			} else {
				p.Log.Info("ObjectStore is not ready, waiting")
			}
		}
	}
	// Register and start Kubelet's device plugin
	err := NewNSMDevicePlugin(p.Deps.Log, os)
	if err != nil {
		return err
	}

	return nil
}

// Close is called when the plugin is being stopped
func (p *Plugin) Close() error {
	return p.IdempotentClose(p.close)
}

func (p *Plugin) close() error {
	p.Log.Info("Close")
	p.pluginStopCh <- true
	return p.Log.Close()
}
