package nsm

import (
	"fmt"
	"github.com/golang/protobuf/proto"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/nsm"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/remote/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/remote/networkservice"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

//// Remote NSM Connection Client
type nsmClient struct {
	client     networkservice.NetworkServiceClient
	connection *grpc.ClientConn
}

func (c *nsmClient) Request(ctx context.Context, request nsm.NSMRequest) (nsm.NSMConnection, error) {
	if c == nil || c.client == nil {
		return nil, fmt.Errorf("Remote NSM Connection is not initialized...")
	}
	response, err := c.client.Request(ctx, request.(*networkservice.NetworkServiceRequest))
	return proto.Clone(response).(*connection.Connection), err
}
func (c *nsmClient) Close(ctx context.Context, conn nsm.NSMConnection) error {
	if c == nil || c.client == nil {
		return fmt.Errorf("Remote NSM Connection is not initialized...")
	}
	_, err := c.client.Close(ctx, conn.(*connection.Connection))
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
