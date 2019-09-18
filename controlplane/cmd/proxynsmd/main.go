package main

import (
	"net"
	"os"
	"strings"
	"time"

	"github.com/networkservicemesh/networkservicemesh/pkg/probes/health"

	"github.com/networkservicemesh/networkservicemesh/pkg/probes"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/monitor/remote"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/nsmd"
	proxynetworkserviceserver "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/remote/proxy_network_service_server"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/serviceregistry"

	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/remote/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/remote/networkservice"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
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
	goals := &proxyNsmdProbeGoals{}
	nsmdProbes := probes.New("Prxoy NSMD liveness/readiness healthcheck", goals)
	nsmdProbes.BeginHealthCheck()

	apiRegistry := nsmd.NewApiRegistry()
	serviceRegistry := nsmd.NewServiceRegistry()
	defer serviceRegistry.Stop()

	// Choose a public API listener

	sock, err := apiRegistry.NewPublicListener(getProxyNSMDAPIAddress())
	if err != nil {
		logrus.Errorf("Failed to start Public API server...")
		return
	}
	logrus.Info("Public listener is ready")
	goals.SetPublicListenerReady()

	startAPIServerAt(sock, serviceRegistry, nsmdProbes)
	logrus.Info("API server is ready")
	goals.SetServerAPIReady()

	elapsed := time.Since(start)
	logrus.Debugf("Starting Proxy NSMD took: %s", elapsed)

	<-c
}

func getProxyNSMDAPIAddress() string {
	result := os.Getenv(ProxyNsmdAPIAddressEnv)
	if strings.TrimSpace(result) == "" {
		result = ProxyNsmdAPIAddressDefaults
	}
	return result
}

// StartAPIServerAt starts GRPC API server at sock
func startAPIServerAt(sock net.Listener, serviceRegistry serviceregistry.ServiceRegistry, probes probes.Probes) {
	grpcServer := tools.NewServer()
	remoteConnectionMonitor := remote.NewProxyMonitorServer()
	connection.RegisterMonitorConnectionServer(grpcServer, remoteConnectionMonitor)
	probes.Append(health.NewGrpcHealth(grpcServer, sock.Addr(), time.Minute))
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
