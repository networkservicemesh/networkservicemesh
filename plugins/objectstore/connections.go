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

	"github.com/ligato/networkservicemesh/pkg/nsm/apis/dataplane"
)

type connectionStore struct {
	connections map[string]*dataplane.Connection
	sync.RWMutex
}

func newConnectionStore() *connectionStore {
	return &connectionStore{
		connections: map[string]*dataplane.Connection{},
	}
}

// Add connection to connetionStore
func (c *connectionStore) Add(connection *dataplane.Connection) {
	c.Lock()
	defer c.Unlock()
	if _, ok := c.connections[connection.ConnectionId]; !ok {
		c.connections[connection.ConnectionId] = connection
	}
}

// Get connection from connectionStore by connectionID
func (c *connectionStore) Get(connectionID string) *dataplane.Connection {
	c.Lock()
	defer c.Unlock()
	return c.connections[connectionID] // will be nil of we don't have one already
}

// Delete connection with connectionID from connectionStore
func (c *connectionStore) Delete(connectionID string) {
	c.Lock()
	defer c.Unlock()
	delete(c.connections, connectionID)
}

// List all connections in connectionStore
func (c *connectionStore) List() []*dataplane.Connection {
	c.Lock()
	defer c.Unlock()
	rv := make([]*dataplane.Connection, len(c.connections))
	i := 0
	for _, value := range c.connections {
		rv[i] = value
		i++
	}
	return rv
}
