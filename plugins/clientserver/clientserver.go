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

package clientserver

import (
	"fmt"
	"time"

	"github.com/ligato/cn-infra/flavors/local"
	"github.com/ligato/cn-infra/logging"
	"github.com/ligato/networkservicemesh/netmesh/model/netmesh"
	"github.com/ligato/networkservicemesh/plugins/objectstore"
)

// Plugin is the base plugin object for this CRD handler
type Plugin struct {
	Deps
	pluginStopCh chan struct{}
}

// Deps defines dependencies of netmesh plugin.
type Deps struct {
	local.PluginInfraDeps
}

// Init initializes ObjectStore
func (p *Plugin) Init() error {
	p.Log.SetLevel(logging.DebugLevel)
	p.pluginStopCh = make(chan struct{})

	return nil
}

// AfterInit is called for post init processing
func (p *Plugin) AfterInit() error {
	p.Log.Info("AfterInit")
	var err error
	var objectStore objectstore.Interface

	ticker := time.NewTicker(objectstore.ObjectStoreReadyInterval)
	defer ticker.Stop()
	// Loop to wait for the initialization of ObjectStore
	ready := false
	for !ready {
		select {
		case <-ticker.C:
			if objectStore = objectstore.SharedPlugin(); objectStore != nil {
				ready = true
				ticker.Stop()
			}
		}
	}

	go func() {
		ObjectStoreCommunicator(p, objectStore)
		// p.Log.Errorf("ClientServer.AfterInit failed to start ObjectStoreCommunicator with error: %+v", err)
	}()
	return err
}

// Close is called when the plugin is being stopped
func (p *Plugin) Close() error {
	p.Log.Info("Close")

	return nil
}

// ObjectStoreCommunicator is used to communicate with ObjectStore
func ObjectStoreCommunicator(p *Plugin, objectStore objectstore.Interface) {
	ns := netmesh.NetworkService{
		Metadata: &netmesh.Metadata{},
	}
	for i := 0; i < 5; i++ {
		ns = netmesh.NetworkService{
			Metadata: &netmesh.Metadata{
				Name:      "Network" + fmt.Sprintf("-%d", i),
				Namespace: "default",
			},
		}
		objectStore.ObjectCreated(ns)
	}
	for {
		for _, n := range objectStore.ListNetworkServices() {
			p.Log.Infof("%+v", n)
		}
		time.Sleep(1 * time.Minute)
	}
}
