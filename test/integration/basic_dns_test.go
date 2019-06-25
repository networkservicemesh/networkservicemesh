// +build basic

package nsmd_integration_tests

import (
	"github.com/networkservicemesh/networkservicemesh/test/kubetest"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	"strings"
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
	defer kubetest.FailLogger(k8s, nodes, t)
	kubetest.CreateCorednsConfig(k8s, "nsc-dns-core-file", `. {
	log
	hosts {
		172.16.1.2 test
	}
}`)
	kubetest.DeployICMP(k8s, nodes[0].Node, "icmp-responder", defaultTimeout)
	nscAndDns := kubetest.DeployNSCDns(k8s, nodes[0].Node, "nsc", "nsc-dns-core-file", defaultTimeout)
	resp, _, err := k8s.Exec(nscAndDns, "alpine-img", "ping", "test", "-c", "4")
	Expect(err).Should(BeNil())
	logrus.Info(resp)
	Expect(strings.TrimSpace(resp)).ShouldNot(BeEmpty())
	Expect(strings.Contains(resp, "bad")).Should(BeFalse())
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
	defer kubetest.FailLogger(k8s, nodes, t)

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
	kubetest.DeployICMPDns(k8s, nodes[0].Node, "icmp-responder", "nse-dns-core-file", defaultTimeout)
	nscAndDns := kubetest.DeployNSCDns(k8s, nodes[0].Node, "nsc", "nsc-dns-core-file", defaultTimeout)
	resp, _, err := k8s.Exec(nscAndDns, "alpine-img", "ping", "my.google.com", "-c", "4")
	Expect(err).Should(BeNil())
	logrus.Info(resp)
	Expect(strings.TrimSpace(resp)).ShouldNot(BeEmpty())
	Expect(strings.Contains(resp, "bad")).Should(BeFalse())
}
