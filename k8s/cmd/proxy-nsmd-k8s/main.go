package main

import (
	"net"
	"os"
	"strings"

	"github.com/networkservicemesh/networkservicemesh/k8s/pkg/registryserver"
	"github.com/networkservicemesh/networkservicemesh/k8s/pkg/utils"

	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/k8s/pkg/proxyregistryserver"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
)

var version string

func main() {
	logrus.Info("Starting proxy nsmd-k8s...")
	logrus.Infof("Version: %v", version)

	// Capture signals to cleanup before exiting
	c := tools.NewOSSignalChannel()

	tracer, closer := tools.InitJaeger("proxy-nsmd-k8s")
	opentracing.SetGlobalTracer(tracer)
	defer func() {
		if err := closer.Close(); err != nil {
			logrus.Errorf("Failed to close tracer: %v", err)
		}
	}()

	address := os.Getenv("PROXY_NSMD_K8S_ADDRESS")
	if strings.TrimSpace(address) == "" {
		address = "0.0.0.0:5005"
	}

	logrus.Println("Starting NSMD Kubernetes on " + address)
	nsmClientSet, config, err := utils.NewClientSet()

	clusterInfoService, err := registryserver.NewK8sClusterInfoService(config)
	if err != nil {
		logrus.Fatalln("Fail to start NSMD Kubernetes service", err)
	}

	server := proxyregistryserver.New(nsmClientSet, clusterInfoService)

	listener, err := net.Listen("tcp", address)
	if err != nil {
		logrus.Fatalln(err)
	}

	logrus.Print("proxy-nsmd-k8s initialized and waiting for connection")
	go func() {
		err = server.Serve(listener)
		if err != nil {
			logrus.Fatalln(err)
		}
	}()
	<-c
}
