// +build dns

package nsmd_integration_tests

import (
	"testing"

	"github.com/onsi/gomega"

	"github.com/networkservicemesh/networkservicemesh/test/kubetest"
	"github.com/networkservicemesh/networkservicemesh/test/kubetest/pods"
)

//
//func TestBasicDns(t *testing.T) {
//	if testing.Short() {
//		t.Skip("Skip, please run without -short")
//		return
//	}
//	assert := gomega.NewWithT(t)
//	gomega.RegisterTestingT(t)
//	k8s, err := kubetest.NewK8s(assert, true)
//	assert.Expect(err).To(gomega.BeNil())
//	defer k8s.Cleanup()
//
//	configs, err := kubetest.SetupNodesConfig(k8s, 1, defaultTimeout, []*pods.NSMgrPodConfig{}, k8s.GetK8sNamespace())
//	assert.Expect(err).To(gomega.BeNil())
//	defer kubetest.MakeLogsSnapshot(k8s, t)
//	err = kubetest.DeployCorefile(k8s, "basic-corefile", `. {
//    log
//    hosts {
//        172.16.1.2 my.app
//    }
//}`)
//
//	assert.Expect(err).Should(gomega.BeNil())
//	kubetest.DeployICMP(k8s, configs[0].Node, "icmp-responder-nse", defaultTimeout)
//	nsc := kubetest.DeployNscAndNsmCoredns(k8s, configs[0].Node, "nsc", "basic-corefile", defaultTimeout)
//	assert.Expect(kubetest.PingByHostName(k8s, nsc, "my.app")).Should(gomega.BeTrue())
//}

func TestRepeat1(t *testing.T) {
	for i := 0; i < 15; i++ {
		TestHypothesis1(t)
	}
}

func TestRepeat3(t *testing.T) {
	for i := 0; i < 15; i++ {
		TestHypothesis3(t)
	}
}

func TestRepeat4(t *testing.T) {
	for i := 0; i < 15; i++ {
		TestHypothesis4(t)
	}
}

func TestHypothesis1(t *testing.T) {
	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}
	assert := gomega.NewWithT(t)

	k8s, err := kubetest.NewK8s(assert, true)
	assert.Expect(err).Should(gomega.BeNil())
	defer k8s.Cleanup()
	defer kubetest.MakeLogsSnapshot(k8s, t)

	nseCorefileContent := `icmp.app {
    hosts {
        172.16.1.2 icmp.app
    }
}`
	err = kubetest.DeployCorefile(k8s, "icmp-responder-corefile", nseCorefileContent)
	assert.Expect(err).Should(gomega.BeNil())

	configs, err := kubetest.SetupNodes(k8s, 1, defaultTimeout)
	assert.Expect(err).To(gomega.BeNil())

	kubetest.DeployICMPAndCoredns(k8s, configs[0].Node, "icmp-responder", "icmp-responder-corefile", defaultTimeout)
	nsc := kubetest.DeployMonitoringNSCAndCoredns(k8s, configs[0].Node, "nsc", defaultTimeout)
	assert.Expect(kubetest.PingByHostName(k8s, nsc, "icmp.app")).Should(gomega.BeTrue())
}

func TestHypothesis3(t *testing.T) {
	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}
	assert := gomega.NewWithT(t)
	gomega.RegisterTestingT(t)
	k8s, err := kubetest.NewK8s(assert, true)
	assert.Expect(err).To(gomega.BeNil())
	defer k8s.Cleanup()

	configs, err := kubetest.SetupNodesConfig(k8s, 1, defaultTimeout, []*pods.NSMgrPodConfig{}, k8s.GetK8sNamespace())
	assert.Expect(err).To(gomega.BeNil())
	defer kubetest.MakeLogsSnapshot(k8s, t)
	err = kubetest.DeployCorefile(k8s, "icmp-responder-corefile", `. {
   log
   hosts {
	   172.16.1.2 my.app
   }
}`)
	assert.Expect(err).Should(gomega.BeNil())
	err = kubetest.DeployCorefile(k8s, "basic-corefile", `. {
	fanout 172.16.1.2
}`)

	assert.Expect(err).Should(gomega.BeNil())
	kubetest.DeployICMPAndCoredns(k8s, configs[0].Node, "icmp-responder-nse", "icmp-responder-corefile", defaultTimeout)
	nsc := kubetest.DeployNscAndNsmCoredns(k8s, configs[0].Node, "nsc", "basic-corefile", defaultTimeout)
	assert.Expect(kubetest.PingByHostName(k8s, nsc, "my.app")).Should(gomega.BeTrue())
}

func TestHypothesis4(t *testing.T) {
	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}
	assert := gomega.NewWithT(t)
	gomega.RegisterTestingT(t)
	k8s, err := kubetest.NewK8s(assert, true)
	assert.Expect(err).To(gomega.BeNil())
	defer k8s.Cleanup()

	configs, err := kubetest.SetupNodesConfig(k8s, 1, defaultTimeout, []*pods.NSMgrPodConfig{}, k8s.GetK8sNamespace())
	assert.Expect(err).To(gomega.BeNil())
	defer kubetest.MakeLogsSnapshot(k8s, t)
	err = kubetest.DeployCorefile(k8s, "icmp-responder-corefile", `. {
   hosts {
	   172.16.1.2 my.app
   }
}`)
	assert.Expect(err).Should(gomega.BeNil())
	err = kubetest.DeployCorefile(k8s, "basic-corefile", `. {
	fanout 10.96.0.10 172.16.1.2
}`)

	assert.Expect(err).Should(gomega.BeNil())
	kubetest.DeployICMPAndCoredns(k8s, configs[0].Node, "icmp-responder-nse", "icmp-responder-corefile", defaultTimeout)
	nsc := kubetest.DeployNscAndNsmCoredns(k8s, configs[0].Node, "nsc", "basic-corefile", defaultTimeout)
	assert.Expect(kubetest.PingByHostName(k8s, nsc, "my.app")).Should(gomega.BeTrue())
}

//func TestNsmCorednsNotBreakDefaultK8sDNS(t *testing.T) {
//	if testing.Short() {
//		t.Skip("Skip, please run without -short")
//		return
//	}
//	assert := gomega.NewWithT(t)
//	k8s, err := kubetest.NewK8s(assert, true)
//	assert.Expect(err).Should(gomega.BeNil())
//	defer k8s.Cleanup()
//	configs, err := kubetest.SetupNodes(k8s, 1, defaultTimeout)
//	assert.Expect(err).To(gomega.BeNil())
//	defer kubetest.MakeLogsSnapshot(k8s, t)
//	kubetest.DeployICMP(k8s, configs[0].Node, "icmp-responder", defaultTimeout)
//	nsc := kubetest.DeployMonitoringNSCAndCoredns(k8s, configs[0].Node, "nsc", defaultTimeout)
//	assert.Expect(kubetest.NSLookup(k8s, nsc, "kubernetes.default")).Should(gomega.BeTrue())
//}
