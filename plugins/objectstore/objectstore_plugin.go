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
	*dataplaneStore
}

func newObjectStore() *objectStore {
	objectStore := &objectStore{}
	objectStore.networkServicesStore = newNetworkServicesStore()
	objectStore.dataplaneStore = newDataplaneStore()
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
	p.Log.Infof("ObjectStore.ObjectCreated: %s", reflect.TypeOf(obj))

	switch obj.(type) {
	case *v1.NetworkService:
		ns := obj.(*v1.NetworkService)
		p.objects.networkServicesStore.Add(ns)
		p.Log.Infof("number of network services in Object Store %d", len(p.objects.networkServicesStore.List()))
	case *Dataplane:
		dp := obj.(*Dataplane)
		p.objects.dataplaneStore.Add(dp)
		p.Log.Infof("number of dataplanes in Object Store %d", len(p.objects.dataplaneStore.List()))
	default:
		p.Log.Infof("found object of unknown type: %s", reflect.TypeOf(obj))
	}
}

// GetDataplane get Dataplane object by registration name
func (p *Plugin) GetDataplane(registeredName string) *Dataplane {
	p.Log.Info("ObjectStore.GetDataplane.")
	return p.objects.dataplaneStore.Get(registeredName)
}

// ListDataplanes lists all stored Dataplane objects
func (p *Plugin) ListDataplanes() []*Dataplane {
	p.Log.Info("ObjectStore.ListDataplane.")
	return p.objects.dataplaneStore.List()
}

// GetNetworkService get NetworkService object for name and namespace specified
func (p *Plugin) GetNetworkService(nsName string) *v1.NetworkService {
	p.Log.Info("ObjectStore.GetNetworkService.")
	return p.objects.networkServicesStore.Get(nsName)
}

// ListNetworkServices lists all stored NetworkService objects
func (p *Plugin) ListNetworkServices() []*v1.NetworkService {
	p.Log.Info("ObjectStore.ListNetworkServices.")
	return p.objects.networkServicesStore.List()
}

// ObjectDeleted is called when an object is deleted
func (p *Plugin) ObjectDeleted(obj interface{}) {
	p.Log.Infof("ObjectStore.ObjectDeleted: %s", reflect.TypeOf(obj))
	switch obj.(type) {
	case *v1.NetworkService:
		ns := obj.(*v1.NetworkService)
		p.objects.networkServicesStore.Delete(ns.ObjectMeta.Name)
	case *Dataplane:
		dp := obj.(*Dataplane)
		p.objects.dataplaneStore.Delete(dp.RegisteredName)
	}
}
