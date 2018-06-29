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
	"time"

	"github.com/ligato/networkservicemesh/netmesh/model/netmesh"
)

const (
	// ObjectStoreReadyInterval defines readiness retry interval
	ObjectStoreReadyInterval = time.Second * 10
)

// Interface is the interface to a ObjectStore handler plugin
type Interface interface {
	ObjectCreated(obj interface{})
	ObjectDeleted(obj interface{})
	ListNetworkServices() []*netmesh.NetworkService
	ListNetworkServiceChannels() []*netmesh.NetworkService_NetmeshChannel
	ListNetworkServiceEndpoints() []*netmesh.NetworkServiceEndpoint
}
