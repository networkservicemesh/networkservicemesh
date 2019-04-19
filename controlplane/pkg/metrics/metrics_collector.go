package metrics

import (
	"context"
	"fmt"
	"github.com/gogo/protobuf/proto"
	"github.com/grpc-ecosystem/grpc-opentracing/go/otgrpc"
	rpc "github.com/ligato/vpp-agent/api/configurator"
	interfaces "github.com/ligato/vpp-agent/api/models/vpp/interfaces"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"io"
	"time"
)

type MetricsCollector struct {
	stopChannel     chan struct{}
	nextRequestTime time.Duration
}

func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		stopChannel:     make(chan struct{}),
		nextRequestTime: time.Second * 10,
	}
}

func (m *MetricsCollector) CollectAsync(monitor MetricsMonitor, endpoint string) {
	go m.collect(monitor, endpoint)
}

func (m *MetricsCollector) Close() {
	close(m.stopChannel)
}

func (m *MetricsCollector) collect(monitor MetricsMonitor, endpoint string) {
	tracer := opentracing.GlobalTracer()
	conn, err := grpc.Dial(endpoint, grpc.WithInsecure(),
		grpc.WithUnaryInterceptor(
			otgrpc.OpenTracingClientInterceptor(tracer, otgrpc.LogPayloads())),
		grpc.WithStreamInterceptor(
			otgrpc.OpenTracingStreamClientInterceptor(tracer)))
	if err != nil {
		logrus.Errorf("can't dial grpc server: %v", err)
		return
	}

	for {
		select {
		case <-m.stopChannel:
			logrus.Info("Metrics collector: stop")
			return
		default:
			logrus.Info("Metrics collector: waiting new client")
			notificationClient := rpc.NewConfiguratorClient(conn)
			m.handleNotifications(monitor, notificationClient)
		}
	}
}

func (m *MetricsCollector) handleNotifications(monitor MetricsMonitor, client rpc.ConfiguratorClient) {
	var nextIdx uint32 = 0
	logrus.Info("Metrics collector: start handle notifications")
	for {
		logrus.Infof("Metrics collector: request %v", nextIdx)
		request := &rpc.NotificationRequest{
			Idx: nextIdx,
		}
		stream, err := client.Notify(context.Background(), request)
		if err != nil {
			logrus.Errorf("Metrics collector: an problem during get stream %v", err)
			return
		}
		for {
			notification, err := stream.Recv()
			if err == io.EOF {
				logrus.Info("Metrics collector: EOF")
				break;
			}
			if err != nil {
				logrus.Errorf("Metrics collector: an problem during recv notification %v", err)
				return
			}
			nextIdx = notification.NextIdx
			stat := convertStatistics(notification.Notification.GetVppNotification().Interface.State)
			logrus.Infof("Metrics collector: new statistics %v", proto.MarshalTextString(notification.Notification))
			monitor.HandleMetrics(stat)
			logrus.Info("Metrics collector: handle metrics")
		}
		time.Sleep(m.nextRequestTime)
	}
}

func convertStatistics(state *interfaces.InterfaceState) *Statistics {
	stats := state.Statistics
	metrics := make(map[string]string)
	metrics["rx_bytes"] = fmt.Sprint(stats.InBytes)
	metrics["tx_bytes"] = fmt.Sprint(stats.OutBytes)
	metrics["rx_packets"] = fmt.Sprint(stats.InPackets)
	metrics["tx_packets"] = fmt.Sprint(stats.OutPackets)
	metrics["rx_error_packets"] = fmt.Sprint(stats.InErrorPackets)
	metrics["tx_error_packets"] = fmt.Sprint(stats.OutErrorPackets)
	return &Statistics{
		Name:    state.Name,
		Metrics: metrics,
	}
}
