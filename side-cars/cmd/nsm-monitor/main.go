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
	"github.com/sirupsen/logrus"

	nsm_sidecars "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/sidecars"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
)

var version string

func main() {
	// Capture signals to cleanup before exiting
	c := tools.NewOSSignalChannel()
	logrus.Infof("Starting nsm-monitor....")
	logrus.Infof("Version: %v", version)
	app := nsm_sidecars.NewNSMMonitorApp()

	if MonitorDNSConfigsEnv.GetBooleanOrDefault(false) {
		app.SetHandler(nsm_sidecars.NewNsmDNSMonitorHandler(
			PathToCorefileEnv.GetStringOrDefault(nsm_sidecars.DefaultPathToCorefile),
			ReloadCorefilePeriodEnv.GetOrDefaultDuration(nsm_sidecars.DefaultReloadCorefileTime)))
	}

	go app.Run()
	<-c
}
