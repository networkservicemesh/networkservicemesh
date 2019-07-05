package main

import (
	"time"

	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/model"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/nsm"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/nsmd"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
)

var version string

func main() {
	logrus.Info("Starting nsmd...")
	logrus.Infof("Version: %v", version)
	start := time.Now()

	// Capture signals to cleanup before exiting
	c := tools.NewOSSignalChannel()

	tracer, closer := tools.InitJaeger("nsmd")
	opentracing.SetGlobalTracer(tracer)
	defer closer.Close()

	go nsmd.BeginHealthCheck()

	apiRegistry := nsmd.NewApiRegistry()
	serviceRegistry := nsmd.NewServiceRegistry()

	model := model.NewModel() // This is TCP gRPC server uri to access this NSMD via network.
	defer serviceRegistry.Stop()

	manager := nsm.NewNetworkServiceManager(model, serviceRegistry)

	var server nsmd.NSMServer
	var err error
	// Start NSMD server first, load local NSE/client registry and only then start dataplane/wait for it and recover active connections.
	if server, err = nsmd.StartNSMServer(model, manager, serviceRegistry, apiRegistry); err != nil {
		logrus.Errorf("Error starting nsmd service: %+v", err)
		return
	}
	defer server.Stop()
	nsmd.SetNSMServerReady()

	// Register CrossConnect monitorCrossConnectServer client as ModelListener
	monitorCrossConnectClient := nsmd.NewMonitorCrossConnectClient(server, server.XconManager(), server)
	model.AddListener(monitorCrossConnectClient)

	// Starting dataplane
	logrus.Info("Starting Dataplane registration server...")
	if err := server.StartDataplaneRegistratorServer(); err != nil {
		logrus.Errorf("Error starting dataplane service: %+v", err)
		return
	}

	// Wait for dataplane to be connecting to us
	if err := manager.WaitForDataplane(nsmd.DataplaneTimeout); err != nil {
		logrus.Errorf("Error waiting for dataplane..")
		return
	}
	nsmd.SetDPServerReady()

	// Choose a public API listener
	sock, err := apiRegistry.NewPublicListener()
	if err != nil {
		logrus.Errorf("Failed to start Public API server...")
		return
	}
	nsmd.SetPublicListenerReady()


	quit := make(chan error)
	server.StartAPIServerAt(sock, quit)
	nsmd.SetAPIServerReady()

	elapsed := time.Since(start)
	logrus.Debugf("Starting NSMD took: %s", elapsed)

	select {
		case osSignal := <-c:
			logrus.Errorf("Exited with OS signal: %s", osSignal.String())
		case err = <-quit:
			logrus.Errorf("Failed to start gRPC NSMD API server: %+v", err)
	}
}
