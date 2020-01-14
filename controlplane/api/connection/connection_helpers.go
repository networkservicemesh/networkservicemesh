// Copyright 2018-2019 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0
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

package connection

import (
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connectioncontext"
)

// IsRemote returns if connection is remote
func (c *Connection) IsRemote() bool {
	if c == nil {
		return false
	}
	// If we have two or more, it is remote
	return len(c.GetPath().GetPathSegments()) > 1
}

// GetSourceNetworkServiceManagerName - return source network service manager name
func (c *Connection) GetSourceNetworkServiceManagerName() string {
	if c == nil {
		return ""
	}
	if len(c.GetPath().GetPathSegments()) > 0 {
		return c.GetPath().GetPathSegments()[0].GetName()
	}
	return ""
}

// GetDestinationNetworkServiceManagerName - return destination network service manager name
func (c *Connection) GetDestinationNetworkServiceManagerName() string {
	if c == nil {
		return ""
	}
	if len(c.GetPath().GetPathSegments()) >= 2 {
		return c.GetPath().GetPathSegments()[1].GetName()
	}
	return ""
}

// Equals returns if connection equals given connection
func (c *Connection) Equals(connection *Connection) bool {
	return proto.Equal(c, connection)
}

// Clone clones connection
func (c *Connection) Clone() *Connection {
	return proto.Clone(c).(*Connection)
}

// UpdateContext checks and tries to set connection context
func (c *Connection) UpdateContext(context *connectioncontext.ConnectionContext) error {
	if err := context.MeetsRequirements(c.Context); err != nil {
		return err
	}

	oldContext := c.Context
	c.Context = context

	if err := c.IsValid(); err != nil {
		c.Context = oldContext
		return err
	}

	return nil
}

// IsValid checks if connection is minimally valid
func (c *Connection) IsValid() error {
	if c == nil {
		return errors.New("connection cannot be nil")
	}

	if c.GetNetworkService() == "" {
		return errors.Errorf("connection.NetworkService cannot be empty: %v", c)
	}

	if c.GetMechanism() != nil {
		if err := c.GetMechanism().IsValid(); err != nil {
			return errors.Wrapf(err, "invalid Mechanism in %v", c)
		}
	}

	if err := c.GetPath().IsValid(); err != nil {
		return err
	}

	return nil
}

// IsComplete checks if connection is complete valid
func (c *Connection) IsComplete() error {
	if err := c.IsValid(); err != nil {
		return err
	}

	if c.GetId() == "" {
		return errors.Errorf("connection.Id cannot be empty: %v", c)
	}

	if err := c.GetContext().IsValid(); err != nil {
		return err
	}

	return nil
}

func (c *Connection) MatchesMonitorScopeSelector(selector *MonitorScopeSelector) bool {
	if c == nil {
		return false
	}
	// Note: Empty selector always matches a non-nil Connection
	if len(selector.GetPathSegments()) == 0 {
		return true
	}
	// Iterate through the Connection.NetworkServiceManagers array looking for a subarray that matches
	// the selector.NetworkServiceManagers array, treating "" in the selector.NetworkServiceManagers array
	// as a wildcard
	for i := range c.GetPath().GetPathSegments() {
		// If there aren't enough elements left in the Connection.NetworkServiceManagers array to match
		// all of the elements in the select.NetworkServiceManager array...clearly we can't match
		if i+len(selector.GetPathSegments()) > len(c.GetPath().GetPathSegments()) {
			return false
		}
		// Iterate through the selector.NetworkServiceManagers array to see is the subarray starting at
		// Connection.NetworkServiceManagers[i] matches selector.NetworkServiceManagers
		for j := range selector.GetPathSegments() {
			// "" matches as a wildcard... failure to match either as wildcard or exact match means the subarray
			// starting at Connection.NetworkServiceManagers[i] doesn't match selectors.NetworkServiceManagers
			if selector.GetPathSegments()[j].GetName() != "" && c.GetPath().GetPathSegments()[i+j].GetName() != selector.GetPathSegments()[j].GetName() {
				break
			}
			// If this is the last element in the selector.NetworkServiceManagers array and we still are matching...
			// return true
			if j == len(selector.GetPathSegments())-1 {
				return true
			}
		}
	}
	return false
}

func FilterMapOnManagerScopeSelector(c map[string]*Connection, selector *MonitorScopeSelector) map[string]*Connection {
	rv := make(map[string]*Connection)
	for k, v := range c {
		if v != nil && v.MatchesMonitorScopeSelector(selector) {
			rv[k] = v
		}
	}
	return rv
}
