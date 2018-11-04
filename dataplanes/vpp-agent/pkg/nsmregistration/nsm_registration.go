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

package nsmregistration

import (
	"context"
	"os"
	"path"
	"time"

	"github.com/ligato/networkservicemesh/dataplanes/vpp-agent/pkg/nsmdataplane"
	"github.com/ligato/networkservicemesh/dataplanes/vpp-agent/pkg/nsmvpp"
	"github.com/ligato/networkservicemesh/pkg/nsm/apis/common"
	dataplaneregistrarapi "github.com/ligato/networkservicemesh/pkg/nsm/apis/dataplaneregistrar"
	"github.com/ligato/networkservicemesh/pkg/tools"
	"github.com/ligato/networkservicemesh/plugins/dataplaneregistrar"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/status"
)

var (
	registrarSocket           = path.Join(dataplaneregistrar.DataplaneRegistrarSocketBaseDir, dataplaneregistrar.DataplaneRegistrarSocket)
	registrationRetryInterval = 30 * time.Second
)

// livenessMonitor is a stream initiated by NSM to inform the dataplane that NSM is still alive and
// no re-registration is required. Detection a failure on this "channel" will mean
// that NSM is gone and the dataplane needs to start re-registration logic.
func livenessMonitor(vpp *nsmvpp.VPPAgentClient, registrationConnection dataplaneregistrarapi.DataplaneRegistrationClient) {
	// The only reason liveness monitor exit is when it looses connection with NSM. When it happens, calling UnRegister to clear
	// possibly broken connectivity with NSM. At the end of UnRegister call a new attempt to re-register will be initiated.
	defer func(vpp *nsmvpp.VPPAgentClient) {
		logrus.Warnf("liveness monitor is about to exit, marking VPP Dataplane controller as Unregisteres and starting new registration routine.")
		go UnRegisterDataplane(vpp)
	}(vpp)

	stream, err := registrationConnection.RequestLiveness(context.Background())
	if err != nil {
		logrus.Errorf("nsm-vpp-dataplane: fail to create liveness grpc channel with NSM with error: %s, grpc code: %+v", err.Error(), status.Convert(err).Code())
		return
	}
	for {
		err := stream.RecvMsg(&common.Empty{})
		if err != nil {
			logrus.Errorf("nsm-vpp-dataplane: fail to receive from liveness grpc channel with error: %s, grpc code: %+v", err.Error(), status.Convert(err).Code())
			return
		}
	}
}

// RegisterDataplane is a go routine which is attempting to register vppDataplane controller with NSM.
// Once it is done, NSM liveness monitor will be started.
// Questions: Check for VPP disconnect, if disconnected exit it or just wait and retry?
func RegisterDataplane(vpp *nsmvpp.VPPAgentClient) {
	logrus.Info("Registration routine has been invoked")
	var registrarConnection dataplaneregistrarapi.DataplaneRegistrationClient
	ticker := time.NewTicker(registrationRetryInterval)
	for {
		select {
		case <-ticker.C:
			// First need to check if VPP state is actually connected, there is no reason to proceed with registration
			// if VPP is not in Connected state
			if !vpp.IsConnected() {
				logrus.Warn("nsm-vpp-dataplane: Delaying registration as VPP's state is not Connected.")
				continue
			}
			if _, err := os.Stat(registrarSocket); err != nil {
				logrus.Errorf("nsm-vpp-dataplane: failure to access nsm socket at %s with error: %+v, exiting...", registrarSocket, err)
				continue
			}
			conn, err := tools.SocketOperationCheck(registrarSocket)
			if err != nil {
				logrus.Errorf("nsm-vpp-dataplane: failure to communicate with the socket %s with error: %+v", registrarSocket, err)
				continue
			}
			logrus.Infof("nsm-vpp-dataplane: connection to dataplane registrar socket %s succeeded.", registrarSocket)

			registrarConnection = dataplaneregistrarapi.NewDataplaneRegistrationClient(conn)
			// TODO (sbezverk) Probably dataplane configuration should not be static. Consider adding dynamic
			// ways to configure the dataplane controller. Maybe ConfigMaps?
			dataplane := dataplaneregistrarapi.DataplaneRegistrationRequest{
				DataplaneName:   "nsm-vpp-dataplane",
				DataplaneSocket: nsmdataplane.DataplaneSocket,
			}
			if _, err := registrarConnection.RequestDataplaneRegistration(context.Background(), &dataplane); err != nil {
				logrus.Fatalf("nsm-vpp-dataplane: failure to communicate with the socket %s with error: %+v", registrarSocket, err)
				continue
			}
			logrus.Infof("nsm-vpp-dataplane: dataplane has successfully been registered, starting NSM's liveness monitor.")
			// Registration succeeded
			// Setting up UnRegister callback for the case of VPP Disconnected event
			//vpp.SetUnRegisterCallback(UnRegisterDataplane)
			// Marking VPP Dataplane controller as registered
			//vpp.SetRegistered()
			// Starting NSM liveness monitor
			go livenessMonitor(vpp, registrarConnection)
			return
		}
	}
}

// UnRegisterDataplane informs NSM that VPP Dataplane controller lost connectivity with VPP
// as a result, it cannot accept NSM's request to program clients' dataplane. In order to prevent NSM using this dataplane
// this call will remove VPP Dataplane controller from NSM's dataplane list.
func UnRegisterDataplane(vpp *nsmvpp.VPPAgentClient) {
	logrus.Info("UnRegistration routine has been invoked")
	var unRegistrarConnection dataplaneregistrarapi.DataplaneUnRegistrationClient

	// This function will be called right before any return. Even if we cannot gracefully unregister,
	// the dataplane controller will assume as major issue with NSM and mark itself as Unregistered anyway
	defer func(vpp *nsmvpp.VPPAgentClient) {
		logrus.Info("Unregister deferred function was called.")
		//vpp.SetUnRegistered()
		go RegisterDataplane(vpp)
	}(vpp)

	if _, err := os.Stat(registrarSocket); err != nil {
		logrus.Errorf("nsm-vpp-dataplane: failure to access nsm socket at %s with error: %+v, exiting...", registrarSocket, err)
		return
	}
	conn, err := tools.SocketOperationCheck(registrarSocket)
	if err != nil {
		logrus.Errorf("nsm-vpp-dataplane: failure to communicate with the socket %s with error: %+v", registrarSocket, err)
		return
	}
	defer conn.Close()
	logrus.Infof("nsm-vpp-dataplane: connection to dataplane registrar socket %s succeeded.", registrarSocket)

	unRegistrarConnection = dataplaneregistrarapi.NewDataplaneUnRegistrationClient(conn)
	dataplane := dataplaneregistrarapi.DataplaneUnRegistrationRequest{
		DataplaneName: "nsm-vpp-dataplane",
	}
	if _, err := unRegistrarConnection.RequestDataplaneUnRegistration(context.Background(), &dataplane); err != nil {
		logrus.Fatalf("nsm-vpp-dataplane: failure to communicate with the socket %s with error: %+v", registrarSocket, err)
		return
	}
	logrus.Infof("nsm-vpp-dataplane: dataplane has successfully been un-registered, starting re-registration routine.")
	// UnRegistration succeeded
	// Exiting and deferred function will do the magic.

	return
}
