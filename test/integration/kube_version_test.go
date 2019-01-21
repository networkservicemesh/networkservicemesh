package nsmd_integration_tests

import (
	"github.com/networkservicemesh/networkservicemesh/test/kube_testing"
	. "github.com/onsi/gomega"
	"strings"
	"testing"
)

func TestKubernetesAreOk(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	k8s, err := kube_testing.NewK8s()
	defer k8s.Cleanup()

	Expect(err).To(BeNil())
	v := k8s.GetVersion()
	Expect(strings.Contains(v, "1.")).To(Equal(true))

}
