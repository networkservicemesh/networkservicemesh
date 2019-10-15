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

type localNetworkServiceClientAdapter struct {
	networkservice.NetworkServiceClient
}

func (l localNetworkServiceClientAdapter) Request(ctx context.Context, request *local.NetworkServiceRequest, opts ...grpc.CallOption) (*local_connection.Connection, error) {
	rv, err := l.NetworkServiceClient.Request(ctx, NetworkServiceRequestLocalToUnified(request))
	return ConnectionUnifiedToLocal(rv), err
}

func (l localNetworkServiceClientAdapter) Close(ctx context.Context, conn *local_connection.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	return l.NetworkServiceClient.Close(ctx, ConnectionLocalToUnified(conn))
}

func NewLocalNetworkServiceClient(conn *grpc.ClientConn) local.NetworkServiceClient {
	return &localNetworkServiceClientAdapter{
		NetworkServiceClient: networkservice.NewNetworkServiceClient(conn),
	}
}

type remoteNetworkServiceClientAdapter struct {
	networkservice.NetworkServiceClient
}

func (l remoteNetworkServiceClientAdapter) Request(ctx context.Context, request *remote.NetworkServiceRequest, opts ...grpc.CallOption) (*remote_connection.Connection, error) {
	rv, err := l.NetworkServiceClient.Request(ctx, NetworkServiceRequestRemoteToUnified(request), opts...)
	return ConnectionUnifiedToRemote(rv), err
}

func (l remoteNetworkServiceClientAdapter) Close(ctx context.Context, conn *remote_connection.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	return l.NetworkServiceClient.Close(ctx, ConnectionRemoteToUnified(conn), opts...)
}

func NewRemoteNetworkServiceClient(conn *grpc.ClientConn) remote.NetworkServiceClient {
	return &remoteNetworkServiceClientAdapter{
		NetworkServiceClient: networkservice.NewNetworkServiceClient(conn),
	}
}
