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
	"fmt"
	"os"
	"path"
	"strconv"

	"github.com/docker/docker/pkg/mount"

	"github.com/golang/protobuf/ptypes/empty"

	local "github.com/ligato/networkservicemesh/controlplane/pkg/apis/local/connection"
	remote "github.com/ligato/networkservicemesh/controlplane/pkg/apis/remote/connection"
	"github.com/ligato/networkservicemesh/dataplane/pkg/apis/dataplane"
	"github.com/ligato/networkservicemesh/dataplane/vppagent/pkg/converter"
	"github.com/ligato/vpp-agent/plugins/vpp/model/rpc"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

const (
	MemifBaseDirectory = "/memif"
)

type VPPAgent struct {
	// Parameters set in constructor
	vppAgentEndpoint string
	mechanisms       *Mechanisms

	// Internal state from here on
	updateCh chan *Mechanisms

	crossConnectEventCh      chan *dataplane.CrossConnectEvent
	newMonitorRecipientCh    chan dataplane.Dataplane_MonitorCrossConnectsServer
	closedMonitorRecipientCh chan dataplane.Dataplane_MonitorCrossConnectsServer

	// monitorRecipients and crossConnects should only ever be updated by the monitor() method
	monitorRecipients []dataplane.Dataplane_MonitorCrossConnectsServer
	crossConnects     map[string]*dataplane.CrossConnect
}

func NewVPPAgent(vppAgentEndpoint string) *VPPAgent {
	// TODO provide some validations here for inputs
	rv := &VPPAgent{
		crossConnects:         make(map[string]*dataplane.CrossConnect),
		updateCh:              make(chan *Mechanisms, 1),
		crossConnectEventCh:   make(chan *dataplane.CrossConnectEvent, 10),
		newMonitorRecipientCh: make(chan dataplane.Dataplane_MonitorCrossConnectsServer, 10),
		vppAgentEndpoint:      vppAgentEndpoint,
		mechanisms: &Mechanisms{
			localMechanisms: []*local.Mechanism{
				&local.Mechanism{
					Type: local.MechanismType_KERNEL_INTERFACE,
				},
				&local.Mechanism{
					Type: local.MechanismType_MEM_INTERFACE,
				},
			},
		},
	}
	rv.reset()
	go rv.monitorCrossConnects()
	return rv
}

// Mechanisms is a message used to communicate any changes in operational parameters and constraints
type Mechanisms struct {
	remoteMechanisms []*remote.Mechanism
	localMechanisms  []*local.Mechanism
}

func (v *VPPAgent) MonitorMechanisms(empty *empty.Empty, updateSrv dataplane.Dataplane_MonitorMechanismsServer) error {
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

func (v *VPPAgent) Request(ctx context.Context, crossConnect *dataplane.CrossConnect) (*dataplane.CrossConnect, error) {
	logrus.Infof("Request(ConnectRequest) called with %v", crossConnect)
	xcon, err := v.ConnectOrDisConnect(ctx, crossConnect, true)
	xconEvent := &dataplane.CrossConnectEvent{
		Type: dataplane.CrossConnectEventType_UPDATE,
		CrossConnects: map[string]*dataplane.CrossConnect{
			xcon.GetId(): xcon,
		},
	}
	v.crossConnectEventCh <- xconEvent
	return xcon, err
}

func createDirectory(path string) error {
	if err := os.MkdirAll(path, 0777); err != nil {
		return err
	}
	logrus.Infof("Create directory: %s", path)
	return nil
}

func buildMemifDirectory(mechanism *local.Mechanism) string {
	return path.Join(mechanism.Parameters[local.Workspace], MemifBaseDirectory)
}

func masterSlave(src, dst *local.Mechanism) (*local.Mechanism, *local.Mechanism) {
	if isMaster, _ := strconv.ParseBool(src.GetParameters()[local.Master]); isMaster {
		return src, dst
	}
	return dst, src
}

func (v *VPPAgent) ConnectOrDisConnect(ctx context.Context, crossConnect *dataplane.CrossConnect, connect bool) (*dataplane.CrossConnect, error) {
	if crossConnect.GetLocalSource().GetMechanism().GetType() == local.MechanismType_MEM_INTERFACE &&
		crossConnect.GetLocalDestination().GetMechanism().GetType() == local.MechanismType_MEM_INTERFACE {

		//memif direct connection
		srcMechanism := crossConnect.GetLocalSource().GetMechanism()
		dstMechanism := crossConnect.GetLocalDestination().GetMechanism()
		master, slave := masterSlave(srcMechanism, dstMechanism)

		masterSocketDir := path.Join(buildMemifDirectory(master), crossConnect.Id)
		slaveSocketDir := path.Join(buildMemifDirectory(slave), crossConnect.Id)

		if err := createDirectory(masterSocketDir); err != nil {
			return nil, err
		}

		if err := createDirectory(slaveSocketDir); err != nil {
			return nil, err
		}

		if err := mount.Mount(masterSocketDir, slaveSocketDir, "hard", "bind"); err != nil {
			return nil, err
		}
		logrus.Infof("Successfully mount folder %s to %s", masterSocketDir, slaveSocketDir)

		if master.GetParameters()[local.SocketFilename] != slave.GetParameters()[local.SocketFilename] {
			masterSocket := path.Join(masterSocketDir, master.GetParameters()[local.SocketFilename])
			slaveSocket := path.Join(slaveSocketDir, slave.GetParameters()[local.SocketFilename])

			if err := os.Symlink(masterSocket, slaveSocket); err != nil {
				return nil, fmt.Errorf("failed to create symlink: %s", err)
			}
		}

		return crossConnect, nil
	}

	// TODO look at whether keepin a single conn might be better
	conn, err := grpc.Dial(v.vppAgentEndpoint, grpc.WithInsecure())
	if err != nil {
		logrus.Errorf("can't dial grpc server: %v", err)
		return nil, err
	}
	defer conn.Close()
	client := rpc.NewDataChangeServiceClient(conn)
	dataChange, err := converter.NewCrossConnectConverter(crossConnect).ToDataRequest(nil)
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
		// TODO handle teardown of any partial config that happened
		return crossConnect, err
	}
	return crossConnect, nil
}

