package nsm

import (
	"golang.org/x/net/context"
	"google.golang.org/grpc"

	"github.com/pkg/errors"

	local_connection "github.com/networkservicemesh/networkservicemesh/controlplane/api/local/connection"
	local_networkservice "github.com/networkservicemesh/networkservicemesh/controlplane/api/local/networkservice"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/nsm/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/nsm/networkservice"
)

//// Endpoint Connection Client
type endpointClient struct {
	client     local_networkservice.NetworkServiceClient
	connection *grpc.ClientConn
}

func (c *endpointClient) Request(ctx context.Context, request networkservice.Request) (connection.Connection, error) {
	if c == nil || c.client == nil {
		return nil, errors.New("NSE Connection is not initialized...")
	}

	response, err := c.client.Request(ctx, request.(*local_networkservice.NetworkServiceRequest))
	if err != nil {
		return nil, err
	}

	return response.Clone(), nil
}
func (c *endpointClient) Cleanup() error {
	if c == nil || c.client == nil {
		return errors.New("NSE Connection is not initialized...")
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
		return errors.New("Remote NSM Connection is already cleaned...")
	}
	_, err := c.client.Close(ctx, conn.(*local_connection.Connection))
	_ = c.Cleanup()
	return err
}
