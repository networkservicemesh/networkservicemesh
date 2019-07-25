// +build basic

package nsmd_integration_tests

import (
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/crossconnect"
	"github.com/networkservicemesh/networkservicemesh/test/kubetest"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	"testing"
)

func TestSingleCrossConnect(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	k8s, err := kubetest.NewK8s(true)
	defer k8s.Cleanup()
	Expect(err).To(BeNil())

	nodesCount := 2

	nodes, err := kubetest.SetupNodes(k8s, nodesCount, defaultTimeout)
	Expect(err).To(BeNil())
	defer kubetest.ShowLogs(k8s, t)
	kubetest.DeployICMP(k8s, nodes[nodesCount-1].Node, "icmp-responder-nse-1", defaultTimeout)
	kubetest.DeployNSC(k8s, nodes[0].Node, "nsc-1", defaultTimeout)

	// monitor client for node0
	eventCh0, closeFunc0 := kubetest.XconProxyMonitor(k8s, nodes[0], "0")
	defer closeFunc0()

	// monitor client for node1
	eventCh1, closeFunc1 := kubetest.XconProxyMonitor(k8s, nodes[1], "1")
	defer closeFunc1()

	// checking goroutine for node0
	expectedFunc0, waitFunc0 := kubetest.NewEventChecker(t, eventCh0)

	// checking goroutine for node1
	expectedFunc1, waitFunc1 := kubetest.NewEventChecker(t, eventCh1)

	expectedFunc0(kubetest.EventDescription{
		EventType: crossconnect.CrossConnectEventType_INITIAL_STATE_TRANSFER,
		SrcUp:     true,
		DstUp:     true,
		LastEvent: true,
	})

	expectedFunc1(kubetest.EventDescription{
		EventType: crossconnect.CrossConnectEventType_INITIAL_STATE_TRANSFER,
		SrcUp:     true,
		DstUp:     true,
		LastEvent: true,
	})

	waitFunc0()
	waitFunc1()
}

func TestSingleCrossConnectMonitorBeforeXcons(t *testing.T) {
	if !kubetest.IsBrokeTestsEnabled() {
		t.Skip("https://github.com/networkservicemesh/networkservicemesh/issues/1385")
	}
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	k8s, err := kubetest.NewK8s(true)
	defer k8s.Cleanup()
	Expect(err).To(BeNil())

	nodesCount := 2

	nodes, err := kubetest.SetupNodes(k8s, nodesCount, defaultTimeout)
	Expect(err).To(BeNil())
	defer kubetest.ShowLogs(k8s, t)

	// monitor client for node0
	eventCh0, closeFunc0 := kubetest.XconProxyMonitor(k8s, nodes[0], "0")
	defer closeFunc0()

	// monitor client for node1
	eventCh1, closeFunc1 := kubetest.XconProxyMonitor(k8s, nodes[1], "1")
	defer closeFunc1()

	kubetest.DeployICMP(k8s, nodes[nodesCount-1].Node, "icmp-responder-nse-1", defaultTimeout)
	kubetest.DeployNSC(k8s, nodes[0].Node, "nsc-1", defaultTimeout)

	_, err = kubetest.CollectXcons(eventCh0, 2, fastTimeout)
	Expect(err).To(BeNil())

	_, err = kubetest.CollectXcons(eventCh1, 2, fastTimeout)
	Expect(err).To(BeNil())
}

func TestSeveralCrossConnects(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	k8s, err := kubetest.NewK8s(true)
	defer k8s.Cleanup()
	Expect(err).To(BeNil())

	nodesCount := 2

	nodes, err := kubetest.SetupNodes(k8s, nodesCount, defaultTimeout)
	Expect(err).To(BeNil())
	defer kubetest.ShowLogs(k8s, t)
	kubetest.DeployICMP(k8s, nodes[nodesCount-1].Node, "icmp-responder-nse-1", defaultTimeout)
	kubetest.DeployNSC(k8s, nodes[0].Node, "nsc-1", defaultTimeout)
	kubetest.DeployNSC(k8s, nodes[0].Node, "nsc-2", defaultTimeout)

	// monitor client for node0
	eventCh0, closeFunc0 := kubetest.XconProxyMonitor(k8s, nodes[0], "0")
	defer closeFunc0()

	// monitor client for node1
	eventCh1, closeFunc1 := kubetest.XconProxyMonitor(k8s, nodes[1], "1")
	defer closeFunc1()

	_, err = kubetest.CollectXcons(eventCh0, 2, fastTimeout)
	Expect(err).To(BeNil())

	_, err = kubetest.CollectXcons(eventCh1, 2, fastTimeout)
	Expect(err).To(BeNil())
}

func TestCrossConnectMonitorRestart(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	k8s, err := kubetest.NewK8s(true)
	defer k8s.Cleanup()
	Expect(err).To(BeNil())

	nodesCount := 2

	nodes, err := kubetest.SetupNodes(k8s, nodesCount, defaultTimeout)
	Expect(err).To(BeNil())
	kubetest.DeployICMP(k8s, nodes[nodesCount-1].Node, "icmp-responder-nse-1", defaultTimeout)
	kubetest.DeployNSC(k8s, nodes[0].Node, "nsc-1", defaultTimeout)
	kubetest.DeployNSC(k8s, nodes[0].Node, "nsc-2", defaultTimeout)

	// monitor client for node0
	eventCh0, closeFunc0 := kubetest.XconProxyMonitor(k8s, nodes[0], "0")

	_, err = kubetest.CollectXcons(eventCh0, 2, fastTimeout)
	Expect(err).To(BeNil())
	closeFunc0()

	logrus.Info("Restarting monitor")

	eventCh1, closeFunc1 := kubetest.XconProxyMonitor(k8s, nodes[0], "0")
	defer closeFunc1()

	_, err = kubetest.CollectXcons(eventCh1, 2, fastTimeout)
	Expect(err).To(BeNil())
}
