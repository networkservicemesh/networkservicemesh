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

	"github.com/networkservicemesh/networkservicemesh/pkg/tools/jaeger"

	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/pkg/tools/spanhelper"

	"github.com/networkservicemesh/networkservicemesh/pkg/probes"

	"github.com/networkservicemesh/networkservicemesh/forwarder/pkg/common"
	"github.com/networkservicemesh/networkservicemesh/forwarder/vppagent/pkg/vppagent"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
)

var version string

func main() {
	logrus.Info("Starting vppagent-forwarder...")
	logrus.Infof("Version: %v", version)

	closer := jaeger.InitJaeger("vppagent-forwarder")
	defer func() { _ = closer.Close() }()

	span := spanhelper.FromContext(context.Background(), "Start.VPPAgent.Forwarder")
	defer span.Finish()
	// Capture signals to cleanup before exiting
	c := tools.NewOSSignalChannel()
	forwarderGoals := &common.ForwarderProbeGoals{}
	forwarderProbes := probes.New("Vppagent forwarder liveness/readiness healthcheck", forwarderGoals)
	forwarderProbes.BeginHealthCheck()

	agent := vppagent.CreateVPPAgent()

	registration := common.CreateForwarder(span.Context(), agent, forwarderGoals)

	span.Finish()
	<-c
	logrus.Info("Closing Forwarder Registration")
	registration.Close()
}
