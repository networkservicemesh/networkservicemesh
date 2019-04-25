// +build basic

package nsmd_integration_tests

import (
	"strings"
	"testing"

	"github.com/networkservicemesh/networkservicemesh/test/kube_testing"
	. "github.com/onsi/gomega"
)

func TestKubernetesAreOk(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	k8s, err := kube_testing.NewK8sWithoutRoles(false)
	defer k8s.Cleanup()

	Expect(err).To(BeNil())
	v := k8s.GetVersion()
	Expect(strings.Contains(v, "1.")).To(Equal(true))

}
