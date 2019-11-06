package endpoint

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/open-policy-agent/opa/rego"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/networkservice"
)

type AuthEndpoint struct {
	policy rego.PreparedEvalQuery
}

func (auth *AuthEndpoint) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*connection.Connection, error) {
	rs, err := auth.policy.Eval(ctx, rego.EvalInput(request))
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	for _, r := range rs {
		for _, e := range r.Expressions {
			t, ok := e.Value.(bool)
			if !ok {
				return nil, status.Error(codes.Internal, "non boolean expression")
			}

			if !t {
				return nil, status.Error(codes.PermissionDenied, "no sufficient privileges to call Request")
			}
		}
	}

	if Next(ctx) != nil {
		return Next(ctx).Request(ctx, request)
	}
	return request.GetConnection(), nil
}

func (auth *AuthEndpoint) Close(ctx context.Context, connection *connection.Connection) (*empty.Empty, error) {
	if Next(ctx) != nil {
		return Next(ctx).Close(ctx, connection)
	}
	return &empty.Empty{}, nil
}

func NewAuthEndpoint(policy rego.PreparedEvalQuery) networkservice.NetworkServiceServer {
	return &AuthEndpoint{
		policy: policy,
	}
}
