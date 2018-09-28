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
	"sync"

	"google.golang.org/grpc"

	"github.com/ligato/networkservicemesh/pkg/nsm/apis/common"
	"github.com/ligato/networkservicemesh/pkg/nsm/apis/dataplaneinterface"
)

// dataplaneStore map stores all registered dataplane providers
// with a key of its registered name.
type dataplaneStore struct {
	dataplanes map[string]*Dataplane
	sync.RWMutex
}

// Dataplane defines an object describing the dataplane module and its capabilities/parameters and
// operational constraints.
type Dataplane struct {
	RegisteredName  string
	SocketLocation  string
	RemoteMechanism []*common.RemoteMechanism
	// List of supported interface types by a dataplane module.
	SupportedInterface []*common.Interface
	// Mutex is required here to protect Parameters while they are being updated
	// by dataplaneMonitoring routine.
	sync.RWMutex
	// Conn is grpc connection to Dataplane module, it is instantiated by dataplaneMonitor function
	Conn *grpc.ClientConn
	// DataplaneInterface is a pointer to all available Dataplane client related operations, defined
	// in dataplaneinterface.proto API.
	DataplaneClient dataplaneinterface.DataplaneOperationsClient
}

// newDataplaneStore instantiates a new instance of a dataplane store. The store will be populated
// with incoming dataplane registration requests.
func newDataplaneStore() *dataplaneStore {
	return &dataplaneStore{
		dataplanes: map[string]*Dataplane{}}
}

// Add method adds registered dataplane if it does not
// already exit in the store.
func (d *dataplaneStore) Add(dp *Dataplane) {
	d.Lock()
	defer d.Unlock()

	if _, ok := d.dataplanes[dp.RegisteredName]; !ok {
		// Not in the store, adding it.
		d.dataplanes[dp.RegisteredName] = dp
	}
}

// Get method returns dataplane, if it does not
// already it returns nil.
func (d *dataplaneStore) Get(registeredName string) *Dataplane {
	d.Lock()
	defer d.Unlock()

	dp, ok := d.dataplanes[registeredName]
	if !ok {
		return nil
	}
	return dp
}

// Delete method deletes dataplane from the store.
func (d *dataplaneStore) Delete(registeredName string) {
	d.Lock()
	defer d.Unlock()

	if _, ok := d.dataplanes[registeredName]; ok {
		delete(d.dataplanes, registeredName)
	}
}

// List method lists all known dataplane objects.
func (d *dataplaneStore) List() []*Dataplane {
	d.Lock()
	defer d.Unlock()
	dps := make([]*Dataplane, 0)
	for _, dp := range d.dataplanes {
		dps = append(dps, dp)
	}
	return dps
}
