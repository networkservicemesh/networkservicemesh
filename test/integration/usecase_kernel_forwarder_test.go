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

// +build usecase

package integration

import (
	"strconv"
	"testing"

	vpp_interfaces "go.ligato.io/vpp-agent/v3/proto/ligato/vpp/interfaces"
	v1 "k8s.io/api/core/v1"

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

func TestVPPAgentForwarder_MEMIF_RemoteWireguard(t *testing.T) {
	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	// MemifLink_IP is used for wireguard
	additionEnv := v1.EnvVar{
		Name:  "MEMIF_MODE",
		Value: strconv.Itoa(int(vpp_interfaces.MemifLink_IP)),
	}
	testGenericForwarderNSCAndICMP(t, 2, pods.EnvForwardingPlaneVPP, "WIREGUARD", kubetest.VppAgentTestingPodFixture(NewWithT(t)), additionEnv)
}

func TestVPPAgentForwarder_WIREGUARD_KernelForwarder(t *testing.T) {
	g := NewWithT(t)

	k8s, err := kubetest.NewK8s(g, true)
	g.Expect(err).To(BeNil())

	const nodesCount = 2

	defer k8s.Cleanup()
	defer k8s.SaveTestArtifacts(t)

	var planes = []string{pods.EnvForwardingPlaneVPP, pods.EnvForwardingPlaneKernel}
	var config []*pods.NSMgrPodConfig

	fixture := kubetest.VppAgentTestingPodFixture(g)

	for i := 0; i < nodesCount; i++ {
		cfg := &pods.NSMgrPodConfig{
			Variables:      pods.DefaultNSMD(),
			ForwarderPlane: &planes[i],
		}
		cfg.Variables[remote.PreferredRemoteMechanism.Name()] = "WIREGUARD"
		cfg.Namespace = k8s.GetK8sNamespace()
		cfg.ForwarderVariables = kubetest.DefaultForwarderVariables(planes[i])
		config = append(config, cfg)
	}
	nodesSetup, err := kubetest.SetupNodesConfig(k8s, nodesCount, defaultTimeout, config, k8s.GetK8sNamespace())
	g.Expect(err).To(BeNil())

	// Run ICMP on latest node
	_ = kubetest.DeployICMP(k8s, nodesSetup[nodesCount-1].Node, "icmp-responder-nse-1", defaultTimeout)

	// MemifLink_IP is used for wireguard
	additionEnv := v1.EnvVar{
		Name:  "MEMIF_MODE",
		Value: strconv.Itoa(int(vpp_interfaces.MemifLink_IP)),
	}

	//Check mechanism parameters selection on two clients
	nscPodNode1 := fixture.DeployNsc(k8s, nodesSetup[0].Node, "nsc-1", defaultTimeout, additionEnv)
	nscPodNode2 := fixture.DeployNsc(k8s, nodesSetup[0].Node, "nsc-2", defaultTimeout, additionEnv)

	fixture.CheckNsc(k8s, nscPodNode1)
	fixture.CheckNsc(k8s, nscPodNode2)
}

func testGenericForwarderNSCAndICMP(t *testing.T, nodesCount int, forwarderPlane, remoteMechanism string, fixture kubetest.TestingPodFixture, envs ...v1.EnvVar) {
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
	_ = fixture.DeployNse(k8s, nodesSetup[nodesCount-1].Node, "icmp-responder-nse-1", defaultTimeout, envs...)

	//Check mechanism parameters selection on two clients
	nscPodNode1 := fixture.DeployNsc(k8s, nodesSetup[0].Node, "nsc-1", defaultTimeout, envs...)
	nscPodNode2 := fixture.DeployNsc(k8s, nodesSetup[0].Node, "nsc-2", defaultTimeout, envs...)

	fixture.CheckNsc(k8s, nscPodNode1)
	fixture.CheckNsc(k8s, nscPodNode2)
}
