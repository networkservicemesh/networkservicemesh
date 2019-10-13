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

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection/mechanisms/kernel"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools/jaeger"

	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/sdk/common"

	nsmmonitor "github.com/networkservicemesh/networkservicemesh/side-cars/pkg/nsm-monitor"

	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
	"github.com/networkservicemesh/networkservicemesh/sdk/client"
)

const (
	nscLogWithParamFormat = "NSM Client: %v: %v"
	nscLogFormat          = "NSM Client: %v"
)

var version string

func main() {
	// Capture signals to cleanup before exiting
	c := tools.NewOSSignalChannel()
	logrus.Info("Starting monitoring-nsc...")
	logrus.Infof("Version: %v", version)

	closer := jaeger.InitJaeger("monitoring-nsc")
	defer func() { _ = closer.Close() }()

	configuration := common.FromEnv()
	nsc, err := client.NewNSMClient(context.Background(), configuration)
	if err != nil {
		logrus.Fatalf(nscLogWithParamFormat, "Unable to create the NSM client", err)
	}
	logrus.Info(nscLogFormat, "nsm client: initialization is completed successfully")

	_, err = nsc.ConnectRetry(context.Background(), "nsm", kernel.MECHANISM, "Primary interface", client.ConnectionRetry, client.RequestDelay)
	if err != nil {
		logrus.Fatalf(nscLogWithParamFormat, "Failed to connect", err)
	}
	monitor := nsmmonitor.NewNSMMonitorApp(configuration)
	monitor.Run()
	<-c
}
