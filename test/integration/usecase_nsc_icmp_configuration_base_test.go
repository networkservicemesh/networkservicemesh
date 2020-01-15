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

// +build srv6 single_cluster_suite

package nsmd_integration_tests

import (
	"testing"

	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/remote"
	"github.com/networkservicemesh/networkservicemesh/test/kubetest"
	"github.com/networkservicemesh/networkservicemesh/test/kubetest/pods"
)

/**
If passed 1 both will be on same node, if not on different.
*/
func testNSCAndICMP(t *testing.T, nodesCount int, useWebhook, disableVHost bool, remoteMechanism string) {
	g := NewWithT(t)

	k8s, err := kubetest.NewK8s(g, kubetest.ReuseNSMResources)
	defer k8s.Cleanup()
	defer k8s.ProcessArtifacts(t)
	g.Expect(err).To(BeNil())

	if useWebhook {
		awc, awDeployment, awService := kubetest.DeployAdmissionWebhook(k8s, "nsm-admission-webhook", "networkservicemesh/admission-webhook", k8s.GetK8sNamespace(), defaultTimeout)
		defer kubetest.DeleteAdmissionWebhook(k8s, "nsm-admission-webhook-certs", awc, awDeployment, awService, k8s.GetK8sNamespace())
	}

	config := []*pods.NSMgrPodConfig{}
	for i := 0; i < nodesCount; i++ {
		cfg := &pods.NSMgrPodConfig{
			Variables: pods.DefaultNSMD(),
		}
		cfg.Variables[remote.PreferredRemoteMechanism.Name()] = remoteMechanism
		cfg.Namespace = k8s.GetK8sNamespace()
		cfg.ForwarderVariables = kubetest.DefaultForwarderVariables(k8s.GetForwardingPlane())
		if disableVHost {
			cfg.ForwarderVariables["FORWARDER_ALLOW_VHOST"] = "false"
		}
		config = append(config, cfg)
	}
	nodes_setup, err := kubetest.SetupNodesConfig(k8s, nodesCount, defaultTimeout, config, k8s.GetK8sNamespace())
	g.Expect(err).To(BeNil())

	// Run ICMP on latest node
	_ = kubetest.DeployICMP(k8s, nodes_setup[nodesCount-1].Node, "icmp-responder-nse-1", defaultTimeout)

	var nscPodNode *v1.Pod
	if useWebhook {
		nscPodNode = kubetest.DeployNSCWebhook(k8s, nodes_setup[0].Node, "nsc-1", defaultTimeout)
	} else {
		nscPodNode = kubetest.DeployNSC(k8s, nodes_setup[0].Node, "nsc-1", defaultTimeout)
	}

	kubetest.CheckNSC(k8s, nscPodNode)
}
