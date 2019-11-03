package nsm

import (
	"golang.org/x/net/context"
	"google.golang.org/grpc"

	"github.com/pkg/errors"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/networkservice"
)

//// Remote NSM Connection Client
type nsmClient struct {
	client     networkservice.NetworkServiceClient
	connection *grpc.ClientConn
}

func (c *nsmClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*connection.Connection, error) {
	if c == nil || c.client == nil {
		return nil, errors.New("Remote NSM Connection is not initialized...")
	}

	response, err := c.client.Request(ctx, request)
	if err != nil {
		return nil, err
	}

	return response.Clone(), nil
}

func (c *nsmClient) Close(ctx context.Context, conn *connection.Connection) error {
	if c == nil || c.client == nil {
		return errors.New("Remote NSM Connection is not initialized...")
	}
	_, err := c.client.Close(ctx, conn)
	_ = c.Cleanup()
	return err
}

func (c *nsmClient) Cleanup() error {
	if c.client == nil {
		return errors.Errorf("Remote NSM Connection is already cleaned...")
	}
	var err error
	if c.connection != nil { // Required for testing
		err = c.connection.Close()
	}
	c.connection = nil
	c.client = nil
	return err
}
