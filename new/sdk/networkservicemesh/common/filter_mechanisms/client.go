package filter_mechanisms

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection/mechanisms/cls"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/networkservice"
	"github.com/networkservicemesh/networkservicemesh/new/sdk/networkservicemesh/common/client_url"
	"github.com/networkservicemesh/networkservicemesh/new/sdk/networkservicemesh/core/next"
)

type filterMechanismsClient struct{}

func NewClient() networkservice.NetworkServiceClient {
	return &filterMechanismsClient{}
}

func (f *filterMechanismsClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*connection.Connection, error) {
	u := client_url.ClientUrl(ctx)
	if u.Scheme == "unix" {
		var mechanisms []*connection.Mechanism
		for _, mechanism := range request.GetMechanismPreferences() {
			if mechanism.Cls == cls.LOCAL {
				mechanisms = append(mechanisms, mechanism)
			}
		}
		request.MechanismPreferences = mechanisms
		return next.Client(ctx).Request(ctx, request, opts...)
	}
	var mechanisms []*connection.Mechanism
	for _, mechanism := range request.GetMechanismPreferences() {
		if mechanism.Cls == cls.REMOTE {
			mechanisms = append(mechanisms, mechanism)
		}
	}
	return next.Client(ctx).Request(ctx, request, opts...)
}

func (f *filterMechanismsClient) Close(ctx context.Context, conn *connection.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	return next.Client(ctx).Close(ctx, conn, opts...)
}
