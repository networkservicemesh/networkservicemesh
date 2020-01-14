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

type commitClient struct {
	vppagentCC     *grpc.ClientConn
	vppagentClient configurator.ConfiguratorClient
}

func NewClient(vppagentCC *grpc.ClientConn) networkservice.NetworkServiceClient {
	return &commitClient{
		vppagentCC:     vppagentCC,
		vppagentClient: configurator.NewConfiguratorClient(vppagentCC),
	}
}

func (c *commitClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*connection.Connection, error) {
	conf := vppagent.Config(ctx)
	rv, err := next.Client(ctx).Request(ctx, request, opts...)
	if err != nil {
		return nil, err
	}
	_, err = c.vppagentClient.Update(ctx, &configurator.UpdateRequest{Update: conf})
	if err != nil {
		return nil, errors.Wrapf(err, "error sending config to vppagent %s: ", conf)
	}
	return rv, nil
}

func (c *commitClient) Close(ctx context.Context, conn *connection.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	conf := vppagent.Config(ctx)
	rv, err := next.Client(ctx).Close(ctx, conn)
	if err != nil {
		return nil, err
	}
	_, err = c.vppagentClient.Update(ctx, &configurator.UpdateRequest{Update: conf})
	if err != nil {
		return nil, errors.Wrapf(err, "error sending config to vppagent %s: ", conf)
	}
	return rv, nil
}
