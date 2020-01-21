// Copyright (c) 2020 Cisco Systems, Inc.
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

package nsmd

import (
	"testing"

	"github.com/onsi/gomega"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/registry"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/model"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/services"
)

func TestNSMDCrossConnectClient_ShouldCorrectlyWork_WhenRemotePeerCanceled(t *testing.T) {
	assert := gomega.NewWithT(t)
	m := model.NewModel()
	xconMgr := services.NewClientConnectionManager(m, nil, nil)
	c := NewMonitorCrossConnectClient(m, nil, xconMgr, nil)

	c.startPeerMonitor(&model.ClientConnection{
		ConnectionID: "1",
		RemoteNsm: &registry.NetworkServiceManager{
			Name: "a",
		},
	})
	c.remotePeerLock.Lock()
	peer := c.remotePeers["a"]
	c.remotePeerLock.Unlock()
	assert.Expect(peer).ShouldNot(gomega.BeNil())
	assert.Expect(peer.IsCanceled()).Should(gomega.BeFalse())
	peer.Cancel()
	assert.Expect(peer.IsCanceled()).Should(gomega.BeTrue())
	c.startPeerMonitor(&model.ClientConnection{
		ConnectionID: "1",
		RemoteNsm: &registry.NetworkServiceManager{
			Name: "a",
		},
	})
	assert.Expect(peer.IsCanceled()).Should(gomega.BeFalse())
	peer.Cancel()
	assert.Expect(peer.IsCanceled()).Should(gomega.BeTrue())
}
