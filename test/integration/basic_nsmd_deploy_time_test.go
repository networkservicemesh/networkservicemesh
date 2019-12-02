// +build single_cluster_suite

package nsmd_integration_tests

import (
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/test/kubetest"
)

func TestNSMDDeploy(t *testing.T) {
	g := NewWithT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	logrus.Print("Running NSMgr Deploy test")

	k8s, err := kubetest.NewK8s(g, kubetest.DefaultClear)
	g.Expect(err).To(BeNil())

	// Warmup
	k8s, err = kubetest.NewK8s(g, kubetest.DefaultClear)
	defer k8s.Cleanup()
	defer k8s.ProcessArtifacts(t)
	var node *kubetest.NodeConf
	deploy := measureTime(func() {
		nodes, setupErr := kubetest.SetupNodes(k8s, 1, defaultTimeout)
		node, err = nodes[0], setupErr
	})
	g.Expect(err).To(BeNil())
	k8s.DescribePod(node.Nsmd)
	k8s.DescribePod(node.Forwarder)
	nsmdPullingImageTime := k8s.GetPullingImagesDuration(node.Nsmd)
	forwarderPullingImageTime := k8s.GetPullingImagesDuration(node.Forwarder)
	logrus.Infof("NSMD pulling image time: %v", nsmdPullingImageTime)
	logrus.Infof("VPPAgent Forwarder pulling image time: %v", forwarderPullingImageTime)
	deploy -= nsmdPullingImageTime + forwarderPullingImageTime
	destroy := measureTime(k8s.Cleanup)

	logrus.Infof("Pods deploy time: %v", deploy)
	g.Expect(deploy < time.Second*60).To(Equal(true))
	logrus.Infof("Pods Cleanup time: %v", destroy)
	g.Expect(destroy < time.Second*60).To(Equal(true))
}

func measureTime(f func()) time.Duration {
	t := time.Now()
	f()
	return time.Now().Sub(t)
}
