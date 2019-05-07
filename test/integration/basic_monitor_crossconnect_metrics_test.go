// +build basic

package nsmd_integration_tests

import (
	"context"
	"fmt"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/crossconnect"
	"github.com/networkservicemesh/networkservicemesh/dataplane/vppagent/pkg/vppagent"
	"github.com/networkservicemesh/networkservicemesh/test/integration/nsmd_test_utils"
	"github.com/networkservicemesh/networkservicemesh/test/kube_testing"
	"github.com/networkservicemesh/networkservicemesh/test/kube_testing/pods"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"testing"
	"time"
)

func TestSimpleMetrics(t *testing.T) {
	RegisterTestingT(t)
	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}
	k8s, err := kube_testing.NewK8s(true)
	Expect(err).To(BeNil())

	defer k8s.Cleanup()

	nodesCount := 2
	requestPeriod := time.Second

	nodes := nsmd_test_utils.SetupNodesConfig(k8s, nodesCount, defaultTimeout, []*pods.NSMgrPodConfig{
		{
			DataplaneVariables: map[string]string{
				vppagent.DataplaneMetricsCollectorEnabledKey:       "true",
				vppagent.DataplaneMetricsCollectorRequestPeriodKey: requestPeriod.String(),
			},
			Variables: pods.DefaultNSMD,
		},
	}, k8s.GetK8sNamespace())

	nsmd_test_utils.DeployICMP(k8s, nodes[nodesCount-1].Node, "icmp-responder-nse-1", defaultTimeout)
	fwd, err := k8s.NewPortForwarder(nodes[0].Nsmd, 5001)
	Expect(err).To(BeNil())

	defer fwd.Stop()

	err = fwd.Start()
	Expect(err).To(BeNil())

	nsmdMonitor, close := crossConnectClient(fmt.Sprintf("localhost:%d", fwd.ListenPort))
	defer close()
	metricsCh := make(chan map[string]string)
	monitorCrossConnectsMetrics(nsmdMonitor, metricsCh)
	nsc := nsmd_test_utils.DeployNSC(k8s, nodes[0].Node, "nsc1", defaultTimeout)

	response, _, err := k8s.Exec(nsc, nsc.Spec.Containers[0].Name, "ping", "172.16.1.2", "-A", "-c", "4")
	logrus.Infof("response = %v", response)
	Expect(err).To(BeNil())
	<-time.Tick(requestPeriod * 5)
	k8s.DeletePods(nsc)
	select {
	case metrics := <-metricsCh:
		Expect(isMetricsEmpty(metrics)).Should(Equal(false))
		Expect(metrics["rx_error_packets"]).Should(Equal("0"))
		Expect(metrics["tx_error_packets"]).Should(Equal("0"))
		return
	case <-time.Tick(defaultTimeout):
		t.Fatalf("Fail to get metrics during %v", defaultTimeout)
	}
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
						logrus.Infof("Statistics: %v %v is empty", k, v)
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
