// +build basic

package nsmd_integration_tests

import (
	"testing"

	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/test/kubetest"
)

func TestDeployNSMMonitor(t *testing.T) {
	RegisterTestingT(t)

	k8s, err := kubetest.NewK8s(true)
	defer k8s.Cleanup()
	defer kubetest.ShowLogs(k8s, t)
	Expect(err).To(BeNil())

	nodes_setup, err := kubetest.SetupNodes(k8s, 2, defaultTimeout)
	Expect(err).To(BeNil())

	// Run ICMP on latest node
	_ = kubetest.DeployICMP(k8s, nodes_setup[1].Node, "icmp-responder-nse-1", defaultTimeout)

	nscPodNode := kubetest.DeployNSCMonitor(k8s, nodes_setup[0].Node, "nsc-1", defaultTimeout)
	var nscInfo *kubetest.NSCCheckInfo

	failures := InterceptGomegaFailures(func() {
		nscInfo = kubetest.CheckNSC(k8s, nscPodNode)
	})

	k8s.WaitLogsContains(nscPodNode, "nsm-monitor", "", defaultTimeout)
	// Do dumping of container state to dig into what is happened.
	if len(failures) > 0 {
		logrus.Errorf("Failures: %v", failures)

		nscInfo.PrintLogs()

		t.Fail()
	}

}
