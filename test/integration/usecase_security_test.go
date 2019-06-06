// +build usecase
package nsmd_integration_tests

import (
	"github.com/networkservicemesh/networkservicemesh/test/kubetest"
	. "github.com/onsi/gomega"
	"testing"
)

func TestCertSidecar(t *testing.T) {
	RegisterTestingT(t)

	k8s, err := kubetest.NewK8s(true)
	Expect(err).To(BeNil())
	defer k8s.Cleanup()

	nodesConf, err := kubetest.SetupNodes(k8s, 1, defaultTimeout)
	Expect(err).To(BeNil())

	nsc := kubetest.DeployNSC(k8s, nodesConf[0].Node, "nsc-1", defaultTimeout)

}
