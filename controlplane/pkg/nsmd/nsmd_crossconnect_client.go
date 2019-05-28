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

package nsmd

import (
	"context"
	"github.com/golang/protobuf/proto"
	"net"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/grpc-ecosystem/grpc-opentracing/go/otgrpc"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/crossconnect"
	local_connection "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/registry"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/remote/connection"
	remote_connection "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/remote/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/model"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/monitor"
	monitor_crossconnect "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/monitor/crossconnect"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/services"
)

type NsmMonitorCrossConnectClient struct {
	monitorManager MonitorManager
	xconManager    *services.ClientConnectionManager
	endpoints      map[string]context.CancelFunc
	remotePeers    map[string]*remotePeerDescriptor
	dataplanes     map[string]context.CancelFunc
	model.ModelListenerImpl
}

type remotePeerDescriptor struct {
	xconCounter int
	cancel      context.CancelFunc
}

// MonitorManager is an interface to provide access to different monitors
type MonitorManager interface {
	CrossConnectMonitor() monitor_crossconnect.MonitorServer
	RemoteConnectionMonitor() monitor.Server
	LocalConnectionMonitor(workspace string) monitor.Server
}

// NewMonitorCrossConnectClient creates a new NsmMonitorCrossConnectClient
func NewMonitorCrossConnectClient(monitorManager MonitorManager, xconManager *services.ClientConnectionManager) *NsmMonitorCrossConnectClient {
	rv := &NsmMonitorCrossConnectClient{
		monitorManager: monitorManager,
		xconManager:    xconManager,
		endpoints:      map[string]context.CancelFunc{},
		remotePeers:    map[string]*remotePeerDescriptor{},
		dataplanes:     map[string]context.CancelFunc{},
	}
	return rv
}

func dial(ctx context.Context, network, address string) (*grpc.ClientConn, error) {
	tracer := opentracing.GlobalTracer()
	conn, err := grpc.DialContext(ctx, address, grpc.WithInsecure(), grpc.WithBlock(),
		grpc.WithDialer(func(addr string, timeout time.Duration) (net.Conn, error) {
			return net.Dial(network, addr)
		}),
		grpc.WithUnaryInterceptor(
			otgrpc.OpenTracingClientInterceptor(tracer)),
		grpc.WithStreamInterceptor(
			otgrpc.OpenTracingStreamClientInterceptor(tracer)))

	return conn, err
}

// DataplaneAdded implements method from ModelListener
func (client *NsmMonitorCrossConnectClient) DataplaneAdded(dp *model.Dataplane) {
	ctx, cancel := context.WithCancel(context.Background())
	client.dataplanes[dp.RegisteredName] = cancel
	logrus.Infof("Starting Dataplane crossconnect monitoring client...")
	go client.dataplaneCrossConnectMonitor(dp, ctx)
}

// DataplaneDeleted implements method from ModelListener
func (client *NsmMonitorCrossConnectClient) DataplaneDeleted(dp *model.Dataplane) {
	client.xconManager.DataplaneDown(dp)
	client.dataplanes[dp.RegisteredName]()
	delete(client.dataplanes, dp.RegisteredName)
}

