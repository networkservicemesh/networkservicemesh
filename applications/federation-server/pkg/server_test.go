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

package server_test

import (
	"context"
	"fmt"
	"net"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	"github.com/spiffe/spire/proto/spire/common"
	"google.golang.org/grpc"

	federation "github.com/networkservicemesh/networkservicemesh/applications/federation-server/api"
	server "github.com/networkservicemesh/networkservicemesh/applications/federation-server/pkg"
)

func TestFederationServer(t *testing.T) {
	// TODO: unskip the test
	t.Skip()

	g := NewWithT(t)

	ln, err := net.Listen("tcp", "localhost:0")
	g.Expect(err).To(BeNil())

	srv := grpc.NewServer()
	federation.RegisterRegistrationServer(srv, server.New())

	go func() {
		if err := srv.Serve(ln); err != nil {
			t.Fatal(err)
		}
	}()

	addr := fmt.Sprintf("localhost:%d", ln.Addr().(*net.TCPAddr).Port)
	cc, err := grpc.Dial(addr, grpc.WithInsecure())
	g.Expect(err).To(BeNil())

	regc := federation.NewRegistrationClient(cc)
	for i := 0; i < 10; i++ {
		_, err := regc.CreateFederatedBundle(context.Background(), &common.Bundle{
			TrustDomainId: fmt.Sprintf("td-%d", i),
		})
		g.Expect(err).To(BeNil())
	}

	stream, err := regc.ListFederatedBundles(context.Background(), &common.Empty{})
	g.Expect(err).To(BeNil())

	for {
		msg, err := stream.Recv()
		g.Expect(err).To(BeNil())

		logrus.Infof("received - %v", msg.Type)
		for _, b := range msg.Bundles {
			logrus.Infof("%v: %v", b.GetTrustDomainId(), b)
		}
	}
}
