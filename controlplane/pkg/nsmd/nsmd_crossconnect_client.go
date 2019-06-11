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
	"fmt"
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
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/monitor/remote"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/services"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
)

const (
	endpointLogFormat          = "NSM-EndpointMonitor(%v): %v"
	endpointLogWithParamFormat = "NSM-EndpointMonitor(%v): %v: %v"

	dataplaneLogFormat          = "NSM-DataplaneMonitor(%v): %v"
	dataplaneLogWithParamFormat = "NSM-DataplaneMonitor(%v): %v: %v"

	peerLogFormat          = "NSM-PeerMonitor(%v): %v"
	peerLogWithParamFormat = "NSM-PeerMonitor(%v): %v: %v"

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
	go client.dataplaneCrossConnectMonitor(ctx, dp)
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
	go client.remotePeerConnectionMonitor(ctx, clientConnection.RemoteNsm)
}

// ClientConnectionUpdated implements method from Listener
func (client *NsmMonitorCrossConnectClient) ClientConnectionUpdated(old, new *model.ClientConnection) {
	logrus.Infof("ClientConnectionUpdated: old - %v; new - %v", old, new)

	if conn := new.Xcon.GetLocalSource(); conn != nil && !proto.Equal(old.Xcon.GetLocalSource(), conn) {
		if workspace, ok := conn.GetMechanism().GetParameters()[local_connection.Workspace]; ok {
			if localConnectionMonitor := client.monitorManager.LocalConnectionMonitor(workspace); localConnectionMonitor != nil {
				localConnectionMonitor.Update(conn)
			}
		}
	}

	if conn := new.Xcon.GetRemoteSource(); conn != nil && !proto.Equal(old.Xcon.GetRemoteSource(), conn) {
		client.monitorManager.RemoteConnectionMonitor().Update(conn)
	}
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

type grpcConnectionSupplier func() (*grpc.ClientConn, error)
type monitorClientSupplier func(conn *grpc.ClientConn) (monitor.Client, error)

type entityHandler func(entity monitor.Entity, eventType monitor.EventType) error
type eventHandler func(event monitor.Event) error

func (client *NsmMonitorCrossConnectClient) monitor(
	ctx context.Context,
	logFormat, logWithParamFormat, name string,
	grpcConnectionSupplier grpcConnectionSupplier, monitorClientSupplier monitorClientSupplier,
	entityHandler entityHandler, eventHandler eventHandler,
) error {
	logrus.Infof(logFormat, name, "Added")

	conn, err := grpcConnectionSupplier()
	if err != nil {
		logrus.Errorf(logWithParamFormat, name, "Failed to connect", err)
		return err
	}
	logrus.Infof(logFormat, name, "Connected")
	defer func() { _ = conn.Close() }()

	monitorClient, err := monitorClientSupplier(conn)
	if err != nil {
		logrus.Errorf(logWithParamFormat, name, "Failed to start monitor", err)
		return err
	}
	logrus.Infof(logFormat, name, "Started monitor")
	defer monitorClient.Close()

	for {
		select {
		case <-ctx.Done():
			logrus.Infof(logFormat, name, "Removed")
			return nil
		case err = <-monitorClient.ErrorChannel():
			logrus.Errorf(logWithParamFormat, name, "Connection closed", err)
			return err
		case event := <-monitorClient.EventChannel():
			if event != nil {
				logrus.Infof(logWithParamFormat, name, "Received event", event)
				for _, entity := range event.Entities() {
					if err = entityHandler(entity, event.EventType()); err != nil {
						logrus.Errorf(logWithParamFormat, name, "Error handling entity", err)
					}
				}

				if eventHandler != nil {
					if err = eventHandler(event); err != nil {
						logrus.Errorf(logWithParamFormat, name, "Error handling event", err)
					}
				}
			}
		}
	}
}

func (client *NsmMonitorCrossConnectClient) endpointConnectionMonitor(ctx context.Context, endpoint *model.Endpoint) {
	grpcConnectionSupplier := func() (*grpc.ClientConn, error) {
		logrus.Infof(endpointLogWithParamFormat, endpoint.EndpointName(), "Connecting to", endpoint.SocketLocation)
		return client.connectToEndpoint(endpoint)
	}

	err := client.monitor(
		ctx,
		endpointLogFormat, endpointLogWithParamFormat, endpoint.EndpointName(),
		grpcConnectionSupplier, local.NewMonitorClient,
		client.handleLocalConnection, nil)

	if err != nil {
		if err = client.endpointManager.DeleteEndpointWithBrokenConnection(endpoint); err != nil {
			logrus.Errorf(endpointLogWithParamFormat, endpoint.EndpointName(), "Failed to delete endpoint", err)
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

func (client *NsmMonitorCrossConnectClient) handleLocalConnection(entity monitor.Entity, eventType monitor.EventType) error {
	localConnection, ok := entity.(*local_connection.Connection)
	if !ok {
		return fmt.Errorf("unable to cast %v to local.Connection", entity)
	}

	if cc := client.xconManager.GetClientConnectionByLocalDst(localConnection.GetId()); cc != nil {
		switch eventType {
		case monitor.EventTypeUpdate:
			// DST connection is updated, we most probable need to re-programm our data plane.
			client.xconManager.LocalDestinationUpdated(cc, localConnection)
		case monitor.EventTypeDelete:
			// DST is down, we need to choose new NSE in any case.
			client.xconManager.DestinationDown(cc, false)
		}
	}

	return nil
}

// dataplaneCrossConnectMonitor is per registered dataplane crossconnect monitoring routine.
// It creates a grpc client for the socket advertsied by the dataplane and listens for a stream of Cross Connect Events.
// If it detects a failure of the connection, it will indicate that dataplane is no longer operational. In this case
// monitor will remove all dataplane connections and will terminate itself.
func (client *NsmMonitorCrossConnectClient) dataplaneCrossConnectMonitor(ctx context.Context, dataplane *model.Dataplane) {
	grpcConnectionSupplier := func() (*grpc.ClientConn, error) {
		logrus.Infof(dataplaneLogWithParamFormat, dataplane.RegisteredName, "Connecting to", dataplane.SocketLocation)
		return dial(context.Background(), "unix", dataplane.SocketLocation)
	}

	eventHandler := func(event monitor.Event) error {
		return client.handleXconEvent(event, dataplane)
	}

	_ = client.monitor(
		ctx,
		dataplaneLogFormat, dataplaneLogWithParamFormat, dataplane.RegisteredName,
		grpcConnectionSupplier, monitor_crossconnect.NewMonitorClient,
		client.handleXcon, eventHandler)
}

func (client *NsmMonitorCrossConnectClient) handleXcon(entity monitor.Entity, eventType monitor.EventType) error {
	xcon, ok := entity.(*crossconnect.CrossConnect)
	if !ok {
		return fmt.Errorf("unable to cast %v to CrossConnect", entity)
	}

	if cc := client.xconManager.GetClientConnectionByXcon(xcon); cc != nil {
		switch eventType {
		case monitor.EventTypeInitialStateTransfer:
			client.monitorManager.CrossConnectMonitor().Update(xcon)
		case monitor.EventTypeUpdate:
			client.monitorManager.CrossConnectMonitor().Update(xcon)
			client.xconManager.UpdateXcon(cc, xcon)
		case monitor.EventTypeDelete:
			if cc.ConnectionState == model.ClientConnectionClosing {
				client.monitorManager.CrossConnectMonitor().Delete(xcon)
			}
		}
	}

	return nil
}

func (client *NsmMonitorCrossConnectClient) handleXconEvent(event monitor.Event, dataplane *model.Dataplane) error {
	xconEvent, ok := event.(*monitor_crossconnect.Event)
	if !ok {
		return fmt.Errorf("unable to cast %v to crossconnect.Event", event)
	}

	if len(xconEvent.Statistics) > 0 {
		client.monitorManager.CrossConnectMonitor().HandleMetrics(xconEvent.Statistics)
	}

	if xconEvent.EventType() == monitor.EventTypeInitialStateTransfer {
		xcons := []*crossconnect.CrossConnect{}
		for _, entity := range event.Entities() {
			xcons = append(xcons, entity.(*crossconnect.CrossConnect))
		}

		client.xconManager.UpdateFromInitialState(xcons, dataplane)
	}

	return nil
}

func (client *NsmMonitorCrossConnectClient) remotePeerConnectionMonitor(ctx context.Context, remotePeer *registry.NetworkServiceManager) {
	grpcConnectionSupplier := func() (*grpc.ClientConn, error) {
		logrus.Infof(peerLogWithParamFormat, remotePeer.Name, "Connecting to", remotePeer.Url)
		return grpc.Dial(remotePeer.Url, grpc.WithInsecure())
	}
	monitorClientSupplier := func(conn *grpc.ClientConn) (monitor.Client, error) {
		return remote.NewMonitorClient(conn, &remote_connection.MonitorScopeSelector{
			NetworkServiceManagerName: client.xconManager.GetNsmName(),
		})
	}

	err := client.monitor(
		ctx,
		peerLogFormat, peerLogWithParamFormat, remotePeer.Name,
		grpcConnectionSupplier, monitorClientSupplier,
		client.handleRemoteConnection, nil)

	if err != nil {
		// Same as DST down case, we need to wait for same NSE and updated NSMD.
		connections := client.xconManager.GetClientConnectionByRemote(remotePeer)
		for _, cc := range connections {
			client.xconManager.DestinationDown(cc, true)
		}
	}
}

func (client *NsmMonitorCrossConnectClient) handleRemoteConnection(entity monitor.Entity, eventType monitor.EventType) error {
	remoteConnection, ok := entity.(*remote_connection.Connection)
	if !ok {
		return fmt.Errorf("unable to cast %v to remote.Connection", entity)
	}

	if cc := client.xconManager.GetClientConnectionByRemoteDst(remoteConnection.GetId()); cc != nil {
		switch eventType {
		case monitor.EventTypeUpdate:
			// DST connection is updated, we most probable need to re-programm our data plane.
			client.xconManager.RemoteDestinationUpdated(cc, remoteConnection)
		case monitor.EventTypeDelete:
			// DST is down, we need to choose new NSE in any case.
			downConnection := proto.Clone(remoteConnection).(*remote_connection.Connection)
			downConnection.State = remote_connection.State_DOWN

			xconToSend := &crossconnect.CrossConnect{
				Source: &crossconnect.CrossConnect_LocalSource{
					LocalSource: cc.Xcon.GetLocalSource(),
				},
				Destination: &crossconnect.CrossConnect_RemoteDestination{
					RemoteDestination: downConnection,
				},
			}

			client.monitorManager.CrossConnectMonitor().Update(xconToSend)
			client.xconManager.DestinationDown(cc, false)
		}
	}

	return nil
}
