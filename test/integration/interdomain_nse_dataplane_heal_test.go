// +build interdomain

package nsmd_integration_tests

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/networkservicemesh/networkservicemesh/test/kubetest/pods"

	"github.com/networkservicemesh/networkservicemesh/test/kubetest"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
)

func TestInterdomainNSCAndICMPDataplaneHealLocal(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	testInterdomainDataplaneHeal(t, 2, 2, 0)
}

func TestInterdomainNSCAndICMPDataplaneHealRemote(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	testInterdomainDataplaneHeal(t, 2, 2, 1)
}

func testInterdomainDataplaneHeal(t *testing.T, clustersCount int, nodesCount int, killIndex int) {
	k8ss := []*kubetest.ExtK8s{}

	for i := 0; i < clustersCount; i++ {
		kubeconfig := os.Getenv(fmt.Sprintf("KUBECONFIG_CLUSTER_%d", i+1))
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
			K8s:        k8s,
			NodesSetup: nodesSetup,
		})

		for j := 0; j < nodesCount; j++ {
			pnsmdName := fmt.Sprintf("pnsmgr-%s", nodesSetup[j].Node.Name)
			kubetest.DeployProxyNSMgr(k8s, nodesSetup[j].Node, pnsmdName, defaultTimeout)
		}

		serviceCleanup := kubetest.RunProxyNSMgrService(k8s)
		defer serviceCleanup()

		defer k8ss[i].K8s.Cleanup()
	}

	// Run ICMP on latest node
	_ = kubetest.DeployICMP(k8ss[clustersCount-1].K8s, k8ss[clustersCount-1].NodesSetup[nodesCount-1].Node, "icmp-responder-nse-1", defaultTimeout)

	nseExternalIP, err := kubetest.GetNodeExternalIP(k8ss[clustersCount-1].NodesSetup[0].Node)
	if err != nil {
		nseExternalIP, err = kubetest.GetNodeInternalIP(k8ss[clustersCount-1].NodesSetup[0].Node)
		Expect(err).To(BeNil())
	}

	nscPodNode := kubetest.DeployNSCWithEnv(k8ss[0].K8s, k8ss[0].NodesSetup[0].Node, "nsc-1", defaultTimeout, map[string]string{
		"OUTGOING_NSC_LABELS": "app=icmp",
		"OUTGOING_NSC_NAME":   fmt.Sprintf("icmp-responder@%s", nseExternalIP),
	})

	kubetest.CheckNSC(k8ss[0].K8s, nscPodNode)

	nodeKillIndex := 0
	if killIndex > 0 {
		nodeKillIndex = nodesCount - 1
	}

	logrus.Infof("Delete Selected dataplane")
	k8ss[killIndex].K8s.DeletePods(k8ss[killIndex].NodesSetup[nodeKillIndex].Dataplane)

	logrus.Infof("Wait NSMD is waiting for dataplane recovery")
	k8ss[killIndex].K8s.WaitLogsContains(k8ss[killIndex].NodesSetup[nodeKillIndex].Nsmd, "nsmd", "Waiting for Dataplane to recovery...", defaultTimeout)
	// Now are are in dataplane dead state, and in Heal procedure waiting for dataplane.
	dpName := fmt.Sprintf("nsmd-dataplane-recovered-%d", killIndex)

	logrus.Infof("Starting recovered dataplane...")
	startTime := time.Now()
	k8ss[killIndex].NodesSetup[0].Dataplane = k8ss[killIndex].K8s.CreatePod(pods.ForwardingPlane(dpName, k8ss[killIndex].NodesSetup[nodeKillIndex].Node, k8ss[killIndex].K8s.GetForwardingPlane()))
	logrus.Printf("Started new Dataplane: %v on node %s", time.Since(startTime), k8ss[killIndex].NodesSetup[nodeKillIndex].Node.Name)

	// Check NSMd goint into HEAL state.

	logrus.Infof("Waiting for connection recovery...")
	if killIndex != 0 {
		k8ss[killIndex].K8s.WaitLogsContains(k8ss[killIndex].NodesSetup[nodeKillIndex].Nsmd, "nsmd", "Healing will be continued on source side...", defaultTimeout)
		k8ss[0].K8s.WaitLogsContains(k8ss[0].NodesSetup[0].Nsmd, "nsmd", "Heal: Connection recovered:", defaultTimeout)
	} else {
		k8ss[killIndex].K8s.WaitLogsContains(k8ss[killIndex].NodesSetup[nodeKillIndex].Nsmd, "nsmd", "Heal: Connection recovered:", defaultTimeout)
	}
	logrus.Infof("Waiting for connection recovery Done...")

	kubetest.HealTestingPodFixture().CheckNsc(k8ss[0].K8s, nscPodNode)
}
