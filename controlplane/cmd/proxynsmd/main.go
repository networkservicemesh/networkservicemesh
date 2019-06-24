package main

import (
	"github.com/grpc-ecosystem/grpc-opentracing/go/otgrpc"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/nsmd"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/remote/proxy_network_service_server"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/serviceregistry"
	"google.golang.org/grpc"
	"net"
	"os"
	"strings"
	"time"

	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/remote/networkservice"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
)

const (
	ProxyNsmdApiAddressEnv 		= "PROXY_NSMD_API_ADDRESS"
	ProxyNsmdApiAddressDefaults = "0.0.0.0:5006"
)

func main() {
	start := time.Now()

	// Capture signals to cleanup before exiting
	c := tools.NewOSSignalChannel()

	tracer, closer := tools.InitJaeger("proxy-nsmd")
	opentracing.SetGlobalTracer(tracer)
	defer closer.Close()

	go nsmd.BeginHealthCheck()

	apiRegistry := nsmd.NewApiRegistry()
	serviceRegistry := nsmd.NewServiceRegistry()
	defer serviceRegistry.Stop()

	// Choose a public API listener
	nsmdApiAddress := os.Getenv(ProxyNsmdApiAddressEnv)
	if strings.TrimSpace(nsmdApiAddress) == "" {
		nsmdApiAddress = ProxyNsmdApiAddressDefaults
	}
	sock, err := apiRegistry.NewPublicListener(nsmdApiAddress)
	if err != nil {
		logrus.Errorf("Failed to start Public API server...")
		nsmd.SetPublicListenerFailed()
	}

	startAPIServerAt(sock, serviceRegistry)

	elapsed := time.Since(start)
	logrus.Debugf("Starting Proxy NSMD took: %s", elapsed)

	<-c
}

// StartAPIServerAt starts GRPC API server at sock
func startAPIServerAt(sock net.Listener, serviceRegistry serviceregistry.ServiceRegistry) {
	tracer := opentracing.GlobalTracer()
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(
			otgrpc.OpenTracingServerInterceptor(tracer, otgrpc.LogPayloads())),
		grpc.StreamInterceptor(
			otgrpc.OpenTracingStreamServerInterceptor(tracer)))

	// Register Remote NetworkServiceManager
	remoteServer := proxy_network_service_server.NewProxyNetworkServiceServer(serviceRegistry)
	networkservice.RegisterNetworkServiceServer(grpcServer, remoteServer)

	go func() {
		if err := grpcServer.Serve(sock); err != nil {
			logrus.Errorf("failed to start gRPC NSMD API server %+v", err)
		}
	}()
	logrus.Infof("NSM gRPC API Server: %s is operational", sock.Addr().String())
}
