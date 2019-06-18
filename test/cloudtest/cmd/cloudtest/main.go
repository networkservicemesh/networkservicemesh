package main

import (
	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/commands"

	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"	// This is required for GKE authentication to work properly
)

func main() {
	commands.ExecuteCloudTest()
}