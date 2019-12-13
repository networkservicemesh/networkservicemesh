package test

import (
	"testing"

	"github.com/networkservicemesh/networkservicemesh/new/sdk/networkservicemesh/common/heal"
	"github.com/networkservicemesh/networkservicemesh/new/sdk/networkservicemesh/common/refresh"
	"github.com/networkservicemesh/networkservicemesh/new/sdk/networkservicemesh/common/setid"
	"github.com/networkservicemesh/networkservicemesh/new/sdk/networkservicemesh/common/timeout"
	"github.com/networkservicemesh/networkservicemesh/new/sdk/networkservicemesh/common/update_path"
	"github.com/networkservicemesh/networkservicemesh/new/sdk/networkservicemesh/core/adapters"
	"github.com/networkservicemesh/networkservicemesh/new/sdk/networkservicemesh/core/chain"
)

func TestBasic(t *testing.T) {
	nsmgr_forwarder := chain.NewNetworkServiceServer(
		authorize.NewServer(),
		setid.NewServer(),
		timeout.NewServer(),
		discover_candidates.NewServer(),
		select_endpoint.NewServer(),
		update_path.NewServer(name),
		adapters.NewClientToServer(client),
		forwarder,
		endpoint.NewServer(),
	)

	forwarder := chain.NewNetworkServiceServer(
		timeout.NewServer(),
		discover_candidates.NewStaticServer(networkServiceName),
		select_endpoint.NewServer(),
	)

	//
	//nsmgr := chain.NewNetworkServiceServer(
	//	authorize.NewServer(),
	//	setid.NewServer(),
	//	timeout.NewServer(),
	//	update_path.NewServer(name),
	//	discover_candidates.NewStaticServer(),
	//	order_candidates.NewServer(),
	//	select_endpoint.NewServer(),
	//	client.NewServer(),
	//	adapters.NewDiscoveryClientToServer(client),
	//)

	client := chain.NewNetworkServiceClient(
		setid.NewClient(),
		update_path.NewClient(name),
		heal.NewClient(monitorClient),
		refresh.NewClient(),
	)
}
