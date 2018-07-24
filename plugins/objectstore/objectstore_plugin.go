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
	"reflect"

	"github.com/ligato/networkservicemesh/pkg/apis/networkservicemesh.io/v1"
	"github.com/ligato/networkservicemesh/pkg/nsm/apis/netmesh"
	"github.com/ligato/networkservicemesh/plugins/logger"
	"github.com/ligato/networkservicemesh/utils/helper/deptools"
	"github.com/ligato/networkservicemesh/utils/helper/plugintools"
	"github.com/ligato/networkservicemesh/utils/idempotent"
	"github.com/ligato/networkservicemesh/utils/registry"
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
	Log  logger.FieldLogger
}

// Init initializes ObjectStore plugin
func (p *Plugin) Init() error {
	return p.IdempotentInit(plugintools.LoggingInitFunc(p.Log, p, p.init))
}

func (p *Plugin) init() error {
	p.pluginStopCh = make(chan struct{})
	p.objects = newObjectStore()
	return nil
}

// Close is called when the plugin is being stopped
func (p *Plugin) Close() error {
	return p.IdempotentClose(plugintools.LoggingCloseFunc(p.Log, p, p.close))
}

func (p *Plugin) close() error {
	p.Log.Info("Close")
	registry.Shared().Delete(p)
	return deptools.Close(p)
}

// ObjectCreated is called when an object is created
func (p *Plugin) ObjectCreated(obj interface{}) {
	p.Log.Infof("ObjectStore.ObjectCreated: %+v", obj)

	switch obj.(type) {
	case *v1.NetworkService:
		ns := obj.(*v1.NetworkService).Spec
		p.objects.networkServicesStore.Add(&ns)
		p.Log.Infof("number of network services in Object Store %d", len(p.objects.networkServicesStore.List()))
	case *v1.NetworkServiceChannel:
		nsc := obj.(*v1.NetworkServiceChannel).Spec
		p.objects.networkServiceChannelsStore.AddChannel(&nsc)
	case *v1.NetworkServiceEndpoint:
		nse := obj.(*v1.NetworkServiceEndpoint).Spec
		p.objects.networkServiceEndpointsStore.Add(&nse)
	default:
		p.Log.Infof("found object of unknown type: %s", reflect.TypeOf(obj))
	}
}

// GetNetworkService get NetworkService object for name and namespace specified
func (p *Plugin) GetNetworkService(nsName, nsNamespace string) *netmesh.NetworkService {
	p.Log.Info("ObjectStore.GetNetworkService.")
	return p.objects.networkServicesStore.Get(nsName, nsNamespace)
}

// AddChannelToNetworkService adds a channel to Existing in the ObjectStore NetworkService object
func (p *Plugin) AddChannelToNetworkService(nsName string, nsNamespace string, ch *netmesh.NetworkServiceChannel) error {
	p.Log.Info("ObjectStore.AddChannelToNetworkService.")
	return p.objects.networkServicesStore.AddChannelToNetworkService(nsName, nsNamespace, ch)
}

// DeleteChannelFromNetworkService deletes a channel from the ObjectStore NetworkService object
func (p *Plugin) DeleteChannelFromNetworkService(nsName string, nsNamespace string, ch *netmesh.NetworkServiceChannel) error {
	p.Log.Info("ObjectStore.DeleteChannelFromNetworkService.")
	return p.objects.networkServicesStore.DeleteChannelFromNetworkService(nsName, nsNamespace, ch)
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
	case *v1.NetworkService:
		ns := obj.(*v1.NetworkService).Spec
		p.objects.networkServicesStore.Delete(meta{name: ns.Metadata.Name, namespace: ns.Metadata.Namespace})
	case *v1.NetworkServiceChannel:
		nsc := obj.(*v1.NetworkServiceChannel).Spec
		p.objects.networkServiceChannelsStore.DeleteChannel(&nsc)
	case *v1.NetworkServiceEndpoint:
		nse := obj.(*v1.NetworkServiceEndpoint).Spec
		p.objects.networkServiceEndpointsStore.Delete(meta{name: nse.Metadata.Name, namespace: nse.Metadata.Namespace})
	}
}

// GetChannelsByNSEServerProvider lists all stored NetworkServiceChannel objects for a given nse server
func (p *Plugin) GetChannelsByNSEServerProvider(nseServer, namespace string) []*netmesh.NetworkServiceChannel {
	p.Log.Info("ObjectStore.GetChannelsByNSEServerProvider for %s/%s", nseServer, namespace)
	return p.objects.networkServiceChannelsStore.GetChannelsByNSEServerProvider(nseServer, namespace)
}

// DeleteNSE delete all channels associated with given NSE
func (p *Plugin) DeleteNSE(nseServer, namespace string) {
	p.Log.Info("ObjectStore.DeleteNSE")
	p.objects.networkServiceChannelsStore.DeleteNSE(nseServer, namespace)
}

// DeleteChannel delete all channels associated with given NSE
func (p *Plugin) DeleteChannel(nch *netmesh.NetworkServiceChannel) {
	p.Log.Info("ObjectStore.DeleteChannel")
	p.objects.networkServiceChannelsStore.DeleteChannel(nch)
}

// AddChannel checks if advertised NSE already exists and then add given channel to its list,
// othewise NSE gets created and then new channel gets added.
func (p *Plugin) AddChannel(nch *netmesh.NetworkServiceChannel) {
	p.Log.Info("ObjectStore.AddChannel")
	p.objects.networkServiceChannelsStore.AddChannel(nch)
}
