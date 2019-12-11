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
	"flag"
	"os"

	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/utils"
)

var version string

func main() {
	logrus.Info("Starting nsm-generate-sriov-configmap...")
	logrus.Infof("Version: %v", version)
	utils.PrintAllEnv(logrus.StandardLogger())
	flag.Set("logtostderr", "true")
	flag.Parse()
	discoveredVFs := newVFs()
	if err := discoverNetworks(discoveredVFs); err != nil {
		logrus.Errorf("%+v", err)
		os.Exit(1)
	}
	if len(discoveredVFs.vfs) == 0 {
		logrus.Info("no VF were discovered, exiting...")
		os.Exit(0)
	}
	logrus.Infof("%d VFs were discovered on the host.", len(discoveredVFs.vfs))
	// Check if noRebind is selected
	if !*noRebind {
		// Building vfio device for each VF
		if err := buildVFIODevices(discoveredVFs); err != nil {
			logrus.Errorf("failed to build VFIO devices for VFs with error: %+v", err)
			os.Exit(1)
		}
	}
	// Check if noConfigMap is selected
	if !*noConfigMap {
		if err := generateConfigMap(discoveredVFs); err != nil {
			logrus.Errorf("%+v", err)
			os.Exit(1)
		}
	}
}
