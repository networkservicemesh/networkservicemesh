// +build suite basic

package nsmd_integration_tests

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/networkservicemesh/networkservicemesh/test/kubetest"
)

func TestDeployNSMMonitor(t *testing.T) {
	g := NewWithT(t)

	k8s, err := kubetest.NewK8s(g, kubetest.ReuseNSMResources)
	defer k8s.Cleanup()
	defer k8s.ProcessArtifacts(t)
	g.Expect(err).To(BeNil())

	nodes_setup, err := kubetest.SetupNodes(k8s, 2, defaultTimeout)
	g.Expect(err).To(BeNil())

	// Run ICMP on latest node
	_ = kubetest.DeployICMP(k8s, nodes_setup[1].Node, "icmp-responder-nse-1", defaultTimeout)

	nscPodNode := kubetest.DeployNSCMonitor(k8s, nodes_setup[0].Node, "nsc-1", defaultTimeout)
	kubetest.CheckNSC(k8s, nscPodNode)

	k8s.WaitLogsContains(nscPodNode, "nsm-monitor", "", defaultTimeout)
	// Do dumping of container state to dig into what is happened.
}
