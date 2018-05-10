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

package plugin

import (
	"github.com/ligato/cn-infra/db/keyval"
	"github.com/ligato/cn-infra/db/keyval/kvproto"
	"github.com/ligato/cn-infra/servicelabel"
	"github.com/ligato/cn-infra/utils/safeclose"
)

// Connection defines an access to a particular key-value data store implementation.
type Connection interface {
	keyval.CoreBrokerWatcher
}

// Skeleton of a KV plugin is a generic part of KV plugin.
type Skeleton struct {
	serviceLabel servicelabel.ReaderAPI
	name         string
	protoWrapper *kvproto.ProtoWrapper
	connection   Connection
}

// NewSkeleton creates a new instance of the Skeleton with the given connector.
// The connection is established in AfterInit phase.
func NewSkeleton(name string, serviceLabel servicelabel.ReaderAPI,
	connection Connection) *Skeleton {
	return &Skeleton{serviceLabel: serviceLabel, name: name, connection: connection}
}

// Init is called on plugin startup.
func (plugin *Skeleton) Init() (err error) {
	plugin.protoWrapper = kvproto.NewProtoWrapperWithSerializer(plugin.connection, &keyval.SerializerJSON{})
	return err
}

// AfterInit is called once all plugins have been initialized. The connection to datastore
// is established in this phase.
func (plugin *Skeleton) AfterInit() (err error) {
	return nil
}

// Close cleans up the resources.
func (plugin *Skeleton) Close() error {
	return safeclose.Close(plugin.connection)
}

// NewBroker creates new instance of prefixed broker that provides API with arguments of type proto.Message.
func (plugin *Skeleton) NewBroker(keyPrefix string) keyval.ProtoBroker {
	return plugin.protoWrapper.NewBroker(keyPrefix)
}

// NewWatcher creates new instance of prefixed broker that provides API with arguments of type proto.Message.
func (plugin *Skeleton) NewWatcher(keyPrefix string) keyval.ProtoWatcher {
	return plugin.protoWrapper.NewWatcher(keyPrefix)
}
