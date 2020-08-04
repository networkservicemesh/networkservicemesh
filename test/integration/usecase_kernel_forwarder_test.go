// Copyright (c) 2020 Doc.ai and/or its affiliates.
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

// +build wireguard

package integration

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/remote"
	"github.com/networkservicemesh/networkservicemesh/test/kubetest"
	"github.com/networkservicemesh/networkservicemesh/test/kubetest/pods"
)

func TestKernelForwarderLocal(t *testing.T) {
	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	testGenericForwarderNSCAndICMP(t, 2, pods.EnvForwardingPlaneKernel, "", kubetest.DefaultTestingPodFixture(NewWithT(t)))
}

func TestKernelForwarder_RemoteVXLAN(t *testing.T) {
	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}
	testGenericForwarderNSCAndICMP(t, 2, pods.EnvForwardingPlaneKernel, "VXLAN", kubetest.DefaultTestingPodFixture(NewWithT(t)))
}

func TestKernelForwarder_RemoteWireguard(t *testing.T) {
	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}
	testGenericForwarderNSCAndICMP(t, 2, pods.EnvForwardingPlaneKernel, "WIREGUARD", kubetest.DefaultTestingPodFixture(NewWithT(t)))
}

func TestVPPAgentForwarder_TAP_RemoteWireguard(t *testing.T) {
	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}
	testGenericForwarderNSCAndICMP(t, 2, "vpp", "WIREGUARD", kubetest.DefaultTestingPodFixture(NewWithT(t)))
}

func TestVPPAgentForwarder_MEMIF_RemoteWireguard(t *testing.T) {
	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}
	testGenericForwarderNSCAndICMP(t, 2, "vpp", "WIREGUARD", kubetest.VppAgentTestingPodFixture(NewWithT(t)))
}

func testGenericForwarderNSCAndICMP(t *testing.T, nodesCount int, forwarderPlane, remoteMechanism string, fixture kubetest.TestingPodFixture) {
	g := NewWithT(t)

	k8s, err := kubetest.NewK8s(g, true)
	g.Expect(err).To(BeNil())

	defer k8s.Cleanup()
	defer k8s.SaveTestArtifacts(t)

	err = k8s.SetForwardingPlane(forwarderPlane)
	g.Expect(err).To(BeNil())

	var config []*pods.NSMgrPodConfig
	for i := 0; i < nodesCount; i++ {
		cfg := &pods.NSMgrPodConfig{
			Variables: pods.DefaultNSMD(),
		}
		cfg.Variables[remote.PreferredRemoteMechanism.Name()] = remoteMechanism
		cfg.Namespace = k8s.GetK8sNamespace()
		cfg.ForwarderVariables = kubetest.DefaultForwarderVariables(forwarderPlane)
		config = append(config, cfg)
	}
	nodesSetup, err := kubetest.SetupNodesConfig(k8s, nodesCount, defaultTimeout, config, k8s.GetK8sNamespace())
	g.Expect(err).To(BeNil())

	// Run ICMP on latest node
	_ = fixture.DeployNse(k8s, nodesSetup[nodesCount-1].Node, "icmp-responder-nse-1", defaultTimeout)

	// Check mechanism parameters selection on two clients
	nscPodNode1 := fixture.DeployNsc(k8s, nodesSetup[0].Node, "nsc-1", defaultTimeout)
	nscPodNode2 := fixture.DeployNsc(k8s, nodesSetup[0].Node, "nsc-2", defaultTimeout)

	fixture.CheckNsc(k8s, nscPodNode1)
	fixture.CheckNsc(k8s, nscPodNode2)
}
