// +build basic

package nsmd_integration_tests

import (
	"testing"

	"github.com/networkservicemesh/networkservicemesh/test/kube_testing"
	"github.com/networkservicemesh/networkservicemesh/test/kube_testing/pods"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
)

func TestExec(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	k8s, err := kube_testing.NewK8sWithoutRoles(false)
	defer k8s.Cleanup()

	Expect(err).To(BeNil())

	k8s.Prepare("alpine-pod")

	alpinePod := k8s.CreatePod(pods.AlpinePod("alpine-pod", nil))

	ipResponse, errResponse, error := k8s.Exec(alpinePod, alpinePod.Spec.Containers[0].Name, "ip", "addr")
	Expect(error).To(BeNil())
	Expect(errResponse).To(Equal(""))
	logrus.Printf("NSC IP status:%s", ipResponse)
	logrus.Printf("End of test")
	k8s.DeletePods(alpinePod)

}
