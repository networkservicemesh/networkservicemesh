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
	"fmt"

	"github.com/golang/protobuf/proto"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connectioncontext"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/api/nsm/connection"
)

// IsRemote returns if connection is remote
func (c *Connection) IsRemote() bool {
	return true
}

// Equals returns if connection equals given connection
func (c *Connection) Equals(connection connection.Connection) bool {
	if other, ok := connection.(*Connection); ok {
		return proto.Equal(c, other)
	}

	return false
}

// Clone clones connection
func (c *Connection) Clone() connection.Connection {
	return proto.Clone(c).(*Connection)
}

// SetID sets connection id
func (c *Connection) SetID(id string) {
	c.Id = id
}

// SetNetworkService sets connection networkService
func (c *Connection) SetNetworkService(networkService string) {
	c.NetworkService = networkService
}

// GetConnectionMechanism returns connection mechanism
func (c *Connection) GetConnectionMechanism() connection.Mechanism {
	return c.Mechanism
}

// SetConnectionMechanism sets connection mechanism
func (c *Connection) SetConnectionMechanism(mechanism connection.Mechanism) {
	c.Mechanism = mechanism.(*Mechanism)
}

// SetContext sets connection context
func (c *Connection) SetContext(context *connectioncontext.ConnectionContext) {
	c.Context = context
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

// GetConnectionState returns connection state
func (c *Connection) GetConnectionState() connection.State {
	switch c.State {
	case State_UP:
		return connection.StateUp
	case State_DOWN:
		return connection.StateDown
	default:
		panic(fmt.Sprintf("state is out of range: %v", c.State))
	}
}

// SetConnectionState sets connection state
func (c *Connection) SetConnectionState(state connection.State) {
	switch state {
	case connection.StateUp:
		c.State = State_UP
	case connection.StateDown:
		c.State = State_DOWN
	default:
		panic(fmt.Sprintf("state is out of range: %v", state))
	}
}

// IsValid checks if connection is minimally valid
func (c *Connection) IsValid() error {
	if c == nil {
		return fmt.Errorf("connection cannot be nil")
	}

	if c.GetNetworkService() == "" {
		return fmt.Errorf("connection.NetworkService cannot be empty: %v", c)
	}

	if c.GetMechanism() != nil {
		if err := c.GetMechanism().IsValid(); err != nil {
			return fmt.Errorf("invalid Mechanism in %v: %s", c, err)
		}
	}
	return nil
}

// IsComplete checks if connection is complete valid
func (c *Connection) IsComplete() error {
	if err := c.IsValid(); err != nil {
		return err
	}

	if c.GetId() == "" {
		return fmt.Errorf("connection.Id cannot be empty: %v", c)
	}

	if err := c.GetContext().IsValid(); err != nil {
		return err
	}

	return nil
}
