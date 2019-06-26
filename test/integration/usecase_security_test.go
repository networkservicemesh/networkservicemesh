// +build usecase

package nsmd_integration_tests

import (
	"bufio"
	"github.com/networkservicemesh/networkservicemesh/test/kubetest"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	"os"
	"testing"
)

func TestCertSidecar(t *testing.T) {
	RegisterTestingT(t)

	k8s, err := kubetest.NewK8s(true)
	defer k8s.Cleanup()
	Expect(err).To(BeNil())

	nodesConf, err := kubetest.SetupNodes(k8s, 1, defaultTimeout)
	Expect(err).To(BeNil())

	kubetest.DeployICMP(k8s, nodesConf[0].Node, "icmp-responder-nse-1", defaultTimeout)
	kubetest.DeployNSC(k8s, nodesConf[0].Node, "nsc-1", defaultTimeout)
	return
}

func TestReader(t *testing.T) {
	RegisterTestingT(t)

	writer := bufio.NewWri
	reader := bufio.NewReader
	for {
		str, err := reader.ReadString('\n')
		if err != nil {
			logrus.Error(err)
			return
		}
		logrus.Info(str)
	}
}
