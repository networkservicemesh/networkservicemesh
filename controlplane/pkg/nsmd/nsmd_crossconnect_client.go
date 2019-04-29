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
	"net"
	"time"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/monitor/crossconnect_monitor"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/monitor/remote_connection_monitor"
	"github.com/opentracing/opentracing-go"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/grpc-ecosystem/grpc-opentracing/go/otgrpc"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/crossconnect"
	local_connection "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/registry"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/remote/connection"
	remote_connection "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/remote/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/model"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/services"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

type NsmMonitorCrossConnectClient struct {
	crossConnectMonitor *crossconnect_monitor.CrossConnectMonitor
	connectionMonitor   *remote_connection_monitor.RemoteConnectionMonitor
	remotePeers         map[string]*remotePeerDescriptor
	dataplanes          map[string]context.CancelFunc
	xconManager         *services.ClientConnectionManager
	model.ModelListenerImpl
}

type remotePeerDescriptor struct {
	xconCounter int
	cancel      context.CancelFunc
}

func NewMonitorCrossConnectClient(crossConnectMonitor *crossconnect_monitor.CrossConnectMonitor,
	connectionMonitor *remote_connection_monitor.RemoteConnectionMonitor, xconManager *services.ClientConnectionManager) *NsmMonitorCrossConnectClient {
	rv := &NsmMonitorCrossConnectClient{
		crossConnectMonitor: crossConnectMonitor,
		connectionMonitor:   connectionMonitor,
		remotePeers:         map[string]*remotePeerDescriptor{},
		dataplanes:          map[string]context.CancelFunc{},
		xconManager:         xconManager,
	}
	return rv
}

func dial(ctx context.Context, network string, address string) (*grpc.ClientConn, error) {
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

func (client *NsmMonitorCrossConnectClient) DataplaneAdded(dataplane *model.Dataplane) {
	ctx, cancel := context.WithCancel(context.Background())
	client.dataplanes[dataplane.RegisteredName] = cancel
	logrus.Infof("Starting Dataplane crossconnect monitoring client...")
	go client.dataplaneCrossConnectMonitor(dataplane, ctx)
}

func (client *NsmMonitorCrossConnectClient) DataplaneDeleted(dataplane *model.Dataplane) {
	clientConnections := client.xconManager.GetClientConnectionsByDataplane(dataplane.RegisteredName)
	client.xconManager.UpdateClientConnectionDataplaneStateDown(clientConnections)
	client.dataplanes[dataplane.RegisteredName]()
	delete(client.dataplanes, dataplane.RegisteredName)
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
func (client *NsmMonitorCrossConnectClient) ClientConnectionUpdated(clientConnection *model.ClientConnection) {
	logrus.Infof("ClientConnectionUpdated: %v", clientConnection)

	if conn := clientConnection.Xcon.GetRemoteSource(); conn != nil {
		client.connectionMonitor.Update(conn)
	}
}

func (client *NsmMonitorCrossConnectClient) ClientConnectionDeleted(clientConnection *model.ClientConnection) {
	logrus.Infof("ClientConnectionDeleted: %v", clientConnection)

	client.crossConnectMonitor.Delete(clientConnection.Xcon)
	if conn := clientConnection.Xcon.GetRemoteSource(); conn != nil {
		client.connectionMonitor.Delete(conn)
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
					if src := xcon.GetLocalSource(); src != nil && src.State == local_connection.State_DOWN {
						client.xconManager.UpdateClientConnectionSrcStateDown(clientConnection)
					}
					if dst := xcon.GetLocalDestination(); dst != nil && dst.State == local_connection.State_DOWN {
						client.xconManager.UpdateClientConnectionDstStateDown(clientConnection, false)
					}
					clientConnection.Xcon = xcon
					client.xconManager.UpdateClientConnection(clientConnection)
					client.crossConnectMonitor.Update(xcon)
				case crossconnect.CrossConnectEventType_DELETE:
					client.crossConnectMonitor.Delete(xcon)
				case crossconnect.CrossConnectEventType_INITIAL_STATE_TRANSFER:
					client.crossConnectMonitor.Update(xcon)
				}
			}
			if event.Metrics != nil {
				client.crossConnectMonitor.HandleMetrics(event.Metrics)
			}
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
				logrus.Errorf("NSM-PeerMonitor(%v) Uexpected error: %v", remotePeer.Name, err)
				connections := client.xconManager.GetClientConnectionByRemote(remotePeer)
				for _, c := range connections {
					// Same as DST down case, we need to wait for same NSE and updated NSMD.
					client.xconManager.UpdateClientConnectionDstStateDown(c, true)
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
					client.xconManager.UpdateClientConnectionDstUpdated(clientConnection, remoteConnection)
				case connection.ConnectionEventType_DELETE:
					// DST is down, we need to choose new NSE in any case.
					client.xconManager.UpdateClientConnectionDstStateDown(clientConnection, false)
				}
			}
		}
	}
}
