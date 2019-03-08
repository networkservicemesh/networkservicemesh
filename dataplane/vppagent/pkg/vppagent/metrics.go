package vppagent

import (
	"context"
	"github.com/grpc-ecosystem/grpc-opentracing/go/otgrpc"
	"github.com/ligato/vpp-agent/plugins/vpp/model/rpc"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/monitor/crossconnect_monitor"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"io"
	"time"
)

func startMetricsCollector(crossConnectServer  *crossconnect_monitor.CrossConnectMonitor, vppAgentEndpoint string) {
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
			notificationClient := rpc.NewNotificationServiceClient(conn)
			nr := &rpc.NotificationRequest{}
			stream, err := notificationClient.Get(context.Background(), nr)
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
				name := msg.GetNIf().GetState().Name
				stat := msg.GetNIf().GetState().Statistics
				crossConnectServer.UpdateStatistics(name, stat)

				//logrus.Infof("Monitor msg: %v", msg)
			}
		}
	}()
}
