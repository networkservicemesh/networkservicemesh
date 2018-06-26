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

package objectstore

import (
	"github.com/ligato/cn-infra/flavors/local"
	"github.com/ligato/cn-infra/logging"
	"github.com/ligato/networkservicemesh/netmesh/model/netmesh"
	"github.com/ligato/networkservicemesh/utils/idempotent"
)

// meta is used as a key for each NetworkService object
type meta struct {
	name      string
	namespace string
}

// ObjectStore stores information about all objects learned by CRDs controller
// TODO add NetworkServiceEndpoint and NetworkServiceChannel
type objectStore struct {
	*networkServicesStore
}

// sharedPlugin is used to provide access to ObjectStore to other plugins
var sharedPlugin *Plugin

// SharedPlugin returns a pointer to the actual plugin struct
func SharedPlugin() Interface {
	return sharedPlugin
}

func newObjectStore() *objectStore {
	objectStore := &objectStore{}
	objectStore.networkServicesStore = newNetworkServicesStore()
	// TODO add initialization of NetworkServiceEndpoint and NetworkServiceChannel
	return objectStore
}

// Plugin is the base plugin object for this CRD handler
type Plugin struct {
	objects *objectStore
	Deps
	pluginStopCh chan struct{}
	idempotent.Impl
}

// Deps defines dependencies of netmesh plugin.
type Deps struct {
	local.PluginInfraDeps
}

// Init initializes ObjectStore plugin
func (p *Plugin) Init() error {
	return p.IdempotentInit(p.init)
}

func (p *Plugin) init() error {
	p.Log.SetLevel(logging.DebugLevel)
	p.pluginStopCh = make(chan struct{})
	p.objects = newObjectStore()
	sharedPlugin = p

	return nil
}

// AfterInit is called for post init processing
func (p *Plugin) AfterInit() error {
	p.Log.Info("AfterInit")

	return nil
}

// Close is called when the plugin is being stopped
func (p *Plugin) Close() error {
	return p.IdempotentClose(p.close)
}

func (p *Plugin) close() error {
	p.Log.Info("Close")
	return nil
}

// ObjectCreated is called when an object is created
func (p *Plugin) ObjectCreated(obj interface{}) {
	p.Log.Infof("ObjectStore.ObjectCreated: %s", obj)

	switch obj.(type) {
	case netmesh.NetworkService:
		ns := obj.(netmesh.NetworkService)
		p.objects.networkServicesStore.Add(&ns)
	case netmesh.NetworkServiceEndpoint:
	case netmesh.NetworkService_NetmeshChannel:
	}
}

// ListNetworkServices lists all stored NetworkService objects
func (p *Plugin) ListNetworkServices() []*netmesh.NetworkService {
	p.Log.Info("ObjectStore.ListNetworkServices.")
	return p.objects.networkServicesStore.List()
}

// ObjectDeleted is called when an object is deleted
func (p *Plugin) ObjectDeleted(obj interface{}) {
	p.Log.Infof("ObjectStore.ObjectDeleted: %s", obj)
	switch obj.(type) {
	case netmesh.NetworkService:
		ns := obj.(netmesh.NetworkService)
		p.objects.networkServicesStore.Delete(meta{name: ns.Metadata.Name, namespace: ns.Metadata.Namespace})
	case netmesh.NetworkServiceEndpoint:
	case netmesh.NetworkService_NetmeshChannel:
	}
}
