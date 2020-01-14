package next

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/networkservice"
)

type nextClient struct {
	clients []networkservice.NetworkServiceClient
	index   int
}

type ClientWrapper func(networkservice.NetworkServiceClient) networkservice.NetworkServiceClient
type ClientChainer func(...networkservice.NetworkServiceClient) networkservice.NetworkServiceClient

func NewWrappedNetworkServiceClient(wrapper ClientWrapper, clients ...networkservice.NetworkServiceClient) networkservice.NetworkServiceClient {
	rv := &nextClient{
		clients: clients,
	}
	for i := range rv.clients {
		rv.clients[i] = wrapper(rv.clients[i])
	}
	return rv
}

func NewNetworkServiceClient(clients ...networkservice.NetworkServiceClient) networkservice.NetworkServiceClient {
	return NewWrappedNetworkServiceClient(nil, clients...)
}

func (n *nextClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*connection.Connection, error) {
	if n.index+1 < len(n.clients) {
		return n.clients[n.index].Request(withNextClient(ctx, &nextClient{clients: n.clients, index: n.index + 1}), request, opts...)
	}
	return n.clients[n.index].Request(withNextClient(ctx, newTailClient()), request, opts...)
}

func (n *nextClient) Close(ctx context.Context, conn *connection.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	if n.index+1 < len(n.clients) {
		return n.clients[n.index].Close(withNextClient(ctx, &nextClient{clients: n.clients, index: n.index + 1}), conn, opts...)
	}
	return n.clients[n.index].Close(withNextClient(ctx, newTailClient()), conn, opts...)
}
