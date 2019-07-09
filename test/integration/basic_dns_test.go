// +build basic

package nsmd_integration_tests

import (
	"github.com/networkservicemesh/networkservicemesh/test/kubetest"
	. "github.com/onsi/gomega"
	"testing"
)

func TestBasicLocalDns(t *testing.T) {
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
	defer kubetest.ShowLogs(k8s, t)
	kubetest.CreateCorednsConfig(k8s, "nsc-dns-core-file", `. {
	log
	hosts {
		172.16.1.2 test
	}
}`)
	kubetest.DeployICMP(k8s, nodes[0].Node, "icmp-responder", defaultTimeout)
	nsc := kubetest.DeployNSCAndCoredns(k8s, nodes[0].Node, "nsc", "nsc-dns-core-file", defaultTimeout)
	Expect(kubetest.PingByHostName(k8s, nsc, "test")).Should(BeTrue())
}

func TestBasicProxyDns(t *testing.T) {
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
	defer kubetest.ShowLogs(k8s, t)

	kubetest.CreateCorednsConfig(k8s, "nsc-dns-core-file", `. {
	log
	forward . 172.16.1.2:53
}`)
	kubetest.CreateCorednsConfig(k8s, "nse-dns-core-file", `.:53 {
	log
	hosts {
		172.16.1.1 my.google.com
	}
}`)
	kubetest.DeployICMPAndCoredns(k8s, nodes[0].Node, "icmp-responder", "nse-dns-core-file", defaultTimeout)
	nsc := kubetest.DeployNSCAndCoredns(k8s, nodes[0].Node, "nsc", "nsc-dns-core-file", defaultTimeout)
	Expect(kubetest.PingByHostName(k8s, nsc, "my.google.com")).Should(BeTrue())
}
