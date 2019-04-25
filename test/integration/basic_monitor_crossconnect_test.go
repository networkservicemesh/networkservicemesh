// +build basic

package nsmd_integration_tests

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/crossconnect"
	"github.com/networkservicemesh/networkservicemesh/test/integration/nsmd_test_utils"
	"github.com/networkservicemesh/networkservicemesh/test/kube_testing"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

func TestSingleCrossConnect(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	k8s, err := kube_testing.NewK8s(true)
	defer k8s.Cleanup()
	Expect(err).To(BeNil())

	nodesCount := 2

	nodes := nsmd_test_utils.SetupNodes(k8s, nodesCount, defaultTimeout)
	nsmd_test_utils.DeployICMP(k8s, nodes[nodesCount-1].Node, "icmp-responder-nse-1", defaultTimeout)
	nsmd_test_utils.DeployNSC(k8s, nodes[0].Node, "nsc-1", defaultTimeout)

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

	nsmdMonitor1, close1, cancel1 := createCrossConnectClient(fmt.Sprintf("localhost:%d", fwd.ListenPort))
	defer close1()
	nsmdMonitor2, close2, cancel2 := createCrossConnectClient(fmt.Sprintf("localhost:%d", fwd2.ListenPort))
	defer close2()

	_, err = getCrossConnectsFromMonitor(nsmdMonitor1, cancel1, 1, fastTimeout)
	Expect(err).To(BeNil())
	_, err = getCrossConnectsFromMonitor(nsmdMonitor2, cancel2, 1, fastTimeout)
	Expect(err).To(BeNil())
}

func TestSingleCrossConnectMonitorBeforeXcons(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	k8s, err := kube_testing.NewK8s(true)
	defer k8s.Cleanup()
	Expect(err).To(BeNil())

	nodesCount := 2

	nodes := nsmd_test_utils.SetupNodes(k8s, nodesCount, defaultTimeout)

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

	nsmdMonitor1, close1, cancel1 := createCrossConnectClient(fmt.Sprintf("localhost:%d", fwd.ListenPort))
	defer close1()
	nsmdMonitor2, close2, cancel2 := createCrossConnectClient(fmt.Sprintf("localhost:%d", fwd2.ListenPort))
	defer close2()

	nsmd_test_utils.DeployICMP(k8s, nodes[nodesCount-1].Node, "icmp-responder-nse-1", defaultTimeout)
	nsmd_test_utils.DeployNSC(k8s, nodes[0].Node, "nsc-1", defaultTimeout)

	_, err = getCrossConnectsFromMonitor(nsmdMonitor1, cancel1, 1, fastTimeout)
	Expect(err).To(BeNil())
	_, err = getCrossConnectsFromMonitor(nsmdMonitor2, cancel2, 1, fastTimeout)
	Expect(err).To(BeNil())
}

func TestSeveralCrossConnects(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	k8s, err := kube_testing.NewK8s(true)
	defer k8s.Cleanup()
	Expect(err).To(BeNil())

	nodesCount := 2

	nodes := nsmd_test_utils.SetupNodes(k8s, nodesCount, defaultTimeout)
	nsmd_test_utils.DeployICMP(k8s, nodes[nodesCount-1].Node, "icmp-responder-nse-1", defaultTimeout)
	nsmd_test_utils.DeployNSC(k8s, nodes[0].Node, "nsc-1", defaultTimeout)
	nsmd_test_utils.DeployNSC(k8s, nodes[0].Node, "nsc-2", defaultTimeout)

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

	nsmdMonitor1, close1, cancel1 := createCrossConnectClient(fmt.Sprintf("localhost:%d", fwd.ListenPort))
	defer close1()

	nsmdMonitor2, close2, cancel2 := createCrossConnectClient(fmt.Sprintf("localhost:%d", fwd2.ListenPort))
	defer close2()

	_, err = getCrossConnectsFromMonitor(nsmdMonitor1, cancel1, 2, fastTimeout)
	Expect(err).To(BeNil())
	_, err = getCrossConnectsFromMonitor(nsmdMonitor2, cancel2, 2, fastTimeout)
	Expect(err).To(BeNil())
}
func TestCrossConnectMonitorRestart(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	k8s, err := kube_testing.NewK8s(true)
	defer k8s.Cleanup()
	Expect(err).To(BeNil())

	nodesCount := 2

	nodes := nsmd_test_utils.SetupNodes(k8s, nodesCount, defaultTimeout)
	nsmd_test_utils.DeployICMP(k8s, nodes[nodesCount-1].Node, "icmp-responder-nse-1", defaultTimeout)
	nsmd_test_utils.DeployNSC(k8s, nodes[0].Node, "nsc-1", defaultTimeout)
	nsmd_test_utils.DeployNSC(k8s, nodes[0].Node, "nsc-2", defaultTimeout)

	fwd, err := k8s.NewPortForwarder(nodes[0].Nsmd, 5001)
	Expect(err).To(BeNil())
	defer fwd.Stop()

	err = fwd.Start()
	Expect(err).To(BeNil())

	nsmdMonitor, closeFunc, cancel := createCrossConnectClient(fmt.Sprintf("localhost:%d", fwd.ListenPort))
	_, err = getCrossConnectsFromMonitor(nsmdMonitor, cancel, 2, fastTimeout)
	Expect(err).To(BeNil())
	closeFunc()

	logrus.Info("Restarting monitor")
	nsmdMonitor, closeFunc, cancel = createCrossConnectClient(fmt.Sprintf("localhost:%d", fwd.ListenPort))
	defer closeFunc()
	_, err = getCrossConnectsFromMonitor(nsmdMonitor, cancel, 2, fastTimeout)
	Expect(err).To(BeNil())
}

func createCrossConnectClient(address string) (crossconnect.MonitorCrossConnect_MonitorCrossConnectsClient, func(), context.CancelFunc) {
	var err error
	logrus.Infof("Starting CrossConnections Monitor on %s", address)
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		Expect(err).To(BeNil())
		return nil, nil, nil
	}

	monitorClient := crossconnect.NewMonitorCrossConnectClient(conn)
	ctx, cancel := context.WithCancel(context.Background())
	stream, err := monitorClient.MonitorCrossConnects(ctx, &empty.Empty{})
	if err != nil {
		Expect(err).To(BeNil())
		cancel()
		return nil, nil, nil
	}

	closeFunc := func() {
		if err := conn.Close(); err != nil {
			logrus.Errorf("Closing the stream with: %v", err)
		}
	}
	return stream, closeFunc, cancel
}

func getCrossConnectsFromMonitor(stream crossconnect.MonitorCrossConnect_MonitorCrossConnectsClient, cancel context.CancelFunc,
	xconAmount int, timeout time.Duration) (map[string]*crossconnect.CrossConnect, error) {

	xcons := map[string]*crossconnect.CrossConnect{}
	events := make(chan *crossconnect.CrossConnectEvent)

	go func() {
		for {
			select {
			case <-stream.Context().Done():
				return
			default:
				event, _ := stream.Recv()
				if event != nil {
					events <- event
				}
			}
		}
	}()

	for {
		select {
		case event := <-events:
			logrus.Infof("Receive event type: %v", event.GetType())

			for _, xcon := range event.CrossConnects {
				logrus.Infof("xcon: %v", xcon)
				xcons[xcon.GetId()] = xcon
			}
			if len(xcons) == xconAmount {
				cancel()
				return xcons, nil
			}
		case <-time.After(timeout):
			cancel()
			return nil, fmt.Errorf("timeout exceeded")
		}
	}
}
