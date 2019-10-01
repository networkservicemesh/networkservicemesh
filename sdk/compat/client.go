package compat

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc"

	local_connection "github.com/networkservicemesh/networkservicemesh/controlplane/api/local/connection"
	local "github.com/networkservicemesh/networkservicemesh/controlplane/api/local/networkservice"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/networkservice"
	remote_connection "github.com/networkservicemesh/networkservicemesh/controlplane/api/remote/connection"
	remote "github.com/networkservicemesh/networkservicemesh/controlplane/api/remote/networkservice"
)

type localClientAdapter struct {
	networkservice.NetworkServiceClient
}

func (l localClientAdapter) Request(ctx context.Context, request *local.NetworkServiceRequest, opts ...grpc.CallOption) (*local_connection.Connection, error) {
	rv, err := l.NetworkServiceClient.Request(ctx, NetworkServiceRequestLocalToUnified(request))
	return ConnectionUnifiedToLocal(rv), err
}

func (l localClientAdapter) Close(ctx context.Context, conn *local_connection.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	return l.NetworkServiceClient.Close(ctx, ConnectionLocalToUnified(conn))
}

func NewLocalClient(conn *grpc.ClientConn) local.NetworkServiceClient {
	return &localClientAdapter{
		NetworkServiceClient: networkservice.NewNetworkServiceClient(conn),
	}
}

type remoteClientAdapter struct {
	networkservice.NetworkServiceClient
}

func (l remoteClientAdapter) Request(ctx context.Context, request *remote.NetworkServiceRequest, opts ...grpc.CallOption) (*remote_connection.Connection, error) {
	rv, err := l.NetworkServiceClient.Request(ctx, NetworkServiceRequestRemoteToUnified(request), opts...)
	return ConnectionUnifiedToRemote(rv), err
}

func (l remoteClientAdapter) Close(ctx context.Context, conn *remote_connection.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	return l.NetworkServiceClient.Close(ctx, ConnectionRemoteToUnified(conn), opts...)
}

func NewRemoteClient(conn *grpc.ClientConn) remote.NetworkServiceClient {
	return &remoteClientAdapter{
		NetworkServiceClient: networkservice.NewNetworkServiceClient(conn),
	}
}
