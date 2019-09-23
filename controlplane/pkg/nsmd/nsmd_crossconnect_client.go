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
	"sync"
	"time"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/log"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/common"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/crossconnect"
	local "github.com/networkservicemesh/networkservicemesh/controlplane/api/local/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/nsm/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/registry"
	remote "github.com/networkservicemesh/networkservicemesh/controlplane/api/remote/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/model"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/services"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
	"github.com/networkservicemesh/networkservicemesh/sdk/monitor"
	monitor_crossconnect "github.com/networkservicemesh/networkservicemesh/sdk/monitor/crossconnect"
	monitor_local "github.com/networkservicemesh/networkservicemesh/sdk/monitor/local"
	monitor_remote "github.com/networkservicemesh/networkservicemesh/sdk/monitor/remote"
)

const (
	endpointLogFormat          = "NSM-EndpointMonitor(%v): %v"
	endpointLogWithParamFormat = "NSM-EndpointMonitor(%v): %v: %v"

	dataplaneLogFormat          = "NSM-DataplaneMonitor(%v): %v"
	dataplaneLogWithParamFormat = "NSM-DataplaneMonitor(%v): %v: %v"

	peerLogFormat          = "NSM-PeerMonitor(%v): %v"
	peerLogWithParamFormat = "NSM-PeerMonitor(%v): %v: %v"

	endpointConnectionTimeout = 10 * time.Second
	eventConnectionTimeout    = 30 * time.Second

	PeerName = "PeerName"
)

type NsmMonitorCrossConnectClient struct {
	model.ListenerImpl

	monitorManager MonitorManager
	xconManager    *services.ClientConnectionManager
	endpoints      sync.Map
	dataplanes     sync.Map
	remotePeers    map[string]*remotePeerDescriptor

	endpointManager EndpointManager
	model           model.Model

	remotePeerLock sync.Mutex
}

