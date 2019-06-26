package main

import (
	"context"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/monitor"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/monitor/local"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
	"github.com/networkservicemesh/networkservicemesh/sdk/client"
	"github.com/networkservicemesh/networkservicemesh/test/applications/cmd/monitoring-dns-nsc/corefile"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
)

var version string

func main() {
	logrus.Info("Starting monitoring-dns-nsc...")
	logrus.Infof("Version: %v", version)
	c := tools.NewOSSignalChannel()
	tracer, closer := tools.InitJaeger("nsc")
	opentracing.SetGlobalTracer(tracer)
	defer func() { _ = closer.Close() }()

	manager, err := corefile.NewDefaultDNSConfigManager()
	if err != nil {
		logrus.Errorf("Can not create dns config manager, reason: %v", err)
	}

	nsc, err := client.NewNSMClient(context.Background(), nil)
	if err != nil {
		logrus.Fatalf("Unable to create the NSM client: %v", err)
	}
	go startMonitor(nsc, manager)

	logrus.Info("nsm client: initialization is completed successfully.")
	<-c
}

func startMonitor(nsc *client.NsmClient, manager corefile.DNSConfigManager) {
	currentConn, err := nsc.Connect("nsm", "kernel", "Primary interface")
	if err != nil {
		logrus.Errorf("An error during nsc connecting: %v", err)
	}
	monitorClient, err := local.NewMonitorClient(nsc.NsmConnection.GrpcClient)
	if err != nil {
		logrus.Errorf("An error during creating monitor client: %v", err)
	}
	defer monitorClient.Close()
	for {
		select {
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
				switch event.EventType() {
				case monitor.EventTypeInitialStateTransfer, monitor.EventTypeUpdate:
					if currentConn.Context.DnsConfig != nil {
						if err := manager.UpdateDNSConfig(currentConn.Context.DnsConfig); err != nil {
							logrus.Errorf("An error during updating dns config %v", err)
						} else {
							logrus.Infof("dns config %v has been successfully updated", currentConn.Context.DnsConfig)
						}
					}
				case monitor.EventTypeDelete:
					logrus.Info("Connection closed")
					if currentConn.Context.DnsConfig != nil {
						if err := manager.RemoveDNSConfig(currentConn.Context.DnsConfig); err != nil {
							logrus.Errorf("An error during removing dns config %v", err)
						} else {
							logrus.Infof("dns config %v has been successfully deleted", currentConn.Context.DnsConfig)
						}
					}
					return
				}
			}
		}
	}
}
