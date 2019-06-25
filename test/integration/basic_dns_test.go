// +build basic

package nsmd_integration_tests

import (
	"github.com/networkservicemesh/networkservicemesh/test/kubetest"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	"strings"
	"testing"
)

func TestBasicDNS(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	k8s, err := kubetest.NewK8s(true)
	Expect(err).Should(BeNil())
	defer k8s.Cleanup()

	nodes, err := kubetest.SetupNodes(k8s, 1, defaultTimeout)
	Expect(err).To(BeNil())
	defer kubetest.FailLogger(k8s, nodes, t)

	kubetest.DeployICMP(k8s, nodes[0].Node, "icmp-responder", defaultTimeout)
	nscAndDns := kubetest.DeployMonitoringNSCDns(k8s, nodes[0].Node, "nsc", defaultTimeout)
	resp, _, err := k8s.Exec(nscAndDns, "nsc", "ping", "test", "-c", "4")
	Expect(err).Should(BeNil())
	logrus.Info(resp)
	Expect(strings.TrimSpace(resp)).ShouldNot(BeEmpty())
	Expect(strings.Contains(resp, "bad")).Should(BeFalse())
}
