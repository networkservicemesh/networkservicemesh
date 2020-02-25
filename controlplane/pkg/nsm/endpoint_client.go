package nsm

import (
	"golang.org/x/net/context"
	"google.golang.org/grpc"

	"github.com/pkg/errors"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
)

//// Endpoint Connection Client
type endpointClient struct {
	client     networkservice.NetworkServiceClient
	connection *grpc.ClientConn
}

func (c *endpointClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	if c == nil || c.client == nil {
		return nil, errors.New("NSE Connection is not initialized...")
	}

	response, err := c.client.Request(ctx, request)
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

func (c *endpointClient) Close(ctx context.Context, conn *networkservice.Connection) error {
	if c.client == nil {
		return errors.New("Remote NSM Connection is already cleaned...")
	}
	_, err := c.client.Close(ctx, conn)
	_ = c.Cleanup()
	return err
}
