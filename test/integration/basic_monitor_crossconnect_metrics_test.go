// +build basic

package nsmd_integration_tests

import (
	"context"
	"fmt"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/crossconnect"
	"github.com/networkservicemesh/networkservicemesh/test/integration/nsmd_test_utils"
	"github.com/networkservicemesh/networkservicemesh/test/kube_testing"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"testing"
)

func TestSimpleMetrics(t *testing.T) {
	RegisterTestingT(t)
	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}
	if !nsmd_test_utils.IsBrokeTestsEnabled() {
		t.Skip("Skipped for a while, will be enabled soon")
		return
	}
	k8s, err := kube_testing.NewK8s()
	Expect(err).To(BeNil())

	defer k8s.Cleanup()

	k8s.PrepareDefault()

	nodesCount := 2

	nodes := nsmd_test_utils.SetupNodes(k8s, nodesCount, defaultTimeout)
	nsmd_test_utils.DeployICMP(k8s, nodes[nodesCount-1].Node, "icmp-responder-nse-1", defaultTimeout)
	nsc := nsmd_test_utils.DeployNSC(k8s, nodes[0].Node, "nsc1", defaultTimeout)

	fwd, err := k8s.NewPortForwarder(nodes[0].Nsmd, 5001)
	Expect(err).To(BeNil())

	defer fwd.Stop()

	err = fwd.Start()
	Expect(err).To(BeNil())

	nsmdMonitor, close := crossConnectClient(fmt.Sprintf("localhost:%d", fwd.ListenPort))
	defer close()

	metricsCh := make(chan map[string]string)
	monitorCrossConnectsMetrics(nsmdMonitor, metricsCh)

	response, _, err := k8s.Exec(nsc, nsc.Spec.Containers[0].Name, "ping", "172.16.1.2", "-A", "-c", "4")
	logrus.Infof("response = %v", response)
	Expect(err).To(BeNil())

	k8s.DeletePods(nsc)
	metrics := <-metricsCh

	Expect(isMetricsEmpty(metrics)).Should(Equal(false))
	Expect(metrics["rx_error_packets"]).Should(Equal("0"))
	Expect(metrics["tx_error_packets"]).Should(Equal("0"))
}

func crossConnectClient(address string) (crossconnect.MonitorCrossConnect_MonitorCrossConnectsClient, func()) {
	var err error
	logrus.Infof("Starting CrossConnections Monitor on %s", address)
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		Expect(err).To(BeNil())
		return nil, nil
	}
	monitorClient := crossconnect.NewMonitorCrossConnectClient(conn)
	stream, err := monitorClient.MonitorCrossConnects(context.Background(), &empty.Empty{})
	if err != nil {
		Expect(err).To(BeNil())
		return nil, nil
	}

	closeFunc := func() {
		if err := conn.Close(); err != nil {
			logrus.Errorf("Closing the stream with: %v", err)
		}
	}
	return stream, closeFunc
}

func monitorCrossConnectsMetrics(stream crossconnect.MonitorCrossConnect_MonitorCrossConnectsClient, metricsCh chan<- map[string]string) {
	go func() {
		for {
			select {
			case <-stream.Context().Done():
				return
			default:
				event, err := stream.Recv()
				if err != nil {
					println(err)
					continue
				}
				if event.Metrics == nil {
					continue
				}
				for k, v := range event.Metrics {
					logrus.Infof("New statistics: %v %v", k, v)
					if isMetricsEmpty(v.Metrics) {
						logrus.Infof("Statistics: %v %v skipped", k, v)
						continue
					}
					metricsCh <- v.Metrics
				}
			}
		}
	}()
}

func isMetricsEmpty(metrics map[string]string) bool {
	for _, v := range metrics {
		if v != "0" && v != "" {
			return false
		}
	}
	return true
}
