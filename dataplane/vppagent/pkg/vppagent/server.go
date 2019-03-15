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

package vppagent

import (
	"os"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/monitor/crossconnect_monitor"
	"github.com/sirupsen/logrus"

	"github.com/grpc-ecosystem/grpc-opentracing/go/otgrpc"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/crossconnect"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/monitor_crossconnect_server"
	"github.com/networkservicemesh/networkservicemesh/dataplane/pkg/apis/dataplane"
	"github.com/networkservicemesh/networkservicemesh/dataplane/pkg/common"
	"github.com/opentracing/opentracing-go"
	"google.golang.org/grpc"
)

func NewServer(baseDir string, egressInterface *common.EgressInterface) *grpc.Server {
	tracer := opentracing.GlobalTracer()
	server := grpc.NewServer(
		grpc.UnaryInterceptor(
			otgrpc.OpenTracingServerInterceptor(tracer, otgrpc.LogPayloads())),
		grpc.StreamInterceptor(
			otgrpc.OpenTracingStreamServerInterceptor(tracer)))

	monitor := crossconnect_monitor.NewCrossConnectMonitor()
	crossconnect.RegisterMonitorCrossConnectServer(server, monitor)

	vppAgentEndpoint, ok := os.LookupEnv(common.DataplaneVPPAgentEndpointKey)
	if !ok {
		logrus.Infof("%s not set, using default %s", common.DataplaneVPPAgentEndpointKey, common.DefaultVPPAgentEndpoint)
		vppAgentEndpoint = common.DefaultVPPAgentEndpoint
	}
	logrus.Infof("vppAgentEndpoint: %s", vppAgentEndpoint)

	vppagent := NewVPPAgent(vppAgentEndpoint, monitor, baseDir, egressInterface)
	monitor_crossconnect_server.NewMonitorNetNsInodeServer(monitor)
	dataplane.RegisterDataplaneServer(server, vppagent)
	return server
}
