// +build basic

package nsmd_integration_tests

import (
	"strings"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/networkservicemesh/networkservicemesh/test/kubetest"
)

func TestKubernetesAreOk(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	k8s, err := kubetest.NewK8sWithoutRoles(false)
	defer k8s.Cleanup()
	Expect(err).To(BeNil())
	defer kubetest.ShowLogs(k8s, t)
	v := k8s.GetVersion()
	Expect(strings.Contains(v, "1.")).To(Equal(true))

}
