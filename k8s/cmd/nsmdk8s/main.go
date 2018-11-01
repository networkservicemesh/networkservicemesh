package main

import (
	"github.com/ligato/networkservicemesh/k8s/pkg/registryserver"
	"github.com/sirupsen/logrus"
)

func main() {
	address := "127.0.0.1:5000"
	logrus.Println("Starting NSMD Kubernetes on " + address)

	// Start NSC Server
	logrus.Println(registryserver.New(address))

	// Start NSC Client

	// Start InterNSM

}
