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
	"context"

	"github.com/ligato/networkservicemesh/pkg/nsm/apis/common"
	"github.com/ligato/networkservicemesh/pkg/nsm/apis/dataplane"
	"github.com/ligato/vpp-agent/plugins/vpp/model/rpc"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

type VPPAgent struct {
	// Parameters set in constructor
	vppAgentEndpoint string
	mechanisms       *Mechanisms

	// Internal state
	connections map[string]*dataplane.Connection
	updateCh    chan *Mechanisms
}

func NewVPPAgent(vppAgentEndpoint string) *VPPAgent {
	// TODO provide some validations here for inputs
	return &VPPAgent{
		connections:      make(map[string]*dataplane.Connection),
		vppAgentEndpoint: vppAgentEndpoint,
		mechanisms: &Mechanisms{
			localMechanisms: []*common.LocalMechanism{
				&common.LocalMechanism{
					Type: common.LocalMechanismType_KERNEL_INTERFACE,
				},
				&common.LocalMechanism{
					Type: common.LocalMechanismType_MEM_INTERFACE,
				},
			},
		},
	}
}

// Mechanisms is a message used to communicate any changes in operational parameters and constraints
type Mechanisms struct {
	remoteMechanisms []*common.RemoteMechanism
	localMechanisms  []*common.LocalMechanism
}

func (v VPPAgent) MonitorMechanisms(empty *common.Empty, updateSrv dataplane.DataplaneOperations_MonitorMechanismsServer) error {
	logrus.Infof("MonitorMechanisms was called")
	if err := updateSrv.Send(&dataplane.MechanismUpdate{
		RemoteMechanisms: v.mechanisms.remoteMechanisms,
		LocalMechanisms:  v.mechanisms.localMechanisms,
	}); err != nil {
		logrus.Errorf("vpp-agent dataplane server: Detected error %s, grpc code: %+v on grpc channel", err.Error(), status.Convert(err).Code())
		return nil
	}
	for {
		select {
		// Waiting for any updates which might occur during a life of dataplane module and communicating
		// them back to NSM.
		case update := <-v.updateCh:
			v.mechanisms = update
			if err := updateSrv.Send(&dataplane.MechanismUpdate{
				RemoteMechanisms: update.remoteMechanisms,
				LocalMechanisms:  update.localMechanisms,
			}); err != nil {
				logrus.Errorf("vpp dataplane server: Deteced error %s, grpc code: %+v on grpc channel", err.Error(), status.Convert(err).Code())
				return nil
			}
		}
	}
}

func (v VPPAgent) ConnectRequest(ctx context.Context, connection *dataplane.Connection) (*dataplane.Reply, error) {
	logrus.Infof("vppagent.ConnectRequest called with %#v", connection)
	return v.ConnectOrDisConnect(ctx, connection, true)
}

func (v VPPAgent) ConnectOrDisConnect(ctx context.Context, connection *dataplane.Connection, connect bool) (*dataplane.Reply, error) {
	// TODO look at whether keepin a single conn might be better
	conn, err := grpc.Dial(v.vppAgentEndpoint, grpc.WithInsecure())
	if err != nil {
		logrus.Errorf("can't dial grpc server: %v", err)
		return nil, err
	}
	defer conn.Close()
	client := rpc.NewDataChangeServiceClient(conn)
	dataChange, err := DataRequestFromConnection(connection, nil)
	if err != nil {
		logrus.Error(err)
		return nil, err
	}
	logrus.Infof("Sending DataChange to vppagent: %v", dataChange)
	if connect {
		_, err = client.Put(ctx, dataChange)
	} else {
		_, err = client.Del(ctx, dataChange)
	}
	if err != nil {
		logrus.Error(err)
		// TODO handle connection tracking
		return &dataplane.Reply{
			Connection:    connection,
			Success:       false,
			ExtendedError: err.Error(),
		}, err
	}
	return &dataplane.Reply{
		Connection: connection,
		Success:    true,
	}, nil
}

func (v VPPAgent) DisconnectRequest(ctx context.Context, connection *dataplane.Connection) (*dataplane.Reply, error) {
	logrus.Infof("vppagent.DisconnectRequest called with %#v", connection)
	return v.ConnectOrDisConnect(ctx, connection, false)
}
