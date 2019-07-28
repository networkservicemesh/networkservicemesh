// +build basic

package nsmd_integration_tests

import (
	"testing"

	"github.com/onsi/gomega"

	"github.com/networkservicemesh/networkservicemesh/test/kubetest"
)

func TestDataplaneVersion(t *testing.T) {
	gomega.RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}
	k8s, err := kubetest.NewK8s(true)
	defer k8s.Cleanup()

	gomega.Expect(err).To(gomega.BeNil())

	nodes, err := kubetest.SetupNodes(k8s, 1, defaultTimeout)
	gomega.Expect(err).To(gomega.BeNil())
	defer kubetest.ShowLogs(k8s, t)

	gomega.Expect(len(nodes) > 0).Should(gomega.BeTrue())
	dataplane := nodes[0].Dataplane
	k8s.PrintImageVersion(dataplane)

}
