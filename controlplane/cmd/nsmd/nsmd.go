package main

import (
	"time"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/model"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/nsm"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/nsmd"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
)

func main() {
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
		nsmd.SetNSMServerFailed()
		return
	}
	defer server.Stop()

	// Starting dataplane
	logrus.Info("Starting Dataplane registration server...")
	if err := server.StartDataplaneRegistratorServer(); err != nil {
		logrus.Errorf("Error starting dataplane service: %+v", err)
		nsmd.SetDPServerFailed()
	}

	// Wait for dataplane to be connecting to us
	if err := manager.WaitForDataplane(nsmd.DataplaneTimeout); err != nil {
		logrus.Errorf("Error waiting for dataplane")
	}

	// Choose a public API listener
	sock, err := apiRegistry.NewPublicListener()
	if err != nil {
		logrus.Errorf("Failed to start Public API server...")
		nsmd.SetPublicListenerFailed()
	}

	if err := nsmd.StartAPIServerAt(server, sock); err != nil {
		logrus.Errorf("Error starting NSMD API service: %+v", err)
		nsmd.SetAPIServerFailed()
	}

	elapsed := time.Since(start)
	logrus.Debugf("Starting NSMD took: %s", elapsed)

	<-c
}
