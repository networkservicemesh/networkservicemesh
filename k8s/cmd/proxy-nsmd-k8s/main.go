package main

import (
	"net"
	"os"
	"strings"

	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools/jaeger"

	"github.com/networkservicemesh/networkservicemesh/k8s/pkg/proxyregistryserver"
	k8s_utils "github.com/networkservicemesh/networkservicemesh/k8s/pkg/utils"

	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/utils"
)

var version string

func main() {
	logrus.Info("Starting proxy nsmd-k8s...")
	logrus.Infof("Version: %v", version)
	utils.PrintAllEnv(logrus.StandardLogger())
	// Capture signals to cleanup before exiting
	c := tools.NewOSSignalChannel()

	closer := jaeger.InitJaeger("proxy-nsmd-k8s")
	defer func() { _ = closer.Close() }()

	address := os.Getenv("PROXY_NSMD_K8S_ADDRESS")
	if strings.TrimSpace(address) == "" {
		address = "0.0.0.0:5005"
	}

	logrus.Println("Starting NSMD Kubernetes on " + address)
	nsmClientSet, config, err := k8s_utils.NewClientSet()

	clusterInfoService, err := proxyregistryserver.NewK8sClusterInfoService(config)
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
