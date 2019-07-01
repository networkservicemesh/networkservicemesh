// +build usecase

package nsmd_integration_tests

import (
	"github.com/networkservicemesh/networkservicemesh/test/kubetest"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	"testing"
)

func TestCertSidecar(t *testing.T) {
	RegisterTestingT(t)

	k8s, err := kubetest.NewK8s(true)
	defer k8s.Cleanup()
	Expect(err).To(BeNil())

	nodesConf, err := kubetest.SetupNodes(k8s, 2, defaultTimeout)
	Expect(err).To(BeNil())

	kubetest.DeployICMP(k8s, nodesConf[1].Node, "icmp-responder-nse-1", defaultTimeout)
	nsc := kubetest.DeployNSC(k8s, nodesConf[0].Node, "nsc-1", defaultTimeout)

	logs, err := k8s.GetLogs(nsc, "nsm-init")
	logrus.Infof(logs)
	return
}
