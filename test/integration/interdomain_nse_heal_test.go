// +build interdomain_suite

package nsmd_integration_tests

import (
	"fmt"
	"os"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/test/kubetest"
)

func TestInterdomainNSEHealLocal(t *testing.T) {
	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	g := NewWithT(t)

	testInterdomainNSEHeal(t, 2, 2, map[string]int{
		"icmp-responder-nse-1": 0,
		"icmp-responder-nse-2": 0,
	}, kubetest.DefaultTestingPodFixture(g))
}

func TestInterdomainNSEHealRemote(t *testing.T) {
	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	g := NewWithT(t)

	testInterdomainNSEHeal(t, 2, 2, map[string]int{
		"icmp-responder-nse-1": 1,
		"icmp-responder-nse-2": 1,
	}, kubetest.DefaultTestingPodFixture(g))
}

func TestInterdomainNSEHealLocalToRemote(t *testing.T) {
	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	g := NewWithT(t)

	testInterdomainNSEHeal(t, 2, 2, map[string]int{
		"icmp-responder-nse-1": 0,
		"icmp-responder-nse-2": 1,
	}, kubetest.DefaultTestingPodFixture(g))
}

func testInterdomainNSEHeal(t *testing.T, clustersCount int, nodesCount int, affinity map[string]int, fixture kubetest.TestingPodFixture) {
	g := NewWithT(t)

	k8ss := []*kubetest.ExtK8s{}

	for i := 0; i < clustersCount; i++ {
		kubeconfig := os.Getenv(fmt.Sprintf("KUBECONFIG_CLUSTER_%d", i+1))
		g.Expect(len(kubeconfig)).ToNot(Equal(0))

		k8s, err := kubetest.NewK8sForConfig(g, kubetest.ReuseNSMResources, kubeconfig)
		g.Expect(err).To(BeNil())
		defer k8s.Cleanup()
		defer k8s.ProcessArtifacts(t)

		nodesSetup, err := kubetest.SetupNodes(k8s, nodesCount, defaultTimeout)
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

	// Run ICMP
	node := affinity["icmp-responder-nse-1"]
	nse1 := kubetest.DeployICMP(k8ss[clustersCount-1].K8s, k8ss[clustersCount-1].NodesSetup[node].Node, "icmp-responder-nse-1", defaultTimeout)

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

	// Since all is fine now, we need to add new ICMP responder and delete previous one.
	node = affinity["icmp-responder-nse-2"]
	kubetest.DeployICMP(k8ss[clustersCount-1].K8s, k8ss[clustersCount-1].NodesSetup[node].Node, "icmp-responder-nse-2", defaultTimeout)

	logrus.Infof("Delete first NSE")
	k8ss[clustersCount-1].K8s.DeletePods(nse1)

	logrus.Infof("Waiting for connection recovery...")
	k8ss[0].K8s.WaitLogsContains(k8ss[0].NodesSetup[0].Nsmd, "nsmd", "Heal: Connection recovered:", defaultTimeout)

	kubetest.DefaultTestingPodFixture(g).CheckNsc(k8ss[0].K8s, nscPodNode)
}
