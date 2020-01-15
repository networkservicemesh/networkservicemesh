package client

import (
	"google.golang.org/grpc"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/networkservice"
	"github.com/networkservicemesh/networkservicemesh/new/sdk/networkservicemesh/common/authorize"
	"github.com/networkservicemesh/networkservicemesh/new/sdk/networkservicemesh/common/heal"
	"github.com/networkservicemesh/networkservicemesh/new/sdk/networkservicemesh/common/refresh"
	"github.com/networkservicemesh/networkservicemesh/new/sdk/networkservicemesh/common/setid"
	"github.com/networkservicemesh/networkservicemesh/new/sdk/networkservicemesh/common/update_path"
	"github.com/networkservicemesh/networkservicemesh/new/sdk/networkservicemesh/core/chain"
)

func NewClient(name string, onHeal networkservice.NetworkServiceClient, cc *grpc.ClientConn, additionalFunctionality ...networkservice.NetworkServiceClient) networkservice.NetworkServiceClient {
	return chain.NewNetworkServiceClient(
		append(
			append([]networkservice.NetworkServiceClient{
				authorize.NewClient(),
				setid.NewClient(name),
				heal.NewClient(connection.NewMonitorConnectionClient(cc), onHeal),
				refresh.NewClient(),
				update_path.NewClient(name),
			}, additionalFunctionality...),
			networkservice.NewNetworkServiceClient(cc),
		)...)
}

func NewClientFactory(name string, onHeal networkservice.NetworkServiceClient, additionalFunctionality ...networkservice.NetworkServiceClient) func(cc *grpc.ClientConn) networkservice.NetworkServiceClient {
	return func(cc *grpc.ClientConn) networkservice.NetworkServiceClient {
		return NewClient(name, onHeal, cc, additionalFunctionality...)
	}
}
