// +build recover_suite

package integration

import (
	"fmt"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/test/kubetest"
	"github.com/networkservicemesh/networkservicemesh/test/kubetest/pods"
)

func TestForwarderHealLocal(t *testing.T) {
	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	g := NewWithT(t)

	testForwarderHeal(t, 0, 1, kubetest.DefaultTestingPodFixture(g))
}

func TestForwarderHealLocalMemif(t *testing.T) {
	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	g := NewWithT(t)

	testForwarderHeal(t, 0, 1, kubetest.VppAgentTestingPodFixture(g))
}

func TestForwarderHealMultiNodesLocal(t *testing.T) {
	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	g := NewWithT(t)
	testForwarderHeal(t, 0, 2, kubetest.DefaultTestingPodFixture(g))
}

func TestForwarderHealMultiNodesRemote(t *testing.T) {
	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	g := NewWithT(t)

	testForwarderHeal(t, 1, 2, kubetest.DefaultTestingPodFixture(g))
}

/**
If passed 1 both will be on same node, if not on different.
*/
func testForwarderHeal(t *testing.T, killForwarderIndex, nodesCount int, fixture kubetest.TestingPodFixture) {
	g := NewWithT(t)

	g.Expect(nodesCount > 0).Should(BeTrue())
	g.Expect(killForwarderIndex >= 0 && killForwarderIndex < nodesCount).Should(BeTrue())
	k8s, err := kubetest.NewK8s(g, kubetest.ReuseNSMResources)
	defer k8s.Cleanup(t)
	g.Expect(err).To(BeNil())

	// Deploy open tracing to see what happening.
	nodes_setup, err := kubetest.SetupNodes(k8s, nodesCount, defaultTimeout)
	g.Expect(err).To(BeNil())
	defer k8s.SaveTestArtifacts(t)
	// Run ICMP on latest node
	fixture.DeployNse(k8s, nodes_setup[nodesCount-1].Node, "icmp-responder-nse-1", defaultTimeout)

	nscPodNode := fixture.DeployNsc(k8s, nodes_setup[0].Node, "nsc-1", defaultTimeout)
	fixture.CheckNsc(k8s, nscPodNode)

	logrus.Infof("Delete Selected forwarder")
	k8s.DeletePods(nodes_setup[killForwarderIndex].Forwarder)

	logrus.Infof("Wait NSMD is waiting for forwarder recovery")
	k8s.WaitLogsContains(nodes_setup[killForwarderIndex].Nsmd, "nsmd", "Waiting for Forwarder to recovery...", defaultTimeout)
	// Now are are in forwarder dead state, and in Heal procedure waiting for forwarder.
	dpName := fmt.Sprintf("nsmd-forwarder-recovered-%d", killForwarderIndex)

	logrus.Infof("Starting recovered forwarder...")
	startTime := time.Now()
	nodes_setup[killForwarderIndex].Forwarder = k8s.CreatePod(pods.ForwardingPlane(dpName, nodes_setup[killForwarderIndex].Node, k8s.GetForwardingPlane()))
	logrus.Printf("Started new Forwarder: %v on node %s", time.Since(startTime), nodes_setup[killForwarderIndex].Node.Name)

	// Check NSMd goint into HEAL state.

	logrus.Infof("Waiting for connection recovery...")
	if nodesCount > 1 && killForwarderIndex != 0 {
		k8s.WaitLogsContains(nodes_setup[nodesCount-1].Nsmd, "nsmd", "Healing will be continued on source side...", defaultTimeout)
		k8s.WaitLogsContains(nodes_setup[0].Nsmd, "nsmd", "Heal: Connection recovered:", defaultTimeout)
	} else {
		k8s.WaitLogsContains(nodes_setup[killForwarderIndex].Nsmd, "nsmd", "Heal: Connection recovered:", defaultTimeout)
	}
	logrus.Infof("Waiting for connection recovery Done...")
	fixture.CheckNsc(k8s, nscPodNode)
}
