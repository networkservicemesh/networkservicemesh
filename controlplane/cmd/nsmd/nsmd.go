package main

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/model"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/nsm"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/nsmd"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
)

func main() {
	start := time.Now()

	// Capture signals to cleanup before exiting
	c := make(chan os.Signal, 1)
	signal.Notify(c,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)

	tracer, closer := tools.InitJaeger("nsmd")
	opentracing.SetGlobalTracer(tracer)
	defer closer.Close()

	go nsmd.BeginHealthCheck()

	apiRegistry := nsmd.NewApiRegistry()
	serviceRegistry := nsmd.NewServiceRegistry()

	model := model.NewModel() // This is TCP gRPC server uri to access this NSMD via network.

	defer serviceRegistry.Stop()

	if err := nsmd.StartDataplaneRegistrarServer(model); err != nil {
		logrus.Fatalf("Error starting dataplane service: %+v", err)
		nsmd.SetDPServerFailed()
	}

	manager := nsm.NewNetworkServiceManager(model, serviceRegistry, nsmd.GetExcludedPrefixes())

	if err := nsmd.StartNSMServer(model, manager, serviceRegistry, apiRegistry); err != nil {
		logrus.Fatalf("Error starting nsmd service: %+v", err)
		nsmd.SetNSMServerFailed()
	}

	if err := nsmd.StartAPIServer(model, manager, apiRegistry, serviceRegistry); err != nil {
		logrus.Fatalf("Error starting nsmd api service: %+v", err)
		nsmd.SetAPIServerFailed()
	}

	elapsed := time.Since(start)
	logrus.Debugf("Starting NSMD took: %s", elapsed)

	<-c
}
