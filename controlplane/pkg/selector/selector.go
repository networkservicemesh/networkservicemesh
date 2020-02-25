package selector

import (
	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/registry"
)

type Selector interface {
	SelectEndpoint(requestConnection *networkservice.Connection, ns *registry.NetworkService, networkServiceEndpoints []*registry.NetworkServiceEndpoint) *registry.NetworkServiceEndpoint
}
