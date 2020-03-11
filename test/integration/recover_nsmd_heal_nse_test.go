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

package integration

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/nsmd"
	"github.com/networkservicemesh/networkservicemesh/test/kubetest"
	"github.com/networkservicemesh/networkservicemesh/test/kubetest/pods"
)

func TestNSMDRecoverNSE(t *testing.T) {
	g := NewWithT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	k8s, err := kubetest.NewK8s(g, kubetest.DefaultClear)
	defer k8s.Cleanup(t)

	g.Expect(err).To(BeNil())

	nodes, err := kubetest.SetupNodesConfig(k8s, 1, defaultTimeout, []*pods.NSMgrPodConfig{
		{
			Variables: map[string]string{
				nsmd.NsmdDeleteLocalRegistry: "true",
			},
			Namespace:          k8s.GetK8sNamespace(),
			ForwarderVariables: kubetest.DefaultForwarderVariables(k8s.GetForwardingPlane()),
		},
	}, k8s.GetK8sNamespace())
	g.Expect(err).To(BeNil())
	icmpPod := kubetest.DeployICMP(k8s, nodes[0].Node, "icmp-responder-nse-1", defaultTimeout)

	nsmdName := nodes[0].Nsmd.Name

	k8s.DeletePods(nodes[0].Nsmd, icmpPod)

	nodes[0].Nsmd = k8s.CreatePod(pods.NSMgrPodWithConfig(nsmdName, nodes[0].Node, &pods.NSMgrPodConfig{
		Namespace: k8s.GetK8sNamespace(),
	})) // Recovery NSEs
	// Wait for NSMgr to be deployed, to not get admission error
	kubetest.WaitNSMgrDeployed(k8s, nodes[0].Nsmd, defaultTimeout)
	icmpPod = kubetest.DeployICMP(k8s, nodes[0].Node, "icmp-responder-nse-2", defaultTimeout)
	g.Expect(icmpPod).ToNot(BeNil())
}
