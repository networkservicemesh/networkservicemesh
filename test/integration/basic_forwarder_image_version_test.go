// +build basic_suite

package integration

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/networkservicemesh/networkservicemesh/test/kubetest"
)

func TestForwarderVersion(t *testing.T) {
	g := NewWithT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}
	k8s, err := kubetest.NewK8s(g, kubetest.ReuseNSMResources)
	defer k8s.Cleanup(t)

	g.Expect(err).To(BeNil())

	nodes, err := kubetest.SetupNodes(k8s, 1, defaultTimeout)
	g.Expect(err).To(BeNil())
	defer k8s.SaveTestArtifacts(t)

	g.Expect(len(nodes) > 0).Should(BeTrue())
	forwarder := nodes[0].Forwarder
	k8s.PrintImageVersion(forwarder)

}
