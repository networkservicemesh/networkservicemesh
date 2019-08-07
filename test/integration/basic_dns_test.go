// +build basic

package nsmd_integration_tests

import (
	"testing"

	"github.com/onsi/gomega"

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
	k8s, err := kubetest.NewK8s(assert, true)
	assert.Expect(err).To(gomega.BeNil())
	defer k8s.Cleanup()

	configs, err := kubetest.SetupNodesConfig(k8s, 1, defaultTimeout, []*pods.NSMgrPodConfig{}, k8s.GetK8sNamespace())
	assert.Expect(err).To(gomega.BeNil())
	defer kubetest.MakeLogsSnapshot(k8s, t)
	err = kubetest.DeployCorefile(k8s, "basic-corefile", `. {
    log
    hosts {
        172.16.1.2 my.app
    }
}`)

	assert.Expect(err).Should(gomega.BeNil())
	kubetest.DeployICMP(k8s, configs[0].Node, "icmp-responder-nse", defaultTimeout)
	nsc := kubetest.DeployNscAndNsmCoredns(k8s, configs[0].Node, "nsc", "basic-corefile", defaultTimeout)
	assert.Expect(kubetest.PingByHostName(k8s, nsc, "my.app")).Should(gomega.BeTrue())
}
