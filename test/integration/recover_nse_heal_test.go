// +build recover

package nsmd_integration_tests

import (
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/test/kubetest"
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
	}, kubetest.HealTestingPodFixture())
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
	}, kubetest.HealTestingPodFixture())
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
	}, kubetest.HealTestingPodFixture())
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
	}, kubetest.HealTestingPodFixture())
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
	}, kubetest.VppAgentTestingPodFixture())
}

/**
If passed 1 both will be on same node, if not on different.
*/
func testNSEHeal(t *testing.T, nodesCount int, affinity map[string]int,
	fixture kubetest.TestingPodFixture) {
	k8s, err := kubetest.NewK8s(true)
	defer k8s.Cleanup()
	Expect(err).To(BeNil())

	// Deploy open tracing to see what happening.
	nodesSetup, err := kubetest.SetupNodes(k8s, nodesCount, defaultTimeout)
	Expect(err).To(BeNil())
	defer kubetest.ShowLogs(k8s, t)

	// Run ICMP
	node := affinity["icmp-responder-nse-1"]
	nse1 := fixture.DeployNse(k8s, nodesSetup[node].Node, "icmp-responder-nse-1", defaultTimeout)

	nscPodNode := fixture.DeployNsc(k8s, nodesSetup[0].Node, "nsc-1", defaultTimeout)
	fixture.CheckNsc(k8s, nscPodNode)

	// Since all is fine now, we need to add new ICMP responder and delete previous one.
	node = affinity["icmp-responder-nse-2"]
	fixture.DeployNse(k8s, nodesSetup[node].Node, "icmp-responder-nse-2", defaultTimeout)

	logrus.Infof("Delete first NSE")
	k8s.DeletePods(nse1)

	logrus.Infof("Waiting for connection recovery...")

	k8s.WaitLogsContains(nodesSetup[0].Nsmd, "nsmd", "Heal: Connection recovered:", defaultTimeout)

	if len(nodesSetup) > 1 {
		l2, err := k8s.GetLogs(nodesSetup[1].Nsmd, "nsmd")
		Expect(err).To(BeNil())
		if strings.Contains(l2, "Dataplane request failed:") {
			logrus.Infof("Dataplane first attempt was failed: %v", l2)
		}
	}

	fixture.CheckNsc(k8s, nscPodNode)
}
