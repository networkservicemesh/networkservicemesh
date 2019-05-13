// +build recover

package nsmd_integration_tests

import (
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/test/integration/utils"
	"github.com/networkservicemesh/networkservicemesh/test/kube_testing"
)

func TestNSEHealLocal(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	testNSEHeal(t, 1, map[string]int{
		"icmp-responder-nse-1": 0,
		"icmp-responder-nse-2": 0,
	}, utils.DeployNSC, utils.DeployICMP, utils.CheckNSC)
}

func TestNSEHealLocalToRemote(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	testNSEHeal(t, 2, map[string]int{
		"icmp-responder-nse-1": 0,
		"icmp-responder-nse-2": 1,
	}, utils.DeployNSC, utils.DeployICMP, utils.CheckNSC)
}

func TestNSEHealRemoteToLocal(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	testNSEHeal(t, 2, map[string]int{
		"icmp-responder-nse-1": 1,
		"icmp-responder-nse-2": 0,
	}, utils.DeployNSC, utils.DeployICMP, utils.CheckNSC)
}

func TestNSEHealRemote(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	testNSEHeal(t, 2, map[string]int{
		"icmp-responder-nse-1": 1,
		"icmp-responder-nse-2": 1,
	}, utils.DeployNSC, utils.DeployICMP, utils.CheckNSC)
}

func TestNSEHealLocalMemif(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	testNSEHeal(t, 1, map[string]int{
		"icmp-responder-nse-1": 0,
		"icmp-responder-nse-2": 0,
	}, utils.DeployVppAgentNSC, utils.DeployVppAgentICMP, utils.CheckVppAgentNSC)
}

/**
If passed 1 both will be on same node, if not on different.
*/
func testNSEHeal(t *testing.T, nodesCount int, affinity map[string]int,
	nscDeploy, icmpDeploy utils.PodSupplier, nscCheck utils.NscChecker) {
	k8s, err := kube_testing.NewK8s(true)
	defer k8s.Cleanup()
	Expect(err).To(BeNil())

	// Deploy open tracing to see what happening.
	nodes_setup, err := utils.SetupNodes(k8s, nodesCount, defaultTimeout)
	Expect(err).To(BeNil())

	// Run ICMP
	node := affinity["icmp-responder-nse-1"]
	nse1 := icmpDeploy(k8s, nodes_setup[node].Node, "icmp-responder-nse-1", defaultTimeout)

	nscPodNode := nscDeploy(k8s, nodes_setup[0].Node, "nsc-1", defaultTimeout)
	var nscInfo *utils.NSCCheckInfo
	failures := InterceptGomegaFailures(func() {
		nscInfo = nscCheck(k8s, t, nscPodNode)
	})
	// Do dumping of container state to dig into what is happened.Heal: Connection recovered
	utils.PrintErrors(failures, k8s, nodes_setup, nscInfo, t)

	// Since all is fine now, we need to add new ICMP responder and delete previous one.
	node = affinity["icmp-responder-nse-2"]
	icmpDeploy(k8s, nodes_setup[node].Node, "icmp-responder-nse-2", defaultTimeout)

	logrus.Infof("Delete first NSE")
	k8s.DeletePods(nse1)

	logrus.Infof("Waiting for connection recovery...")
	failures = InterceptGomegaFailures(func() {
		k8s.WaitLogsContains(nodes_setup[0].Nsmd, "nsmd", "Heal: Connection recovered:", defaultTimeout)
	})
	if len(failures) > 0 {
		utils.PrintErrors(failures, k8s, nodes_setup, nscInfo, t)
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
	utils.PrintErrors(failures, k8s, nodes_setup, nscInfo, t)
}
