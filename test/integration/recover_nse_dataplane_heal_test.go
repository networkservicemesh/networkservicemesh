// +build recover

package nsmd_integration_tests

import (
	"fmt"
	"testing"
	"time"

	"github.com/networkservicemesh/networkservicemesh/test/integration/nsmd_test_utils"
	"github.com/networkservicemesh/networkservicemesh/test/kube_testing/pods"

	"github.com/networkservicemesh/networkservicemesh/test/kube_testing"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
)

func TestDataplaneHealLocal(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	testDataplaneHeal(t, 1, nsmd_test_utils.DeployNSC, nsmd_test_utils.DeployICMP, nsmd_test_utils.CheckNSC)
}

func TestDataplaneHealLocalMemif(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	testDataplaneHeal(t, 1, nsmd_test_utils.DeployVppAgentNSC, nsmd_test_utils.DeployVppAgentICMP, nsmd_test_utils.CheckVppAgentNSC)
}

func TestDataplaneHealRemote(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	testDataplaneHeal(t, 2, nsmd_test_utils.DeployNSC, nsmd_test_utils.DeployICMP, nsmd_test_utils.CheckNSC)
}

/**
If passed 1 both will be on same node, if not on different.
*/
func testDataplaneHeal(t *testing.T, nodesCount int, createNSC, createICMP nsmd_test_utils.PodSupplier, checkNsc nsmd_test_utils.NscChecker) {
	k8s, err := kube_testing.NewK8s(true)
	defer k8s.Cleanup()

	Expect(err).To(BeNil())

	// Deploy open tracing to see what happening.
	nodes_setup := nsmd_test_utils.SetupNodes(k8s, nodesCount, defaultTimeout)

	// Run ICMP on latest node
	createICMP(k8s, nodes_setup[nodesCount-1].Node, "icmp-responder-nse-1", defaultTimeout)

	nscPodNode := createNSC(k8s, nodes_setup[0].Node, "nsc-1", defaultTimeout)
	var nscInfo *nsmd_test_utils.NSCCheckInfo
	failures := InterceptGomegaFailures(func() {
		nscInfo = checkNsc(k8s, t, nscPodNode)
	})
	// Do dumping of container state to dig into what is happened.
	nsmd_test_utils.PrintErrors(failures, k8s, nodes_setup, nscInfo, t)

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
		nscInfo = checkNsc(k8s, t, nscPodNode)
	})
	nsmd_test_utils.PrintErrors(failures, k8s, nodes_setup, nscInfo, t)
}
