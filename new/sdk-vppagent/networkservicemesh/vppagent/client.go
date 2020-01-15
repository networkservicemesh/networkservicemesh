package vppagent

import (
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/networkservice"
	"github.com/networkservicemesh/networkservicemesh/new/sdk/networkservicemesh/core/adapters"
)

type configClient struct{}

func NewClient() networkservice.NetworkServiceClient {
	return adapters.NewServerToClient(&configServer{})
}
