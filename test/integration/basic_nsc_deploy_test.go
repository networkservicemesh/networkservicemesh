// +build basic

package nsmd_integration_tests

import (
	"testing"

	"github.com/networkservicemesh/networkservicemesh/test/kube_testing"
	"github.com/networkservicemesh/networkservicemesh/test/kube_testing/pods"

	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
)

func TestDeployPodIntoInvalidEnv(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	logrus.Print("Running NSMD Deploy test")

	k8s, err := kube_testing.NewK8s(true)
	defer k8s.Cleanup()
	Expect(err).To(BeNil())

	nodes := k8s.GetNodesWait(1, defaultTimeout)

	if len(nodes) < 1 {
		logrus.Printf("At least two Kubernetes nodes are required for this test")
		Expect(len(nodes)).To(Equal(1))
		return
	}

	nsmdPodNode1, err := k8s.CreatePodsRaw(fastTimeout, false, pods.NSCPod("nsc-1", &nodes[0], map[string]string{}))
	Expect(len(nsmdPodNode1)).To(Equal(1))

	k8s.DeletePods(nsmdPodNode1...)

}
