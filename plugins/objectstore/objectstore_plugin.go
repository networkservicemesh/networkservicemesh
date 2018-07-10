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
	"github.com/ligato/networkservicemesh/pkg/apis/networkservicemesh.io/v1"
	"github.com/ligato/networkservicemesh/pkg/nsm/apis/netmesh"
	"github.com/ligato/networkservicemesh/plugins/logger"
	"github.com/ligato/networkservicemesh/utils/idempotent"
)

// meta is used as a key for each NetworkService object
type meta struct {
	name      string
	namespace string
}

// ObjectStore stores information about all objects learned by CRDs controller
type objectStore struct {
	*networkServicesStore
	*networkServiceChannelsStore
	*networkServiceEndpointsStore
}

func newObjectStore() *objectStore {
	objectStore := &objectStore{}
	objectStore.networkServicesStore = newNetworkServicesStore()
	objectStore.networkServiceChannelsStore = newNetworkServiceChannelsStore()
	objectStore.networkServiceEndpointsStore = newNetworkServiceEndpointsStore()
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
	Name string
	Log  logger.FieldLoggerPlugin
}

// Init initializes ObjectStore plugin
func (p *Plugin) Init() error {
	return p.IdempotentInit(p.init)
}

func (p *Plugin) init() error {
	err := p.Log.Init()
	if err != nil {
		return err
	}
	p.pluginStopCh = make(chan struct{})
	p.objects = newObjectStore()
	return nil
}

// Close is called when the plugin is being stopped
func (p *Plugin) Close() error {
	return p.IdempotentClose(p.close)
}

func (p *Plugin) close() error {
	p.Log.Info("Close")
	return p.Log.Close()
}

// ObjectCreated is called when an object is created
func (p *Plugin) ObjectCreated(obj interface{}) {
	p.Log.Infof("ObjectStore.ObjectCreated: %s", obj)

	switch obj.(type) {
	case *v1.NetworkService:
		ns := obj.(*v1.NetworkService).Spec
		p.objects.networkServicesStore.Add(&ns)
	case *v1.NetworkServiceChannel:
		nsc := obj.(*v1.NetworkServiceChannel).Spec
		p.objects.networkServiceChannelsStore.Add(&nsc)
	case *v1.NetworkServiceEndpoint:
		nse := obj.(*v1.NetworkServiceEndpoint).Spec
		p.objects.networkServiceEndpointsStore.Add(&nse)
	}
}

// ListNetworkServices lists all stored NetworkService objects
func (p *Plugin) ListNetworkServices() []*netmesh.NetworkService {
	p.Log.Info("ObjectStore.ListNetworkServices.")
	return p.objects.networkServicesStore.List()
}

// ListNetworkServiceChannels lists all stored NetworkServiceChannel objects
func (p *Plugin) ListNetworkServiceChannels() []*netmesh.NetworkServiceChannel {
	p.Log.Info("ObjectStore.ListNetworkServiceChannels")
	return p.objects.networkServiceChannelsStore.List()
}

// ListNetworkServiceEndpoints lists all stored NetworkServiceEndpoint objects
func (p *Plugin) ListNetworkServiceEndpoints() []*netmesh.NetworkServiceEndpoint {
	p.Log.Info("ObjectStore.ListNetworkServiceEndpoints")
	return p.objects.networkServiceEndpointsStore.List()
}

// ObjectDeleted is called when an object is deleted
func (p *Plugin) ObjectDeleted(obj interface{}) {
	p.Log.Infof("ObjectStore.ObjectDeleted: %s", obj)
	switch obj.(type) {
	case *v1.NetworkService:
		ns := obj.(*v1.NetworkService).Spec
		p.objects.networkServicesStore.Delete(meta{name: ns.Metadata.Name, namespace: ns.Metadata.Namespace})
	case *v1.NetworkServiceChannel:
		nsc := obj.(*v1.NetworkServiceChannel).Spec
		p.objects.networkServiceChannelsStore.Delete(meta{name: nsc.Metadata.Name, namespace: nsc.Metadata.Namespace})
	case *v1.NetworkServiceEndpoint:
		nse := obj.(*v1.NetworkServiceEndpoint).Spec
		p.objects.networkServiceEndpointsStore.Delete(meta{name: nse.Metadata.Name, namespace: nse.Metadata.Namespace})
	}
}
