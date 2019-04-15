package vppagent

import (
	"context"
	"github.com/grpc-ecosystem/grpc-opentracing/go/otgrpc"
	rpc "github.com/ligato/vpp-agent/api/configurator"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/monitor/crossconnect_monitor"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"io"
	"time"
)

func startMetricsCollector(crossConnectServer *crossconnect_monitor.CrossConnectMonitor, vppAgentEndpoint string) {
	go func() {
		tracer := opentracing.GlobalTracer()
		conn, err := grpc.Dial(vppAgentEndpoint, grpc.WithInsecure(),
			grpc.WithUnaryInterceptor(
				otgrpc.OpenTracingClientInterceptor(tracer, otgrpc.LogPayloads())),
			grpc.WithStreamInterceptor(
				otgrpc.OpenTracingStreamClientInterceptor(tracer)))
		if err != nil {
			logrus.Errorf("can't dial grpc server: %v", err)
			return
		}
		for {
			<-time.Tick(10 * time.Second)
			notificationClient := rpc.NewConfiguratorClient(conn)
			nr := &rpc.NotificationRequest{}
			stream, err := notificationClient.Notify(context.Background(), nr)
			if err != nil {
				logrus.Errorf("Can't get notification stream: %v", err)
				continue
			}

			for {
				msg, err := stream.Recv()
				if err != nil {
					if err != io.EOF {
						logrus.Errorf("Notification stream error: %v", err)
					}
					break
				}
				name := msg.GetNotification().GetVppNotification().Interface.State.Name
				stat := msg.GetNotification().GetVppNotification().Interface.State.Statistics
				crossConnectServer.UpdateStatistics(name, stat)

				logrus.Infof("Monitor msg: %v", msg)
			}
		}
	}()
}
