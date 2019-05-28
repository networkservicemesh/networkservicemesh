// +build recover

package nsmd_integration_tests

import (
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/test/kubetest"
)

func TestUpdateConnectionOnNSEChange(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	k8s, err := kubetest.NewK8s(true)
	Expect(err).To(BeNil())
	defer k8s.Cleanup()

	nodesConf, err := kubetest.SetupNodes(k8s, 2, defaultTimeout)
	Expect(err).To(BeNil())
	defer kubetest.FailLogger(k8s, nodesConf, t)

	nse1 := kubetest.DeployICMP(k8s, nodesConf[0].Node, "icmp-responder-nse-1", defaultTimeout)

	nsc := kubetest.DeployMonitoringNSC(k8s, nodesConf[0].Node, "monitoring-nsc", defaultTimeout)
	k8s.WaitLogsContains(nsc, "monitoring-nsc", "Monitor started", defaultTimeout)

	time.Sleep(5 * time.Second) // we need to be sure that NSC is updated only with the initial connection

	l, err := k8s.GetLogs(nsc, "monitoring-nsc")
	Expect(err).To(BeNil())
	Expect(l).NotTo(ContainSubstring("Connection updated"))

	kubetest.DeployICMP(k8s, nodesConf[1].Node, "icmp-responder-nse-2", defaultTimeout)
	k8s.DeletePods(nse1)

	logrus.Info("Waiting for NSMD to heal broken NSE connection")
	k8s.WaitLogsContains(nodesConf[0].Nsmd, "nsmd", "Heal: Connection recovered:", defaultTimeout)

	logrus.Infof("Waiting for NSMD to update NSC connection")
	k8s.WaitLogsContains(nsc, "monitoring-nsc", "Connection updated", defaultTimeout)
}
