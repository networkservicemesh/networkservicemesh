// +build basic_suite

package nsmd_integration_tests

import (
	"testing"

	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/crossconnect"
	"github.com/networkservicemesh/networkservicemesh/test/kubetest"
)

func TestSingleCrossConnect(t *testing.T) {
	g := NewWithT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	k8s, err := kubetest.NewK8s(g, kubetest.ReuseNSMResources)
	defer k8s.Cleanup()
	g.Expect(err).To(BeNil())

	nodesCount := 2

	nodes, err := kubetest.SetupNodes(k8s, nodesCount, defaultTimeout)
	g.Expect(err).To(BeNil())
	defer k8s.ProcessArtifacts(t)
	kubetest.DeployICMP(k8s, nodes[nodesCount-1].Node, "icmp-responder-nse-1", defaultTimeout)
	kubetest.DeployNSC(k8s, nodes[0].Node, "nsc-1", defaultTimeout)

	// monitor client for node0
	eventCh0, closeFunc0 := kubetest.CrossConnectClientAt(k8s, nodes[0].Nsmd)
	defer closeFunc0()

	// monitor client for node1
	eventCh1, closeFunc1 := kubetest.CrossConnectClientAt(k8s, nodes[1].Nsmd)
	defer closeFunc1()

	// checking goroutine for node0
	expectedFunc0, waitFunc0 := kubetest.NewEventChecker(t, eventCh0)

	// checking goroutine for node1
	expectedFunc1, waitFunc1 := kubetest.NewEventChecker(t, eventCh1)

	expectedFunc0(&kubetest.SingleEventChecker{
		EventType: crossconnect.CrossConnectEventType_INITIAL_STATE_TRANSFER,
		SrcUp:     true,
		DstUp:     true,
	})

	expectedFunc1(&kubetest.SingleEventChecker{
		EventType: crossconnect.CrossConnectEventType_INITIAL_STATE_TRANSFER,
		SrcUp:     true,
		DstUp:     true,
	})

	waitFunc0()
	waitFunc1()
}

func TestSingleCrossConnectMonitorBeforeXcons(t *testing.T) {
	g := NewWithT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	k8s, err := kubetest.NewK8s(g, kubetest.ReuseNSMResources)
	defer k8s.Cleanup()
	g.Expect(err).To(BeNil())

	nodesCount := 2

	nodes, err := kubetest.SetupNodes(k8s, nodesCount, defaultTimeout)
	g.Expect(err).To(BeNil())
	defer k8s.ProcessArtifacts(t)

	// monitor client for node0
	eventCh0, closeFunc0 := kubetest.CrossConnectClientAt(k8s, nodes[0].Nsmd)
	defer closeFunc0()

	// monitor client for node1
	eventCh1, closeFunc1 := kubetest.CrossConnectClientAt(k8s, nodes[1].Nsmd)
	defer closeFunc1()

	kubetest.DeployICMP(k8s, nodes[nodesCount-1].Node, "icmp-responder-nse-1", defaultTimeout)
	kubetest.DeployNSC(k8s, nodes[0].Node, "nsc-1", defaultTimeout)

	_, err = kubetest.CollectXcons(eventCh0, 1, fastTimeout)
	g.Expect(err).To(BeNil())

	_, err = kubetest.CollectXcons(eventCh1, 1, fastTimeout)
	g.Expect(err).To(BeNil())
}

func TestSeveralCrossConnects(t *testing.T) {
	g := NewWithT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	k8s, err := kubetest.NewK8s(g, kubetest.ReuseNSMResources)
	defer k8s.Cleanup()
	g.Expect(err).To(BeNil())

	nodesCount := 2

	nodes, err := kubetest.SetupNodes(k8s, nodesCount, defaultTimeout)
	g.Expect(err).To(BeNil())
	defer k8s.ProcessArtifacts(t)
	kubetest.DeployICMP(k8s, nodes[nodesCount-1].Node, "icmp-responder-nse-1", defaultTimeout)
	kubetest.DeployNSC(k8s, nodes[0].Node, "nsc-1", defaultTimeout)
	kubetest.DeployNSC(k8s, nodes[0].Node, "nsc-2", defaultTimeout)

	// monitor client for node0
	eventCh0, closeFunc0 := kubetest.CrossConnectClientAt(k8s, nodes[0].Nsmd)
	defer closeFunc0()

	// monitor client for node1
	eventCh1, closeFunc1 := kubetest.CrossConnectClientAt(k8s, nodes[1].Nsmd)
	defer closeFunc1()

	_, err = kubetest.CollectXcons(eventCh0, 2, fastTimeout)
	g.Expect(err).To(BeNil())

	_, err = kubetest.CollectXcons(eventCh1, 2, fastTimeout)
	g.Expect(err).To(BeNil())
}

func TestCrossConnectMonitorRestart(t *testing.T) {
	g := NewWithT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	k8s, err := kubetest.NewK8s(g, kubetest.ReuseNSMResources)
	defer k8s.Cleanup()
	g.Expect(err).To(BeNil())

	nodesCount := 2

	nodes, err := kubetest.SetupNodes(k8s, nodesCount, defaultTimeout)
	g.Expect(err).To(BeNil())
	kubetest.DeployICMP(k8s, nodes[nodesCount-1].Node, "icmp-responder-nse-1", defaultTimeout)
	kubetest.DeployNSC(k8s, nodes[0].Node, "nsc-1", defaultTimeout)
	kubetest.DeployNSC(k8s, nodes[0].Node, "nsc-2", defaultTimeout)

	// monitor client for node0
	eventCh0, closeFunc0 := kubetest.CrossConnectClientAt(k8s, nodes[0].Nsmd)

	_, err = kubetest.CollectXcons(eventCh0, 2, fastTimeout)
	g.Expect(err).To(BeNil())
	closeFunc0()

	logrus.Info("Restarting monitor")

	eventCh1, closeFunc1 := kubetest.CrossConnectClientAt(k8s, nodes[0].Nsmd)
	defer closeFunc1()

	_, err = kubetest.CollectXcons(eventCh1, 2, fastTimeout)
	g.Expect(err).To(BeNil())
}
