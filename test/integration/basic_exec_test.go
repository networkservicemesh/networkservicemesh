// +build basic

package nsmd_integration_tests

import (
	"testing"

	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/test/kubetest"
	"github.com/networkservicemesh/networkservicemesh/test/kubetest/pods"
)

func TestExec(t *testing.T) {
	g := NewWithT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	k8s, err := kubetest.NewK8sWithoutRoles(g, kubetest.ReuseNSMResources)
	defer k8s.Cleanup()
	g.Expect(err).To(BeNil())
	defer k8s.ProcessArtifacts(t)

	k8s.DeletePodsByName("alpine-pod")

	alpinePod := k8s.CreatePod(pods.AlpinePod("alpine-pod", nil))

	ipResponse, errResponse, error := k8s.Exec(alpinePod, alpinePod.Spec.Containers[0].Name, "ip", "addr")
	g.Expect(error).To(BeNil())
	g.Expect(errResponse).To(Equal(""))
	logrus.Printf("NSC IP status:%s", ipResponse)
	logrus.Printf("End of test")
	k8s.DeletePods(alpinePod)
}
