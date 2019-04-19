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

func TestSimpleMetrics(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	k8s, err := kube_testing.NewK8s()
	defer k8s.Cleanup()
	Expect(err).To(BeNil())

	s1 := time.Now()
	k8s.PrepareDefault()
	logrus.Printf("Cleanup done: %v", time.Since(s1))

	nodesCount := 2

	nodes := nsmd_test_utils.SetupNodes(k8s, nodesCount, defaultTimeout)
	nsmd_test_utils.DeployICMP(k8s, nodes[nodesCount-1].Node, "icmp-responder-nse-1", defaultTimeout)
	nsmd_test_utils.DeployNSC(k8s, nodes[0].Node, "nsc1", defaultTimeout)

	fwd, err := k8s.NewPortForwarder(nodes[0].Nsmd, 5001)
	Expect(err).To(BeNil())
	defer fwd.Stop()

	err = fwd.Start()
	Expect(err).To(BeNil())

	nsmdMonitor, close, cancel := createCrossConnectClient(fmt.Sprintf("localhost:%d", fwd.ListenPort))
	defer close()
	monitorCrossConnects(nsmdMonitor, cancel)
	time.Sleep(time.Minute * 5)
	cancel()

	Expect(err).To(BeNil())
}

func crossConnectClient(address string) (crossconnect.MonitorCrossConnect_MonitorCrossConnectsClient, func(), context.CancelFunc) {
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

func monitorCrossConnects(stream crossconnect.MonitorCrossConnect_MonitorCrossConnectsClient, cancel context.CancelFunc) {
	go func() {
		for {
			select {
			case <-stream.Context().Done():
				logrus.Info("GG")
				return
			default:
				event, _ := stream.Recv()
				logrus.Infof("Receive event type: %v", event.GetType())
				for k, v := range event.Metrics {
					logrus.Infof("New statistics: %v %v", k, v)
				}
			}
		}
	}()

}
