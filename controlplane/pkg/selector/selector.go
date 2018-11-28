package selector

import (
	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/local/connection"
	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/registry"
)

type Selector interface {
	SelectEndpoint(requestConnection *connection.Connection, ns *registry.NetworkService, networkServiceEndpoints []*registry.NetworkServiceEndpoint) *registry.NetworkServiceEndpoint
}