type remotePeerDescriptor struct {
	connections map[string]*model.ClientConnection
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
func NewMonitorCrossConnectClient(model model.Model, monitorManager MonitorManager, xconManager *services.ClientConnectionManager,
	endpointManager EndpointManager) *NsmMonitorCrossConnectClient {
	rv := &NsmMonitorCrossConnectClient{
		monitorManager:  monitorManager,
		xconManager:     xconManager,
		endpointManager: endpointManager,
		endpoints:       sync.Map{},
		dataplanes:      sync.Map{},
		remotePeers:     map[string]*remotePeerDescriptor{},
		model:           model,
	}
	return rv
}

// EndpointAdded implements method from Listener
func (client *NsmMonitorCrossConnectClient) EndpointAdded(endpoint *model.Endpoint) {
	ctx, cancel := context.WithCancel(context.Background())
	client.endpoints.Store(endpoint.EndpointName(), cancel)
	go client.endpointConnectionMonitor(ctx, endpoint)
}

// EndpointDeleted implements method from Listener
func (client *NsmMonitorCrossConnectClient) EndpointDeleted(endpoint *model.Endpoint) {
	if cancel, ok := client.endpoints.Load(endpoint.EndpointName()); ok {
		cancel.(context.CancelFunc)()
		client.endpoints.Delete(endpoint.EndpointName())
	}
}

// DataplaneAdded implements method from Listener
func (client *NsmMonitorCrossConnectClient) DataplaneAdded(dp *model.Dataplane) {
	ctx, cancel := context.WithCancel(context.Background())
	client.dataplanes.Store(dp.RegisteredName, cancel)
	logrus.Infof("Starting Dataplane crossconnect monitoring client...")
	go client.dataplaneCrossConnectMonitor(ctx, dp)
}

// DataplaneDeleted implements method from Listener
func (client *NsmMonitorCrossConnectClient) DataplaneDeleted(dp *model.Dataplane) {
	client.xconManager.DataplaneDown(dp)
	if cancel, ok := client.dataplanes.Load(dp.RegisteredName); ok {
		cancel.(context.CancelFunc)()
		client.dataplanes.Delete(dp.RegisteredName)
	}
}

func (client *NsmMonitorCrossConnectClient) startPeerMonitor(clientConnection *model.ClientConnection) {
	client.remotePeerLock.Lock()
	defer client.remotePeerLock.Unlock()
	if clientConnection.RemoteNsm == nil {
		return
	}
	if remotePeer, exist := client.remotePeers[clientConnection.RemoteNsm.Name]; exist {
		remotePeer.connections[clientConnection.ConnectionID] = clientConnection
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	client.remotePeers[clientConnection.RemoteNsm.Name] = &remotePeerDescriptor{
		cancel: cancel,
		connections: map[string]*model.ClientConnection{
			clientConnection.ConnectionID: clientConnection,
		},
	}
	go client.remotePeerConnectionMonitor(ctx, clientConnection.RemoteNsm)
}

func (client *NsmMonitorCrossConnectClient) ClientConnectionAdded(clientConnection *model.ClientConnection) {
	client.startPeerMonitor(clientConnection)
}

// ClientConnectionUpdated implements method from Listener
func (client *NsmMonitorCrossConnectClient) ClientConnectionUpdated(old, new *model.ClientConnection) {
	client.startPeerMonitor(new)

	conn := new.Xcon.GetSourceConnection()
	if conn.Equals(old.Xcon.GetSourceConnection()) {
		logrus.Infof("ClientConnectionUpdated: old - %v == new - %v. No event send.", old, new)
		return
	}

	var monitorServer monitor.Server
	if conn.IsRemote() {
		monitorServer = client.monitorManager.RemoteConnectionMonitor()
	} else {
		monitorServer = client.monitorManager.LocalConnectionMonitor(new.Workspace)
	}

	if monitorServer != nil {
		logrus.Infof("ClientConnectionUpdated: old - %v; new - %v. Send to Monitor...", old, new)
		monitorServer.Update(conn)
	} else {
		logrus.Infof("ClientConnectionUpdated: old - %v; new - %v.", old, new)
	}
}

func (client *NsmMonitorCrossConnectClient) ClientConnectionDeleted(clientConnection *model.ClientConnection) {
	client.remotePeerLock.Lock()
	defer client.remotePeerLock.Unlock()

	logrus.Infof("ClientConnectionDeleted: %v", clientConnection)

	client.monitorManager.CrossConnectMonitor().Delete(clientConnection.Xcon)
	if conn := clientConnection.GetConnectionSource(); conn != nil && conn.IsRemote() {
		client.monitorManager.RemoteConnectionMonitor().Delete(conn)
	}

	if clientConnection.RemoteNsm == nil {
		return
	}
	remotePeer := client.remotePeers[clientConnection.RemoteNsm.Name]
	if remotePeer != nil {
		delete(remotePeer.connections, clientConnection.ConnectionID)
		if len(remotePeer.connections) == 0 {
			remotePeer.cancel()
			delete(client.remotePeers, clientConnection.RemoteNsm.Name)
		}
	} else {
		logrus.Errorf("Remote peer for NSM is already closed: %v", clientConnection)
	}
}

type grpcConnectionSupplier func() (*grpc.ClientConn, error)
type monitorClientSupplier func(conn *grpc.ClientConn) (monitor.Client, error)

type entityHandler func(entity monitor.Entity, eventType monitor.EventType, parameters map[string]string) error
type eventHandler func(event monitor.Event, parameters map[string]string) error

func (client *NsmMonitorCrossConnectClient) monitor(
	ctx context.Context,
	logFormat, logWithParamFormat, name string,
	grpcConnectionSupplier grpcConnectionSupplier, monitorClientSupplier monitorClientSupplier,
	entityHandler entityHandler, eventHandler eventHandler, parameters map[string]string) error {
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
					if err = entityHandler(entity, event.EventType(), parameters); err != nil {
						logrus.Errorf(logWithParamFormat, name, "Error handling entity", err)
					}
				}

				if eventHandler != nil {
					if err = eventHandler(event, parameters); err != nil {
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
		//<- time.After(1000* time.Second)
		return client.connectToEndpoint(endpoint)
	}

	err := client.monitor(
		ctx,
		endpointLogFormat, endpointLogWithParamFormat, endpoint.EndpointName(),
		grpcConnectionSupplier, monitor_local.NewMonitorClient,
		client.handleLocalConnection, nil, map[string]string{"endpointName": endpoint.EndpointName()})

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
		if conn, err = tools.DialUnix(endpoint.SocketLocation); err == nil {
			break
		}
	}

	return conn, err
}

func (client *NsmMonitorCrossConnectClient) handleLocalConnection(entity monitor.Entity, eventType monitor.EventType, parameters map[string]string) error {
	localConnection, ok := entity.(*local.Connection)
	if !ok {
		return fmt.Errorf("unable to cast %v to local.Connection", entity)
	}

	// We could do so because for local NSE connections ID is assigned by NSMgr itself.
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
		//<- time.After(1000* time.Second)
		return tools.DialUnix(dataplane.SocketLocation)
	}

	eventHandler := func(event monitor.Event, parameters map[string]string) error {
		return client.handleXconEvent(event, dataplane, parameters)
	}

	_ = client.monitor(
		ctx,
		dataplaneLogFormat, dataplaneLogWithParamFormat, dataplane.RegisteredName,
		grpcConnectionSupplier, monitor_crossconnect.NewMonitorClient,
		client.handleXcon, eventHandler, nil)
}

