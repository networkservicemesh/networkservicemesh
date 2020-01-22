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

package tests

import (
	"testing"

	"github.com/golang/mock/gomock"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/tests/mock"

	. "github.com/onsi/gomega"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection/mechanisms/kernel"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection/mechanisms/vxlan"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/networkservice"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/common"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/model"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/vni"
)

func TestFindLocalMechanism(t *testing.T) {
	g := NewWithT(t)
	fwd := &model.Forwarder{
		LocalMechanisms: []*connection.Mechanism{
			{
				Type: kernel.MECHANISM,
			},
		},
	}

	selector := common.NewLocalMechanismSelector()
	m := selector.Find(fwd, kernel.MECHANISM)

	g.Expect(m).NotTo(BeNil())
	g.Expect(m.Type).To(Equal(kernel.MECHANISM))
}

func TestFindRemoteMechanism(t *testing.T) {
	g := NewWithT(t)
	ctrl := gomock.NewController(t)

	fwd := &model.Forwarder{
		RemoteMechanisms: []*connection.Mechanism{
			{
				Type: vxlan.MECHANISM,
			},
		},
	}

	serviceReg := mock.NewMockServiceRegistry(ctrl)
	serviceReg.EXPECT().VniAllocator().Return(vni.NewVniAllocator())
	selector := common.NewRemoteMechanismSelector(serviceReg)
	m := selector.Find(fwd, vxlan.MECHANISM)

	g.Expect(m).NotTo(BeNil())
	g.Expect(m.Type).To(Equal(vxlan.MECHANISM))
}

func TestSelectLocalMechanism(t *testing.T) {
	g := NewWithT(t)
	fwd := &model.Forwarder{
		LocalMechanisms: []*connection.Mechanism{
			{
				Type: kernel.MECHANISM,
			},
		},
	}

	req := &networkservice.NetworkServiceRequest{MechanismPreferences: []*connection.Mechanism{
		{
			Type: kernel.MECHANISM,
		},
	},
	}

	selector := common.NewLocalMechanismSelector()

	m, err := selector.Select(req, fwd)

	g.Expect(err).To(BeNil())
	g.Expect(m).NotTo(BeNil())
	g.Expect(m.Type).To(Equal(kernel.MECHANISM))
}

func TestSelectRemoteMechanism(t *testing.T) {
	g := NewWithT(t)
	ctrl := gomock.NewController(t)

	fwd := &model.Forwarder{
		RemoteMechanisms: []*connection.Mechanism{
			{
				Type:       vxlan.MECHANISM,
				Parameters: map[string]string{vxlan.SrcIP: "127.0.0.1"},
			},
		},
	}

	req := &networkservice.NetworkServiceRequest{MechanismPreferences: []*connection.Mechanism{
		{
			Type:       vxlan.MECHANISM,
			Parameters: map[string]string{vxlan.SrcIP: "127.0.0.1"},
		},
	},
	}

	serviceReg := mock.NewMockServiceRegistry(ctrl)
	serviceReg.EXPECT().VniAllocator().Return(vni.NewVniAllocator())
	selector := common.NewRemoteMechanismSelector(serviceReg)

	m, err := selector.Select(req, fwd)

	g.Expect(err).To(BeNil())
	g.Expect(m).NotTo(BeNil())
	g.Expect(m.Type).To(Equal(vxlan.MECHANISM))
}
