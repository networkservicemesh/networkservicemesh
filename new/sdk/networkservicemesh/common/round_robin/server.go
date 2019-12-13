package round_robin

import (
	"context"
	"net/url"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/networkservice"
	"github.com/networkservicemesh/networkservicemesh/new/sdk/networkservicemesh/common/client_url"
	"github.com/networkservicemesh/networkservicemesh/new/sdk/networkservicemesh/common/discover_candidates"
	"github.com/networkservicemesh/networkservicemesh/new/sdk/networkservicemesh/core/next"
	"github.com/pkg/errors"
)

type selectEndpointServer struct {
	selector Selector
}

func NewServer() networkservice.NetworkServiceServer {
	return &selectEndpointServer{
		selector: NewRoundRobinSelector(),
	}
}

func (s *selectEndpointServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*connection.Connection, error) {
	ctx, err := s.withClientUrl(ctx, request.GetConnection())
	if err != nil {
		return nil, err
	}
	// TODO - set Request.Connection.NetworkServiceEndpoint if unset
	return next.Server(ctx).Request(ctx, request)
	panic("implement me")
}

func (s *selectEndpointServer) Close(ctx context.Context, conn *connection.Connection) (*empty.Empty, error) {
	ctx, err := s.withClientUrl(ctx, conn)
	if err != nil {
		return nil, err
	}
	panic("implement me")
}

func (s *selectEndpointServer) withClientUrl(ctx context.Context, conn *connection.Connection) (context.Context, error) {
	// TODO - handle local case
	candidates := discover_candidates.Candidates(ctx)
	endpoint := s.selector.SelectEndpoint(conn, candidates.GetNetworkService(), candidates.GetNetworkServiceEndpoints())
	url_string := candidates.GetNetworkServiceManagers()[endpoint.GetNetworkServiceManagerName()].GetUrl()
	u, err := url.Parse(url_string)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	ctx = client_url.WithClientUrl(ctx, u)
	return ctx, nil
}
