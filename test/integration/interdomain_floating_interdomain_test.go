package nsmd_integration_tests

import (
	"fmt"
	"github.com/networkservicemesh/networkservicemesh/test/kubetest"
	"github.com/networkservicemesh/networkservicemesh/test/kubetest/pods"
	. "github.com/onsi/gomega"
	"os"
	"testing"
)

func TestFloatingInterdomain(t *testing.T) {
	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	testFloatingInterdomain(t, 2)
}

func testFloatingInterdomain(t *testing.T, clustersCount int) {
	g := NewWithT(t)

	k8ss := []*kubetest.ExtK8s{}

	for i := 0; i < clustersCount; i++ {
		kubeconfig := os.Getenv(fmt.Sprintf("KUBECONFIG_CLUSTER_%d", i+1))
		g.Expect(len(kubeconfig)).ToNot(Equal(0))

		k8s, err := kubetest.NewK8sForConfig(g, true, kubeconfig)
		g.Expect(err).To(BeNil())
		//defer k8s.Cleanup()
		//defer kubetest.MakeLogsSnapshot(k8s, t)

		nseNoHealPodConfig.Namespace = k8s.GetK8sNamespace()
		nseNoHealPodConfig.DataplaneVariables = kubetest.DefaultDataplaneVariables(k8s.GetForwardingPlane())

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
	nsmrsNode := &k8ss[clustersCount-1].K8s.GetNodesWait(2, defaultTimeout)[1]
	kubetest.DeployNSMRS(k8ss[clustersCount - 1].K8s, nsmrsNode, "nsmrs", defaultTimeout)

	//<- time.After(1 * time.Minute)

	_ = kubetest.DeployICMP(k8ss[clustersCount-1].K8s, k8ss[clustersCount-1].NodesSetup[0].Node, "icmp-responder-nse-1", defaultTimeout)

	nseExternalIP, err := kubetest.GetNodeExternalIP(nsmrsNode)
	if err != nil {
		nseExternalIP, err = kubetest.GetNodeInternalIP(nsmrsNode)
		g.Expect(err).To(BeNil())
	}

	nscPodNode := kubetest.DeployNSCWithEnv(k8ss[0].K8s, k8ss[0].NodesSetup[0].Node, "nsc-1", defaultTimeout, map[string]string{
		"OUTGOING_NSC_LABELS": "app=icmp",
		"OUTGOING_NSC_NAME":   fmt.Sprintf("icmp-responder@%s", nseExternalIP),
	})

	kubetest.CheckNSC(k8ss[0].K8s, nscPodNode)

	//<- time.After(1 * time.Minute)
}
