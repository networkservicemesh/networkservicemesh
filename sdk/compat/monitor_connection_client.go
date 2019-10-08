package compat

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	local "github.com/networkservicemesh/networkservicemesh/controlplane/api/local/connection"
	remote "github.com/networkservicemesh/networkservicemesh/controlplane/api/remote/connection"
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
	if l.MonitorConnection_MonitorConnectionsClient != nil {
		m, err := l.MonitorConnection_MonitorConnectionsClient.Recv()
		if err != nil {
			return nil, err
		}
		return ConnectionEventUnifiedToLocal(m), nil
	}
	return nil, errors.New("localMonitorConnection_MonitorConnectionsClientAdapter.MonitorConnection_MonitorConnectionsClient == nil")
}

func newLocalMonitorConnection_MonitorConnectionsClientAdapter(adapted connection.MonitorConnection_MonitorConnectionsClient) *localMonitorConnection_MonitorConnectionsClientAdapter {
	return &localMonitorConnection_MonitorConnectionsClientAdapter{MonitorConnection_MonitorConnectionsClient: adapted}
}

type remoteMonitorConnection_MonitorConnectionsClientAdapter struct {
	connection.MonitorConnection_MonitorConnectionsClient
}

func (r remoteMonitorConnection_MonitorConnectionsClientAdapter) Recv() (*remote.ConnectionEvent, error) {
	if r.MonitorConnection_MonitorConnectionsClient != nil {
		m, err := r.MonitorConnection_MonitorConnectionsClient.Recv()
		if err != nil {
			return nil, err
		}
		return ConnectionEventUnifiedToRemote(m), nil
	}
	return nil, errors.New("remoteMonitorConnection_MonitorConnectionsClientAdapter.MonitorConnection_MonitorConnectionsClient == nil")
}

func newRemoteMonitorConnection_MonitorConnectionsClientAdapter(adapted connection.MonitorConnection_MonitorConnectionsClient) *remoteMonitorConnection_MonitorConnectionsClientAdapter {
	return &remoteMonitorConnection_MonitorConnectionsClientAdapter{MonitorConnection_MonitorConnectionsClient: adapted}
}

type remoteMonitorAdapter struct {
	connection.MonitorConnectionClient
}

func NewRemoteMonitorAdapter(conn *grpc.ClientConn) remote.MonitorConnectionClient {
	return &remoteMonitorAdapter{
		MonitorConnectionClient: connection.NewMonitorConnectionClient(conn),
	}
}

func (r remoteMonitorAdapter) MonitorConnections(ctx context.Context, selector *remote.MonitorScopeSelector, opts ...grpc.CallOption) (remote.MonitorConnection_MonitorConnectionsClient, error) {
	value, err := r.MonitorConnectionClient.MonitorConnections(ctx, MonitorScopeSelectorRemoteToUnified(selector), opts...)
	return newRemoteMonitorConnection_MonitorConnectionsClientAdapter(value), err
}
