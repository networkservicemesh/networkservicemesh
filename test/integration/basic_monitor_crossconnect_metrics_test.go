// +build basic

package integration

import (
	"bufio"
	"bytes"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/crossconnect"
	metricspkg "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/metrics"

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

	defer k8s.Cleanup(t)

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
	defer k8s.SaveTestArtifacts(t)

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

func collectMetrics() {

	prometheusCtx := metricspkg.BuildPrometheusMetricsContext("alpine-nsc-8485cb6857-jk8d2", "nsm-system", "icmp-responder-nse-5cdf65d447-ft4k5", "nsm-system")
	prometheusMetrics := metricspkg.BuildPrometheusMetrics()

	metricsData := crossconnect.Metrics{
		Metrics: map[string]string{
			"rx_bytes":         "140",
			"rx_packets":       "3",
			"rx_error_packets": "0",
			"tx_bytes":         "140",
			"tx_packets":       "3",
			"tx_error_packets": "0",
		},
	}
	metricspkg.CollectAllMetrics(prometheusCtx, prometheusMetrics, &metricsData)
}

func verifyMetrics(t *testing.T, hostPort string) string {
	wg := sync.WaitGroup{}
	wg.Add(1)

	res := ""
	go func() {
		defer wg.Done()
		for {
			resp, err := http.Get("http://" + hostPort + "/metrics")
			if err != nil {
				t.Logf("failed to get metrics from %s with error: %v", hostPort, err)
				break
			}
			defer resp.Body.Close()
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				t.Logf("failed to read response body (resp: %v) with err: %v", resp, err)
				break
			}

			bytesReader := bytes.NewReader(body)
			bufReader := bufio.NewReader(bytesReader)

			checkDone := false
			for {
				value, _, err := bufReader.ReadLine()
				if err != nil {
					checkDone = true
					t.Logf("finished reading response from prometheus metrics")
					break
				}
				val := string(value)
				if strings.Contains(val, "rx") || strings.Contains(val, "tx") {
					res += val + "\n"
				}
			}
			if checkDone {
				t.Log("done receiving prometheus metrics")
				break
			}
			time.Sleep(2 * time.Second)
		}
		return
	}()
	wg.Wait()
	if len(res) == 0 {
		t.Fatal("failed to get metrics from Prometheus")
	}
	t.Logf("metrics triggered: \n%s", res)
	return res
}

func TestPrometheusMetrics(t *testing.T) {

	promServer := httptest.NewServer(promhttp.Handler())
	promServer.URL = strings.TrimPrefix(promServer.URL, "http://")

	collectMetrics()
	verifyMetrics(t, promServer.URL)

	time.Sleep(10 * time.Second)
	promServer.Close()
}

func TestGetMetricsIdentifiers(t *testing.T) {
	cc := &crossconnect.CrossConnect{
		Id: "1",
		Source: &connection.Connection{
			Id: "1",
			Labels: map[string]string{
				"app":       "icmp",
				"namespace": "nsm-system",
				"podName":   "icmp-responder-nsc-6475749466-qbcq5",
			},
		},
		Destination: &connection.Connection{
			Id: "2",
			Labels: map[string]string{
				"app":       "icmp",
				"namespace": "nsm-system",
				"podName":   "icmp-responder-nse-59c456b6d8-c7nc5",
			},
		},
	}

	metricsIdentifiers, err := metricspkg.GetMetricsIdentifiers(cc)
	if err != nil {
		t.Fatalf("failed to get metrics identifier from crossconnect: %v", err)
	}
	expectedMetricsIdentifiers := map[string]string{
		"src_pod":       "icmp-responder-nsc-6475749466-qbcq5",
		"src_namespace": "nsm-system",
		"dst_pod":       "icmp-responder-nse-59c456b6d8-c7nc5",
		"dst_namespace": "nsm-system",
	}

	if metricsIdentifiers["src_pod"] != expectedMetricsIdentifiers["src_pod"] ||
		metricsIdentifiers["dst_pod"] != expectedMetricsIdentifiers["dst_pod"] ||
		metricsIdentifiers["src_namespace"] != expectedMetricsIdentifiers["src_namespace"] ||
		metricsIdentifiers["dst_namespace"] != expectedMetricsIdentifiers["dst_namespace"] {
		t.Fatalf("failed to correct metrics identifier from crossconnect: want: %v, got: %v",
			expectedMetricsIdentifiers, metricsIdentifiers)
	}
}
