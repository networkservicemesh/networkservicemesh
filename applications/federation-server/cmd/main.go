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

package main

import (
	federation "github.com/networkservicemesh/networkservicemesh/applications/federation-server/api"
	server "github.com/networkservicemesh/networkservicemesh/applications/federation-server/pkg"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"net"
)

const (
	address = ":7002"
)

func main() {
	ln, err := net.Listen("tcp", address)
	if err != nil {
		logrus.Fatal(err)
	}

	srv := grpc.NewServer()
	federation.RegisterRegistrationServer(srv, server.New())

	if err := srv.Serve(ln); err != nil {
		logrus.Fatal(err)
	}
}
