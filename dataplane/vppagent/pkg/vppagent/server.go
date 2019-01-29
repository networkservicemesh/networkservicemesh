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
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/monitor/crossconnect_monitor"
	"net"

	"github.com/grpc-ecosystem/grpc-opentracing/go/otgrpc"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/crossconnect"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/monitor_crossconnect_server"
	"github.com/networkservicemesh/networkservicemesh/dataplane/pkg/apis/dataplane"
	"github.com/opentracing/opentracing-go"
	"google.golang.org/grpc"
)

func NewServer(vppAgentEndpoint string, baseDir string, srcIp net.IP, srcIPNet net.IPNet, mgmtIfaceName string) *grpc.Server {
	tracer := opentracing.GlobalTracer()
	server := grpc.NewServer(
		grpc.UnaryInterceptor(
			otgrpc.OpenTracingServerInterceptor(tracer, otgrpc.LogPayloads())),
		grpc.StreamInterceptor(
			otgrpc.OpenTracingStreamServerInterceptor(tracer)))

	monitor := crossconnect_monitor.NewCrossConnectMonitor()
	crossconnect.RegisterMonitorCrossConnectServer(server, monitor)

	vppagent := NewVPPAgent(vppAgentEndpoint, monitor, baseDir, srcIp, srcIPNet, mgmtIfaceName)
	monitor_crossconnect_server.NewMonitorNetNsInodeServer(monitor)
	dataplane.RegisterDataplaneServer(server, vppagent)
	return server
}
