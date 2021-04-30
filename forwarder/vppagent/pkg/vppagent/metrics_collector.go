package vppagent

import (
	"context"
	"fmt"
	"time"

	"go.ligato.io/vpp-agent/v3/proto/ligato/configurator"
	vpp_interfaces "go.ligato.io/vpp-agent/v3/proto/ligato/vpp/interfaces"

	"github.com/google/go-cmp/cmp"

	"github.com/networkservicemesh/networkservicemesh/pkg/tools"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/crossconnect"
	"github.com/networkservicemesh/networkservicemesh/sdk/monitor/metrics"

	"github.com/sirupsen/logrus"
)

type MetricsCollector struct {
	requestPeriod time.Duration
}

// NewMetricsCollector creates a new metrics collector instance
func NewMetricsCollector(requestPeriod time.Duration) *MetricsCollector {
	return &MetricsCollector{
		requestPeriod: requestPeriod,
	}
}

// CollectAsync starts ago routine for asynchronous metrics collection
func (m *MetricsCollector) CollectAsync(monitor metrics.MetricsMonitor, endpoint string) {
	go m.collect(monitor, endpoint)
}

func (m *MetricsCollector) collect(monitor metrics.MetricsMonitor, endpoint string) {
	conn, err := tools.DialTCPInsecure(endpoint)

	if err != nil {
		logrus.Errorf("Metrics collector: can't dial %v", err)
		return
	}
	logrus.Infof("Metrics collector: creating notification client for %v", endpoint)

	notificationClient := configurator.NewStatsPollerServiceClient(conn)
	m.startListenNotifications(monitor, notificationClient)
}

func (m *MetricsCollector) startListenNotifications(monitor metrics.MetricsMonitor, client configurator.StatsPollerServiceClient) {
	var prevStats = make(map[string]*vpp_interfaces.InterfaceStats)
	req := &configurator.PollStatsRequest{
		PeriodSec: uint32(m.requestPeriod.Seconds()),
	}

	ctx := context.Background()
	stream, err := client.PollStats(ctx, req)
	if err != nil {
		logrus.Errorf("MetricsCollector: PollStats err: %v", err)
		return
	}

	for {
		resp, err := stream.Recv()
		if err != nil {
			logrus.Errorf("MetricsCollector: stream.Recv() err: %v", err)
		} else {
			vppStats := resp.GetStats().GetVppStats()
			if vppStats.Interface != nil {
				if v, ok := prevStats[vppStats.Interface.Name]; ok && cmp.Equal(v, vppStats.Interface) {
					continue
				}
				prevStats[vppStats.Interface.Name] = vppStats.Interface
				monitor.HandleMetrics(convertStatistics(vppStats.Interface))
			}
			logrus.Infof("MetricsCollector: GetStats(): %v", vppStats)
		}

		<-time.After(m.requestPeriod)
	}
}

func convertStatistics(stats *vpp_interfaces.InterfaceStats) map[string]*crossconnect.Metrics {
	metrics := make(map[string]string)
	metrics["rx_bytes"] = fmt.Sprint(stats.Rx.Bytes)
	metrics["tx_bytes"] = fmt.Sprint(stats.Tx.Bytes)
	metrics["rx_packets"] = fmt.Sprint(stats.Rx.Packets)
	metrics["tx_packets"] = fmt.Sprint(stats.Tx.Packets)
	metrics["rx_error_packets"] = fmt.Sprint(stats.RxError)
	metrics["tx_error_packets"] = fmt.Sprint(stats.TxError)
	return map[string]*crossconnect.Metrics{
		stats.Name: {Metrics: metrics},
	}
}
