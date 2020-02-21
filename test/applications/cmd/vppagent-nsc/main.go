// Copyright (c) 2018-2019 Cisco and/or its affiliates.
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
	"os"
	"sync"

	nsmmonitor "github.com/networkservicemesh/networkservicemesh/side-cars/pkg/nsm-monitor"

	"github.com/networkservicemesh/networkservicemesh/pkg/tools/jaeger"
	"github.com/networkservicemesh/networkservicemesh/utils"

	"github.com/networkservicemesh/networkservicemesh/sdk/common"

	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/memif"

	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
	"github.com/networkservicemesh/networkservicemesh/sdk/client"
)

const (
	defaultVPPAgentEndpoint = "localhost:9113"
)

type nsClientBackend struct {
	*nsmmonitor.EmptyNSMMonitorHandler
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

func (nscb *nsClientBackend) Connect(connection *networkservice.Connection) error {
	logrus.Infof("nsClientBackend received: %v", connection)
	err := CreateVppInterface(connection, nscb.workspace, nscb.vppAgentEndpoint)
	if err != nil {
		logrus.Errorf("VPPAgent failed creating the requested interface with: %v", err)
	}
	return err
}

var version string

func (nscb *nsClientBackend) Updated(_, new *networkservice.Connection) {
	err := nscb.Connect(new)
	if err != nil {
		logrus.Fatalf("Unable to re-connect %v", err)
	}
}

func main() {
	logrus.Info("Starting vppagent-nsc...")
	logrus.Infof("Version: %v", version)
	utils.PrintAllEnv(logrus.StandardLogger())
	// Capture signals to cleanup before exiting
	c := tools.NewOSSignalChannel()
	closer := jaeger.InitJaeger("vppagent-nsc")
	defer func() { _ = closer.Close() }()
	workspace, ok := os.LookupEnv(common.WorkspaceEnv)
	if !ok {
		logrus.Fatalf("Failed getting %s", common.WorkspaceEnv)
	}

	backend := &nsClientBackend{
		workspace:        workspace,
		vppAgentEndpoint: defaultVPPAgentEndpoint,
	}
	ctx, cancelProc := context.WithTimeout(context.Background(), client.ConnectTimeout)
	defer cancelProc()

	configuration := common.FromEnv()

	nsmClient, err := client.NewNSMClient(ctx, configuration)
	if err != nil {
		logrus.Fatalf("Unable to create the NSM client %v", err)
	}

	err = backend.New()
	if err != nil {
		logrus.Fatalf("Unable to create the backend %v", err)
	}

	ctx, cancelProc = context.WithTimeout(context.Background(), client.ConnectionRetry*(client.RequestDelay+client.ConnectTimeout))
	defer cancelProc()

	var outgoingConnection *networkservice.Connection
	outgoingConnection, err = nsmClient.ConnectRetry(ctx, "if1", memif.MECHANISM, "Primary interface", client.ConnectionRetry, client.RequestDelay)
	if err != nil {
		logrus.Fatalf("Unable to connect %v", err)
	}

	err = backend.Connect(outgoingConnection)
	if err != nil {
		logrus.Fatalf("Unable to connect %v", err)
	}

	connMonitor := nsmmonitor.NewNSMMonitorApp(common.FromEnv())
	connMonitor.SetHandler(backend)
	go connMonitor.Run()
	logrus.Info("nsm client: initialization is completed successfully, wait for Ctrl+C...")
	var wg sync.WaitGroup
	wg.Add(1)

	<-c
}
