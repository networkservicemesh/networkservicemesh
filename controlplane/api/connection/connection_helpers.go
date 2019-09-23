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
)

// IsRemote returns if connection is remote
func (c *Connection) IsRemote() bool {
	return true
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
