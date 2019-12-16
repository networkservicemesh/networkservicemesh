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

// +build single_cluster_suite

package nsmd_integration_tests

import (
	"testing"

	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/test/kubetest"
	"github.com/networkservicemesh/networkservicemesh/test/kubetest/pods"
)

func TestDeleteDirtyNSE(t *testing.T) {
	g := NewWithT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	logrus.Print("Running delete dirty NSE test")

	k8s, err := kubetest.NewK8s(g, kubetest.ReuseNSMResources)
	g.Expect(err).To(BeNil())
	defer k8s.Cleanup()

	nodesConf, err := kubetest.SetupNodesConfig(k8s, 1, defaultTimeout, []*pods.NSMgrPodConfig{}, k8s.GetK8sNamespace())
	g.Expect(err).To(BeNil())
	defer k8s.ProcessArtifacts(t)

	nsePod := kubetest.DeployDirtyICMP(k8s, nodesConf[0].Node, "dirty-icmp-responder-nse", defaultTimeout)

	kubetest.ExpectNSEsCountToBe(k8s, 0, 1)

	k8s.DeletePods(nsePod)

	kubetest.ExpectNSEsCountToBe(k8s, 1, 0)
}

func TestDeleteDirtyNSEWithClient(t *testing.T) {
	g := NewWithT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	logrus.Print("Running delete dirty NSE with client test")

	k8s, err := kubetest.NewK8s(g, kubetest.ReuseNSMResources)
	g.Expect(err).To(BeNil())
	defer k8s.Cleanup()

	nodesConf, err := kubetest.SetupNodesConfig(k8s, 1, defaultTimeout, []*pods.NSMgrPodConfig{}, k8s.GetK8sNamespace())
	g.Expect(err).To(BeNil())
	defer k8s.ProcessArtifacts(t)

	nsePod := kubetest.DeployDirtyICMP(k8s, nodesConf[0].Node, "dirty-icmp-responder-nse", defaultTimeout)
	kubetest.DeployNSC(k8s, nodesConf[0].Node, "nsc-1", defaultTimeout)

	kubetest.ExpectNSEsCountToBe(k8s, 0, 1)
	k8s.DeletePods(nsePod)

	kubetest.ExpectNSEsCountToBe(k8s, 1, 0)
}
