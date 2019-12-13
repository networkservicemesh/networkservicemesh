package round_robin

import (
	"sync"

	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/registry"
)

type roundRobinSelector struct {
	// TODO - replace this with something like a sync.Map that doesn't require us to lock all requests for a simple
	// selection
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
	logrus.Infof("RoundRobin selected %v", endpoint)
	return endpoint
}
