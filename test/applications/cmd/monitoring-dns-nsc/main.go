package main

import (
	"context"
	"fmt"
	"github.com/gogo/protobuf/proto"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/connectioncontext"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
	"strings"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/monitor"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/monitor/local"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
	"github.com/networkservicemesh/networkservicemesh/sdk/client"
)

var version string

//TODO: cleanup
func main() {
	corefile := NewCorefile("/etc/coredns/Corefile")
	corefile.WriteScope(".:53").Write("log")
	logrus.Info("Starting monitoring-dns-nsc...")
	logrus.Infof("Version: %v", version)
	// Capture signals to cleanup before exiting
	c := tools.NewOSSignalChannel()

	tracer, closer := tools.InitJaeger("nsc")
	opentracing.SetGlobalTracer(tracer)
	defer func() { _ = closer.Close() }()

	nsc, err := client.NewNSMClient(context.Background(), nil)
	if err != nil {
		logrus.Fatalf("Unable to create the NSM client: %v", err)
	}

	currentConn, err := nsc.Connect("nsm", "kernel", "Primary interface")
	if err != nil {
		logrus.Fatalf("Failed to connect: %v", err)
	}

	logrus.Info("nsm client: initialization is completed successfully.")

	monitorClient, err := local.NewMonitorClient(nsc.NsmConnection.GrpcClient)
	if err != nil {
		logrus.Fatalf("Failed to start monitor client: %v", err)
	}
	defer monitorClient.Close()

	for {
		select {
		case <-c:
			return
		case err = <-monitorClient.ErrorChannel():
			logrus.Fatalf("Monitor failed: %v", err)
		case event := <-monitorClient.EventChannel():
			if event.EventType() == monitor.EventTypeInitialStateTransfer {
				logrus.Info("Monitor started")
			}

			for _, entity := range event.Entities() {
				conn, ok := entity.(*connection.Connection)
				if !ok || conn.GetId() != currentConn.GetId() {
					continue
				}
				if conn.Context.DnsConfig != nil {
					updateCoreFile(corefile, conn.Context.DnsConfig)
				}
				switch event.EventType() {
				case monitor.EventTypeInitialStateTransfer, monitor.EventTypeUpdate:
					if !proto.Equal(conn, currentConn) {
						logrus.Infof("Connection updated: %v", conn)
						currentConn = conn
					}
				case monitor.EventTypeDelete:
					logrus.Info("Connection closed")
					return
				}
			}
		}
	}
}

func updateCoreFile(corefile Corefile, config *connectioncontext.DNSConfig) {
	forwards := strings.Join(config.DnsServerIps, " ")
	corefile.Remove(".:53")
	corefile.WriteScope(".:53").Write("log").Write(fmt.Sprintf("forward . %v", forwards))
	corefile.Save()
}
