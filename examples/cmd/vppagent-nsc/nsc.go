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

package main

import (
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/nsmd"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
	"github.com/networkservicemesh/networkservicemesh/sdk/client"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
	"os"
	"sync"
)

const (
	defaultVPPAgentEndpoint = "localhost:9113"
)

type nsClientBackend struct {
	workspace        string
	vppAgentEndpoint string
}

func (nscb *nsClientBackend) New() error {
	if err := Reset(nscb.vppAgentEndpoint); err != nil {
		logrus.Fatal(err)
	}
	logrus.Infof("workspace: %s", nscb.workspace)
	return nil
}

func (nscb *nsClientBackend) Connect(connection *connection.Connection) error {
	logrus.Infof("nsClientBackend received: %v", connection)
	err := CreateVppInterface(connection, nscb.workspace, nscb.vppAgentEndpoint)
	if err != nil {
		logrus.Errorf("VPPAgent failed creating the requested interface with: %v", err)
	}
	return err
}

func main() {
	// Capture signals to cleanup before exiting
	c := tools.NewOSSignalChannel()

	tracer, closer := tools.InitJaeger("nsc")
	opentracing.SetGlobalTracer(tracer)
	defer closer.Close()

	workspace, ok := os.LookupEnv(nsmd.WorkspaceEnv)
	if !ok {
		logrus.Fatalf("Failed getting %s", nsmd.WorkspaceEnv)
	}

	backend := &nsClientBackend{
		workspace:        workspace,
		vppAgentEndpoint: defaultVPPAgentEndpoint,
	}

	client, err := client.NewNSMClient(nil, nil)
	if err != nil {
		logrus.Fatalf("Unable to create the NSM client %v", err)
	}

	err = backend.New()
	if err != nil {
		logrus.Fatalf("Unable to create the backend %v", err)
	}

	var outgoingConnection *connection.Connection
	outgoingConnection, err = client.Connect("if1", "mem", "Primary interface")
	if err != nil {
		logrus.Fatalf("Unable to connect %v", err)
	}

	err = backend.Connect(outgoingConnection)
	if err != nil {
		logrus.Fatalf("Unable to connect %v", err)
	}

	logrus.Info("nsm client: initialization is completed successfully, wait for Ctrl+C...")
	var wg sync.WaitGroup
	wg.Add(1)

	<-c
}
