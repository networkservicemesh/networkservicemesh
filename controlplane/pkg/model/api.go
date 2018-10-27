package model

import (
	"sync"

	"github.com/ligato/networkservicemesh/pkg/nsm/apis/netmesh"
)

type Model interface {
	GetNetworkService(name string) *netmesh.NetworkService
	GetNetworkServiceEndpoints(name string) []*netmesh.NetworkServiceEndpoint

	GetEndpoint(name string) (*netmesh.NetworkServiceEndpoint, bool)
	AddEndpoint(endpoint *netmesh.NetworkServiceEndpoint)

	GetDataplane(name string) *Dataplane
	AddDataplane(dataplane *Dataplane)
	DeleteDataplane(name string)
}

type impl struct {
	sync.RWMutex
	endpoints map[string]*netmesh.NetworkServiceEndpoint
}

func (i *impl) GetNetworkService(name string) *netmesh.NetworkService {
	return nil
}

func (i *impl) GetNetworkServiceEndpoints(name string) []*netmesh.NetworkServiceEndpoint {
	return nil
}

func (i *impl) GetEndpoint(name string) (*netmesh.NetworkServiceEndpoint, bool) {
	i.RLock()
	defer i.RUnlock()
	r, ok := i.endpoints[name]
	return r, ok
}

func (i *impl) AddEndpoint(endpoint *netmesh.NetworkServiceEndpoint) {
	i.Lock()
	i.endpoints[endpoint.NseProviderName] = endpoint
	i.Unlock()
}

func (i *impl) GetDataplane(name string) *Dataplane {
	return nil
}

func (i *impl) AddDataplane(dataplane *Dataplane) {}

func (i *impl) DeleteDataplane(name string) {}

func NewModel() Model {
	return &impl{
		endpoints: make(map[string]*netmesh.NetworkServiceEndpoint),
	}
}
