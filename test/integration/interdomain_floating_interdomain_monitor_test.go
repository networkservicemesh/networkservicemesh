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

// +build interdomain

package nsmd_integration_tests

import (
	"fmt"
	"os"
	"testing"

	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/nsmd"

	"github.com/networkservicemesh/networkservicemesh/k8s/pkg/proxyregistryserver"
	"github.com/networkservicemesh/networkservicemesh/test/kubetest"
	"github.com/networkservicemesh/networkservicemesh/test/kubetest/pods"
)

func TestFloatingInterdomainMonitorDieNSMRS(t *testing.T) {
	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	testFloatingInterdomainMonitor(t, "nsmrs")
}

func TestFloatingInterdomainMonitorDieProxyNSMGR(t *testing.T) {
	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	testFloatingInterdomainMonitor(t, "proxy-nsmgr")
}

func testFloatingInterdomainMonitor(t *testing.T, killPod string) {
	g := NewWithT(t)

	k8ss := []*kubetest.ExtK8s{}
	for i := 0; i < 2; i++ {
		kubeconfig := os.Getenv(fmt.Sprintf("KUBECONFIG_CLUSTER_%d", i+1))
		g.Expect(len(kubeconfig)).ToNot(Equal(0))

		k8s, err := kubetest.NewK8sForConfig(g, true, kubeconfig)
		g.Expect(err).To(BeNil())

		defer k8s.Cleanup()
		defer kubetest.MakeLogsSnapshot(k8s, t)

		k8ss = append(k8ss, &kubetest.ExtK8s{
			K8s:        k8s,
			NodesSetup: nil,
		})
	}

	nsmrsNode := &k8ss[1].K8s.GetNodesWait(2, defaultTimeout)[1]
	nsmrsPod := kubetest.DeployNSMRS(k8ss[1].K8s, nsmrsNode, "nsmrs", defaultTimeout)

	nsmrsExternalIP, err := kubetest.GetNodeExternalIP(nsmrsNode)
	if err != nil {
		nsmrsExternalIP, err = kubetest.GetNodeInternalIP(nsmrsNode)
		g.Expect(err).To(BeNil())
	}

	k8s := k8ss[0].K8s
	nodesSetup, err := kubetest.SetupNodesConfig(k8s, 1, defaultTimeout, []*pods.NSMgrPodConfig{
		{
			Variables: map[string]string{
				nsmd.NSETrackingIntervalSecondsEnv.Name(): "10s",
			},
			Namespace:          k8s.GetK8sNamespace(),
			ForwarderVariables: kubetest.DefaultForwarderVariables(k8s.GetForwardingPlane()),
		},
	}, k8s.GetK8sNamespace())
	g.Expect(err).To(BeNil())

	k8ss[0].NodesSetup = nodesSetup
	pnsmdName := fmt.Sprintf("pnsmgr-%s", nodesSetup[0].Node.Name)
	proxyNSMGRPod := startProxyNSMGRPod(g, pnsmdName, k8s, nodesSetup, nsmrsExternalIP)

	serviceCleanup := kubetest.RunProxyNSMgrService(k8s)
	defer serviceCleanup()

	kubetest.DeployICMP(k8ss[0].K8s, k8ss[0].NodesSetup[0].Node, "icmp-responder-nse-1", defaultTimeout)
	k8ss[1].K8s.WaitLogsContains(nsmrsPod, "nsmrs", "Returned from RegisterNSE", defaultTimeout)

	switch killPod {
	case "nsmrs":
		k8ss[1].K8s.DeletePods(nsmrsPod)
		nsmrsPod := kubetest.DeployNSMRS(k8ss[1].K8s, nsmrsNode, "nsmrs-recovered", defaultTimeout)
		k8ss[1].K8s.WaitLogsContains(nsmrsPod, "nsmrs", "Registered NSE entry", defaultTimeout)
	case "proxy-nsmgr":
		k8ss[0].K8s.DeletePods(proxyNSMGRPod)
		k8ss[1].K8s.DeletePods(nsmrsPod)
		nsmrsPod := kubetest.DeployNSMRS(k8ss[1].K8s, nsmrsNode, "nsmrs", defaultTimeout)
		startProxyNSMGRPod(g, pnsmdName+"-recovered", k8ss[0].K8s, k8ss[0].NodesSetup, nsmrsExternalIP)
		k8ss[1].K8s.WaitLogsContains(nsmrsPod, "nsmrs", "Registered NSE entry", defaultTimeout)
	}
}

func startProxyNSMGRPod(g *WithT, pnsmdName string, k8s *kubetest.K8s, nodesSetup []*kubetest.NodeConf, nsmrsExternalIP string) *v1.Pod {
	proxyNSMgrConfig := &pods.NSMgrPodConfig{
		Variables: pods.DefaultProxyNSMD(),
		Namespace: k8s.GetK8sNamespace(),
	}
	proxyNSMgrConfig.Variables[proxyregistryserver.NSMRSAddressEnv] = nsmrsExternalIP + ":80"
	pnsmd, err := kubetest.DeployProxyNSMgrWithConfig(k8s, nodesSetup[0].Node, pnsmdName, defaultTimeout, proxyNSMgrConfig)
	g.Expect(err).To(BeNil())

	return pnsmd
}
