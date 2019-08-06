package nsm

import (
	"fmt"

	"golang.org/x/net/context"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/nsm/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/nsm/networkservice"
	remote_connection "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/remote/connection"
	remote_networkservice "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/remote/networkservice"
)

//// Remote NSM Connection Client
type nsmClient struct {
	client     remote_networkservice.NetworkServiceClient
	connection *grpc.ClientConn
}

func (c *nsmClient) Request(ctx context.Context, request networkservice.Request) (networkservice.Reply, error) {
	if c == nil || c.client == nil {
		return nil, fmt.Errorf("Remote NSM Connection is not initialized...")
	}

	reply, err := c.client.Request(ctx, request.(*remote_networkservice.NetworkServiceRequest))
	if err != nil {
		return nil, err
	}

	return reply.Clone(), nil
}

func (c *nsmClient) Close(ctx context.Context, conn connection.Connection) error {
	if c == nil || c.client == nil {
		return fmt.Errorf("Remote NSM Connection is not initialized...")
	}
	_, err := c.client.Close(ctx, conn.(*remote_connection.Connection))
	_ = c.Cleanup()
	return err
}

func (c *nsmClient) Cleanup() error {
	if c.client == nil {
		return fmt.Errorf("Remote NSM Connection is already cleaned...")
	}
	var err error
	if c.connection != nil { // Required for testing
		err = c.connection.Close()
	}
	c.connection = nil
	c.client = nil
	return err
}
