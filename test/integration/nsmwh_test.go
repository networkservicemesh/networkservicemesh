// +build performance

package nsmd_integration_tests

import (
	"testing"
	"time"

	"github.com/onsi/gomega"
	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/test/kubetest"
)

func TestSimpleDeploy(t *testing.T) {
	assert := gomega.NewWithT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	k8s, err := kubetest.NewK8s(assert, true)

	defer k8s.Cleanup()
	defer kubetest.MakeLogsSnapshot(k8s, t)

	configs, err := kubetest.SetupNodes_(k8s, 1, defaultTimeout)
	assert.Expect(err).To(gomega.BeNil())
	configs = configs
	clearResources := kubetest.DeployNSMWH(k8s, configs[0].Node, defaultTimeout)
	defer clearResources()

	now := time.Now()
	kubetest.DeployICMP(k8s, configs[0].Node, "icmp-responder", defaultTimeout)
	nsc := kubetest.DeployNSC(k8s, configs[0].Node, "vppagent-nsc", defaultTimeout)
	elapsed := now.Sub(time.Now())
	logrus.Infof("Elapsed time for creating nsc/nse %v", elapsed)
	assert.Expect(true, kubetest.IsNsePinged(k8s, nsc))
}
