// +build basic_suite

package nsmd_integration_tests

import (
	"testing"

	"github.com/onsi/gomega"

	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/test/kubetest"
	"github.com/networkservicemesh/networkservicemesh/test/kubetest/pods"
)

func TestBasicDns(t *testing.T) {
	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}
	assert := gomega.NewWithT(t)
	gomega.RegisterTestingT(t)
	k8s, err := kubetest.NewK8s(assert, kubetest.ReuseNSMResources)
	assert.Expect(err).To(gomega.BeNil())
	defer k8s.Cleanup()

	configs, err := kubetest.SetupNodesConfig(k8s, 1, defaultTimeout, []*pods.NSMgrPodConfig{}, k8s.GetK8sNamespace())
	assert.Expect(err).To(gomega.BeNil())
	defer k8s.ProcessArtifacts(t)
	err = kubetest.DeployCorefile(k8s, "basic-corefile", `. {
    log
    hosts {
		no_recursive
        172.16.1.2 my.app
    }
}`)

	assert.Expect(err).Should(gomega.BeNil())
	kubetest.DeployICMP(k8s, configs[0].Node, "icmp-responder-nse", defaultTimeout)
	nsc := kubetest.DeployNscAndNsmCoredns(k8s, configs[0].Node, "nsc", "basic-corefile", defaultTimeout)
	assert.Expect(kubetest.PingByHostName(k8s, nsc, "my.app.")).Should(gomega.BeTrue())
}

func TestDNSMonitoringNsc(t *testing.T) {
	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}
	assert := gomega.NewWithT(t)

	k8s, err := kubetest.NewK8s(assert, kubetest.ReuseNSMResources)
	assert.Expect(err).Should(gomega.BeNil())
	defer k8s.Cleanup()

	nseCorefileContent := `. {
    hosts {
		no_recursive
        172.16.1.2 icmp.app
    }
}`
	err = kubetest.DeployCorefile(k8s, "icmp-responder-corefile", nseCorefileContent)
	assert.Expect(err).Should(gomega.BeNil())

	configs, err := kubetest.SetupNodes(k8s, 1, defaultTimeout)
	assert.Expect(err).To(gomega.BeNil())
	defer k8s.ProcessArtifacts(t)

	kubetest.DeployICMPAndCoredns(k8s, configs[0].Node, "icmp-responder", "icmp-responder-corefile", defaultTimeout)
	nsc := kubetest.DeployMonitoringNSCAndCoredns(k8s, configs[0].Node, "nsc", defaultTimeout)
	assert.Expect(kubetest.PingByHostName(k8s, nsc, "icmp.app.")).Should(gomega.BeTrue())
}

func TestDNSExternalClient(t *testing.T) {
	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}
	assert := gomega.NewWithT(t)
	k8s, err := kubetest.NewK8s(assert, kubetest.ReuseNSMResources)
	defer k8s.Cleanup()
	defer k8s.ProcessArtifacts(t)
	assert.Expect(err).To(gomega.BeNil())

	deleteWebhook := func() func() {
		awc, awDeployment, awService := kubetest.DeployAdmissionWebhook(k8s, "nsm-admission-webhook", "networkservicemesh/admission-webhook", k8s.GetK8sNamespace(), defaultTimeout)
		return func() {
			kubetest.DeleteAdmissionWebhook(k8s, "nsm-admission-webhook-certs", awc, awDeployment, awService, k8s.GetK8sNamespace())
		}
	}()
	defer deleteWebhook()
	coreFile := `. {
    hosts {
		no_recursive
        172.16.1.2 icmp.app
    }
}`
	err = kubetest.DeployCorefile(k8s, "icmp-responder-corefile", coreFile)
	assert.Expect(err).Should(gomega.BeNil())
	nodes, err := kubetest.SetupNodes(k8s, 1, defaultTimeout)
	assert.Expect(err).To(gomega.BeNil())

	kubetest.DeployICMPAndCoredns(k8s, nodes[0].Node, "icmp-responder", "icmp-responder-corefile", defaultTimeout)
	nsc := kubetest.DeployNSCWebhook(k8s, nodes[0].Node, "nsc-1", defaultTimeout)
	assert.Expect(kubetest.PingByHostName(k8s, nsc, "icmp.app.")).Should(gomega.BeTrue())
}

func TestNsmCorednsNotBreakDefaultK8sDNS(t *testing.T) {
	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}
	assert := gomega.NewWithT(t)
	k8s, err := kubetest.NewK8s(assert, kubetest.ReuseNSMResources)
	assert.Expect(err).Should(gomega.BeNil())
	defer k8s.Cleanup()
	configs, err := kubetest.SetupNodes(k8s, 1, defaultTimeout)
	assert.Expect(err).To(gomega.BeNil())
	defer k8s.ProcessArtifacts(t)

	nse := kubetest.DeployICMP(k8s, configs[0].Node, "icmp-responder", defaultTimeout)
	nsc := kubetest.DeployMonitoringNSCAndCoredns(k8s, configs[0].Node, "nsc", defaultTimeout)

	if !kubetest.NSLookup(k8s, nse, "kubernetes.default") {
		logrus.Info("Cluster DNS not works")
		t.SkipNow()
	}
	assert.Expect(kubetest.NSLookup(k8s, nsc, "kubernetes.default")).Should(gomega.BeTrue())
}
