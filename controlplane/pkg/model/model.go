package model

import (
	"fmt"
	"strconv"
	"sync"

	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/registry"

	local "github.com/ligato/networkservicemesh/controlplane/pkg/apis/local/connection"
	remote "github.com/ligato/networkservicemesh/controlplane/pkg/apis/remote/connection"
	"github.com/sirupsen/logrus"
)

type Dataplane struct {
	RegisteredName   string
	SocketLocation   string
	LocalMechanisms  []*local.Mechanism
	RemoteMechanisms []*remote.Mechanism
}

type Model interface {
	GetNetworkServiceEndpoints(name string) []*registry.NetworkServiceEndpoint

	GetEndpoint(name string) *registry.NetworkServiceEndpoint
	AddEndpoint(endpoint *registry.NetworkServiceEndpoint)
	DeleteEndpoint(name string) error

	GetDataplane(name string) *Dataplane
	AddDataplane(dataplane *Dataplane)
	DeleteDataplane(name string)
	SelectDataplane() (*Dataplane, error)

	GetNsmUrl() string
	ConnectionId() string
}

type impl struct {
	sync.RWMutex
	endpoints         map[string]*registry.NetworkServiceEndpoint
	networkServices   map[string][]*registry.NetworkServiceEndpoint
	dataplanes        map[string]*Dataplane
	lastConnnectionId uint64
	nsmUrl            string
}

func (i *impl) GetNetworkServiceEndpoints(name string) []*registry.NetworkServiceEndpoint {
	i.RLock()
	defer i.RUnlock()
	var endpoints = i.networkServices[name]
	if endpoints == nil {
		endpoints = []*registry.NetworkServiceEndpoint{}
	}
	return endpoints
}

func (i *impl) GetEndpoint(name string) *registry.NetworkServiceEndpoint {
	i.RLock()
	defer i.RUnlock()
	return i.endpoints[name]
}

func (i *impl) AddEndpoint(endpoint *registry.NetworkServiceEndpoint) {
	i.Lock()
	defer i.Unlock()
	i.endpoints[endpoint.EndpointName] = endpoint
	serviceName := endpoint.NetworkServiceName
	services := i.networkServices[serviceName]
	if services == nil {
		services = []*registry.NetworkServiceEndpoint{endpoint}
	} else {
		services = append(services, endpoint)
	}
	i.networkServices[serviceName] = services

	logrus.Infof("Endpoint added: %v", endpoint)
}

func (i *impl) DeleteEndpoint(name string) error {
	i.Lock()
	defer i.Unlock()

	endpoint := i.endpoints[name]
	if endpoint != nil {
		services := i.networkServices[endpoint.NetworkServiceName]
		if len(services) > 1 {
			for idx, e := range services {
				if e == endpoint {
					services = append(services[:idx], services[idx+1:]...)
					break
				}
			}
			// Update services with removed item.
			i.networkServices[endpoint.NetworkServiceName] = services
		} else {
			delete(i.networkServices, endpoint.NetworkServiceName)
		}

		delete(i.endpoints, name)
	}

	return fmt.Errorf("no endpoint with name: %s", name)
}

func (i *impl) GetDataplane(name string) *Dataplane {
	i.RLock()
	defer i.RUnlock()
	for _, dp := range i.dataplanes {
		if dp.RegisteredName == name {
			return dp
		}
	}
	return nil
}

func (i *impl) SelectDataplane() (*Dataplane, error) {
	i.Lock()
	defer i.Unlock()
	for _, v := range i.dataplanes {
		return v, nil // TODO: Return first for now
	}
	return nil, fmt.Errorf("no dataplanes registered")
}

func (i *impl) AddDataplane(dataplane *Dataplane) {
	i.Lock()
	defer i.Unlock()
	i.dataplanes[dataplane.RegisteredName] = dataplane
	logrus.Infof("Dataplane added: %v", dataplane)
}

func (i *impl) DeleteDataplane(name string) {
	i.Lock()
	defer i.Unlock()

	delete(i.dataplanes, name)
}

func (i *impl) GetNsmUrl() string {
	return i.nsmUrl
}

func NewModel(nsmUrl string) Model {
	return &impl{
		nsmUrl:          nsmUrl,
		dataplanes:      make(map[string]*Dataplane),
		networkServices: make(map[string][]*registry.NetworkServiceEndpoint),
		endpoints:       make(map[string]*registry.NetworkServiceEndpoint),
	}
}

func (i *impl) ConnectionId() string {
	i.Lock()
	defer i.Unlock()
	i.lastConnnectionId++
	return strconv.FormatUint(i.lastConnnectionId, 16)
}
