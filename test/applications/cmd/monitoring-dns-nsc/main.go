//TODO: refactor this after https://github.com/networkservicemesh/networkservicemesh/pull/1352 will be merged
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
	"os"
)

var version string

//TODO: remove this.
const DO_NOT_CREATE_INTERFACE = "DO_NOT_CREATE_INTERFACE"

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
	if os.Getenv(DO_NOT_CREATE_INTERFACE) == "true" {
	} else {
		_, err := nsc.Connect("nsm", "kernel", "Primary interface")
		if err != nil {
			logrus.Errorf("An error during nsc connecting: %v", err)
		}
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
				if !ok {
					continue
				}
				switch event.EventType() {
				case monitor.EventTypeInitialStateTransfer, monitor.EventTypeUpdate:
					if conn.Context.DnsContext != nil {
						if err := manager.UpdateDNSConfig(conn.Context.DnsContext); err != nil {
							logrus.Errorf("An error during updating dns config %v", err)
						} else {
							logrus.Infof("dns config %v has been successfully updated", conn.Context.DnsContext)
						}
					}
				case monitor.EventTypeDelete:
					logrus.Info("Connection closed")
					if conn.Context.DnsContext != nil {
						if err := manager.RemoveDNSConfig(conn.Context.DnsContext); err != nil {
							logrus.Errorf("An error during removing dns config %v", err)
						} else {
							logrus.Infof("dns config %v has been successfully deleted", conn.Context.DnsContext)
						}
					}
					return
				}
			}
		}
	}
}