func (client *NsmMonitorCrossConnectClient) ClientConnectionAdded(clientConnection *model.ClientConnection) {
	if clientConnection.RemoteNsm == nil {
		return
	}

	if remotePeer, exist := client.remotePeers[clientConnection.RemoteNsm.Name]; exist {
		remotePeer.xconCounter++
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	client.remotePeers[clientConnection.RemoteNsm.Name] = &remotePeerDescriptor{cancel: cancel, xconCounter: 1}
	go client.remotePeerConnectionMonitor(clientConnection.RemoteNsm, ctx)
}

// ClientConnectionUpdated implements method from ModelListener
func (client *NsmMonitorCrossConnectClient) ClientConnectionUpdated(old, new *model.ClientConnection) {
	logrus.Infof("ClientConnectionUpdated: old - %v; new - %v", old, new)

	if conn := new.Xcon.GetLocalSource(); conn != nil {
		if workspace, ok := conn.GetMechanism().GetParameters()[local_connection.Workspace]; ok {
			if localConnectionMonitor := client.monitorManager.LocalConnectionMonitor(workspace); localConnectionMonitor != nil {
				localConnectionMonitor.Update(conn)
			}
		}
	}

	if new.Xcon.GetRemoteSource() == nil {
		return
	}

	if proto.Equal(old.Xcon.GetRemoteSource(), new.Xcon.GetRemoteSource()) {
		return
	}

	client.monitorManager.CrossConnectMonitor().Update(new.Xcon.GetRemoteSource())
}

func (client *NsmMonitorCrossConnectClient) ClientConnectionDeleted(clientConnection *model.ClientConnection) {
	logrus.Infof("ClientConnectionDeleted: %v", clientConnection)

	client.monitorManager.CrossConnectMonitor().Delete(clientConnection.Xcon)
	if conn := clientConnection.Xcon.GetRemoteSource(); conn != nil {
		client.monitorManager.RemoteConnectionMonitor().Delete(conn)
	}

	if clientConnection.RemoteNsm == nil {
		return
	}
	remotePeer := client.remotePeers[clientConnection.RemoteNsm.Name]
	if remotePeer != nil {
		remotePeer.xconCounter--
		if remotePeer.xconCounter == 0 {
			remotePeer.cancel()
			delete(client.remotePeers, clientConnection.RemoteNsm.Name)
		}
	} else {
		logrus.Errorf("Remote peer for NSM is already closed: %v", clientConnection)
	}
}

// dataplaneCrossConnectMonitor is per registered dataplane crossconnect monitoring routine.
// It creates a grpc client for the socket advertsied by the dataplane and listens for a stream of Cross Connect Events.
// If it detects a failure of the connection, it will indicate that dataplane is no longer operational. In this case
// monitor will remove all dataplane connections and will terminate itself.
func (client *NsmMonitorCrossConnectClient) dataplaneCrossConnectMonitor(dataplane *model.Dataplane, ctx context.Context) {
	logrus.Infof("Connecting to Dataplane %s %s", dataplane.RegisteredName, dataplane.SocketLocation)
	conn, err := dial(context.Background(), "unix", dataplane.SocketLocation)
	if err != nil {
		logrus.Errorf("failure to communicate with the socket %s with error: %+v", dataplane.SocketLocation, err)
		return
	}
	defer conn.Close()

	monitorClient := crossconnect.NewMonitorCrossConnectClient(conn)
	stream, err := monitorClient.MonitorCrossConnects(ctx, &empty.Empty{})
	if err != nil {
		logrus.Error(err)
		return
	}
	logrus.Infof("Monitoring %v CrossConnections...", dataplane.RegisteredName)
	for {
		select {
		case <-ctx.Done():
			logrus.Info("Context timeout exceeded...")
			return
		default:
			event, err := stream.Recv()
			if err != nil {
				logrus.Error(err)
				return
			}
			logrus.Infof("Receive event from dataplane %s: %s %s", dataplane.RegisteredName, event.Type, event.CrossConnects)

			for _, xcon := range event.GetCrossConnects() {
				clientConnection := client.xconManager.GetClientConnectionByXcon(xcon)
				if clientConnection == nil {
					continue
				}

				switch event.GetType() {
				case crossconnect.CrossConnectEventType_UPDATE:
					client.monitorManager.CrossConnectMonitor().Update(xcon)
					client.xconManager.UpdateXcon(clientConnection, xcon)
				case crossconnect.CrossConnectEventType_DELETE:
					client.monitorManager.CrossConnectMonitor().Delete(xcon)
				case crossconnect.CrossConnectEventType_INITIAL_STATE_TRANSFER:
					client.monitorManager.CrossConnectMonitor().Update(xcon)
				}
			}
			if event.Metrics != nil {
				client.monitorManager.CrossConnectMonitor().HandleMetrics(event.Metrics)
			}
			// TODO: initial_state_transfer event is handled twice
			if event.GetType() == crossconnect.CrossConnectEventType_INITIAL_STATE_TRANSFER {
				connects := []*crossconnect.CrossConnect{}
				for _, xcon := range event.GetCrossConnects() {
					connects = append(connects, xcon)
				}
				client.xconManager.UpdateFromInitialState(connects, dataplane)
			}
		}
	}
}

func (client *NsmMonitorCrossConnectClient) remotePeerConnectionMonitor(remotePeer *registry.NetworkServiceManager, ctx context.Context) {
	logrus.Infof("NSM-PeerMonitor(%v): Connecting...", remotePeer.Name)
	conn, err := grpc.Dial(remotePeer.Url, grpc.WithInsecure())
	if err != nil {
		logrus.Errorf("NSM-PeerMonitor(%v): Failed to dial Network Service Registry at %s: %s", remotePeer.GetName(), remotePeer.Url, err)
		return
	}
	defer conn.Close()

	defer func() {
		logrus.Infof("NSM-PeerMonitor(%v): Remote monitor closed...", remotePeer.Name)
	}()

	monitorClient := remote_connection.NewMonitorConnectionClient(conn)
	selector := &remote_connection.MonitorScopeSelector{NetworkServiceManagerName: client.xconManager.GetNsmName()}
	stream, err := monitorClient.MonitorConnections(context.Background(), selector)
	if err != nil {
		logrus.Error(err)
		return
	}

	for {
		select {
		case <-ctx.Done():
			logrus.Infof("NSM-PeerMonitor(%v): Context timeout exceeded...", remotePeer.Name)
			return
		default:
			logrus.Infof("NSM-PeerMonitor(%v): Waiting for events...", remotePeer.Name)
			event, err := stream.Recv()
			if err != nil {
				logrus.Errorf("NSM-PeerMonitor(%v) Unexpected error: %v", remotePeer.Name, err)
				connections := client.xconManager.GetClientConnectionByRemote(remotePeer)
				for _, c := range connections {
					// Same as DST down case, we need to wait for same NSE and updated NSMD.
					client.xconManager.RemoteDestinationDown(c, true)
				}
				return
			}
			logrus.Infof("NSM-PeerMonitor(%v) Receive event %s %s", remotePeer.GetName(), event.Type, event.Connections)

			for _, remoteConnection := range event.GetConnections() {
				clientConnection := client.xconManager.GetClientConnectionByDst(remoteConnection.GetId())
				if clientConnection == nil {
					continue
				}
				switch event.GetType() {
				case connection.ConnectionEventType_UPDATE:
					// DST connection is updated, we most probable need to re-programm our data plane.
					client.xconManager.RemoteDestinationUpdated(clientConnection, remoteConnection)
				case connection.ConnectionEventType_DELETE:
					// DST is down, we need to choose new NSE in any case.
					client.xconManager.RemoteDestinationDown(clientConnection, false)
				}
			}
		}
	}
}
