package selector

import "github.com/ligato/networkservicemesh/controlplane/pkg/apis/registry"

type Selector interface {
	SelectEndpoint(ns *registry.NetworkService, networkServiceEndpoints []*registry.NetworkServiceEndpoint) *registry.NetworkServiceEndpoint
}
