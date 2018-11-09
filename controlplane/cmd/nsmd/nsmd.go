package main

import (
	"github.com/ligato/networkservicemesh/controlplane/pkg/model/registry"
	"google.golang.org/grpc"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"github.com/ligato/networkservicemesh/controlplane/pkg/model"
	"github.com/ligato/networkservicemesh/controlplane/pkg/nsmd"
	"github.com/sirupsen/logrus"
)

func main() {
	model := model.NewModel()

	registryAddress := os.Getenv("NSM_REGISTRY_ADDRESS")
	registryAddress = strings.TrimSpace(registryAddress)
	if registryAddress == "" {
		registryAddress = "localhost:5000"
	}

	registryConn, err := grpc.Dial(registryAddress, grpc.WithInsecure())
	if err != nil {
		logrus.Fatalln("Unable to connect to registry", err)
	}
	defer registryConn.Close()

	registryClient := registry.NewNetworkServiceRegistryClient(registryConn)

	if err := nsmd.StartDataplaneRegistrarServer(model); err != nil {
		logrus.Fatalf("Error starting dataplane service: %+v", err)
		os.Exit(1)
	}
	if err := nsmd.StartEndpointServer(model, registryClient); err != nil {
		logrus.Fatalf("Error starting endpoint service: %+v", err)
		os.Exit(1)
	}
	if err := nsmd.StartNSMServer(model); err != nil {
		logrus.Fatalf("Error starting nsmd service: %+v", err)
		os.Exit(1)
	}

	var wg sync.WaitGroup
	wg.Add(1)
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		wg.Done()
	}()
	wg.Wait()
}
