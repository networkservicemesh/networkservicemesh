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

	"github.com/golang/protobuf/proto"
	"github.com/grpc-ecosystem/grpc-opentracing/go/otgrpc"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/crossconnect"
	local_connection "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/registry"
	remote_connection "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/remote/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/model"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/monitor"
	monitor_crossconnect "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/monitor/crossconnect"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/monitor/local"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/services"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
)

const (
	endpointLogFormat          = "NSM-EndpointMonitor(%v): %v"
	endpointLogWithParamFormat = "NSM-EndpointMonitor(%v): %v: %v"

	dataplaneLogFormat          = "NSM-DataplaneMonitor(%v): %v"
	dataplaneLogWithParamFormat = "NSM-DataplaneMonitor(%v): %v: %v"

	endpointConnectionTimeout = 10 * time.Second
)

type NsmMonitorCrossConnectClient struct {
	model.ListenerImpl

	monitorManager MonitorManager
	xconManager    *services.ClientConnectionManager
	endpoints      map[string]context.CancelFunc
	remotePeers    map[string]*remotePeerDescriptor
	dataplanes     map[string]context.CancelFunc

	endpointManager EndpointManager
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

// EndpointManager is an interface to delete endpoints with broken connection
type EndpointManager interface {
	DeleteEndpointWithBrokenConnection(endpoint *model.Endpoint) error
}

// NewMonitorCrossConnectClient creates a new NsmMonitorCrossConnectClient
func NewMonitorCrossConnectClient(monitorManager MonitorManager, xconManager *services.ClientConnectionManager,
	endpointManager EndpointManager) *NsmMonitorCrossConnectClient {
	rv := &NsmMonitorCrossConnectClient{
		monitorManager:  monitorManager,
		xconManager:     xconManager,
		endpointManager: endpointManager,
		endpoints:       map[string]context.CancelFunc{},
		remotePeers:     map[string]*remotePeerDescriptor{},
		dataplanes:      map[string]context.CancelFunc{},
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

// EndpointAdded implements method from Listener
func (client *NsmMonitorCrossConnectClient) EndpointAdded(endpoint *model.Endpoint) {
	ctx, cancel := context.WithCancel(context.Background())
	client.endpoints[endpoint.EndpointName()] = cancel
	go client.endpointConnectionMonitor(ctx, endpoint)
}

// EndpointDeleted implements method from Listener
func (client *NsmMonitorCrossConnectClient) EndpointDeleted(endpoint *model.Endpoint) {
	if cancel, ok := client.endpoints[endpoint.EndpointName()]; ok {
		cancel()
		delete(client.endpoints, endpoint.EndpointName())
	}
}

// DataplaneAdded implements method from Listener
func (client *NsmMonitorCrossConnectClient) DataplaneAdded(dp *model.Dataplane) {
	ctx, cancel := context.WithCancel(context.Background())
	client.dataplanes[dp.RegisteredName] = cancel
	logrus.Infof("Starting Dataplane crossconnect monitoring client...")
	go client.dataplaneCrossConnectMonitor(dp, ctx)
}

// DataplaneDeleted implements method from Listener
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

// ClientConnectionUpdated implements method from Listener
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

	client.monitorManager.RemoteConnectionMonitor().Update(new.Xcon.GetRemoteSource())
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

func (client *NsmMonitorCrossConnectClient) endpointConnectionMonitor(ctx context.Context, endpoint *model.Endpoint) {
	logrus.Infof(endpointLogFormat, endpoint.EndpointName(), "Added")

	conn, err := client.connectToEndpoint(endpoint)
	if err != nil {
		logrus.Errorf(endpointLogWithParamFormat, endpoint.EndpointName(), "Failed to connect", err)
		client.deleteEndpoint(endpoint)
		return
	}
	logrus.Infof(endpointLogFormat, endpoint.EndpointName(), "Connected")
	defer func() { _ = conn.Close() }()

	monitorClient, err := local.NewMonitorClient(conn)
	if err != nil {
		logrus.Errorf(endpointLogWithParamFormat, endpoint.EndpointName(), "Failed to start monitor", err)
		client.deleteEndpoint(endpoint)
		return
	}
	logrus.Infof(endpointLogFormat, endpoint.EndpointName(), "Started monitor")
	defer monitorClient.Close()

	for {
		select {
		case <-ctx.Done():
			logrus.Infof(endpointLogFormat, endpoint.EndpointName(), "Removed")
			return
		case err = <-monitorClient.ErrorChannel():
			logrus.Infof(endpointLogWithParamFormat, endpoint.EndpointName(), "Connection closed", err)
			client.deleteEndpoint(endpoint)
			return
		case event := <-monitorClient.EventChannel():
			logrus.Infof(endpointLogWithParamFormat, endpoint.EndpointName(), "Received event", event)
			for _, entity := range event.Entities() {
				cc := client.xconManager.GetClientConnectionByDst(entity.GetId())
				if cc == nil {
					continue
				}

				switch event.EventType() {
				case monitor.EventTypeUpdate:
					// DST connection is updated, we most probable need to re-programm our data plane.
					client.xconManager.LocalDestinationUpdated(cc, entity.(*local_connection.Connection))
				case monitor.EventTypeDelete:
					// DST is down, we need to choose new NSE in any case.
					client.xconManager.DestinationDown(cc, false)
				}
			}
		}
	}
}

func (client *NsmMonitorCrossConnectClient) connectToEndpoint(endpoint *model.Endpoint) (*grpc.ClientConn, error) {
	var conn *grpc.ClientConn
	var err error

	for st := time.Now(); time.Since(st) < endpointConnectionTimeout; <-time.After(100 * time.Millisecond) {
		if conn, err = tools.SocketOperationCheck(tools.SocketPath(endpoint.SocketLocation)); err == nil {
			break
		}
	}

	return conn, err
}

func (client *NsmMonitorCrossConnectClient) deleteEndpoint(endpoint *model.Endpoint) {
	if err := client.endpointManager.DeleteEndpointWithBrokenConnection(endpoint); err != nil {
		logrus.Errorf(endpointLogWithParamFormat, endpoint.EndpointName(), "Failed to delete endpoint", err)
	}
}

// dataplaneCrossConnectMonitor is per registered dataplane crossconnect monitoring routine.
// It creates a grpc client for the socket advertsied by the dataplane and listens for a stream of Cross Connect Events.
// If it detects a failure of the connection, it will indicate that dataplane is no longer operational. In this case
// monitor will remove all dataplane connections and will terminate itself.
func (client *NsmMonitorCrossConnectClient) dataplaneCrossConnectMonitor(dataplane *model.Dataplane, ctx context.Context) {
	logrus.Infof(dataplaneLogFormat, dataplane.RegisteredName, "Added")

	logrus.Infof(dataplaneLogWithParamFormat, dataplane.RegisteredName, "Connecting to Dataplane", dataplane.SocketLocation)
	conn, err := dial(context.Background(), "unix", dataplane.SocketLocation)
	if err != nil {
		logrus.Errorf(dataplaneLogWithParamFormat, dataplane.RegisteredName, "Failed to connect", err)
		return
	}
	logrus.Infof(dataplaneLogFormat, dataplane.RegisteredName, "Connected")
	defer func() { _ = conn.Close() }()

	monitorClient, err := monitor_crossconnect.NewMonitorClient(conn)
	if err != nil {
		logrus.Errorf(dataplaneLogWithParamFormat, dataplane.RegisteredName, "Failed to start monitor", err)
		return
	}
	logrus.Infof(dataplaneLogFormat, dataplane.RegisteredName, "Started monitor")
	defer monitorClient.Close()

	for {
		select {
		case <-ctx.Done():
			logrus.Infof(dataplaneLogFormat, dataplane.RegisteredName, "Removed")
			return
		case err = <-monitorClient.ErrorChannel():
			logrus.Errorf(dataplaneLogWithParamFormat, dataplane.RegisteredName, "Connection closed", err)
			return
		case event := <-monitorClient.EventChannel():
			logrus.Infof(dataplaneLogFormat, dataplane.RegisteredName, "Received event", event)
			xcons := []*crossconnect.CrossConnect{}
			for _, entity := range event.Entities() {
				xcon := entity.(*crossconnect.CrossConnect)
				cc := client.xconManager.GetClientConnectionByXcon(xcon)
				if cc == nil {
					continue
				}

				switch event.EventType() {
				case monitor.EventTypeUpdate:
					client.monitorManager.CrossConnectMonitor().Update(xcon)
					client.xconManager.UpdateXcon(cc, xcon)
				case monitor.EventTypeDelete:
					if cc.ConnectionState == model.ClientConnectionClosing {
						client.monitorManager.CrossConnectMonitor().Delete(xcon)
					}
				case monitor.EventTypeInitialStateTransfer:
					xcons = append(xcons, xcon)
					client.monitorManager.CrossConnectMonitor().Update(xcon)
				}
			}

			if statistics := event.(*monitor_crossconnect.Event).Statistics; len(statistics) > 0 {
				client.monitorManager.CrossConnectMonitor().HandleMetrics(statistics)
			}

			if event.EventType() == monitor.EventTypeInitialStateTransfer {
				client.xconManager.UpdateFromInitialState(xcons, dataplane)
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
					client.xconManager.DestinationDown(c, true)
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
				case remote_connection.ConnectionEventType_UPDATE:
					// DST connection is updated, we most probable need to re-programm our data plane.
					client.xconManager.RemoteDestinationUpdated(clientConnection, remoteConnection)
				case remote_connection.ConnectionEventType_DELETE:
					// DST is down, we need to choose new NSE in any case.
					downConnection := proto.Clone(remoteConnection).(*remote_connection.Connection)
					downConnection.State = remote_connection.State_DOWN

					xconToSend := &crossconnect.CrossConnect{
						Source: &crossconnect.CrossConnect_LocalSource{
							LocalSource: clientConnection.Xcon.GetLocalSource(),
						},
						Destination: &crossconnect.CrossConnect_RemoteDestination{
							RemoteDestination: downConnection,
						},
					}

					client.monitorManager.CrossConnectMonitor().Update(xconToSend)
					client.xconManager.DestinationDown(clientConnection, false)
				}
			}
		}
	}
}
