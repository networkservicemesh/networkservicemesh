// Copyright (c) 2018 Cisco and/or its affiliates.
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

package nsmd

import (
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/ligato/networkservicemesh/controlplane/pkg/model/registry"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

var once sync.Once
var registryClient registry.NetworkServiceRegistryClient
var registryClientConnection *grpc.ClientConn
var stopRedial = true

func RegistryClient() (registry.NetworkServiceRegistryClient, error) {
	var err error
	once.Do(func() {
		// TODO doing registry Address here is ugly
		registryAddress := os.Getenv("NSM_REGISTRY_ADDRESS")
		registryAddress = strings.TrimSpace(registryAddress)
		if registryAddress == "" {
			registryAddress = "localhost:5000"
		}
		for stopRedial {
			conn, err := grpc.Dial(registryAddress, grpc.WithInsecure())
			if err != nil {
				logrus.Errorf("Failed to dial Network Service Registry: %s", err)
				continue
			}
			registryClientConnection = conn
			registryClient = registry.NewNetworkServiceRegistryClient(conn)
			return
		}
		err = fmt.Errorf("Stopped before success trying to dial Network Registry Server")
		logrus.Error(err)
	})
	return registryClient, err
}

func StopRegistryClient() {
	stopRedial = false
	if registryClientConnection != nil {
		registryClientConnection.Close()
	}
}
