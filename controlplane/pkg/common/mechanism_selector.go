// Copyright (c) 2019 Cisco and/or its affiliates.
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

	"github.com/pkg/errors"

	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection/mechanisms/vxlan"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/networkservice"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/model"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/vni"
)

type MechanismSelector interface {
	Select(*networkservice.NetworkServiceRequest, *model.Forwarder) (*connection.Mechanism, error)
	Find(*model.Forwarder, string) *connection.Mechanism
}

type RemoteMechanismSelector struct {
	vniAllocator vni.VniAllocator
}

type LocalMechanismSelector struct{}

func findMechanism(mechanismPreferences []*connection.Mechanism, mechanismType string) *connection.Mechanism {
	for _, m := range mechanismPreferences {
		if m.GetType() == mechanismType {
			return m
		}
	}
	return nil
}

func (selector *RemoteMechanismSelector) Select(request *networkservice.NetworkServiceRequest, dp *model.Forwarder) (*connection.Mechanism, error) {
	for _, mechanism := range request.GetRequestMechanismPreferences() {
		dpMechanism := selector.Find(dp, vxlan.MECHANISM)
		if dpMechanism == nil {
			continue
		}
		// TODO: Add other mechanisms support

		if mechanism.GetType() == vxlan.MECHANISM {
			parameters := mechanism.GetParameters()
			dpParameters := dpMechanism.GetParameters()

			parameters[vxlan.DstIP] = dpParameters[vxlan.SrcIP]
			var vni uint32

			extSrcIP := parameters[vxlan.SrcIP]
			extDstIP := dpParameters[vxlan.SrcIP]
			srcIP := parameters[vxlan.SrcIP]
			dstIP := dpParameters[vxlan.SrcIP]

			if ip, ok := parameters[vxlan.SrcOriginalIP]; ok {
				srcIP = ip
			}

			if ip, ok := parameters[vxlan.DstExternalIP]; ok {
				extDstIP = ip
			}

			if extDstIP != extSrcIP {
				vni = selector.vniAllocator.Vni(extDstIP, extSrcIP)
			} else {
				vni = selector.vniAllocator.Vni(dstIP, srcIP)
			}

			parameters[vxlan.VNI] = strconv.FormatUint(uint64(vni), 10)
		}

		logrus.Infof("NSM:(5.1) Remote mechanism selected %v", mechanism)
		return mechanism, nil
	}

	return nil, errors.New("failed to select mechanism, no matched mechanisms found")
}

func (selector *RemoteMechanismSelector) Find(fwd *model.Forwarder, mechanismType string) *connection.Mechanism {
	return findMechanism(fwd.RemoteMechanisms, mechanismType)
}

func (selector *LocalMechanismSelector) Select(request *networkservice.NetworkServiceRequest, dp *model.Forwarder) (*connection.Mechanism, error) {
	for _, m := range request.GetRequestMechanismPreferences() {
		if dpMechanism := selector.Find(dp, m.GetType()); dpMechanism != nil {
			return m, nil
		}
	}
	return nil, errors.New("failed to select mechanism, no matched mechanisms found")
}

func (selector *LocalMechanismSelector) Find(fwd *model.Forwarder, mechanismType string) *connection.Mechanism {
	return findMechanism(fwd.LocalMechanisms, mechanismType)
}

func NewLocalMechanismSelector() *LocalMechanismSelector {
	return &LocalMechanismSelector{}
}

func NewRemoteMechanismSelector(allocator vni.VniAllocator) *RemoteMechanismSelector {
	return &RemoteMechanismSelector{
		vniAllocator: allocator,
	}
}
