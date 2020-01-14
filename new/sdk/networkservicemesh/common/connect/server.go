package connect

import (
	"context"
	"net/url"
	"sync"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/networkservice"
	"github.com/networkservicemesh/networkservicemesh/new/sdk/networkservicemesh/common/client_url"
	"github.com/networkservicemesh/networkservicemesh/new/sdk/networkservicemesh/core/next"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
)

type connectServer struct {
	clientFactory     func(conn *grpc.ClientConn) networkservice.NetworkServiceClient
	clients           sync.Map
	ccs               sync.Map
	clientConnections sync.Map
}

func NewServer(clientFactory func(conn *grpc.ClientConn) networkservice.NetworkServiceClient) networkservice.NetworkServiceServer {
	return &connectServer{
		clientFactory: clientFactory,
	}
}

func (c *connectServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*connection.Connection, error) {
	client, err := c.client(ctx)
	if err != nil {
		return nil, err
	}
	clientConn, err := client.Request(ctx, request)
	if err != nil {
		return nil, err
	}
	ctx = withClientConnection(ctx, clientConn)
	c.clientConnections.Store(request.GetConnection().GetId(), clientConn)
	return next.Server(ctx).Request(ctx, request)
}

func (c *connectServer) Close(ctx context.Context, conn *connection.Connection) (*empty.Empty, error) {
	client, err := c.client(ctx)
	if err != nil {
		return nil, err
	}
	if untypedClientConn, ok := c.clientConnections.Load(conn.GetId()); ok {
		_, err := client.Close(ctx, untypedClientConn.(*connection.Connection))
		if err != nil {
			return nil, err
		}
	}
	return next.Server(ctx).Close(ctx, conn)
}

func (c *connectServer) cc(u *url.URL) (*grpc.ClientConn, error) {
	untypedCC, found := c.ccs.Load(u.String())
	if found {
		cc, correctType := untypedCC.(*grpc.ClientConn)
		if correctType {
			return cc, nil
		}
		return nil, errors.Errorf("non *grpc.Clientconn stored improperly in sync.Map: %+v", untypedCC)
	}
	cc, err := tools.DialUrl(u)
	if err != nil {
		return nil, err
	}
	untypedCC, found = c.ccs.LoadOrStore(u.String(), cc)
	if found {
		cc, correctType := untypedCC.(*grpc.ClientConn)
		if correctType {
			return cc, nil
		}
	}
	// If we get here... there *is* a value in the map, and its not the one we created... so trying again will return it.
	return c.cc(u)
}

func (c *connectServer) client(ctx context.Context) (networkservice.NetworkServiceClient, error) {
	u := client_url.ClientUrl(ctx)
	untypedClient, found := c.clients.Load(u.String())
	if found {
		client, correctType := untypedClient.(networkservice.NetworkServiceClient)
		if correctType {
			return client, nil
		}
		return nil, errors.Errorf("non networkserviceClient stored improperly in sync.Map: %+v", untypedClient)
	}
	cc, err := c.cc(u)
	if err != nil {
		return nil, err
	}
	return c.clientFactory(cc), nil
}
