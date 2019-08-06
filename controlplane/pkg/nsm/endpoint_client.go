package nsm

import (
	"fmt"

	"golang.org/x/net/context"
	"google.golang.org/grpc"

	local_connection "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/connection"
	local_networkservice "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/networkservice"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/nsm/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/nsm/networkservice"
)

//// Endpoint Connection Client
type endpointClient struct {
	client     local_networkservice.NetworkServiceClient
	connection *grpc.ClientConn
}

func (c *endpointClient) Request(ctx context.Context, request networkservice.Request) (networkservice.Reply, error) {
	if c == nil || c.client == nil {
		return nil, fmt.Errorf("NSE Connection is not initialized...")
	}

	reply, err := c.client.Request(ctx, request.(*local_networkservice.NetworkServiceRequest))
	if err != nil {
		return nil, err
	}

	return reply.Clone(), nil
}
func (c *endpointClient) Cleanup() error {
	if c == nil || c.client == nil {
		return fmt.Errorf("NSE Connection is not initialized...")
	}
	var err error
	if c.connection != nil { // Required for testing
		err = c.connection.Close()
	}
	c.connection = nil
	c.client = nil
	return err
}

func (c *endpointClient) Close(ctx context.Context, conn connection.Connection) error {
	if c.client == nil {
		return fmt.Errorf("Remote NSM Connection is already cleaned...")
	}
	_, err := c.client.Close(ctx, conn.(*local_connection.Connection))
	_ = c.Cleanup()
	return err
}
