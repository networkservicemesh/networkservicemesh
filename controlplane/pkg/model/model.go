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

// Model change listener
type ModelListener interface {
	EndpointAdded(endpoint *registry.NetworkServiceEndpoint)
	EndpointDeleted(endpoint *registry.NetworkServiceEndpoint)

	DataplaneAdded(dataplane *Dataplane)
	DataplaneDeleted(dataplane *Dataplane)
}

type Model interface {
	GetNetworkServiceEndpoints(name string) []*registry.NSERegistration

	GetEndpoint(name string) *registry.NSERegistration
	AddEndpoint(endpoint *registry.NSERegistration)
	DeleteEndpoint(name string) error

	GetDataplane(name string) *Dataplane
	AddDataplane(dataplane *Dataplane)
	DeleteDataplane(name string)
	SelectDataplane() (*Dataplane, error)

	GetNsmUrl() string
	ConnectionId() string

	AddListener(listener ModelListener)
	RemoveListener(listener ModelListener)

	SetNsm(nsm *registry.NetworkServiceManager)
	GetNsm() *registry.NetworkServiceManager
}

type impl struct {
	sync.RWMutex
	endpoints         map[string]*registry.NSERegistration
	networkServices   map[string][]*registry.NSERegistration
	dataplanes        map[string]*Dataplane
	lastConnnectionId uint64
	nsmUrl            string
	nsm               *registry.NetworkServiceManager
	listeners         []ModelListener
}

func (i *impl) AddListener(listener ModelListener) {
	i.listeners = append(i.listeners, listener)
}

func (i *impl) RemoveListener(listener ModelListener) {
	for idx, v := range i.listeners {
		if v == listener {
			i.listeners = append(i.listeners[:idx], i.listeners[idx+1:]...)
			return
		}
	}
}

func (i *impl) GetNetworkServiceEndpoints(name string) []*registry.NSERegistration {
	i.RLock()
	defer i.RUnlock()
	var endpoints = i.networkServices[name]
	if endpoints == nil {
		endpoints = []*registry.NSERegistration{}
	}
	return endpoints
}

func (i *impl) GetEndpoint(name string) *registry.NSERegistration {
	i.RLock()
	defer i.RUnlock()
	return i.endpoints[name]
}

func (i *impl) AddEndpoint(endpoint *registry.NSERegistration) {
	i.Lock()
	defer i.Unlock()
	i.endpoints[endpoint.GetNetworkserviceEndpoint().GetEndpointName()] = endpoint
	serviceName := endpoint.GetNetworkService().GetName()
	services := i.networkServices[serviceName]
	if services == nil {
		services = []*registry.NSERegistration{endpoint}
	} else {
		services = append(services, endpoint)
	}
	i.networkServices[serviceName] = services

	logrus.Infof("Endpoint added: %v", endpoint)

	for _, l := range i.listeners {
		l.EndpointAdded(endpoint.GetNetworkserviceEndpoint())
	}
}

func (i *impl) DeleteEndpoint(name string) error {
	i.Lock()
	defer i.Unlock()

	endpoint := i.endpoints[name]
	if endpoint != nil {
		services := i.networkServices[endpoint.GetNetworkService().GetName()]
		if len(services) > 1 {
			for idx, e := range services {
				if e == endpoint {
					services = append(services[:idx], services[idx+1:]...)
					break
				}
			}
			// Update services with removed item.
			i.networkServices[endpoint.GetNetworkService().GetName()] = services
		} else {
			delete(i.networkServices, endpoint.GetNetworkService().GetName())
		}
		delete(i.endpoints, name)

		for _, l := range i.listeners {
			l.EndpointDeleted(endpoint.GetNetworkserviceEndpoint())
		}
		return nil
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

	for _, l := range i.listeners {
		l.DataplaneAdded(dataplane)
	}
}

func (i *impl) DeleteDataplane(name string) {
	i.Lock()
	defer i.Unlock()

	dataplane := i.dataplanes[name]
	if dataplane != nil {
		delete(i.dataplanes, name)

		for _, l := range i.listeners {
			l.DataplaneDeleted(dataplane)
		}
	}
}

func (i *impl) GetNsmUrl() string {
	return i.nsmUrl
}

func (i *impl) GetNsm() *registry.NetworkServiceManager {
	return i.nsm
}

func (i *impl) SetNsm(nsm *registry.NetworkServiceManager) {
	i.nsm = nsm
}

func NewModel(nsmUrl string) Model {
	return &impl{
		nsmUrl:          nsmUrl,
		dataplanes:      make(map[string]*Dataplane),
		networkServices: make(map[string][]*registry.NSERegistration),
		endpoints:       make(map[string]*registry.NSERegistration),
		listeners:       []ModelListener{},
	}
}

func (i *impl) ConnectionId() string {
	i.Lock()
	defer i.Unlock()
	i.lastConnnectionId++
	return strconv.FormatUint(i.lastConnnectionId, 16)
}
