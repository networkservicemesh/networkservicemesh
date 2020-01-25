// +build basic

package integration

import (
	"testing"
	"time"

	"github.com/networkservicemesh/networkservicemesh/test/kubetest"
	"github.com/networkservicemesh/networkservicemesh/test/kubetest/pods"

	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
)

func TestDeployPodIntoInvalidEnv(t *testing.T) {
	g := NewWithT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	logrus.Print("Running NSMD Deploy test")

	k8s, err := kubetest.NewK8s(g, true)
	defer k8s.Cleanup()
	g.Expect(err).To(BeNil())
	defer kubetest.MakeLogsSnapshot(k8s, t)

	nodes := k8s.GetNodesWait(1, defaultTimeout)

	if len(nodes) < 1 {
		logrus.Printf("At least two Kubernetes nodes are required for this test")
		g.Expect(len(nodes)).To(Equal(1))
		return
	}

	nsmdPodNode1, err := k8s.CreatePodsRaw(3*time.Second, false, pods.NSCPod("nsc-1", &nodes[0], map[string]string{}))
	g.Expect(len(nsmdPodNode1)).To(Equal(1))

	k8s.DeletePods(nsmdPodNode1...)

}
