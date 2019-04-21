package metrics

import (
	"context"
	"fmt"
	"github.com/gogo/protobuf/proto"
	"github.com/grpc-ecosystem/grpc-opentracing/go/otgrpc"
	rpc "github.com/ligato/vpp-agent/api/configurator"
	interfaces "github.com/ligato/vpp-agent/api/models/vpp/interfaces"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/crossconnect"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"io"
	"time"
)

type MetricsCollector struct {
	requestPeriod time.Duration
}

func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		requestPeriod: time.Second * 15,
	}
}

func (m *MetricsCollector) CollectAsync(monitor MetricsMonitor, endpoint string) {
	go m.collect(monitor, endpoint)
}

func (m *MetricsCollector) collect(monitor MetricsMonitor, endpoint string) {
	tracer := opentracing.GlobalTracer()
	conn, err := grpc.Dial(endpoint, grpc.WithInsecure(),
		grpc.WithUnaryInterceptor(
			otgrpc.OpenTracingClientInterceptor(tracer, otgrpc.LogPayloads())),
		grpc.WithStreamInterceptor(
			otgrpc.OpenTracingStreamClientInterceptor(tracer)))
	if err != nil {
		logrus.Errorf("Metrics collector: can't dial %v", err)
		return
	}
	logrus.Infof("Metrics collector: creating notificaiton client for %v", endpoint)
	notificationClient := rpc.NewConfiguratorClient(conn)
	m.startListenNotifications(monitor, notificationClient)
}

func (m *MetricsCollector) startListenNotifications(monitor MetricsMonitor, client rpc.ConfiguratorClient) {
	var nextIdx uint32 = 0
	logrus.Info("Metrics collector: start handle notifications")
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
func (m *MetricsCollector) handleNotifications(monitor MetricsMonitor, stream rpc.Configurator_NotifyClient, nextIndex *uint32) error {
	for {
		notification, err := stream.Recv()
		if err != nil {
			return err
		}
		*nextIndex = notification.NextIdx
		statistics := convertStatistics(notification.Notification.GetVppNotification().Interface.State)
		logrus.Infof("Metrics collector: new statistics %v", proto.MarshalTextString(notification.Notification))
		monitor.HandleMetrics(statistics)
		logrus.Info("Metrics collector: statistics handled")
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
