package vppagent

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/ligato/vpp-agent/api/configurator"
	vpp_interfaces "github.com/ligato/vpp-agent/api/models/vpp/interfaces"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/crossconnect"
	"github.com/networkservicemesh/networkservicemesh/sdk/monitor/metrics"
	"google.golang.org/grpc"

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
	conn, err := grpc.Dial("unix",
		grpc.WithInsecure(),
		grpc.WithDialer(dialer("tcp", endpoint, m.requestPeriod)))

	if err != nil {
		logrus.Errorf("Metrics collector: can't dial %v", err)
		return
	}
	logrus.Infof("Metrics collector: creating notification client for %v", endpoint)
	notificationClient := configurator.NewStatsPollerClient(conn)
	m.startListenNotifications(monitor, notificationClient)
}

func (m *MetricsCollector) startListenNotifications(monitor metrics.MetricsMonitor, client configurator.StatsPollerClient) {
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
				monitor.HandleMetrics(convertStatistics(vppStats.Interface))
			}
			logrus.Infof("MetricsCollector: GetStats(): %v", vppStats)
		}

		<-time.After(m.requestPeriod)
	}

}

// Dialer for unix domain socket
func dialer(socket, address string, timeoutVal time.Duration) func(string, time.Duration) (net.Conn, error) {
	return func(addr string, timeout time.Duration) (net.Conn, error) {
		// Pass values
		addr, timeout = address, timeoutVal
		// Dial with timeout
		return net.DialTimeout(socket, addr, timeoutVal)
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
