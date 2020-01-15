package next

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/networkservice"
)

type nextServer struct {
	servers []networkservice.NetworkServiceServer
	index   int
}

type ServerWrapper func(networkservice.NetworkServiceServer) networkservice.NetworkServiceServer
type ServerChainer func(...networkservice.NetworkServiceServer) networkservice.NetworkServiceServer

func NewWrappedNetworkServiceServer(wrapper ServerWrapper, servers ...networkservice.NetworkServiceServer) networkservice.NetworkServiceServer {
	rv := &nextServer{
		servers: servers,
	}
	for i := range rv.servers {
		rv.servers[i] = wrapper(rv.servers[i])
	}
	return rv
}

func NewNetworkServiceServer(servers ...networkservice.NetworkServiceServer) networkservice.NetworkServiceServer {
	return NewWrappedNetworkServiceServer(nil, servers...)
}

func (n *nextServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*connection.Connection, error) {
	if n.index+1 < len(n.servers) {
		return n.servers[n.index].Request(withNextServer(ctx, &nextServer{servers: n.servers, index: n.index + 1}), request)
	}
	return n.servers[n.index].Request(withNextServer(ctx, newTailServer()), request)
}

func (n *nextServer) Close(ctx context.Context, conn *connection.Connection) (*empty.Empty, error) {
	if n.index+1 < len(n.servers) {
		return n.servers[n.index].Close(withNextServer(ctx, &nextServer{servers: n.servers, index: n.index + 1}), conn)
	}
	return n.servers[n.index].Close(withNextServer(ctx, newTailServer()), conn)
}
