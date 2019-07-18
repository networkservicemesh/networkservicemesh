package main

import (
	"github.com/grpc-ecosystem/grpc-opentracing/go/otgrpc"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/monitor/remote"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/nsmd"
	proxynetworkserviceserver "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/remote/proxy_network_service_server"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/serviceregistry"
	"google.golang.org/grpc"
	"net"
	"os"
	"strings"
	"time"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/remote/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/remote/networkservice"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
)

var version string

// Default values and environment variables of proxy connection
const (
	ProxyNsmdAPIAddressEnv      = "PROXY_NSMD_API_ADDRESS"
	ProxyNsmdAPIAddressDefaults = ":5006"
)

func main() {
	logrus.Info("Starting proxy nsmd...")
	logrus.Infof("Version: %v", version)
	start := time.Now()

	// Capture signals to cleanup before exiting
	c := tools.NewOSSignalChannel()

	tracer, closer := tools.InitJaeger("proxy-nsmd")
	opentracing.SetGlobalTracer(tracer)
	defer func() {
		if err := closer.Close(); err != nil {
			logrus.Errorf("Failed to close tracer: %v", err)
		}
	}()

	nsmdProbes := nsmd.NewProbes()
	go nsmdProbes.BeginHealthCheck()
	// Proxy NSM doesn't start some services - pass that probes by default
	nsmdProbes.SetDPServerReady()
	nsmdProbes.SetNSMServerReady()

	apiRegistry := nsmd.NewApiRegistry()
	serviceRegistry := nsmd.NewServiceRegistry()
	defer serviceRegistry.Stop()

	// Choose a public API listener
	nsmdAPIAddress := os.Getenv(ProxyNsmdAPIAddressEnv)
	if strings.TrimSpace(nsmdAPIAddress) == "" {
		nsmdAPIAddress = ProxyNsmdAPIAddressDefaults
	}
	sock, err := apiRegistry.NewPublicListener(nsmdAPIAddress)
	if err != nil {
		logrus.Errorf("Failed to start Public API server...")
		return
	}
	nsmdProbes.SetPublicListenerReady()

	startAPIServerAt(sock, serviceRegistry)
	nsmdProbes.SetAPIServerReady()

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

	remoteConnectionMonitor := remote.NewProxyMonitorServer()
	connection.RegisterMonitorConnectionServer(grpcServer, remoteConnectionMonitor)

	// Register Remote NetworkServiceManager
	remoteServer := proxynetworkserviceserver.NewProxyNetworkServiceServer(serviceRegistry)
	networkservice.RegisterNetworkServiceServer(grpcServer, remoteServer)

	go func() {
		if err := grpcServer.Serve(sock); err != nil {
			logrus.Fatalf("Failed to start NSM API server %+v", err)
		}
	}()
	logrus.Infof("NSM gRPC API Server: %s is operational", sock.Addr().String())
}