func (v *VPPAgent) reset() error {
	conn, err := grpc.Dial(v.vppAgentEndpoint, grpc.WithInsecure())
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

func (v *VPPAgent) Close(ctx context.Context, crossConnect *dataplane.CrossConnect) (*empty.Empty, error) {
	logrus.Infof("vppagent.DisconnectRequest called with %#v", crossConnect)
	xcon, err := v.ConnectOrDisConnect(ctx, crossConnect, false)
	xconEvent := &dataplane.CrossConnectEvent{
		Type: dataplane.CrossConnectEventType_DELETE,
		CrossConnects: map[string]*dataplane.CrossConnect{
			xcon.GetId(): xcon,
		},
	}
	v.crossConnectEventCh <- xconEvent
	return &empty.Empty{}, err
}

func (v *VPPAgent) MonitorCrossConnects(_ *empty.Empty, recipient dataplane.Dataplane_MonitorCrossConnectsServer) error {
	v.newMonitorRecipientCh <- recipient
	go func() {
		select {
		case <-recipient.Context().Done():
			v.closedMonitorRecipientCh <- recipient
		}
	}()
	return nil
}

func (v *VPPAgent) monitorCrossConnects() {
	for {
		select {
		case newRecipient := <-v.newMonitorRecipientCh:
			initialStateTransferEvent := &dataplane.CrossConnectEvent{
				Type:          dataplane.CrossConnectEventType_INITIAL_STATE_TRANSFER,
				CrossConnects: v.crossConnects,
			}
			newRecipient.Send(initialStateTransferEvent)
			v.monitorRecipients = append(v.monitorRecipients, newRecipient)
			// TODO handle case where a monitorRecipient goes away
		case closedRecipent := <-v.closedMonitorRecipientCh:
			for j, r := range v.monitorRecipients {
				if r == closedRecipent {
					v.monitorRecipients = append(v.monitorRecipients[:j], v.monitorRecipients[j+1:]...)
					break
				}
			}
		case crossConnectEvent := <-v.crossConnectEventCh:
			for _, xcon := range crossConnectEvent.GetCrossConnects() {
				if crossConnectEvent.GetType() == dataplane.CrossConnectEventType_UPDATE {
					v.crossConnects[xcon.GetId()] = xcon
				}
				if crossConnectEvent.GetType() == dataplane.CrossConnectEventType_DELETE {
					delete(v.crossConnects, xcon.GetId())
				}
			}
			for _, recipient := range v.monitorRecipients {
				err := recipient.Send(crossConnectEvent)
				if err != nil {
					logrus.Errorf("Received error trying to send crossConnectEvent %v to %v: %s", crossConnectEvent, recipient, err)
				}
			}
		}
	}
}
