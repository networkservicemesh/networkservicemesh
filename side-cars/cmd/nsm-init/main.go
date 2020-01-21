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
	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/sdk/common"
	"github.com/networkservicemesh/networkservicemesh/utils"

	nsm_init "github.com/networkservicemesh/networkservicemesh/side-cars/pkg/nsm-init"
)

var version string

func main() {
	logrus.Info("Starting nsm-init...")
	logrus.Infof("Version: %v", version)
	utils.PrintAllEnv(logrus.StandardLogger())

	config := common.FromEnv()
	config.MechanismType = "SRIOV_KERNEL_INTERFACE"
	clientApp := nsm_init.NewNSMClientApp(config)
	clientApp.Run()
}
