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
	"context"
	"github.com/networkservicemesh/networkservicemesh/security/manager"

	"github.com/gogo/protobuf/proto"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/monitor"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/monitor/local"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
	"github.com/networkservicemesh/networkservicemesh/sdk/client"
)

const (
	nscLogFormat          = "NSM Client: %v"
	nscLogWithParamFormat = "NSM Client: %v: %v"
)

var version string

func main() {
	logrus.Info("Starting monitoring-nsc...")
	logrus.Infof("Version: %v", version)
	// Capture signals to cleanup before exiting
	c := tools.NewOSSignalChannel()
	security.NewCertificateManager()

	tracer, closer := tools.InitJaeger("nsc")
	opentracing.SetGlobalTracer(tracer)
	defer func() { _ = closer.Close() }()

	nsc, err := client.NewNSMClient(context.Background(), nil)
	if err != nil {
		logrus.Fatalf(nscLogWithParamFormat, "Unable to create the NSM client", err)
	}

	currentConn, err := nsc.Connect("nsm", "kernel", "Primary interface")
	if err != nil {
		logrus.Fatalf(nscLogWithParamFormat, "Failed to connect", err)
	}

	logrus.Info(nscLogFormat, "nsm client: initialization is completed successfully")

	monitorClient, err := local.NewMonitorClient(nsc.NsmConnection.GrpcClient)
	if err != nil {
		logrus.Fatalf(nscLogWithParamFormat, "Failed to start monitor client", err)
	}
	defer monitorClient.Close()

	for {
		select {
		case sig := <-c:
			logrus.Infof(nscLogWithParamFormat, "Received signal", sig)
			return
		case err = <-monitorClient.ErrorChannel():
			logrus.Fatalf(nscLogWithParamFormat, "Monitor failed", err)
		case event := <-monitorClient.EventChannel():
			if event.EventType() == monitor.EventTypeInitialStateTransfer {
				logrus.Infof(nscLogFormat, "Monitor started")
			}

			for _, entity := range event.Entities() {
				conn, ok := entity.(*connection.Connection)
				if !ok || conn.GetId() != currentConn.GetId() {
					continue
				}

				switch event.EventType() {
				case monitor.EventTypeInitialStateTransfer, monitor.EventTypeUpdate:
					if !proto.Equal(conn, currentConn) {
						logrus.Infof(nscLogWithParamFormat, "Connection updated", conn)
						currentConn = conn
					}
				case monitor.EventTypeDelete:
					logrus.Infof(nscLogFormat, "Connection closed")
					return
				}
			}
		}
	}
}
