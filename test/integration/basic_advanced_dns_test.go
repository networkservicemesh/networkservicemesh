// +build basic

package nsmd_integration_tests

import (
	"github.com/networkservicemesh/networkservicemesh/test/kubetest"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	"strings"
	"testing"
)

func TestAdvancedDNS(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	k8s, err := kubetest.NewK8s(true)
	Expect(err).Should(BeNil())
	defer k8s.Cleanup()

	coreFile := `. {
    hosts {
        172.16.1.1 my.domain1
        172.16.1.2 my.domain2
    }
}`
	kubetest.CreateCorednsConfig(k8s, "core", coreFile)

	nodes, err := kubetest.SetupNodes(k8s, 1, defaultTimeout)
	Expect(err).To(BeNil())
	defer kubetest.FailLogger(k8s, nodes, t)

	kubetest.DeployICMPDns(k8s, nodes[0].Node, "icmp-responder", "core", defaultTimeout)

	nscAndDns1 := kubetest.DeployMonitoringNSCDns(k8s, nodes[0].Node, "nsc", defaultTimeout)
	resp, _, err := k8s.Exec(nscAndDns1, "nsc", "ping", "my.domain1", "-c", "4")
	Expect(err).Should(BeNil())
	logrus.Info(resp)
	Expect(strings.TrimSpace(resp)).ShouldNot(BeEmpty())
	Expect(strings.Contains(resp, "bad")).Should(BeFalse())
	resp, _, err = k8s.Exec(nscAndDns1, "nsc", "ping", "my.domain2", "-c", "4")
	Expect(err).Should(BeNil())
	logrus.Info(resp)
	Expect(strings.TrimSpace(resp)).ShouldNot(BeEmpty())
	Expect(strings.Contains(resp, "bad")).Should(BeFalse())

}
