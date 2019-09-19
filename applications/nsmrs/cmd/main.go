package main

import (
	"github.com/networkservicemesh/networkservicemesh/applications/nsmrs/pkg/serviceregistryserver"
	"net"
	"os"
	"strings"

	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
)

const (
	RegistryAPIAddressEnv      = "NSMRS_API_ADDRESS"
	RegistryAPIAddressDefaults = ":5006"
)

var version string

func main() {
	logrus.Info("Starting kube-api-server...")
	logrus.Infof("Version: %v", version)

	c := tools.NewOSSignalChannel()

	tracer, closer := tools.InitJaeger("serviceregistryserver")
	opentracing.SetGlobalTracer(tracer)
	defer closer.Close()

	span := opentracing.StartSpan("serviceregistryserver")
	defer span.Finish()

	address := os.Getenv(RegistryAPIAddressEnv)
	if strings.TrimSpace(address) == "" {
		address = RegistryAPIAddressDefaults
	}

	logrus.Println("Starting NSMD Service Registry Server on " + address)
	serviceRegistryServer := serviceregistryserver.NewNSMDServiceRegistryServer()
	sock, err := serviceRegistryServer.NewPublicListener(address)
	if err != nil {
		logrus.Errorf("Failed to start Public API server...")
		return
	}

	startAPIServerAt(sock)

	<-c
}

func startAPIServerAt(sock net.Listener) {
	grpcServer := serviceregistryserver.New()

	go func() {
		if err := grpcServer.Serve(sock); err != nil {
			logrus.Fatalf("Failed to start Service Registry API server %+v", err)
		}
	}()
	logrus.Infof("Service Registry gRPC API Server: %s is operational", sock.Addr().String())
}