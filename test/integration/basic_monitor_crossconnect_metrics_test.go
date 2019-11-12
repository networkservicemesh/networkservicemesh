// +build basic

package nsmd_integration_tests

import (
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/crossconnect"

	"testing"
	"time"

	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/forwarder/pkg/common"
	"github.com/networkservicemesh/networkservicemesh/test/kubetest"
	"github.com/networkservicemesh/networkservicemesh/test/kubetest/pods"
)

func TestSimpleMetrics(t *testing.T) {
	g := NewWithT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}
	k8s, err := kubetest.NewK8s(g, kubetest.DefaultClear)
	g.Expect(err).To(BeNil())

	defer k8s.Cleanup()

	nodesCount := 2
	requestPeriod := time.Second

	nodes, err := kubetest.SetupNodesConfig(k8s, nodesCount, defaultTimeout, []*pods.NSMgrPodConfig{
		{
			ForwarderVariables: map[string]string{
				common.ForwarderMetricsEnabledKey:       "true",
				"DEBUG_IFSTATES":                        "true",
				common.ForwarderMetricsRequestPeriodKey: requestPeriod.String(),
			},
			Variables: pods.DefaultNSMD(),
		},
	}, k8s.GetK8sNamespace())
	k8s.WaitLogsContains(nodes[0].Forwarder, nodes[0].Forwarder.Spec.Containers[0].Name, "Metrics collector: creating notification client", time.Minute)
	g.Expect(err).To(BeNil())
	kubetest.DeployICMP(k8s, nodes[nodesCount-1].Node, "icmp-responder-nse-1", defaultTimeout)
	defer kubetest.MakeLogsSnapshot(k8s, t)

	eventCh, closeFunc := kubetest.CrossConnectClientAt(k8s, nodes[0].Nsmd)
	defer closeFunc()

	metricsCh := metricsFromEventCh(eventCh)
	nsc := kubetest.DeployNSC(k8s, nodes[0].Node, "nsc1", defaultTimeout)
	for i := 0; i < 10; i++ {
		response, _, _ := k8s.Exec(nsc, nsc.Spec.Containers[0].Name, "ping", "172.16.1.2", "-A", "-c", "4")
		logrus.Infof("response = %v", response)
	}
	<-time.After(requestPeriod * 5)
	k8s.DeletePods(nsc)
	select {
	case metrics := <-metricsCh:
		g.Expect(isMetricsEmpty(metrics)).Should(Equal(false))
		g.Expect(metrics["rx_error_packets"]).Should(Equal("0"))
		g.Expect(metrics["tx_error_packets"]).Should(Equal("0"))
		return
	case <-time.After(defaultTimeout):
		t.Fatalf("Fail to get metrics during %v", defaultTimeout)
	}
}

func metricsFromEventCh(eventCh <-chan *crossconnect.CrossConnectEvent) chan map[string]string {
	metricsCh := make(chan map[string]string)
	go func() {
		defer close(metricsCh)
		for {
			event, ok := <-eventCh
			if !ok {
				return
			}
			logrus.Infof("Received event %v", event)
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
	}()
	return metricsCh
}

func isMetricsEmpty(metrics map[string]string) bool {
	for _, v := range metrics {
		if v != "0" && v != "" {
			return false
		}
	}
	return true
}
