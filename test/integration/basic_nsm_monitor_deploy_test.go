// +build basic

package nsmd_integration_tests

import (
	"testing"

	"github.com/networkservicemesh/networkservicemesh/test/kubetest"
	"github.com/networkservicemesh/networkservicemesh/test/kubetest/pods"

	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
)

func TestDeployNSMMonitor(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	logrus.Print("Running NSM Monitor test")

	k8s, err := kubetest.NewK8s(true)
	defer k8s.Cleanup()
	Expect(err).To(BeNil())
	defer kubetest.ShowLogs(k8s, t)

	nodes, err := kubetest.SetupNodes(k8s, 1, defaultTimeout)
	Expect(err).To(BeNil())
	Expect(nodes).NotTo(BeNil())

	nsmdPodNode1, err := k8s.CreatePodsRaw(fastTimeout, false, pods.NSCMonitorPod("nsc-1", nodes[0].Node, map[string]string{}))
	Expect(len(nsmdPodNode1)).To(Equal(1))

	k8s.DeletePods(nsmdPodNode1...)

}