func (client *NsmMonitorCrossConnectClient) handleXcon(entity monitor.Entity, eventType monitor.EventType, parameters map[string]string) error {
	xcon, ok := entity.(*crossconnect.CrossConnect)
	if !ok {
		return fmt.Errorf("unable to cast %v to CrossConnect", entity)
	}

	// Let's add this into Span.
	clientConnection := client.model.GetClientConnection(xcon.GetId())

	var logger logrus.FieldLogger
	var span opentracing.Span

	if clientConnection != nil {
		if clientConnection.Span != nil {
			ctx := opentracing.ContextWithSpan(context.Background(), clientConnection.Span)
			span, ctx = opentracing.StartSpanFromContext(ctx, "CrossConnectUpdate")

			span.LogFields(log.Object("CrossConnect", xcon))
			defer span.Finish()
		}
	}
	logger = common.LogFromSpan(span)

	if cc := client.xconManager.GetClientConnectionByXcon(xcon); cc != nil {
		switch eventType {
		case monitor.EventTypeInitialStateTransfer:
			logger.Infof("Send initial transfer cross connect event: %v", xcon)
			client.monitorManager.CrossConnectMonitor().Update(xcon)
		case monitor.EventTypeUpdate:
			logger.Infof("Send cross connect event: %v", xcon)
			client.monitorManager.CrossConnectMonitor().Update(xcon)
			client.xconManager.UpdateXcon(cc, xcon)
		case monitor.EventTypeDelete:
			logger.Infof("Send cross connect delete event: %v", xcon)
			if cc.ConnectionState == model.ClientConnectionClosing {
				client.monitorManager.CrossConnectMonitor().Delete(xcon)
			}
		}
	} else {
		logger.Infof("Failed to Send cross connect event: %v. No Client connection is found", xcon)
	}

	return nil
}

func (client *NsmMonitorCrossConnectClient) handleXconEvent(event monitor.Event, dataplane *model.Dataplane, parameters map[string]string) error {
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
		return tools.DialTCP(remotePeer.GetUrl())
	}
	monitorClientSupplier := func(conn *grpc.ClientConn) (monitor.Client, error) {
		return monitor_remote.NewMonitorClient(conn, &remote.MonitorScopeSelector{
			NetworkServiceManagerName:            client.xconManager.GetNsmName(),
			DestinationNetworkServiceManagerName: remotePeer.Name,
		})
	}

	err := client.monitor(
		ctx,
		peerLogFormat, peerLogWithParamFormat, remotePeer.Name,
		grpcConnectionSupplier, monitorClientSupplier,
		client.handleRemoteConnection, nil, map[string]string{PeerName: remotePeer.Name})

	if err != nil {
		// Same as DST down case, we need to wait for same NSE and updated NSMD.
		connections := client.xconManager.GetClientConnectionByRemote(remotePeer)
		for _, cc := range connections {
			client.xconManager.DestinationDown(cc, true)
		}
	}
}

func (client *NsmMonitorCrossConnectClient) handleRemoteConnection(entity monitor.Entity, eventType monitor.EventType, parameters map[string]string) error {
	remoteConnection, ok := entity.(*remote.Connection)
	if !ok {
		return fmt.Errorf("unable to cast %v to remote.Connection", entity)
	}

	if cc := client.xconManager.GetClientConnectionByRemoteDst(remoteConnection.GetId(), parameters[PeerName]); cc != nil {
		client.handleRemoteConnectionEvent(eventType, cc, remoteConnection)
	} else {
		// We need to check if there is connections in requesting status right, now and wait until they status will be finalized
		// Or they will be removed.
		logrus.Errorf("No remote destination found %v. Will wait for pending connections to match", cc)
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), eventConnectionTimeout)
			defer cancel()
			if cc, err := client.xconManager.WaitPendingConnections(ctx, remoteConnection.GetId(), parameters[PeerName]); cc != nil && err == nil {
				client.handleRemoteConnectionEvent(eventType, cc, remoteConnection)
			}
		}()
	}

	return nil
}

func (client *NsmMonitorCrossConnectClient) handleRemoteConnectionEvent(eventType monitor.EventType, cc *model.ClientConnection, remoteConnection *remote.Connection) {
	switch eventType {
	case monitor.EventTypeInitialStateTransfer, monitor.EventTypeUpdate:
		// DST connection is updated, we most probable need to re-programm our data plane.
		client.xconManager.RemoteDestinationUpdated(cc, remoteConnection)
	case monitor.EventTypeDelete:
		// DST is down, we need to choose new NSE in any case.
		downConnection := remoteConnection.Clone()
		downConnection.SetConnectionState(connection.StateDown)

		xconToSend := crossconnect.NewCrossConnect(
			cc.Xcon.GetId(),
			cc.Xcon.GetPayload(),
			cc.Xcon.GetSourceConnection(),
			downConnection,
		)

		client.monitorManager.CrossConnectMonitor().Update(xconToSend)
		client.xconManager.DestinationDown(cc, false)
	}
}
