// Copyright (c) 2020 Doc.ai and/or its affiliates.
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

package client_test

import (
	"context"
	"net"
	"os"
	"testing"
	"time"

	"github.com/networkservicemesh/networkservicemesh/pkg/tools"

	"github.com/onsi/gomega"

	"github.com/golang/protobuf/ptypes/empty"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/networkservice"

	"github.com/networkservicemesh/networkservicemesh/sdk/client"
	"github.com/networkservicemesh/networkservicemesh/sdk/common"
)

type testNSMServer struct {
}

func (t testNSMServer) Request(context.Context, *networkservice.NetworkServiceRequest) (*connection.Connection, error) {
	return new(connection.Connection), nil
}

func (t testNSMServer) Close(context.Context, *connection.Connection) (*empty.Empty, error) {
	return new(empty.Empty), nil
}

func TestNewNSMClientList_MultipleConnects(t *testing.T) {
	assert := gomega.NewWithT(t)
	err := os.Setenv("INSECURE", "true")
	assert.Expect(err).Should(gomega.BeNil())
	err = os.Setenv(client.AnnotationEnv, "bridge-domain?app=bridge,bridge-domain-ipv6?app=bridge-ipv6")
	assert.Expect(err).Should(gomega.BeNil())

	var configuration = new(common.NSConfiguration)
	configuration.NsmServerSocket = "test.sock"
	cleanup, err := startNsmServer(configuration.NsmServerSocket)
	assert.Expect(err).Should(gomega.BeNil())
	defer cleanup()

	err = tools.WaitForPortAvailable(context.Background(), "unix", configuration.NsmServerSocket, 100*time.Millisecond)
	assert.Expect(err).Should(gomega.BeNil())

	l, err := client.NewNSMClientList(context.Background(), configuration)
	assert.Expect(err).Should(gomega.BeNil())

	clients := l.Clients()
	assert.Expect(clients, gomega.HaveLen(2))
	assert.Expect(clients[0].Configuration.ClientNetworkService).Should(gomega.Equal("bridge-domain"))
	assert.Expect(clients[1].Configuration.ClientNetworkService).Should(gomega.Equal("bridge-domain-ipv6"))
}

func startNsmServer(sock string) (func(), error) {
	cleanup := func() {
		_ = os.Remove(sock)
	}
	cleanup()
	s := tools.NewServer(context.Background())
	networkservice.RegisterNetworkServiceServer(s, &testNSMServer{})
	ln, err := net.Listen("unix", sock)
	if err != nil {
		return nil, err
	}
	if err != nil {
		return nil, err
	}
	go func() {
		_ = s.Serve(ln)
	}()
	return func() {
		cleanup()
		s.Stop()
	}, nil
}
