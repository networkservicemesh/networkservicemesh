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

// +build interdomain

package integration

import (
	"fmt"
	"os"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/remote"
	"github.com/networkservicemesh/networkservicemesh/test/kubetest"
	"github.com/networkservicemesh/networkservicemesh/test/kubetest/pods"
)

func TestInterdomainKernelForwarderVXLAN(t *testing.T) {
	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	testInterdomainKernelForwarder(t, 2, 1, "VXLAN")
}

func TestInterdomainKernelForwarderWireguard(t *testing.T) {
	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	testInterdomainKernelForwarder(t, 2, 1, "WIREGUARD")
}

func testInterdomainKernelForwarder(t *testing.T, clustersCount int, nodesCount int, remoteMechanism string) {
	g := NewWithT(t)

	k8ss := []*kubetest.ExtK8s{}

	for i := 0; i < clustersCount; i++ {
		kubeconfig := os.Getenv(fmt.Sprintf("KUBECONFIG_CLUSTER_%d", i+1))
		g.Expect(len(kubeconfig)).ToNot(Equal(0))

		k8s, err := kubetest.NewK8sForConfig(g, true, kubeconfig)
		g.Expect(err).To(BeNil())
		defer k8s.Cleanup()
		defer k8s.SaveTestArtifacts(t)

		err = k8s.SetForwardingPlane(pods.EnvForwardingPlaneKernel)

		config := []*pods.NSMgrPodConfig{}

		cfg := &pods.NSMgrPodConfig{
			Variables: pods.DefaultNSMD(),
		}
		cfg.Variables[remote.PreferredRemoteMechanism.Name()] = remoteMechanism
		cfg.Namespace = k8s.GetK8sNamespace()
		cfg.ForwarderVariables = kubetest.DefaultForwarderVariables(k8s.GetForwardingPlane())
		config = append(config, cfg)

		nodesSetup, err := kubetest.SetupNodesConfig(k8s, nodesCount, defaultTimeout, config, k8s.GetK8sNamespace())
		g.Expect(err).To(BeNil())

		k8ss = append(k8ss, &kubetest.ExtK8s{
			K8s:        k8s,
			NodesSetup: nodesSetup,
		})

		for j := 0; j < nodesCount; j++ {
			pnsmdName := fmt.Sprintf("pnsmgr-%s", nodesSetup[j].Node.Name)
			kubetest.DeployProxyNSMgr(k8s, nodesSetup[j].Node, pnsmdName, defaultTimeout)
		}

		serviceCleanup := kubetest.RunProxyNSMgrService(k8s)
		defer serviceCleanup()
	}

	// Run ICMP on latest node
	_ = kubetest.DeployICMP(k8ss[clustersCount-1].K8s, k8ss[clustersCount-1].NodesSetup[nodesCount-1].Node, "icmp-responder-nse-1", defaultTimeout)

	nseExternalIP, err := kubetest.GetNodeExternalIP(k8ss[clustersCount-1].NodesSetup[0].Node)
	if err != nil {
		nseExternalIP, err = kubetest.GetNodeInternalIP(k8ss[clustersCount-1].NodesSetup[0].Node)
		g.Expect(err).To(BeNil())
	}

	nscPodNode := kubetest.DeployNSCWithEnv(k8ss[0].K8s, k8ss[0].NodesSetup[0].Node, "nsc-1", defaultTimeout, map[string]string{
		"CLIENT_LABELS":          "app=icmp",
		"CLIENT_NETWORK_SERVICE": fmt.Sprintf("icmp-responder@%s", nseExternalIP),
	})

	kubetest.CheckNSC(k8ss[0].K8s, nscPodNode)
}
