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

package kernelforwarder

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/status"

	"github.com/networkservicemesh/networkservicemesh/dataplane/pkg/common"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/crossconnect"
	local "github.com/networkservicemesh/networkservicemesh/controlplane/api/local/connection"
	remote "github.com/networkservicemesh/networkservicemesh/controlplane/api/remote/connection"
	"github.com/networkservicemesh/networkservicemesh/dataplane/api/dataplane"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
)

// KernelForwarder instance
type KernelForwarder struct {
	common *common.DataplaneConfig
}

// CreateKernelForwarder creates an instance of the KernelForwarder
func CreateKernelForwarder() *KernelForwarder {
	return &KernelForwarder{}
}

// MonitorMechanisms handler
func (v *KernelForwarder) MonitorMechanisms(empty *empty.Empty, updateSrv dataplane.MechanismsMonitor_MonitorMechanismsServer) error {
	logrus.Infof("MonitorMechanisms was called")
	initialUpdate := &dataplane.MechanismUpdate{
		RemoteMechanisms: v.common.Mechanisms.RemoteMechanisms,
		LocalMechanisms:  v.common.Mechanisms.LocalMechanisms,
	}
	logrus.Infof("Sending MonitorMechanisms update: %v", initialUpdate)
	if err := updateSrv.Send(initialUpdate); err != nil {
		logrus.Errorf("Kernel forwarding plane server: Detected error %s, grpc code: %+v on grpc channel", err.Error(), status.Convert(err).Code())
		return nil
	}
	// Waiting for any updates which might occur during a life of dataplane module and communicating
	// them back to NSM.
	for update := range v.common.MechanismsUpdateChannel {
		v.common.Mechanisms = update
		logrus.Infof("Sending MonitorMechanisms update: %v", update)
		if err := updateSrv.Send(&dataplane.MechanismUpdate{
			RemoteMechanisms: update.RemoteMechanisms,
			LocalMechanisms:  update.LocalMechanisms,
		}); err != nil {
			logrus.Errorf("Kernel forwarding plane server: Detected error %s, grpc code: %+v on grpc channel", err.Error(), status.Convert(err).Code())
			return nil
		}
	}
	return nil
}

// Request handler for connections
func (v *KernelForwarder) Request(ctx context.Context, crossConnect *crossconnect.CrossConnect) (*crossconnect.CrossConnect, error) {
	logrus.Infof("Request() called with %v", crossConnect)
	xcon, err := v.connectOrDisconnect(crossConnect, cCONNECT)
	if err != nil {
		return nil, err
	}
	v.common.Monitor.Update(ctx, xcon)
	logrus.Infof("Request() called with %v returning: %v", crossConnect, xcon)
	return xcon, err
}

func (v *KernelForwarder) connectOrDisconnect(crossConnect *crossconnect.CrossConnect, connect bool) (*crossconnect.CrossConnect, error) {
	/* 0. Sanity check whether the forwarding plane supports the connection type in the request */
	if err := common.SanityCheckConnectionType(v.common.Mechanisms, crossConnect); err != nil {
		return crossConnect, err
	}
	/* 1. Handle local connection */
	if crossConnect.GetLocalSource().GetMechanism().GetType() == local.MechanismType_KERNEL_INTERFACE &&
		crossConnect.GetLocalDestination().GetMechanism().GetType() == local.MechanismType_KERNEL_INTERFACE {
		return handleLocalConnection(crossConnect, connect)
	}
	/* 2. Handle remote connection */
	return handleRemoteConnection(v.common.EgressInterface, crossConnect, connect)
}

// Close handler for connections
func (v *KernelForwarder) Close(ctx context.Context, crossConnect *crossconnect.CrossConnect) (*empty.Empty, error) {
	logrus.Infof("Close() called with %#v", crossConnect)
	xcon, err := v.connectOrDisconnect(crossConnect, cDISCONNECT)
	if err != nil {
		logrus.Warn(err)
	}
	v.common.Monitor.Delete(ctx, xcon)
	return &empty.Empty{}, nil
}

// Init initializes the Kernel forwarding plane
func (v *KernelForwarder) Init(common *common.DataplaneConfig) error {
	v.common = common
	v.common.Name = "kernel-forwarder"

	if tools.IsOpentracingEnabled() {
		tracer, closer := tools.InitJaeger(v.common.Name)
		opentracing.SetGlobalTracer(tracer)
		defer func() {
			if err := closer.Close(); err != nil {
				logrus.Error("error when closing:", err)
			}
		}()
	}
	v.configureKernelForwarder()
	return nil
}

// configureKernelForwarder setups the Kernel forwarding plane
func (v *KernelForwarder) configureKernelForwarder() {
	v.common.MechanismsUpdateChannel = make(chan *common.Mechanisms, 1)
	v.common.Mechanisms = &common.Mechanisms{
		LocalMechanisms: []*local.Mechanism{
			{
				Type: local.MechanismType_KERNEL_INTERFACE,
			},
		},
		RemoteMechanisms: []*remote.Mechanism{
			{
				Type: remote.MechanismType_VXLAN,
				Parameters: map[string]string{
					remote.VXLANSrcIP: v.common.EgressInterface.SrcIPNet().IP.String(),
				},
			},
		},
	}
}
