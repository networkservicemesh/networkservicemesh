// Copyright (c) 2020 Cisco and/or its affiliates.
//
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

package common

import (
	"strconv"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection/mechanisms/srv6"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/serviceregistry"

	"github.com/pkg/errors"

	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection/mechanisms/vxlan"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/networkservice"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/model"
)

// MechanismSelector is for selecting and finding forwarding mechanisms. It is used in forwarder service.
type MechanismSelector interface {
	Select(*networkservice.NetworkServiceRequest, *model.Forwarder) (*connection.Mechanism, error)
	Find(*model.Forwarder, string) *connection.Mechanism
}

// RemoteMechanismSelector for remote mechanisms
type RemoteMechanismSelector struct {
	serviceRegistry serviceregistry.ServiceRegistry
}

// LocalMechanismSelector for remote mechanisms
type LocalMechanismSelector struct{}

func findMechanism(mechanisms []*connection.Mechanism, mechanismType string) *connection.Mechanism {
	for _, m := range mechanisms {
		if m.GetType() == mechanismType {
			return m
		}
	}
	return nil
}

// Select remote mechanisms of particular type from forwarder
func (selector *RemoteMechanismSelector) Select(request *networkservice.NetworkServiceRequest, fwd *model.Forwarder) (*connection.Mechanism, error) {
	var mechanism *connection.Mechanism
	var fwdMechanism *connection.Mechanism

	if preferredMechanismName := PreferredRemoteMechanism.StringValue(); len(preferredMechanismName) > 0 {
		for _, m := range request.GetRequestMechanismPreferences() {
			if m.GetType() == preferredMechanismName {
				if fwdM := selector.Find(fwd, m.GetType()); fwdM != nil {
					mechanism = m
					fwdMechanism = fwdM
					break
				}
			}
		}
	}

	if mechanism == nil {
		for _, m := range request.GetRequestMechanismPreferences() {
			fwdM := selector.Find(fwd, m.GetType())
			if fwdM != nil {
				mechanism = m
				fwdMechanism = fwdM
				break
			}
		}
	}

	if mechanism == nil || fwdMechanism == nil {
		return nil, errors.Errorf("failed to select mechanism, no matched mechanisms found")
	}

	switch mechanism.GetType() {
	case vxlan.MECHANISM:
		selector.configureVXLANParameters(mechanism.GetParameters(), fwdMechanism.GetParameters())

	case srv6.MECHANISM:
		connectionID := request.GetConnection().GetId()
		parameters := mechanism.GetParameters()
		fwdParameters := fwdMechanism.GetParameters()

		selector.configureSRv6Parameters(connectionID, parameters, fwdParameters)
	}

	logrus.Infof("NSM: Remote mechanism selected %v", mechanism)
	return mechanism, nil
}

func (selector *RemoteMechanismSelector) configureVXLANParameters(parameters, fwdParameters map[string]string) {
	parameters[vxlan.DstIP] = fwdParameters[vxlan.SrcIP]

	extSrcIP := parameters[vxlan.SrcIP]
	extDstIP := fwdParameters[vxlan.SrcIP]
	srcIP := parameters[vxlan.SrcIP]
	dstIP := fwdParameters[vxlan.SrcIP]

	if ip, ok := parameters[vxlan.SrcOriginalIP]; ok {
		srcIP = ip
	}

	if ip, ok := parameters[vxlan.DstExternalIP]; ok {
		extDstIP = ip
	}

	var vni uint32
	if extDstIP != extSrcIP {
		vni = selector.serviceRegistry.VniAllocator().Vni(extDstIP, extSrcIP)
	} else {
		vni = selector.serviceRegistry.VniAllocator().Vni(dstIP, srcIP)
	}

	parameters[vxlan.VNI] = strconv.FormatUint(uint64(vni), 10)
}

func (selector *RemoteMechanismSelector) configureSRv6Parameters(connectionID string, parameters, fwdParameters map[string]string) {
	parameters[srv6.DstHardwareAddress] = fwdParameters[srv6.SrcHardwareAddress]
	parameters[srv6.DstHostIP] = fwdParameters[srv6.SrcHostIP]
	parameters[srv6.DstHostLocalSID] = fwdParameters[srv6.SrcHostLocalSID]
	parameters[srv6.DstBSID] = selector.serviceRegistry.SIDAllocator().SID(connectionID)
	parameters[srv6.DstLocalSID] = selector.serviceRegistry.SIDAllocator().SID(connectionID)
}

// Find remote mechanisms with particular type in forwarder
func (selector *RemoteMechanismSelector) Find(fwd *model.Forwarder, mechanismType string) *connection.Mechanism {
	return findMechanism(fwd.RemoteMechanisms, mechanismType)
}

// Select local mechanisms of particular type from forwarder
func (selector *LocalMechanismSelector) Select(request *networkservice.NetworkServiceRequest, fwd *model.Forwarder) (*connection.Mechanism, error) {
	for _, m := range request.GetRequestMechanismPreferences() {
		if fwdMechanism := selector.Find(fwd, m.GetType()); fwdMechanism != nil {
			return m, nil
		}
	}
	return nil, errors.New("failed to select mechanism, no matched mechanisms found")
}

// Find local mechanisms with particular type in forwarder
func (selector *LocalMechanismSelector) Find(fwd *model.Forwarder, mechanismType string) *connection.Mechanism {
	return findMechanism(fwd.LocalMechanisms, mechanismType)
}

// NewLocalMechanismSelector creates LocalMechanismSelector
func NewLocalMechanismSelector() *LocalMechanismSelector {
	return &LocalMechanismSelector{}
}

// NewRemoteMechanismSelector creates RemoteMechanismSelector
func NewRemoteMechanismSelector(serviceReg serviceregistry.ServiceRegistry) *RemoteMechanismSelector {
	return &RemoteMechanismSelector{
		serviceRegistry: serviceReg,
	}
}
