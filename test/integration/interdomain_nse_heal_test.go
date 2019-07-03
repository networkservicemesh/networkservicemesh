// +build interdomain

package nsmd_integration_tests

import (
	"fmt"
	"github.com/networkservicemesh/networkservicemesh/test/kubetest/pods"
	"os"
	"testing"

	"github.com/networkservicemesh/networkservicemesh/test/kubetest"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
)

func TestInterdomainNSEHealLocal(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	testInterdomainNSEHeal(t, 2, 2, map[string]int{
		"icmp-responder-nse-1": 0,
		"icmp-responder-nse-2": 0,
	}, kubetest.HealTestingPodFixture())
}

func TestInterdomainNSEHealRemote(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	testInterdomainNSEHeal(t, 2, 2, map[string]int{
		"icmp-responder-nse-1": 1,
		"icmp-responder-nse-2": 1,
	}, kubetest.HealTestingPodFixture())
}

func TestInterdomainNSEHealLocalToRemote(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	testInterdomainNSEHeal(t, 2, 2, map[string]int{
		"icmp-responder-nse-1": 0,
		"icmp-responder-nse-2": 1,
	}, kubetest.HealTestingPodFixture())
}

/**
If passed 1 both will be on same node, if not on different.
*/
func testInterdomainNSEHeal(t *testing.T, clustersCount int, nodesCount int, affinity map[string]int, fixture kubetest.TestingPodFixture) {
	k8ss := []* kubetest.ExtK8s{}

	for i := 0; i < clustersCount; i++ {
		kubeconfig := os.Getenv(fmt.Sprintf("KUBECONFIG_CLUSTER_%d", i + 1))
		Expect(len(kubeconfig)).ToNot(Equal(0))

		k8s, err := kubetest.NewK8sForConfig(true, kubeconfig)

		Expect(err).To(BeNil())

		config := []*pods.NSMgrPodConfig{}

		cfg := &pods.NSMgrPodConfig{
			Variables: pods.DefaultNSMD(),
		}
		cfg.Namespace = k8s.GetK8sNamespace()
		cfg.DataplaneVariables = kubetest.DefaultDataplaneVariables(k8s.GetForwardingPlane())

		config = append(config, cfg)

		nodesSetup, err := kubetest.SetupNodesConfig(k8s, nodesCount, defaultTimeout, config, k8s.GetK8sNamespace())
		Expect(err).To(BeNil())
		defer kubetest.ShowLogs(k8s, t)

		k8ss = append(k8ss, &kubetest.ExtK8s{
			K8s:      k8s,
			NodesSetup: nodesSetup,
		})

		for j := 0; j < nodesCount; j ++ {
			pnsmdName := fmt.Sprintf("pnsmgr-%s", nodesSetup[j].Node.Name)
			kubetest.DeployProxyNSMgr(k8s, nodesSetup[j].Node, pnsmdName, defaultTimeout)
		}

		serviceCleanup := kubetest.RunProxyNSMgrService(k8s)
		defer serviceCleanup()

		defer k8ss[i].K8s.Cleanup()
	}

	// Run ICMP
	node := affinity["icmp-responder-nse-1"]
	nse1 := kubetest.DeployICMP(k8ss[clustersCount - 1].K8s, k8ss[clustersCount - 1].NodesSetup[node].Node, "icmp-responder-nse-1", defaultTimeout)

	nseExternalIP, err := kubetest.GetNodeExternalIP(k8ss[clustersCount - 1].NodesSetup[0].Node)
	if err != nil {
		nseExternalIP, err = kubetest.GetNodeInternalIP(k8ss[clustersCount - 1].NodesSetup[0].Node)
		Expect(err).To(BeNil())
	}

	nscPodNode := kubetest.DeployNSCWithEnv(k8ss[0].K8s, k8ss[0].NodesSetup[0].Node, "nsc-1", defaultTimeout, map[string]string{
		"OUTGOING_NSC_LABELS": "app=icmp",
		"OUTGOING_NSC_NAME":   fmt.Sprintf("icmp-responder@%s", nseExternalIP),
	})

	kubetest.CheckNSC(k8ss[0].K8s, nscPodNode)

	// Since all is fine now, we need to add new ICMP responder and delete previous one.
	node = affinity["icmp-responder-nse-2"]
	kubetest.DeployICMP(k8ss[clustersCount - 1].K8s, k8ss[clustersCount - 1].NodesSetup[node].Node, "icmp-responder-nse-2", defaultTimeout)

	logrus.Infof("Delete first NSE")
	k8ss[clustersCount - 1].K8s.DeletePods(nse1)

	logrus.Infof("Waiting for connection recovery...")
	k8ss[0].K8s.WaitLogsContains(k8ss[0].NodesSetup[0].Nsmd, "nsmd", "Heal: Connection recovered:", defaultTimeout)

	kubetest.HealTestingPodFixture().CheckNsc(k8ss[0].K8s, nscPodNode)
}

