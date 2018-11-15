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

package vppagent

import (
	"fmt"

	"github.com/ligato/networkservicemesh/dataplane/pkg/apis/dataplane"
	"github.com/ligato/vpp-agent/plugins/vpp/model/l2"
	"github.com/ligato/vpp-agent/plugins/vpp/model/rpc"
)

type DefaultConnectionConverter struct {
	*dataplane.CrossConnect
	srcMechanismConverter Converter
	dstMechanismConverter Converter
}

func NewDefaultConnectionConverter(c *dataplane.CrossConnect) Converter {
	rv := &DefaultConnectionConverter{
		CrossConnect: c,
	}
	return rv
}

func (c *DefaultConnectionConverter) Name() string {
	return "DefaultConnectionConverter"
}

func (c *DefaultConnectionConverter) Validate() error {
	if c == nil {
		return fmt.Errorf("Cannot Validate nil Converter")
	}
	if c.CrossConnect == nil {
		return fmt.Errorf("Cannot Validate nil CrossConnect")
	}
	c.srcMechanismConverter = NewMechanismConverter(c.CrossConnect, SRC)
	if c.srcMechanismConverter == nil {
		return fmt.Errorf("Unsupported Mechanism")
	}
	err := c.srcMechanismConverter.Validate()
	if err != nil {
		return err
	}
	c.dstMechanismConverter = NewMechanismConverter(c.CrossConnect, DST)
	if c.dstMechanismConverter == nil {
		return fmt.Errorf("UnsupportedMechanism %#v", c.Destination)
	}
	err = c.dstMechanismConverter.Validate()
	if err != nil {
		return err
	}
	return nil
}

func (c *DefaultConnectionConverter) FullySpecify() error {
	err := c.Validate()
	if err != nil {
		return err
	}
	err = c.srcMechanismConverter.FullySpecify()
	if err != nil {
		return err
	}
	err = c.dstMechanismConverter.FullySpecify()
	if err != nil {
		return err
	}
	return nil
}

func (c *DefaultConnectionConverter) ToDataRequest(rv *rpc.DataRequest) (*rpc.DataRequest, error) {
	if rv == nil {
		rv = &rpc.DataRequest{}
	}
	rv, err := c.srcMechanismConverter.ToDataRequest(rv)
	if err != nil {
		return nil, err
	}
	rv, err = c.dstMechanismConverter.ToDataRequest(rv)
	if err != nil {
		return nil, err
	}
	if len(rv.Interfaces) < 2 {
		return nil, fmt.Errorf("Did not create enough interfaces to cross connect, expected at least 2, got %d", len(rv.Interfaces))
	}
	ifaces := rv.Interfaces[len(rv.Interfaces)-2:]
	rv.XCons = append(rv.XCons, &l2.XConnectPairs_XConnectPair{
		ReceiveInterface:  ifaces[0].Name,
		TransmitInterface: ifaces[1].Name,
	})
	rv.XCons = append(rv.XCons, &l2.XConnectPairs_XConnectPair{
		ReceiveInterface:  ifaces[1].Name,
		TransmitInterface: ifaces[0].Name,
	})
	return rv, nil
}
