// +build basic

package nsmd_integration_tests

import (
	"fmt"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/test/kubetest"
	"github.com/networkservicemesh/networkservicemesh/test/kubetest/pods"
)

func TestDeleteDirtyNSE(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	logrus.Print("Running delete dirty NSE test")

	k8s, err := kubetest.NewK8s(true)
	Expect(err).To(BeNil())
	defer k8s.Cleanup()

	nodesConf, err := kubetest.SetupNodesConfig(k8s, 1, defaultTimeout, []*pods.NSMgrPodConfig{}, k8s.GetK8sNamespace())
	Expect(err).To(BeNil())
	defer kubetest.FailLogger(k8s, nodesConf, t)

	nsePod := kubetest.DeployDirtyICMP(k8s, nodesConf[0].Node, "dirty-icmp-responder-nse", defaultTimeout)

	expectNSEs(k8s, 1)

	k8s.DeletePods(nsePod)

	expectNSEs(k8s, 0)
}

func TestDeleteDirtyNSEWithClient(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	logrus.Print("Running delete dirty NSE with client test")

	k8s, err := kubetest.NewK8s(true)
	Expect(err).To(BeNil())
	defer k8s.Cleanup()

	nodesConf, err := kubetest.SetupNodesConfig(k8s, 1, defaultTimeout, []*pods.NSMgrPodConfig{}, k8s.GetK8sNamespace())
	Expect(err).To(BeNil())
	defer kubetest.FailLogger(k8s, nodesConf, t)

	nsePod := kubetest.DeployDirtyICMP(k8s, nodesConf[0].Node, "dirty-icmp-responder-nse", defaultTimeout)
	kubetest.DeployNSC(k8s, nodesConf[0].Node, "nsc-1", defaultTimeout)

	expectNSEs(k8s, 1)

	k8s.DeletePods(nsePod)

	time.Sleep(10 * time.Second) // we need to be sure that NSE with client is not getting deleted

	nses, err := k8s.GetNSEs()

	Expect(err).To(BeNil())
	Expect(len(nses)).To(Equal(1), fmt.Sprint(nses))
}

func expectNSEs(k8s *kubetest.K8s, count int) {
	for i := 0; i < 10; i++ {
		if nses, err := k8s.GetNSEs(); err == nil && len(nses) == count {
			break
		}
		<-time.Tick(1 * time.Second)
	}

	nses, err := k8s.GetNSEs()

	Expect(err).To(BeNil())
	Expect(len(nses)).To(Equal(count), fmt.Sprint(nses))
}
