// +build basic

package nsmd_integration_tests

import (
	"github.com/networkservicemesh/networkservicemesh/test/kubetest"
	. "github.com/onsi/gomega"
	"testing"
)

func TestAdvancedDNSLocal(t *testing.T) {
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
	defer kubetest.ShowLogs(k8s, t)

	kubetest.DeployICMPAndCoredns(k8s, nodes[0].Node, "icmp-responder", "core", defaultTimeout)

	nsc := kubetest.DeployMonitoringNSCAndCoredns(k8s, nodes[0].Node, "nsc", defaultTimeout)
	Expect(kubetest.PingByHostName(k8s, nsc, "my.domain1")).Should(BeTrue())
	Expect(kubetest.PingByHostName(k8s, nsc, "my.domain2")).Should(BeTrue())
}

func TestAdvancedDNSRemote(t *testing.T) {
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

	nodes, err := kubetest.SetupNodes(k8s, 2, defaultTimeout)
	Expect(err).To(BeNil())
	defer kubetest.ShowLogs(k8s, t)

	kubetest.DeployICMPAndCoredns(k8s, nodes[1].Node, "icmp-responder", "core", defaultTimeout)

	nsc := kubetest.DeployMonitoringNSCAndCoredns(k8s, nodes[0].Node, "nsc", defaultTimeout)
	Expect(kubetest.PingByHostName(k8s, nsc, "my.domain1")).Should(BeTrue())
	Expect(kubetest.PingByHostName(k8s, nsc, "my.domain2")).Should(BeTrue())

}

