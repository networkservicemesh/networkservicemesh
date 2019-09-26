package nsmd_integration_tests

import (
	"fmt"
	"os"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/networkservicemesh/networkservicemesh/k8s/pkg/proxyregistryserver"
	"github.com/networkservicemesh/networkservicemesh/test/kubetest"
	"github.com/networkservicemesh/networkservicemesh/test/kubetest/pods"
)

func TestFloatingInterdomainDieNSE(t *testing.T) {
	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	testFloatingInterdomainDie(t, 2, "nse")
}

func TestFloatingInterdomainDieNSMD(t *testing.T) {
	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	testFloatingInterdomainDie(t, 2, "nsmd")
}

func testFloatingInterdomainDie(t *testing.T, clustersCount int, killPod string) {
	g := NewWithT(t)

	k8ss := []*kubetest.ExtK8s{}
	for i := 0; i < clustersCount; i++ {
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

	nsmrsNode := &k8ss[clustersCount-1].K8s.GetNodesWait(2, defaultTimeout)[1]
	nsmrsPod := kubetest.DeployNSMRS(k8ss[clustersCount-1].K8s, nsmrsNode, "nsmrs", defaultTimeout)

	nsmrsExternalIP, err := kubetest.GetNodeExternalIP(nsmrsNode)
	if err != nil {
		nsmrsExternalIP, err = kubetest.GetNodeInternalIP(nsmrsNode)
		g.Expect(err).To(BeNil())
	}
	nsmrsInternalIP, err := kubetest.GetNodeInternalIP(nsmrsNode)
	g.Expect(err).To(BeNil())

	for i := 0; i < clustersCount; i++ {
		k8s := k8ss[i].K8s
		nseNoHealPodConfig.Namespace = k8s.GetK8sNamespace()
		nseNoHealPodConfig.DataplaneVariables = kubetest.DefaultDataplaneVariables(k8s.GetForwardingPlane())

		nodesSetup, err := kubetest.SetupNodesConfig(k8s, 1, defaultTimeout, []*pods.NSMgrPodConfig{
			nseNoHealPodConfig,
			nseNoHealPodConfig,
		}, k8s.GetK8sNamespace())
		g.Expect(err).To(BeNil())

		k8ss[i].NodesSetup = nodesSetup

		pnsmdName := fmt.Sprintf("pnsmgr-%s", nodesSetup[0].Node.Name)
		proxyNSMgrConfig := &pods.NSMgrPodConfig{
			Variables: pods.DefaultProxyNSMD(),
			Namespace: k8s.GetK8sNamespace(),
		}
		proxyNSMgrConfig.Variables[proxyregistryserver.NSMRSAddressEnv] = nsmrsInternalIP
		kubetest.DeployProxyNSMgrWithConfig(k8s, nodesSetup[0].Node, pnsmdName, defaultTimeout, proxyNSMgrConfig)

		serviceCleanup := kubetest.RunProxyNSMgrService(k8s)
		defer serviceCleanup()
	}

	icmpPod := kubetest.DeployICMP(k8ss[clustersCount-1].K8s, k8ss[clustersCount-1].NodesSetup[0].Node, "icmp-responder-nse-1", defaultTimeout)
	k8ss[clustersCount-1].K8s.WaitLogsContains(nsmrsPod, "nsmrs", "Returned from RegisterNSE", defaultTimeout)

	nscPodNode := kubetest.DeployNSCWithEnv(k8ss[0].K8s, k8ss[0].NodesSetup[0].Node, "nsc-1", defaultTimeout, map[string]string{
		"OUTGOING_NSC_LABELS": "app=icmp",
		"OUTGOING_NSC_NAME":   fmt.Sprintf("icmp-responder@%s", nsmrsExternalIP),
	})

	kubetest.CheckNSC(k8ss[0].K8s, nscPodNode)

	switch killPod {
	case "nse":
		k8ss[clustersCount-1].K8s.DeletePods(icmpPod)
	case "nsmd":
		k8ss[clustersCount-1].K8s.DeletePods(k8ss[clustersCount-1].NodesSetup[0].Nsmd)
	}
	k8ss[clustersCount-1].K8s.WaitLogsContains(nsmrsPod, "nsmrs", "RemoveNSE done", defaultTimeout)
}
