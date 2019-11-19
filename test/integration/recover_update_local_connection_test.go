// +build recover_suite

package nsmd_integration_tests

import (
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/test/kubetest"
)

func TestUpdateConnectionOnNSEChange(t *testing.T) {
	g := NewWithT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	k8s, err := kubetest.NewK8s(g, kubetest.ReuseNSMResources)
	g.Expect(err).To(BeNil())
	defer k8s.Cleanup()

	nodesConf, err := kubetest.SetupNodes(k8s, 2, defaultTimeout)
	g.Expect(err).To(BeNil())
	defer k8s.ProcessArtifacts(t)

	nse1 := kubetest.DeployICMP(k8s, nodesConf[0].Node, "icmp-responder-nse-1", defaultTimeout)

	nsc := kubetest.DeployMonitoringNSC(k8s, nodesConf[0].Node, "monitoring-nsc", defaultTimeout)
	k8s.WaitLogsContains(nsc, "monitoring-nsc", "Monitor started", defaultTimeout)

	<-time.After(5 * time.Second) // we need to be sure that NSC is updated only with the initial connection

	l, err := k8s.GetLogs(nsc, "monitoring-nsc")
	g.Expect(err).To(BeNil())
	g.Expect(l).NotTo(ContainSubstring("Connection updated"))

	kubetest.DeployICMP(k8s, nodesConf[1].Node, "icmp-responder-nse-2", defaultTimeout)
	k8s.DeletePods(nse1)

	logrus.Info("Waiting for NSMD to heal broken NSE connection")
	k8s.WaitLogsContains(nodesConf[0].Nsmd, "nsmd", "Heal: Connection recovered:", defaultTimeout)

	logrus.Infof("Waiting for NSMD to update NSC connection")
	k8s.WaitLogsContains(nsc, "monitoring-nsc", "Connection updated", defaultTimeout)
}

func TestUpdateConnectionOnNSEUpdate(t *testing.T) {
	g := NewWithT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	k8s, err := kubetest.NewK8s(g, kubetest.ReuseNSMResources)
	g.Expect(err).To(BeNil())
	defer k8s.Cleanup()

	nodesConf, err := kubetest.SetupNodes(k8s, 1, defaultTimeout)
	g.Expect(err).To(BeNil())
	defer k8s.ProcessArtifacts(t)

	kubetest.DeployUpdatingNSE(k8s, nodesConf[0].Node, "icmp-responder-nse", defaultTimeout)

	nsc := kubetest.DeployMonitoringNSC(k8s, nodesConf[0].Node, "monitoring-nsc", defaultTimeout)
	k8s.WaitLogsContains(nsc, "monitoring-nsc", "Monitor started", defaultTimeout)

	logrus.Info("Waiting for NSMD to heal updated NSE connection")
	k8s.WaitLogsContains(nodesConf[0].Nsmd, "nsmd", "Heal: Connection recovered:", defaultTimeout)

	logrus.Infof("Waiting for NSMD to update NSC connection")
	k8s.WaitLogsContains(nsc, "monitoring-nsc", "Connection updated", defaultTimeout)
}
