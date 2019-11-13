// +build basic

package nsmd_integration_tests

import (
	"strings"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/networkservicemesh/networkservicemesh/test/kubetest"
)

func TestKubernetesAreOk(t *testing.T) {
	g := NewWithT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	k8s, err := kubetest.NewK8sWithoutRoles(g, kubetest.NoClear)
	defer k8s.Cleanup()
	g.Expect(err).To(BeNil())
	defer k8s.ProcessArtifacts(t)
	v := k8s.GetVersion()
	g.Expect(strings.Contains(v, "1.")).To(Equal(true))

}
