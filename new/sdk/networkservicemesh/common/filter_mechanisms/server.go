package filter_mechanisms

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc/peer"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection/mechanisms/cls"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/networkservice"
	"github.com/networkservicemesh/networkservicemesh/new/sdk/networkservicemesh/core/next"
)

type filterMechanismsServer struct{}

func NewServer() networkservice.NetworkServiceServer {
	return &filterMechanismsServer{}
}

func (f *filterMechanismsServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*connection.Connection, error) {
	p, ok := peer.FromContext(ctx)
	if ok {
		if p.Addr.Network() == "unix" {
			var mechanisms []*connection.Mechanism
			for _, mechanism := range request.GetMechanismPreferences() {
				if mechanism.Cls == cls.LOCAL {
					mechanisms = append(mechanisms, mechanism)
				}
			}
			request.MechanismPreferences = mechanisms
			return next.Client(ctx).Request(ctx, request)
		}
		var mechanisms []*connection.Mechanism
		for _, mechanism := range request.GetMechanismPreferences() {
			if mechanism.Cls == cls.REMOTE {
				mechanisms = append(mechanisms, mechanism)
			}
		}
	}
	return next.Server(ctx).Request(ctx, request)
}

func (f *filterMechanismsServer) Close(ctx context.Context, conn *connection.Connection) (*empty.Empty, error) {
	return next.Server(ctx).Close(ctx, conn)
}
