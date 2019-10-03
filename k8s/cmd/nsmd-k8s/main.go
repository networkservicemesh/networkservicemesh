package main

import (
	"context"
	"net"
	"os"
	"strings"

	"github.com/networkservicemesh/networkservicemesh/pkg/tools/jaeger"

	"github.com/sirupsen/logrus"

	pluginsapi "github.com/networkservicemesh/networkservicemesh/controlplane/api/plugins"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/plugins"
	"github.com/networkservicemesh/networkservicemesh/k8s/pkg/prefixcollector"
	"github.com/networkservicemesh/networkservicemesh/k8s/pkg/registryserver"
	"github.com/networkservicemesh/networkservicemesh/k8s/pkg/utils"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
)

var version string

func main() {
	logrus.Info("Starting nsmd-k8s...")
	logrus.Infof("Version: %v", version)
	// Capture signals to cleanup before exiting
	c := tools.NewOSSignalChannel()

	closer := jaeger.InitJaeger("nsmd-k8s")
	defer func() { _ = closer.Close() }()

	address := os.Getenv("NSMD_K8S_ADDRESS")
	if strings.TrimSpace(address) == "" {
		address = "0.0.0.0:5000"
	}
	nsmName, ok := os.LookupEnv("NODE_NAME")
	if !ok {
		logrus.Fatalf("You must set env variable NODE_NAME to match the name of your Node.  See https://kubernetes.io/docs/tasks/inject-data-application/environment-variable-expose-pod-information/")
	}
	logrus.Println("Starting NSMD Kubernetes on " + address + " with NsmName " + nsmName)

	nsmClientSet, config, err := utils.NewClientSet()
	if err != nil {
		logrus.Fatalln("Fail to start NSMD Kubernetes service", err)
	}

	server := registryserver.New(context.Background(), nsmClientSet, nsmName)

	listener, err := net.Listen("tcp", address)
	if err != nil {
		logrus.Fatalln(err)
	}

	logrus.Print("nsmd-k8s initialized and waiting for connection")
	go func() {
		err = server.Serve(listener)
		if err != nil {
			logrus.Fatalln(err)
		}
	}()

	prefixService, err := prefixcollector.NewPrefixService(config)
	if err != nil {
		logrus.Fatalln(err)
	}

	services := make(map[pluginsapi.PluginCapability]interface{}, 1)
	services[pluginsapi.PluginCapability_CONNECTION] = prefixService

	if err = plugins.StartPlugin("k8s-plugin", pluginsapi.PluginRegistrySocket, services); err != nil {
		logrus.Fatalln("Failed to start K8s Plugin", err)
	}

	<-c
}
