package mechanism_kernel_tap

import (
	"context"
	"fmt"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection/mechanisms/kernel"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/networkservice"
	"github.com/networkservicemesh/networkservicemesh/new/sdk/networkservicemesh/core/next"
)

type kernelTapClient struct{}

func NewClient() networkservice.NetworkServiceClient {
	return &kernelTapClient{}
}

func (k *kernelTapClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*connection.Connection, error) {
	mechanism, err := kernel.New("", "")
	if err != nil {
		return nil, err
	}
	request.MechanismPreferences = append(request.MechanismPreferences, mechanism)
	conn, err := next.Client(ctx).Request(ctx, request)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	if err := appendInterfaceConfig(ctx, conn, fmt.Sprintf("client-%s", conn.GetId())); err != nil {
		return nil, err
	}
	return conn, nil
}

func (k *kernelTapClient) Close(ctx context.Context, conn *connection.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	if err := appendInterfaceConfig(ctx, conn, fmt.Sprintf("client-%s", conn.GetId())); err != nil {
		return nil, err
	}
	return next.Client(ctx).Close(ctx, conn)
}
