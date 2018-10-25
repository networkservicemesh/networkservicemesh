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

package nsmvpp

import (
	govppapi "git.fd.io/govpp.git/api"
	"github.com/ligato/networkservicemesh/pkg/nsm/apis/common"
)

var mechanisms = map[common.LocalMechanismType]Mechanism{
	common.LocalMechanismType_KERNEL_INTERFACE: KernelInterface{},
	common.LocalMechanismType_MEM_INTERFACE:    MemifInterface{},
	common.LocalMechanismType_VHOST_INTERFACE:  UnimplementedMechanism{},
	common.LocalMechanismType_SRIOV_INTERFACE:  UnimplementedMechanism{},
	common.LocalMechanismType_HW_INTERFACE:     UnimplementedMechanism{},
}

type CrossConnectionType interface {
	Connect(apiCh govppapi.Channel, src, dst map[string]string) (string, error)
	Disconnect(apiCh govppapi.Channel, src, dst map[string]string) error
	Validate(src, dst map[string]string) error
}

type SameTypeConnection struct {
	mechanism     Mechanism
	srcParameters map[string]string
	dstParameters map[string]string
}

func CreateSameTypeConnection(mechanismType common.LocalMechanismType) SameTypeConnection {
	return SameTypeConnection{
		mechanism: mechanisms[mechanismType],
	}
}

func (c SameTypeConnection) Connect(apiCh govppapi.Channel, src, dst map[string]string) (string, error) {
	return c.mechanism.CreateLocalConnect(apiCh, src, dst)
}

func (c SameTypeConnection) Validate(src, dst map[string]string) error {
	if err := c.mechanism.ValidateParameters(src); err != nil {
		return err
	}

	if err := c.mechanism.ValidateParameters(dst); err != nil {
		return err
	}

	return nil
}

func (c SameTypeConnection) Disconnect(apiCh govppapi.Channel, src, dst map[string]string) error {
	return nil
}

func CreateDifferentTypeConnection(srcMechanism, dstMechanism common.LocalMechanismType) DifferentTypeConnection {
	return DifferentTypeConnection{
		srcMechanism: mechanisms[srcMechanism],
		dstMechanism: mechanisms[dstMechanism],
	}
}

type DifferentTypeConnection struct {
	srcMechanism Mechanism
	dstMechanism Mechanism
}

func (c DifferentTypeConnection) Connect(apiCh govppapi.Channel, src, dst map[string]string) (string, error) {
	srcSwIfIndex, err := c.srcMechanism.CreateVppInterface(src)
	if err != nil {
		return "", err
	}

	dstSwIfIndex, err := c.dstMechanism.CreateVppInterface(dst)
	if err != nil {
		return "", err
	}

	if err := crossConnect(apiCh, srcSwIfIndex, dstSwIfIndex); err != nil {
		return "", err
	}

	return "", nil //todo generate connectionID
}

func (c DifferentTypeConnection) Validate(src, dst map[string]string) error {
	if err := c.srcMechanism.ValidateParameters(src); err != nil {
		return err
	}

	if err := c.dstMechanism.ValidateParameters(dst); err != nil {
		return err
	}

	return nil
}

func (c DifferentTypeConnection) Disconnect(apiCh govppapi.Channel, src, dst map[string]string) error {
	return nil
}

func crossConnect(apiCh govppapi.Channel, srcSwIfIndex, dstSwIfIndex uint32) error {
	return nil
}
