// Copyright (c) 2019 Cisco and/or its affiliates.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at:
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"math/rand"
	"net"
	"os"
	"strings"
	"time"

	"github.com/networkservicemesh/networkservicemesh/pkg/tools/jaeger"

	"github.com/networkservicemesh/networkservicemesh/applications/nsmrs/pkg/serviceregistryserver"

	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
)

const (
	// RegistryAPIAddressEnv - env with NSMRS API address
	RegistryAPIAddressEnv = "NSMRS_API_ADDRESS"
	// RegistryAPIAddressDefaults - default NSMRS API address
	RegistryAPIAddressDefaults = ":5006"
)

var version string

func main() {
	logrus.Info("Starting kube-api-server...")
	logrus.Infof("Version: %v", version)

	rand.Seed(time.Now().Unix())

	c := tools.NewOSSignalChannel()

	closer := jaeger.InitJaeger("serviceregistryserver")
	defer func() { _ = closer.Close() }()

	address := os.Getenv(RegistryAPIAddressEnv)
	if strings.TrimSpace(address) == "" {
		address = RegistryAPIAddressDefaults
	}

	logrus.Println("Starting NSMD Service Registry Server on " + address)
	serviceRegistryServer := serviceregistryserver.NewNSMDServiceRegistryServer()
	sock, err := serviceRegistryServer.NewPublicListener(address)
	if err != nil {
		logrus.Errorf("Failed to start Public API server...")
		return
	}

	startAPIServerAt(sock)

	<-c
}

func startAPIServerAt(sock net.Listener) {
	grpcServer := serviceregistryserver.New()

	go func() {
		if err := grpcServer.Serve(sock); err != nil {
			logrus.Fatalf("Failed to start Service Registry API server %+v", err)
		}
	}()
	logrus.Infof("Service Registry gRPC API Server: %s is operational", sock.Addr().String())
}
