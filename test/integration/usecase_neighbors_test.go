// Copyright (c) 2019 Cisco Systems, Inc.
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

// +build usecase suite

package nsmd_integration_tests

import (
	"strings"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/networkservicemesh/networkservicemesh/test/kubetest"
)

func TestNSCAndICMPNeighbors(t *testing.T) {
	g := NewWithT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	k8s, err := kubetest.NewK8s(g, kubetest.ReuseNSMResources)
	defer k8s.Cleanup()
	defer k8s.ProcessArtifacts(t)
	g.Expect(err).To(BeNil())

	nodes_setup, err := kubetest.SetupNodes(k8s, 1, defaultTimeout)
	g.Expect(err).To(BeNil())
	_ = kubetest.DeployNeighborNSE(k8s, nodes_setup[0].Node, "icmp-responder-nse-1", defaultTimeout)
	nsc := kubetest.DeployNSC(k8s, nodes_setup[0].Node, "nsc-1", defaultTimeout)

	pingCommand := "ping"
	pingIP := "172.16.1.2"
	arpCommand := []string{"arp", "-a"}
	if k8s.UseIPv6() {
		pingCommand = "ping6"
		pingIP = "100::2"
		arpCommand = []string{"ip", "-6", "neigh", "show"}
	}
	pingResponse, errOut, err := k8s.Exec(nsc, nsc.Spec.Containers[0].Name, pingCommand, pingIP, "-A", "-c", "5")
	g.Expect(err).To(BeNil())
	g.Expect(errOut).To(Equal(""))
	g.Expect(strings.Contains(pingResponse, "100% packet loss")).To(Equal(false))

	nsc2 := kubetest.DeployNSC(k8s, nodes_setup[0].Node, "nsc-2", defaultTimeout)
	arpResponse, errOut, err := k8s.Exec(nsc2, nsc.Spec.Containers[0].Name, arpCommand...)
	g.Expect(err).To(BeNil())
	g.Expect(errOut).To(Equal(""))
	g.Expect(strings.Contains(arpResponse, pingIP)).To(Equal(true))
}
