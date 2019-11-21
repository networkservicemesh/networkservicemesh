// +build basic

package nsmd_integration_tests

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/test/kubetest"
)

func TestDeleteNSMCr(t *testing.T) {
	g := NewWithT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	logrus.Print("Running delete NSM Custom Resource test")

	k8s, err := kubetest.NewK8s(g, true)
	g.Expect(err).To(BeNil())
	defer k8s.Cleanup()

	nodes_setup, err := kubetest.SetupNodes(k8s, 1, defaultTimeout)
	g.Expect(err).To(BeNil())

	kubetest.ExpectNSMsCountToBe(k8s, 0, 1)

	logrus.Infof("Deleting NSMD")
	k8s.DeletePods(nodes_setup[0].Nsmd)

	kubetest.ExpectNSMsCountToBe(k8s, 1, 0)
}
