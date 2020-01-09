// Copyright (c) 2020 Cisco and/or its affiliates.
//
// SPDX-License-Identifier: Apache-2.0
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
	"context"
	"math/rand"
	"net"
	"time"

	"github.com/networkservicemesh/networkservicemesh/applications/nsmrs/pkg/serviceregistryserver"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools/jaeger"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools/spanhelper"
	"github.com/networkservicemesh/networkservicemesh/utils"
)

const (
	// RegistryAPIAddressEnv - env with NSMRS API address
	RegistryAPIAddressEnv = utils.EnvVar("NSMRS_API_ADDRESS")
	// RegistryAPIAddressDefaults - default NSMRS API address
	RegistryAPIAddressDefaults = ":5010"
)

var version string

func main() {
	span := spanhelper.FromContext(context.Background(), "Start-NSMD-k8s")
	defer span.Finish()

	span.Logger().Infof("Starting kube-api-server...")
	span.Logger().Infof("Version: %v", version)

	rand.Seed(time.Now().Unix())

	c := tools.NewOSSignalChannel()

	closer := jaeger.InitJaeger("serviceregistryserver")
	defer func() { _ = closer.Close() }()

	address := RegistryAPIAddressEnv.GetStringOrDefault(RegistryAPIAddressDefaults)

	span.Logger().Println("Starting NSMD Service Registry Server on " + address)
	serviceRegistryServer := serviceregistryserver.NewNSMDServiceRegistryServer()
	sock, err := serviceRegistryServer.NewPublicListener(address)
	if err != nil {
		span.Logger().Errorf("Failed to start Public API server...")
		return
	}

	startAPIServerAt(span.Context(), sock)

	span.Finish()

	<-c
}

func startAPIServerAt(ctx context.Context, sock net.Listener) {
	span := spanhelper.FromContext(ctx, "Nsmrs.RegisterNSE")
	defer span.Finish()

	grpcServer := serviceregistryserver.New(ctx)

	go func() {
		if err := grpcServer.Serve(sock); err != nil {
			span.Logger().Fatalf("Failed to start Service Registry API server %+v", err)
		}
	}()
	span.Logger().Infof("Service Registry gRPC API Server: %s is operational", sock.Addr().String())
}
