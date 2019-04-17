// +build recover

package nsmd_integration_tests

import (
	"strings"
	"testing"
	"time"

	"github.com/networkservicemesh/networkservicemesh/test/integration/nsmd_test_utils"

	"github.com/networkservicemesh/networkservicemesh/test/kube_testing"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
)

func TestNSEHealLocal(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	testNSEHeal(t, 1, nsmd_test_utils.DeployNSC, nsmd_test_utils.DeployICMP, nsmd_test_utils.CheckNSC)
}

func TestNSEHealLocalMemif(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	testNSEHeal(t, 1, nsmd_test_utils.DeployVppAgentNSC, nsmd_test_utils.DeployVppAgentICMP, nsmd_test_utils.CheckVppAgentNSC)
}

func TestNSEHealRemote(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	testNSEHeal(t, 2, nsmd_test_utils.DeployNSC, nsmd_test_utils.DeployICMP, nsmd_test_utils.CheckNSC)
}

/**
If passed 1 both will be on same node, if not on different.
*/
func testNSEHeal(t *testing.T, nodesCount int, nscDeploy, icmpDeploy nsmd_test_utils.PodSupplier, nscCheck nsmd_test_utils.NscChecker) {
	k8s, err := kube_testing.NewK8s()
	defer k8s.Cleanup()

	Expect(err).To(BeNil())

	s1 := time.Now()
	k8s.PrepareDefault()
	logrus.Printf("Cleanup done: %v", time.Since(s1))

	// Deploy open tracing to see what happening.
	nodes_setup := nsmd_test_utils.SetupNodes(k8s, nodesCount, defaultTimeout)

	// Run ICMP on latest node
	nse1 := icmpDeploy(k8s, nodes_setup[nodesCount-1].Node, "icmp-responder-nse-1", defaultTimeout)

	nscPodNode := nscDeploy(k8s, nodes_setup[0].Node, "nsc-1", defaultTimeout)
	var nscInfo *nsmd_test_utils.NSCCheckInfo
	failures := InterceptGomegaFailures(func() {
		nscInfo = nscCheck(k8s, t, nscPodNode)
	})
	// Do dumping of container state to dig into what is happened.Heal: Connection recovered
	nsmd_test_utils.PrintErrors(failures, k8s, nodes_setup, nscInfo, t)

	// Since all is fine now, we need to add new ICMP responder and delete previous one.
	icmpDeploy(k8s, nodes_setup[nodesCount-1].Node, "icmp-responder-nse-2", defaultTimeout)

	logrus.Infof("Delete first NSE")
	k8s.DeletePods(nse1)

	logrus.Infof("Waiting for connection recovery...")
	failures = InterceptGomegaFailures(func() {
		k8s.WaitLogsContains(nodes_setup[0].Nsmd, "nsmd", "Heal: Connection recovered:", defaultTimeout)
	})
	if len(failures) > 0 {
		nsmd_test_utils.PrintErrors(failures, k8s, nodes_setup, nscInfo, t)
	}

	if len(nodes_setup) > 1 {
		l2, err := k8s.GetLogs(nodes_setup[1].Nsmd, "nsmd")
		Expect(err).To(BeNil())
		if strings.Contains(l2, "Dataplane request failed:") {
			logrus.Infof("Dataplane first attempt was failed: %v", l2)
		}
	}

	failures = InterceptGomegaFailures(func() {
		nscInfo = nscCheck(k8s, t, nscPodNode)
	})
	nsmd_test_utils.PrintErrors(failures, k8s, nodes_setup, nscInfo, t)
}
