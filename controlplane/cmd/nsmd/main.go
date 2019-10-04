package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/networkservicemesh/networkservicemesh/pkg/tools/jaeger"

	"github.com/networkservicemesh/networkservicemesh/pkg/tools/spanhelper"

	"github.com/networkservicemesh/networkservicemesh/pkg/probes"

	"github.com/sirupsen/logrus"

	pluginsapi "github.com/networkservicemesh/networkservicemesh/controlplane/api/plugins"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/model"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/nsm"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/nsmd"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/plugins"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
)

var version string

// Default values and environment variables of proxy connection
const (
	NsmdAPIAddressEnv      = "NSMD_API_ADDRESS"
	NsmdAPIAddressDefaults = ":5001"
)

func main() {
	logrus.Info("Starting nsmd...")
	logrus.Infof("Version: %v", version)
	start := time.Now()

	// Capture signals to cleanup before exiting
	c := tools.NewOSSignalChannel()

	closer := jaeger.InitJaeger("nsmd")
	defer func() { _ = closer.Close() }()

	// Global NSMgr server span holder
	span := spanhelper.FromContext(context.Background(), "nsmd.server")
	span.LogValue("tracing.init-complete", fmt.Sprintf("%v", time.Since(start)))
	defer span.Finish() // Mark it as finished, since it will be used as root.

	apiRegistry := nsmd.NewApiRegistry()
	serviceRegistry := nsmd.NewServiceRegistry()
	pluginRegistry := plugins.NewPluginRegistry(pluginsapi.PluginRegistrySocket)

	if err := pluginRegistry.Start(span.Context()); err != nil {
		logrus.Errorf("Failed to start Plugin Registry: %v", err)
		return
	}

	span.Logger().Infof("Plugins started")

	defer func() {
		if err := pluginRegistry.Stop(); err != nil {
			logrus.Errorf("Failed to stop Plugin Registry: %v", err)
		}
	}()


	model := model.NewModel() // This is TCP gRPC server uri to access this NSMD via network.
	defer serviceRegistry.Stop()
	manager := nsm.NewNetworkServiceManager(model, serviceRegistry, pluginRegistry)

	var server nsmd.NSMServer
	var srvErr error
	// Start NSMD server first, load local NSE/client registry and only then start dataplane/wait for it and recover active connections.

	if server, srvErr = nsmd.StartNSMServer(span.Context(), model, manager, serviceRegistry, apiRegistry); srvErr != nil {
		logrus.Errorf("error starting nsmd service: %+v", srvErr)
		return
	}
	defer server.Stop()

	nsmdGoals := &nsmdProbeGoals{}
	nsmdProbes := probes.New("NSMD liveness/readiness healthcheck", nsmdGoals)
	nsmdProbes.BeginHealthCheck()

	logrus.Info("NSM server is ready")
	nsmdGoals.SetNsmServerReady()

	// Register CrossConnect monitorCrossConnectServer client as ModelListener
	monitorCrossConnectClient := nsmd.NewMonitorCrossConnectClient(server, server.XconManager(), server)
	model.AddListener(monitorCrossConnectClient)

	// Starting dataplane
	logrus.Info("Starting Dataplane registration server...")
	if err := server.StartDataplaneRegistratorServer(); err != nil {
		span.LogError(fmt.Errorf("error starting dataplane service: %+v", err))
		return
	}

	// Wait for dataplane to be connecting to us
	if err := manager.WaitForDataplane(span.Context(), nsmd.DataplaneTimeout); err != nil {
		span.LogError(fmt.Errorf("error waiting for dataplane: %+v", err))
		return
	}

	span.Logger().Info("Dataplane server is ready")
	nsmdGoals.SetDataplaneServerReady()
	// Choose a public API listener

	nsmdAPIAddress := getNsmdAPIAddress()
	span.LogObject("api-address", nsmdAPIAddress)
	sock, err := apiRegistry.NewPublicListener(nsmdAPIAddress)
	if err != nil {
		span.LogError(fmt.Errorf("failed to start Public API server: %+v", err))
		return
	}
	span.Logger().Info("Public listener is ready")
	nsmdGoals.SetPublicListenerReady()

	server.StartAPIServerAt(span.Context(), sock, nsmdProbes)
	nsmdGoals.SetServerAPIReady()
	span.Logger().Info("Serve api is ready")

	span.LogValue("start-time", fmt.Sprintf("%v", time.Since(start)))
	span.Finish()
	<-c
}

func getNsmdAPIAddress() string {
	result := os.Getenv(NsmdAPIAddressEnv)
	if strings.TrimSpace(result) == "" {
		result = NsmdAPIAddressDefaults
	}
	return result
}
