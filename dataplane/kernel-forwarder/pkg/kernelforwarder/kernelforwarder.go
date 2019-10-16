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
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/status"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/crossconnect"
	local "github.com/networkservicemesh/networkservicemesh/controlplane/api/local/connection"
	remote "github.com/networkservicemesh/networkservicemesh/controlplane/api/remote/connection"
	"github.com/networkservicemesh/networkservicemesh/forwarder/api/forwarder"
	"github.com/networkservicemesh/networkservicemesh/forwarder/kernel-forwarder/pkg/monitoring"
	"github.com/networkservicemesh/networkservicemesh/forwarder/pkg/common"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools/jaeger"
)

// KernelForwarder instance
type KernelForwarder struct {
	common     *common.DataplaneConfig
	monitoring *monitoring.Metrics
}

// CreateKernelForwarder creates an instance of the KernelForwarder
func CreateKernelForwarder() *KernelForwarder {
	return &KernelForwarder{}
}

// Init initializes the Kernel forwarding plane
func (k *KernelForwarder) Init(common *common.DataplaneConfig) error {
	k.common = common
	k.common.Name = "kernel-forwarder"

	closer := jaeger.InitJaeger(k.common.Name)
	defer func() { _ = closer.Close() }()

	k.configureKernelForwarder()
	return nil
}

// CreateDataplaneServer creates an instance of DataplaneServer
func (k *KernelForwarder) CreateDataplaneServer(config *common.DataplaneConfig) forwarder.DataplaneServer {
	return k
}

// Request handler for connections
func (k *KernelForwarder) Request(ctx context.Context, crossConnect *crossconnect.CrossConnect) (*crossconnect.CrossConnect, error) {
	logrus.Infof("Request() called with %v", crossConnect)
	err := k.connectOrDisconnect(crossConnect, cCONNECT)
	if err != nil {
		logrus.Warn("error while handling Request() connection:", err)
		return nil, err
	}
	k.common.Monitor.Update(ctx, crossConnect)
	return crossConnect, err
}

// Close handler for connections
func (k *KernelForwarder) Close(ctx context.Context, crossConnect *crossconnect.CrossConnect) (*empty.Empty, error) {
	logrus.Infof("Close() called with %#v", crossConnect)
	err := k.connectOrDisconnect(crossConnect, cDISCONNECT)
	if err != nil {
		logrus.Warn("error while handling Close() connection:", err)
	}
	k.common.Monitor.Delete(ctx, crossConnect)
	return &empty.Empty{}, nil
}

func (k *KernelForwarder) connectOrDisconnect(crossConnect *crossconnect.CrossConnect, connect bool) error {
	var err error
	var devices map[string]monitoring.Device

	if k.common.MetricsEnabled {
		k.monitoring.GetDevices().Lock()
		defer k.monitoring.GetDevices().Unlock()
	}

	/* 0. Sanity check whether the forwarding plane supports the connection type in the request */
	if err = common.SanityCheckConnectionType(k.common.Mechanisms, crossConnect); err != nil {
		return err
	}

	/* 1. Handle local connection */
	if crossConnect.GetLocalSource().GetMechanism().GetType() == local.MechanismType_KERNEL_INTERFACE &&
		crossConnect.GetLocalDestination().GetMechanism().GetType() == local.MechanismType_KERNEL_INTERFACE {
		devices, err = handleLocalConnection(crossConnect, connect)
	} else {
		/* 2. Handle remote connection */
		devices, err = handleRemoteConnection(k.common.EgressInterface, crossConnect, connect)
	}
	if devices != nil && err == nil {
		if connect {
			logrus.Info("kernel-forwarder: created devices: ", devices)
		} else {
			logrus.Info("kernel-forwarder: deleted devices: ", devices)
		}
		// Metrics monitoring
		if k.common.MetricsEnabled {
			k.monitoring.GetDevices().UpdateDeviceList(devices, connect)
		}
	}
	return err
}

// configureKernelForwarder setups the Kernel forwarding plane
func (k *KernelForwarder) configureKernelForwarder() {
	k.common.MechanismsUpdateChannel = make(chan *common.Mechanisms, 1)
	k.common.Mechanisms = &common.Mechanisms{
		LocalMechanisms: []*local.Mechanism{
			{
				Type: local.MechanismType_KERNEL_INTERFACE,
			},
		},
		RemoteMechanisms: []*remote.Mechanism{
			{
				Type: remote.MechanismType_VXLAN,
				Parameters: map[string]string{
					remote.VXLANSrcIP: k.common.EgressInterface.SrcIPNet().IP.String(),
				},
			},
		},
	}
	// Metrics monitoring
	if k.common.MetricsEnabled {
		k.monitoring = monitoring.CreateMetricsMonitor(k.common.MetricsPeriod)
		k.monitoring.Start(k.common.Monitor)
	}
}

// MonitorMechanisms handler
func (k *KernelForwarder) MonitorMechanisms(empty *empty.Empty, updateSrv forwarder.MechanismsMonitor_MonitorMechanismsServer) error {
	initialUpdate := &forwarder.MechanismUpdate{
		RemoteMechanisms: k.common.Mechanisms.RemoteMechanisms,
		LocalMechanisms:  k.common.Mechanisms.LocalMechanisms,
	}
	logrus.Infof("kernel-forwarder: sending MonitorMechanisms update: %v", initialUpdate)
	if err := updateSrv.Send(initialUpdate); err != nil {
		logrus.Errorf("kernel-forwarder: detected server error %s, gRPC code: %+v on gRPC channel", err.Error(), status.Convert(err).Code())
		return nil
	}
	// Waiting for any updates which might occur during a life of forwarder module and communicating
	// them back to NSM.
	for update := range k.common.MechanismsUpdateChannel {
		k.common.Mechanisms = update
		logrus.Infof("kernel-forwarder: sending MonitorMechanisms update: %v", update)
		if err := updateSrv.Send(&forwarder.MechanismUpdate{
			RemoteMechanisms: update.RemoteMechanisms,
			LocalMechanisms:  update.LocalMechanisms,
		}); err != nil {
			logrus.Errorf("kernel-forwarder: detected server error %s, gRPC code: %+v on gRPC channel", err.Error(), status.Convert(err).Code())
			return nil
		}
	}
	return nil
}
