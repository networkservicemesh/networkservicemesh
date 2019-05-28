// +build recover

package nsmd_integration_tests

import (
	"fmt"
	"testing"
	"time"

	"github.com/networkservicemesh/networkservicemesh/test/kubetest/pods"

	"github.com/networkservicemesh/networkservicemesh/test/kubetest"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
)

func TestDataplaneHealLocal(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	testDataplaneHeal(t, 1, kubetest.DeployNSC, kubetest.DeployICMP, kubetest.CheckNSC)
}

func TestDataplaneHealLocalMemif(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	testDataplaneHeal(t, 1, kubetest.DeployVppAgentNSC, kubetest.DeployVppAgentICMP, kubetest.CheckVppAgentNSC)
}

func TestDataplaneHealRemote(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	testDataplaneHeal(t, 2, kubetest.DeployNSC, kubetest.DeployICMP, kubetest.CheckNSC)
}

/**
If passed 1 both will be on same node, if not on different.
*/
func testDataplaneHeal(t *testing.T, nodesCount int, createNSC, createICMP kubetest.PodSupplier, checkNsc kubetest.NscChecker) {
	k8s, err := kubetest.NewK8s(true)
	defer k8s.Cleanup()

	Expect(err).To(BeNil())

	// Deploy open tracing to see what happening.
	nodes_setup, err := kubetest.SetupNodes(k8s, nodesCount, defaultTimeout)
	Expect(err).To(BeNil())

	// Run ICMP on latest node
	createICMP(k8s, nodes_setup[nodesCount-1].Node, "icmp-responder-nse-1", defaultTimeout)

	nscPodNode := createNSC(k8s, nodes_setup[0].Node, "nsc-1", defaultTimeout)
	var nscInfo *kubetest.NSCCheckInfo
	failures := InterceptGomegaFailures(func() {
		nscInfo = checkNsc(k8s, nscPodNode)
	})
	// Do dumping of container state to dig into what is happened.
	kubetest.PrintErrors(failures, k8s, nodes_setup, nscInfo, t)

	logrus.Infof("Delete Selected dataplane")
	k8s.DeletePods(nodes_setup[nodesCount-1].Dataplane)

	logrus.Infof("Wait NSMD is waiting for dataplane recovery")
	k8s.WaitLogsContains(nodes_setup[nodesCount-1].Nsmd, "nsmd", "Waiting for Dataplane to recovery...", defaultTimeout)
	// Now are are in dataplane dead state, and in Heal procedure waiting for dataplane.
	dpName := fmt.Sprintf("nsmd-dataplane-recovered-%d", nodesCount-1)

	logrus.Infof("Starting recovered dataplane...")
	startTime := time.Now()
	nodes_setup[nodesCount-1].Dataplane = k8s.CreatePod(pods.VPPDataplanePod(dpName, nodes_setup[nodesCount-1].Node))
	logrus.Printf("Started new Dataplane: %v on node %s", time.Since(startTime), nodes_setup[nodesCount-1].Node.Name)

	// Check NSMd goint into HEAL state.

	logrus.Infof("Waiting for connection recovery...")
	if nodesCount > 1 {
		k8s.WaitLogsContains(nodes_setup[nodesCount-1].Nsmd, "nsmd", "Healing will be continued on source side...", defaultTimeout)
		k8s.WaitLogsContains(nodes_setup[0].Nsmd, "nsmd", "Heal: Connection recovered:", defaultTimeout)
	} else {
		k8s.WaitLogsContains(nodes_setup[nodesCount-1].Nsmd, "nsmd", "Heal: Connection recovered:", defaultTimeout)
	}
	logrus.Infof("Waiting for connection recovery Done...")

	failures = InterceptGomegaFailures(func() {
		nscInfo = checkNsc(k8s, nscPodNode)
	})
	kubetest.PrintErrors(failures, k8s, nodes_setup, nscInfo, t)
}
