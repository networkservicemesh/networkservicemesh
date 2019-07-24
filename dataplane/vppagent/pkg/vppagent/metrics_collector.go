package vppagent

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/networkservicemesh/networkservicemesh/pkg/tools"

	"github.com/gogo/protobuf/proto"
	rpc "github.com/ligato/vpp-agent/api/configurator"
	interfaces "github.com/ligato/vpp-agent/api/models/vpp/interfaces"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/crossconnect"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/metrics"
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
	conn, err := tools.DialTCP(endpoint)
	if err != nil {
		logrus.Errorf("Metrics collector: can't dial %v", err)
		return
	}
	logrus.Infof("Metrics collector: creating notificaiton client for %v", endpoint)
	notificationClient := rpc.NewConfiguratorClient(conn)
	m.startListenNotifications(monitor, notificationClient)
}

func (m *MetricsCollector) startListenNotifications(monitor metrics.MetricsMonitor, client rpc.ConfiguratorClient) {
	var nextIdx uint32 = 0
	for {
		logrus.Infof("Metrics collector: request %v", nextIdx)
		request := &rpc.NotificationRequest{
			Idx: nextIdx,
		}
		stream, err := client.Notify(context.Background(), request)
		if err != nil {
			logrus.Errorf("Metrics collector: an error during getting stream %v", err)
			return
		}
		err = m.handleNotifications(monitor, stream, &nextIdx)
		if err != nil && err != io.EOF {
			logrus.Errorf("Metrics collector: an error during handling notifications %v", err)
			return
		}
		time.Sleep(m.requestPeriod)
	}
}
func (m *MetricsCollector) handleNotifications(monitor metrics.MetricsMonitor, stream rpc.Configurator_NotifyClient, nextIndex *uint32) error {
	for {
		notification, err := stream.Recv()
		if err != nil {
			return err
		}
		*nextIndex = notification.NextIdx
		statistics := convertStatistics(notification.Notification.GetVppNotification().Interface.State)
		logrus.Infof("Metrics collector: new statistics %v", proto.MarshalTextString(notification.Notification))
		monitor.HandleMetrics(statistics)
	}
}

func convertStatistics(state *interfaces.InterfaceState) map[string]*crossconnect.Metrics {
	stats := state.Statistics
	metrics := make(map[string]string)
	metrics["rx_bytes"] = fmt.Sprint(stats.InBytes)
	metrics["tx_bytes"] = fmt.Sprint(stats.OutBytes)
	metrics["rx_packets"] = fmt.Sprint(stats.InPackets)
	metrics["tx_packets"] = fmt.Sprint(stats.OutPackets)
	metrics["rx_error_packets"] = fmt.Sprint(stats.InErrorPackets)
	metrics["tx_error_packets"] = fmt.Sprint(stats.OutErrorPackets)
	return map[string]*crossconnect.Metrics{
		state.Name: {Metrics: metrics},
	}
}
