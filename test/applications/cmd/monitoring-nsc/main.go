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

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/sidecars"

	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
	"github.com/networkservicemesh/networkservicemesh/sdk/client"
)

const (
	nscLogWithParamFormat = "NSM Client: %v: %v"
	nscLogFormat          = "NSM Client: %v"
)

var version string

func main() {
	logrus.Info("Starting monitoring-nsc...")
	logrus.Infof("Version: %v", version)
	// Capture signals to cleanup before exiting
	c := tools.NewOSSignalChannel()

	tracer, closer := tools.InitJaeger("nsc")
	opentracing.SetGlobalTracer(tracer)
	defer func() { _ = closer.Close() }()

	nsc, err := client.NewNSMClient(context.Background(), nil)
	if err != nil {
		logrus.Fatalf(nscLogWithParamFormat, "Unable to create the NSM client", err)
	}
	logrus.Info(nscLogFormat, "nsm client: initialization is completed successfully")
	_, err = nsc.Connect("nsm", "kernel", "Primary interface")
	if err != nil {
		logrus.Fatalf(nscLogWithParamFormat, "Failed to connect", err)
	}
	monitor := sidecars.NewNSMMonitorApp()
	monitor.Run()
	<-c
}
