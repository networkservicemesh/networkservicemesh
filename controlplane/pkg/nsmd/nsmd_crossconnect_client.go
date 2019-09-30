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

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/api/nsm"

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

	peerName     = "peerName"
	endpointName = "endpointName"
)

type NsmMonitorCrossConnectClient struct {
	model.ListenerImpl

	monitorManager nsm.MonitorManager
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

// EndpointManager is an interface to delete endpoints with broken connection
type EndpointManager interface {
	DeleteEndpointWithBrokenConnection(ctx context.Context, endpoint *model.Endpoint) error
}

// NewMonitorCrossConnectClient creates a new NsmMonitorCrossConnectClient
func NewMonitorCrossConnectClient(model model.Model, monitorManager nsm.MonitorManager, xconManager *services.ClientConnectionManager,
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
func (client *NsmMonitorCrossConnectClient) EndpointAdded(_ context.Context, endpoint *model.Endpoint) {
	ctx, cancel := context.WithCancel(context.Background())
	client.endpoints.Store(endpoint.EndpointName(), cancel)
	go client.endpointConnectionMonitor(ctx, endpoint)
}

// EndpointDeleted implements method from Listener
func (client *NsmMonitorCrossConnectClient) EndpointDeleted(_ context.Context, endpoint *model.Endpoint) {
	if cancel, ok := client.endpoints.Load(endpoint.EndpointName()); ok {
		cancel.(context.CancelFunc)()
		client.endpoints.Delete(endpoint.EndpointName())
	}
}

// DataplaneAdded implements method from Listener
func (client *NsmMonitorCrossConnectClient) DataplaneAdded(_ context.Context, dp *model.Dataplane) {
	ctx, cancel := context.WithCancel(context.Background())
	client.dataplanes.Store(dp.RegisteredName, cancel)
	logrus.Infof("Starting Dataplane crossconnect monitoring client...")
	go client.dataplaneCrossConnectMonitor(ctx, dp)
}

// DataplaneDeleted implements method from Listener
func (client *NsmMonitorCrossConnectClient) DataplaneDeleted(ctx context.Context, dp *model.Dataplane) {
	client.xconManager.DataplaneDown(ctx, dp)
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

// ClientConnectionAdded - handle connection added
func (client *NsmMonitorCrossConnectClient) ClientConnectionAdded(ctx context.Context, clientConnection *model.ClientConnection) {
	client.startPeerMonitor(clientConnection)

	span := common.SpanHelperFromConnection(ctx, clientConnection, "ClientConnectionAdded")
	defer span.Finish()
	span.LogObject("clientConnection", clientConnection)

	client.xconManager.MarkConnectionAdded(clientConnection)
}

// ClientConnectionUpdated -  implements method from Listener
func (client *NsmMonitorCrossConnectClient) ClientConnectionUpdated(ctx context.Context, old, new *model.ClientConnection) {
	client.startPeerMonitor(new)

	span := common.SpanHelperFromConnection(ctx, new, "ClientConnectionUpdated")
	defer span.Finish()
	span.LogObject("new", new)
	span.LogObject("old", old)

	client.xconManager.MarkConnectionUpdated(new)
}

// ClientConnectionDeleted - handle client connection deleted
func (client *NsmMonitorCrossConnectClient) ClientConnectionDeleted(ctx context.Context, clientConnection *model.ClientConnection) {
	client.remotePeerLock.Lock()
	defer client.remotePeerLock.Unlock()

	span := common.SpanHelperFromConnection(ctx, clientConnection, "ClientConnectionDeleted")
	defer span.Finish()
	span.LogObject("clientConnection", clientConnection)

	span.Logger().Infof("ClientConnectionDeleted: %v", clientConnection)

	client.xconManager.MarkConnectionDeleted(clientConnection)

	if clientConnection.RemoteNsm == nil {
		span.Logger().Infof("Not a remote connection")
		return
	}
	remotePeer := client.remotePeers[clientConnection.RemoteNsm.Name]
	if remotePeer != nil {
		delete(remotePeer.connections, clientConnection.ConnectionID)
		if len(remotePeer.connections) == 0 {
			remotePeer.cancel()
			delete(client.remotePeers, clientConnection.RemoteNsm.Name)
			span.Logger().Infof("stopping remote monitor")
		}
		span.Logger().Infof("connection removed from monitor")
	} else {
		span.LogError(fmt.Errorf("remote peer for NSM is already closed: %v", clientConnection))
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
		return client.connectToEndpoint(endpoint)
	}

	err := client.monitor(
		ctx,
		endpointLogFormat, endpointLogWithParamFormat, endpoint.EndpointName(),
		grpcConnectionSupplier, monitor_local.NewMonitorClient,
		client.handleLocalConnection, nil, map[string]string{endpointName: endpoint.EndpointName()})

	if err != nil {
		if err = client.endpointManager.DeleteEndpointWithBrokenConnection(ctx, endpoint); err != nil {
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
		span := common.SpanHelperFromConnection(context.Background(), cc, "handleLocalConnection")
		defer span.Finish()
		ctx := span.Context()
		span.LogObject("clientConnection", cc)
		span.LogValue("event", entity)
		span.LogObject("eventType", eventType)

		switch eventType {
		case monitor.EventTypeUpdate:
			// DST connection is updated, we most probable need to re-programm our data plane.
			client.xconManager.LocalDestinationUpdated(ctx, cc, localConnection)
		case monitor.EventTypeDelete:
			// DST is down, we need to choose new NSE in any case.
			client.xconManager.DestinationDown(ctx, cc, false)
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
	clientConnection := client.xconManager.GetClientConnectionByXcon(xcon)

	client.xconManager.CleanupDeletedConnections()

	span := common.SpanHelperFromConnection(context.Background(), clientConnection, "CrossConnectUpdate")
	defer span.Finish()
	span.LogObject("clientConnection", clientConnection)
	span.LogValue("event", entity)
	span.LogObject("eventType", eventType)

	if clientConnection != nil {
		switch eventType {
		case monitor.EventTypeInitialStateTransfer:
			span.Logger().Infof("Send initial transfer cross connect event: %v", xcon)
			client.monitorManager.CrossConnectMonitor().Update(span.Context(), xcon)
		case monitor.EventTypeUpdate:
			span.Logger().Infof("Send cross connect event: %v", xcon)
			client.monitorManager.CrossConnectMonitor().Update(span.Context(), xcon)
			client.xconManager.UpdateXcon(span.Context(), clientConnection, xcon)
		case monitor.EventTypeDelete:
			span.Logger().Infof("Send cross connect delete event: %v", xcon)
			client.monitorManager.CrossConnectMonitor().Delete(span.Context(), xcon)
		}
	} else {
		span.LogError(fmt.Errorf("failed to Send cross connect event: %v. No Client connection is found", xcon))
	}

	return nil
}

func (client *NsmMonitorCrossConnectClient) handleXconEvent(event monitor.Event, dataplane *model.Dataplane, _ map[string]string) error {
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

		client.xconManager.UpdateFromInitialState(xcons, dataplane, client.monitorManager)
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
		client.handleRemoteConnection, nil, map[string]string{peerName: remotePeer.Name})

	if err != nil {
		// Same as DST down case, we need to wait for same NSE and updated NSMD.
		connections := client.xconManager.GetClientConnectionByRemote(remotePeer)
		for _, cc := range connections {
			span := common.SpanHelperFromConnection(ctx, cc, "remotePeerConnectionMonitor.update")
			defer span.Finish()
			ctx = span.Context()
			span.LogObject("clientConnection", cc)
			client.xconManager.DestinationDown(ctx, cc, true)
		}
	}
}

func (client *NsmMonitorCrossConnectClient) handleRemoteConnection(entity monitor.Entity, eventType monitor.EventType, parameters map[string]string) error {
	remoteConnection, ok := entity.(*remote.Connection)
	if !ok {
		return fmt.Errorf("unable to cast %v to remote.Connection", entity)
	}
	peerName := parameters[peerName]
	if cc := client.xconManager.GetClientConnectionByRemoteDst(remoteConnection.GetId(), peerName); cc != nil {
		span := common.SpanHelperFromConnection(context.Background(), cc, "handleRemoteConnection")
		defer span.Finish()
		ctx := span.Context()
		span.LogObject("clientConnection", cc)
		span.LogObject("RemoteConnect", remoteConnection)
		span.LogValue("peerName", peerName)
		span.LogValue("eventType", eventType)
		span.LogObject("entity", entity)

		client.handleRemoteConnectionEvent(ctx, eventType, cc, remoteConnection)
	} else {
		// We need to check if there is connections in requesting status right, now and wait until they status will be finalized
		// Or they will be removed.
		logrus.Errorf("No remote destination found %v. Will wait for pending connections to match", cc)
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), eventConnectionTimeout)
			defer cancel()

			currentTime := time.Now()
			if cc, err := client.xconManager.WaitPendingConnections(ctx, remoteConnection.GetId(), parameters[peerName]); cc != nil && err == nil {
				span := common.SpanHelperFromConnection(ctx, cc, "handleRemoteConnection")
				defer span.Finish()
				span.LogObject("clientConnection", cc)
				span.LogObject("RemoteConnect", remoteConnection)
				span.LogValue("peerName", peerName)
				span.LogValue("eventType", eventType)
				span.LogObject("entity", entity)

				span.LogValue("waitTime", fmt.Sprintf("%v", time.Since(currentTime)))

				client.handleRemoteConnectionEvent(span.Context(), eventType, cc, remoteConnection)
			}
		}()
	}

	return nil
}

func (client *NsmMonitorCrossConnectClient) handleRemoteConnectionEvent(ctx context.Context, eventType monitor.EventType, cc *model.ClientConnection, remoteConnection *remote.Connection) {
	switch eventType {
	case monitor.EventTypeInitialStateTransfer, monitor.EventTypeUpdate:
		// DST connection is updated, we most probable need to re-program our data plane.
		client.xconManager.RemoteDestinationUpdated(ctx, cc, remoteConnection)
	case monitor.EventTypeDelete:
		span := common.SpanHelperFromContext(ctx, "handleRemoteConnectionEvent")
		defer span.Finish()
		ctx = span.Context()

		// DST is down, we need to choose new NSE in any case.
		downConnection := remoteConnection.Clone()
		downConnection.SetConnectionState(connection.StateDown)

		xconToSend := crossconnect.NewCrossConnect(
			cc.Xcon.GetId(),
			cc.Xcon.GetPayload(),
			cc.Xcon.GetSourceConnection(),
			downConnection,
		)

		span.LogObject("xcon-event", xconToSend)

		client.monitorManager.CrossConnectMonitor().Update(ctx, xconToSend)
		client.xconManager.DestinationDown(ctx, cc, false)
	}
}
