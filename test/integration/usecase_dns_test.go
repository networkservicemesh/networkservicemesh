// +build usecase

package nsmd_integration_tests

import (
	"github.com/networkservicemesh/networkservicemesh/test/kubetest"
	. "github.com/onsi/gomega"
	"testing"
)

func TestAlpineAndICMPwithDNS(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}
	k8s, err := kubetest.NewK8s(true)
	defer k8s.Cleanup()
	defer kubetest.ShowLogs(k8s, t)
	Expect(err).To(BeNil())

	f := func() func() {
		awc, awDeployment, awService := kubetest.DeployAdmissionWebhook(k8s, "nsm-admission-webhook", "networkservicemesh/admission-webhook", k8s.GetK8sNamespace())
		return func() {
			kubetest.DeleteAdmissionWebhook(k8s, "nsm-admission-webhook-certs", awc, awDeployment, awService, k8s.GetK8sNamespace())
		}
	}()
	defer f()
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
	nsc := kubetest.DeployNSCWebhook(k8s, nodes[0].Node, "nsc-1", defaultTimeout)
	Expect(kubetest.PingByHostName(k8s, nsc, "my.domain1")).Should(BeTrue())
	Expect(kubetest.PingByHostName(k8s, nsc, "my.domain2")).Should(BeTrue())

}
