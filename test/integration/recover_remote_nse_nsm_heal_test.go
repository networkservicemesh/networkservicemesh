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

// +build recover srv6

package integration

import (
	"fmt"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/nsmd"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/properties"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/remote"
	"github.com/networkservicemesh/networkservicemesh/test/kubetest"
	"github.com/networkservicemesh/networkservicemesh/test/kubetest/pods"
)

func testNSMHealRemoteDieNSMD_NSE(t *testing.T, remoteMechanism string) {
	g := NewWithT(t)

	k8s, err := kubetest.NewK8s(g, kubetest.DefaultClear)
	defer k8s.Cleanup(t)

	g.Expect(err).To(BeNil())
	defer k8s.SaveTestArtifacts(t)
	// Deploy open tracing to see what happening.
	config := []*pods.NSMgrPodConfig{
		{
			Variables: map[string]string{
				properties.NsmdHealDSTWaitTimeout: "20", // 20 second delay, since we know both NSM and NSE will die and we need to go with different code branch.
				nsmd.NsmdDeleteLocalRegistry:      "true",
			},
			Namespace:          k8s.GetK8sNamespace(),
			ForwarderVariables: kubetest.DefaultForwarderVariables(k8s.GetForwardingPlane()),
		},
		{
			Namespace:          k8s.GetK8sNamespace(),
			Variables:          pods.DefaultNSMD(),
			ForwarderVariables: kubetest.DefaultForwarderVariables(k8s.GetForwardingPlane()),
		},
	}
	for i := 0; i < 2; i++ {
		config[i].Variables[remote.PreferredRemoteMechanism.Name()] = remoteMechanism
	}
	nodes_setup, err := kubetest.SetupNodesConfig(k8s, 2, defaultTimeout, config, k8s.GetK8sNamespace())
	g.Expect(err).To(BeNil())

	// Run ICMP on latest node
	icmpPod := kubetest.DeployICMPWithConfig(k8s, nodes_setup[1].Node, "icmp-responder-nse-1", defaultTimeout, 30)

	nscPodNode := kubetest.DeployNSC(k8s, nodes_setup[0].Node, "nsc-1", defaultTimeout)
	kubetest.CheckNSC(k8s, nscPodNode)

	logrus.Infof("Delete Remote NSMD/ICMP responder NSE")
	k8s.DeletePods(nodes_setup[1].Nsmd)
	k8s.DeletePods(icmpPod)
	//k8s.DeletePods(nodes_setup[1].Nsmd, icmpPod)
	logrus.Infof("Waiting for NSE with network service")
	k8s.WaitLogsContains(nodes_setup[0].Nsmd, "nsmd", "Waiting for NSE with network service icmp-responder", time.Minute)
	// Now are are in forwarder dead state, and in Heal procedure waiting for forwarder.
	nsmdName := fmt.Sprintf("nsmd-worker-recovered-%d", 1)

	logrus.Infof("Starting recovered NSMD...")
	startTime := time.Now()
	nodes_setup[1].Nsmd = k8s.CreatePod(pods.NSMgrPodWithConfig(nsmdName, nodes_setup[1].Node, &pods.NSMgrPodConfig{Namespace: k8s.GetK8sNamespace()})) // Recovery NSEs
	// Wait for NSMgr to be deployed, to not get admission error
	kubetest.WaitNSMgrDeployed(k8s, nodes_setup[1].Nsmd, defaultTimeout)
	logrus.Printf("Started new NSMD: %v on node %s", time.Since(startTime), nodes_setup[1].Node.Name)

	// Restore ICMP responder pod.
	icmpPod = kubetest.DeployICMP(k8s, nodes_setup[1].Node, "icmp-responder-nse-2", defaultTimeout)

	logrus.Infof("Waiting for connection recovery...")
	k8s.WaitLogsContains(nodes_setup[0].Nsmd, "nsmd", "Heal: Connection recovered:", defaultTimeout)
	logrus.Infof("Waiting for connection recovery Done...")

	kubetest.CheckNSC(k8s, nscPodNode)
}

func testNSMHealRemoteDieNSMD(t *testing.T, remoteMechanism string) {
	g := NewWithT(t)

	k8s, err := kubetest.NewK8s(g, kubetest.DefaultClear)
	defer k8s.Cleanup(t)

	g.Expect(err).To(BeNil())

	// Deploy open tracing to see what happening.
	config := []*pods.NSMgrPodConfig{}
	for i := 0; i < 2; i++ {
		cfg := &pods.NSMgrPodConfig{
			Namespace:          k8s.GetK8sNamespace(),
			Variables:          pods.DefaultNSMD(),
			ForwarderVariables: kubetest.DefaultForwarderVariables(k8s.GetForwardingPlane()),
		}
		cfg.Variables[remote.PreferredRemoteMechanism.Name()] = remoteMechanism
		config = append(config, cfg)
	}
	nodes_setup, err := kubetest.SetupNodesConfig(k8s, 2, defaultTimeout, config, k8s.GetK8sNamespace())
	g.Expect(err).To(BeNil())

	// Run ICMP on latest node
	icmpPod := kubetest.DeployICMP(k8s, nodes_setup[1].Node, "icmp-responder-nse-1", defaultTimeout)
	g.Expect(icmpPod).ToNot(BeNil())

	nscPodNode := kubetest.DeployNSC(k8s, nodes_setup[0].Node, "nsc-1", defaultTimeout)
	kubetest.CheckNSC(k8s, nscPodNode)

	logrus.Infof("Delete Remote NSMD")
	k8s.DeletePods(nodes_setup[1].Nsmd)

	logrus.Infof("Waiting for NSE with network service")
	k8s.WaitLogsContains(nodes_setup[0].Nsmd, "nsmd", "Waiting for NSE with network service icmp-responder", defaultTimeout)
	// Now are are in forwarder dead state, and in Heal procedure waiting for forwarder.
	nsmdName := fmt.Sprintf("nsmd-worker-recovered-%d", 1)

	logrus.Infof("Starting recovered NSMD...")
	startTime := time.Now()
	nodes_setup[1].Nsmd = k8s.CreatePod(pods.NSMgrPodWithConfig(nsmdName, nodes_setup[1].Node, &pods.NSMgrPodConfig{Namespace: k8s.GetK8sNamespace()})) // Recovery NSEs
	// Wait for NSMgr to be deployed, to not get admission error
	kubetest.WaitNSMgrDeployed(k8s, nodes_setup[1].Nsmd, defaultTimeout)
	logrus.Printf("Started new NSMD: %v on node %s", time.Since(startTime), nodes_setup[1].Node.Name)

	logrus.Infof("Waiting for connection recovery...")
	k8s.WaitLogsContains(nodes_setup[0].Nsmd, "nsmd", "Heal: Connection recovered:", defaultTimeout)
	logrus.Infof("Waiting for connection recovery Done...")

	kubetest.CheckNSC(k8s, nscPodNode)
}
