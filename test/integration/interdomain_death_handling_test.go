// +build interdomain

package nsmd_integration_tests

import (
	"fmt"
	"os"
	"strings"
	"testing"

	v1 "k8s.io/api/core/v1"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/nsm"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/nsmd"
	"github.com/networkservicemesh/networkservicemesh/test/kubetest/pods"

	. "github.com/onsi/gomega"

	"github.com/networkservicemesh/networkservicemesh/test/kubetest"
)

var nseNoHealPodConfig = &pods.NSMgrPodConfig{
	Variables: map[string]string{
		nsmd.NsmdDeleteLocalRegistry: "true", // Do not use local registry restore for clients/NSEs
		nsm.NsmdHealDSTWaitTimeout:   "1",    // 1 second
		nsm.NsmdHealEnabled:          "true",
	},
}

func TestInterdomainNSCDies(t *testing.T) {
	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	testInterdomainNSMDies(t, 2, true)
}

func TestInterdomainNSEDies(t *testing.T) {
	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	testInterdomainNSMDies(t, 2, false)
}

func testInterdomainNSMDies(t *testing.T, clustersCount int, killSrc bool) {
	g := NewWithT(t)

	k8ss := []*kubetest.ExtK8s{}

	for i := 0; i < clustersCount; i++ {
		kubeconfig := os.Getenv(fmt.Sprintf("KUBECONFIG_CLUSTER_%d", i+1))
		g.Expect(len(kubeconfig)).ToNot(Equal(0))

		k8s, err := kubetest.NewK8sForConfig(g, true, kubeconfig)
		g.Expect(err).To(BeNil())
		defer k8s.Cleanup()
		defer kubetest.MakeLogsSnapshot(k8s, t)

		nseNoHealPodConfig.Namespace = k8s.GetK8sNamespace()
		nseNoHealPodConfig.ForwarderVariables = kubetest.DefaultForwarderVariables(k8s.GetForwardingPlane())

		nodesSetup, err := kubetest.SetupNodesConfig(k8s, 1, defaultTimeout, []*pods.NSMgrPodConfig{
			nseNoHealPodConfig,
			nseNoHealPodConfig,
		}, k8s.GetK8sNamespace())
		g.Expect(err).To(BeNil())

		k8ss = append(k8ss, &kubetest.ExtK8s{
			K8s:        k8s,
			NodesSetup: nodesSetup,
		})

		pnsmdName := fmt.Sprintf("pnsmgr-%s", nodesSetup[0].Node.Name)
		kubetest.DeployProxyNSMgr(k8s, nodesSetup[0].Node, pnsmdName, defaultTimeout)

		serviceCleanup := kubetest.RunProxyNSMgrService(k8s)
		defer serviceCleanup()
	}

	// Run ICMP
	icmpPodNode := kubetest.DeployICMP(k8ss[clustersCount-1].K8s, k8ss[clustersCount-1].NodesSetup[0].Node, "icmp-responder-nse-1", defaultTimeout)

	nseExternalIP, err := kubetest.GetNodeExternalIP(k8ss[clustersCount-1].NodesSetup[0].Node)
	if err != nil {
		nseExternalIP, err = kubetest.GetNodeInternalIP(k8ss[clustersCount-1].NodesSetup[0].Node)
		g.Expect(err).To(BeNil())
	}

	nscPodNode := kubetest.DeployNSCWithEnv(k8ss[0].K8s, k8ss[0].NodesSetup[0].Node, "nsc-1", defaultTimeout, map[string]string{
		"OUTGOING_NSC_LABELS": "app=icmp",
		"OUTGOING_NSC_NAME":   fmt.Sprintf("icmp-responder@%s", nseExternalIP),
	})

	kubetest.CheckNSC(k8ss[0].K8s, nscPodNode)
	ipResponse, errOut, err := k8ss[clustersCount-1].K8s.Exec(icmpPodNode, icmpPodNode.Spec.Containers[0].Name, "ip", "addr")
	g.Expect(err).To(BeNil())
	g.Expect(errOut).To(Equal(""))
	g.Expect(strings.Contains(ipResponse, "nsm")).To(Equal(true))

	var podToKill *v1.Pod
	var clusterToKill int
	var podToCheck *v1.Pod
	var clusterToCheck int
	if killSrc {
		podToKill = nscPodNode
		clusterToKill = 0
		podToCheck = icmpPodNode
		clusterToCheck = clustersCount - 1
	} else {
		podToKill = icmpPodNode
		clusterToKill = clustersCount - 1
		podToCheck = nscPodNode
		clusterToCheck = 0
	}

	k8ss[clusterToKill].K8s.DeletePods(podToKill)
	k8ss[clusterToCheck].K8s.WaitLogsContains(k8ss[clusterToCheck].NodesSetup[0].Nsmd, "nsmd", "Cross connection successfully closed on forwarder", defaultTimeout)

	ipResponse, errOut, err = k8ss[clusterToCheck].K8s.Exec(podToCheck, podToCheck.Spec.Containers[0].Name, "ip", "addr")
	g.Expect(err).To(BeNil())
	g.Expect(errOut).To(Equal(""))
	g.Expect(strings.Contains(ipResponse, "nsm")).To(Equal(false))
}
