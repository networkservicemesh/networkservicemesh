// Copyright 2018 VMware, Inc.
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
	"context"
	"time"

	"github.com/grpc-ecosystem/grpc-opentracing/go/otgrpc"
	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/crossconnect"
	"github.com/ligato/networkservicemesh/dataplane/vppagent/pkg/converter"
	"github.com/ligato/networkservicemesh/pkg/tools"
	"github.com/ligato/vpp-agent/plugins/vpp/model/rpc"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

func (ns *vppagentBackend) ApplyAclOnVppInterface(ctx context.Context, aclname, ifname string, rules map[string]string) error {
	conn, err := grpc.Dial(ns.vppAgentEndpoint, grpc.WithInsecure())
	if err != nil {
		logrus.Errorf("can't dial grpc server: %v", err)
		return err
	}
	defer conn.Close()
	client := rpc.NewDataChangeServiceClient(conn)

	dataChange, err := converter.NewAclConverter(aclname, ifname, rules).ToDataRequest(nil, true)

	if err != nil {
		logrus.Error(err)
		return err
	}
	logrus.Infof("Sending DataChange to vppagent: %v", dataChange)
	if _, err := client.Put(ctx, dataChange); err != nil {
		logrus.Error(err)
		client.Del(ctx, dataChange)
		return err
	}
	return nil
}

func (ns *vppagentBackend) CrossConnecVppInterfaces(ctx context.Context, crossConnect *crossconnect.CrossConnect, connect bool, baseDir string) (string, error) {

	logrus.Infof("ns.vppAgentEndpoint %v", ns.vppAgentEndpoint)

	conn, err := grpc.Dial(ns.vppAgentEndpoint, grpc.WithInsecure())
	if err != nil {
		logrus.Errorf("can't dial grpc server: %v", err)
		return "", err
	}
	defer conn.Close()
	client := rpc.NewDataChangeServiceClient(conn)

	conversionParameters := &converter.CrossConnectConversionParameters{
		BaseDir: baseDir,
	}

	xconnConverter := converter.NewCrossConnectConverter(crossConnect, conversionParameters)

	dataChange, err := xconnConverter.ToDataRequest(connect)

	if err != nil {
		logrus.Error(err)
		return "", err
	}

	for _, dc := range dataChange {
		logrus.Infof("Sending DataChange to vppagent: %v", dataChange)
		if connect {
			_, err = client.Put(ctx, dc)
		} else {
			_, err = client.Del(ctx, dc)
		}
		if err != nil {
			logrus.Error(err)
			// TODO handle connection tracking
			// TODO handle teardown of any partial config that happened
			return "", err
		}
	}

	return dataChange[0].GetInterfaces()[0].GetName(), nil
}

func (ns *vppagentBackend) Reset() error {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	tools.WaitForPortAvailable(ctx, "tcp", ns.vppAgentEndpoint, 100*time.Millisecond)

	tracer := opentracing.GlobalTracer()
	conn, err := grpc.Dial(ns.vppAgentEndpoint, grpc.WithInsecure(),
		grpc.WithUnaryInterceptor(
			otgrpc.OpenTracingClientInterceptor(tracer, otgrpc.LogPayloads())),
		grpc.WithStreamInterceptor(
			otgrpc.OpenTracingStreamClientInterceptor(tracer)))

	if err != nil {
		logrus.Errorf("can't dial grpc server: %v", err)
		return err
	}
	defer conn.Close()
	client := rpc.NewDataResyncServiceClient(conn)
	logrus.Infof("Resetting vppagent...")
	_, err = client.Resync(context.Background(), &rpc.DataRequest{})
	if err != nil {
		logrus.Errorf("failed to reset vppagent: %s", err)
	}
	logrus.Infof("Finished resetting vppagent...")
	return nil
}
