package selector

import (
	"sync"

	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/local/connection"
	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/registry"
)

type roundRobinSelector struct {
	sync.Mutex
	roundRobin map[string]int
}

func NewRoundRobinSelector() Selector {
	return &roundRobinSelector{
		roundRobin: make(map[string]int),
	}
}

func (rr *roundRobinSelector) SelectEndpoint(requestConnection *connection.Connection, ns *registry.NetworkService, networkServiceEndpoints []*registry.NetworkServiceEndpoint) *registry.NetworkServiceEndpoint {
	if rr == nil {
		return nil
	}
	if len(networkServiceEndpoints) == 0 {
		return nil
	}
	rr.Lock()
	defer rr.Unlock()
	idx := rr.roundRobin[ns.GetName()] % len(networkServiceEndpoints)
	endpoint := networkServiceEndpoints[idx]
	if endpoint == nil {
		return nil
	}
	rr.roundRobin[ns.GetName()] = rr.roundRobin[ns.GetName()] + 1
	return endpoint
}
