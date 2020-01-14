package commit

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/ligato/vpp-agent/api/configurator"
	"github.com/pkg/errors"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/networkservice"
	"github.com/networkservicemesh/networkservicemesh/new/sdk-vppagent/networkservicemesh/vppagent"
	"github.com/networkservicemesh/networkservicemesh/new/sdk/networkservicemesh/core/next"
)

type commitServer struct {
	vppagentCC     *grpc.ClientConn
	vppagentClient configurator.ConfiguratorClient
}

func NewServer(vppagentCC *grpc.ClientConn) networkservice.NetworkServiceServer {
	return &commitServer{
		vppagentCC:     vppagentCC,
		vppagentClient: configurator.NewConfiguratorClient(vppagentCC),
	}
}

func (c *commitServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*connection.Connection, error) {
	conf := vppagent.Config(ctx)
	_, err := c.vppagentClient.Update(ctx, &configurator.UpdateRequest{Update: conf})
	if err != nil {
		return nil, errors.Wrapf(err, "error sending config to vppagent %s: ", conf)
	}
	return next.Server(ctx).Request(ctx, request)
}

func (c *commitServer) Close(ctx context.Context, conn *connection.Connection) (*empty.Empty, error) {
	conf := vppagent.Config(ctx)
	_, err := c.vppagentClient.Delete(ctx, &configurator.DeleteRequest{Delete: conf})
	if err != nil {
		return nil, err
	}
	return next.Server(ctx).Close(ctx, conn)
}
