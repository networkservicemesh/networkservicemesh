// Copyright 2019 VMware, Inc.
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
	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/k8s/pkg/probes"

	"github.com/networkservicemesh/networkservicemesh/dataplane/kernel-forwarder/pkg/kernelforwarder"
	"github.com/networkservicemesh/networkservicemesh/dataplane/pkg/common"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
)

func main() {
	// Capture signals to cleanup before exiting
	logrus.Info("Starting the Kernel-based forwarding plane!")
	c := tools.NewOSSignalChannel()
	dataplaneGoals := &common.DataplaneProbeGoals{}
	dataplaneProbes := probes.NewProbes("Kerner-based forwarding plane liveness/readiness healthcheck", dataplaneGoals)
	dataplaneProbes.BeginHealthCheck()

	plane := kernelforwarder.CreateKernelForwarder()

	registration := common.CreateDataplane(plane, dataplaneGoals)

	for range c {
		logrus.Info("Closing Dataplane Registration")
		registration.Close()
	}
}
