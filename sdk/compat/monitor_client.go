package compat

import (
	"context"
	"fmt"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	local "github.com/networkservicemesh/networkservicemesh/controlplane/api/local/connection"
	remote "github.com/networkservicemesh/networkservicemesh/controlplane/api/remote/connection"
	"google.golang.org/grpc"
)

type localMonitorAdapter struct {
	connection.MonitorConnectionClient
}

func (l localMonitorAdapter) MonitorConnections(ctx context.Context, in *empty.Empty, opts ...grpc.CallOption) (local.MonitorConnection_MonitorConnectionsClient, error) {
	value, err := l.MonitorConnectionClient.MonitorConnections(ctx, &connection.MonitorScopeSelector{NetworkServiceManagers: make([]string, 1)}, opts...)
	return newLocalMonitorConnection_MonitorConnectionsClientAdapter(value), err
}

func NewLocalMonitorAdapter(conn *grpc.ClientConn) local.MonitorConnectionClient {
	return &localMonitorAdapter{
		MonitorConnectionClient: connection.NewMonitorConnectionClient(conn),
	}
}

type localMonitorConnection_MonitorConnectionsClientAdapter struct {
	connection.MonitorConnection_MonitorConnectionsClient
}

func (l localMonitorConnection_MonitorConnectionsClientAdapter) Recv() (*local.ConnectionEvent, error) {
	m := new(connection.ConnectionEvent)
	if err := l.RecvMsg(m); err != nil {
		return nil, err
	}
	return ConnectionEventUnifiedToLocal(m), nil
}

func (l localMonitorConnection_MonitorConnectionsClientAdapter) SendMsg(m interface{}) error {
	if rv, ok := m.(*local.ConnectionEvent); ok {
		unifiedEvent := ConnectionEventLocalToUnified(rv)
		return l.MonitorConnection_MonitorConnectionsClient.SendMsg(unifiedEvent)
	}
	return fmt.Errorf("localMonitorConnection_MonitorConnectionsClientAdapter.SendMsg(m) - m was not of type local.ConnectionEvent - %+v", m)
}

func (l localMonitorConnection_MonitorConnectionsClientAdapter) RecvMsg(m interface{}) error {
	if rv, ok := m.(*local.ConnectionEvent); ok {
		unifiedEvent := new(connection.ConnectionEvent)
		if err := l.MonitorConnection_MonitorConnectionsClient.RecvMsg(unifiedEvent); err != nil {
			return err
		}
		localEvent := ConnectionEventUnifiedToLocal(unifiedEvent)
		rv.Type = localEvent.Type
		rv.Connections = localEvent.Connections
		return nil
	}
	return fmt.Errorf("localMonitorConnection_MonitorConnectionsClientAdapter.RecvMsg(m) - m was not of type local.ConnectionEvent - %+v", m)
}

func newLocalMonitorConnection_MonitorConnectionsClientAdapter(adapted connection.MonitorConnection_MonitorConnectionsClient) *localMonitorConnection_MonitorConnectionsClientAdapter {
	return &localMonitorConnection_MonitorConnectionsClientAdapter{MonitorConnection_MonitorConnectionsClient: adapted}
}

type remoteMonitorConnection_MonitorConnectionsClientAdapter struct {
	connection.MonitorConnection_MonitorConnectionsClient
}

func (r remoteMonitorConnection_MonitorConnectionsClientAdapter) Recv() (*remote.ConnectionEvent, error) {
	m := new(connection.ConnectionEvent)
	if err := r.RecvMsg(m); err != nil {
		return nil, err
	}
	return ConnectionEventUnifiedToRemote(m), nil
}

func (r remoteMonitorConnection_MonitorConnectionsClientAdapter) SendMsg(m interface{}) error {
	if rv, ok := m.(*remote.ConnectionEvent); ok {
		unifiedEvent := ConnectionEventRemoteToUnified(rv)
		return r.MonitorConnection_MonitorConnectionsClient.SendMsg(unifiedEvent)
	}
	return fmt.Errorf("remoteMonitorConnection_MonitorConnectionsClientAdapter.SendMsg(m) - m was not of type remote.ConnectionEvent - %+v", m)
}

func (r remoteMonitorConnection_MonitorConnectionsClientAdapter) RecvMsg(m interface{}) error {
	if rv, ok := m.(*remote.ConnectionEvent); ok {
		unifiedEvent := new(connection.ConnectionEvent)
		if err := r.MonitorConnection_MonitorConnectionsClient.RecvMsg(unifiedEvent); err != nil {
			return err
		}
		remoteEvent := ConnectionEventUnifiedToRemote(unifiedEvent)
		rv.Type = remoteEvent.Type
		rv.Connections = remoteEvent.Connections
		return nil
	}
	return fmt.Errorf("remoteMonitorConnection_MonitorConnectionsClientAdapter.RecvMsg(m) - m was not of type remote.ConnectionEvent - %+v", m)
}

func newRemoteMonitorConnection_MonitorConnectionsClientAdapter(adapted connection.MonitorConnection_MonitorConnectionsClient) *remoteMonitorConnection_MonitorConnectionsClientAdapter {
	return &remoteMonitorConnection_MonitorConnectionsClientAdapter{MonitorConnection_MonitorConnectionsClient: adapted}
}
