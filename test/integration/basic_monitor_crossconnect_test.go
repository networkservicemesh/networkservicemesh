// +build basic

package nsmd_integration_tests

import (
	"fmt"
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
	kubetest.DeployICMP(k8s, nodes[nodesCount-1].Node, "icmp-responder-nse-1", defaultTimeout)
	kubetest.DeployNSC(k8s, nodes[0].Node, "nsc-1", defaultTimeout)

	fwd, err := k8s.NewPortForwarder(nodes[0].Nsmd, 5001)
	Expect(err).To(BeNil())
	defer fwd.Stop()

	err = fwd.Start()
	Expect(err).To(BeNil())

	fwd2, err := k8s.NewPortForwarder(nodes[1].Nsmd, 5001)
	Expect(err).To(BeNil())
	defer fwd2.Stop()

	err = fwd2.Start()
	Expect(err).To(BeNil())

	nsmdMonitor1, close1, cancel1 := kubetest.CreateCrossConnectClient(fmt.Sprintf("localhost:%d", fwd.ListenPort))
	defer close1()
	nsmdMonitor2, close2, cancel2 := kubetest.CreateCrossConnectClient(fmt.Sprintf("localhost:%d", fwd2.ListenPort))
	defer close2()

	_, err = kubetest.GetCrossConnectsFromMonitor(nsmdMonitor1, cancel1, 1, fastTimeout)
	Expect(err).To(BeNil())
	_, err = kubetest.GetCrossConnectsFromMonitor(nsmdMonitor2, cancel2, 1, fastTimeout)
	Expect(err).To(BeNil())
}

func TestSingleCrossConnectMonitorBeforeXcons(t *testing.T) {
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

	fwd, err := k8s.NewPortForwarder(nodes[0].Nsmd, 5001)
	Expect(err).To(BeNil())
	defer fwd.Stop()

	err = fwd.Start()
	Expect(err).To(BeNil())

	fwd2, err := k8s.NewPortForwarder(nodes[1].Nsmd, 5001)
	Expect(err).To(BeNil())
	defer fwd2.Stop()

	err = fwd2.Start()
	Expect(err).To(BeNil())

	nsmdMonitor1, close1, cancel1 := kubetest.CreateCrossConnectClient(fmt.Sprintf("localhost:%d", fwd.ListenPort))
	defer close1()
	nsmdMonitor2, close2, cancel2 := kubetest.CreateCrossConnectClient(fmt.Sprintf("localhost:%d", fwd2.ListenPort))
	defer close2()

	kubetest.DeployICMP(k8s, nodes[nodesCount-1].Node, "icmp-responder-nse-1", defaultTimeout)
	kubetest.DeployNSC(k8s, nodes[0].Node, "nsc-1", defaultTimeout)

	_, err = kubetest.GetCrossConnectsFromMonitor(nsmdMonitor1, cancel1, 1, fastTimeout)
	Expect(err).To(BeNil())
	_, err = kubetest.GetCrossConnectsFromMonitor(nsmdMonitor2, cancel2, 1, fastTimeout)
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
	kubetest.DeployICMP(k8s, nodes[nodesCount-1].Node, "icmp-responder-nse-1", defaultTimeout)
	kubetest.DeployNSC(k8s, nodes[0].Node, "nsc-1", defaultTimeout)
	kubetest.DeployNSC(k8s, nodes[0].Node, "nsc-2", defaultTimeout)

	fwd, err := k8s.NewPortForwarder(nodes[0].Nsmd, 5001)
	Expect(err).To(BeNil())
	defer fwd.Stop()

	err = fwd.Start()
	Expect(err).To(BeNil())

	fwd2, err := k8s.NewPortForwarder(nodes[1].Nsmd, 5001)
	Expect(err).To(BeNil())
	defer fwd2.Stop()

	err = fwd2.Start()
	Expect(err).To(BeNil())

	nsmdMonitor1, close1, cancel1 := kubetest.CreateCrossConnectClient(fmt.Sprintf("localhost:%d", fwd.ListenPort))
	defer close1()

	nsmdMonitor2, close2, cancel2 := kubetest.CreateCrossConnectClient(fmt.Sprintf("localhost:%d", fwd2.ListenPort))
	defer close2()

	_, err = kubetest.GetCrossConnectsFromMonitor(nsmdMonitor1, cancel1, 2, fastTimeout)
	Expect(err).To(BeNil())
	_, err = kubetest.GetCrossConnectsFromMonitor(nsmdMonitor2, cancel2, 2, fastTimeout)
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

	fwd, err := k8s.NewPortForwarder(nodes[0].Nsmd, 5001)
	Expect(err).To(BeNil())
	defer fwd.Stop()

	err = fwd.Start()
	Expect(err).To(BeNil())

	nsmdMonitor, closeFunc, cancel := kubetest.CreateCrossConnectClient(fmt.Sprintf("localhost:%d", fwd.ListenPort))
	_, err = kubetest.GetCrossConnectsFromMonitor(nsmdMonitor, cancel, 2, fastTimeout)
	Expect(err).To(BeNil())
	closeFunc()

	logrus.Info("Restarting monitor")
	nsmdMonitor, closeFunc, cancel = kubetest.CreateCrossConnectClient(fmt.Sprintf("localhost:%d", fwd.ListenPort))
	defer closeFunc()
	_, err = kubetest.GetCrossConnectsFromMonitor(nsmdMonitor, cancel, 2, fastTimeout)
	Expect(err).To(BeNil())
}
