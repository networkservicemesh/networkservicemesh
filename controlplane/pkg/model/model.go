package model

import (
	"fmt"
	"github.com/ligato/networkservicemesh/controlplane/pkg/model/registry"
	"sync"

	"github.com/ligato/networkservicemesh/pkg/nsm/apis/common"
	"github.com/ligato/networkservicemesh/pkg/nsm/apis/netmesh"
	"github.com/sirupsen/logrus"
)

type Dataplane struct {
	RegisteredName   string
	SocketLocation   string
	LocalMechanisms  []*common.LocalMechanism
	RemoteMechanisms []*common.RemoteMechanism
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
}

type impl struct {
	sync.RWMutex
	endpoints  []*registry.NetworkServiceEndpoint
	services   []*netmesh.NetworkService
	dataplanes []*Dataplane
}

func (i *impl) GetNetworkServiceEndpoints(name string) []*registry.NetworkServiceEndpoint {
	i.RLock()
	defer i.RUnlock()
	var endpoints []*registry.NetworkServiceEndpoint
	for _, e := range i.endpoints {
		if e.NetworkServiceName == name {
			endpoints = append(endpoints, e)
		}
	}
	return endpoints
}

func (i *impl) GetEndpoint(name string) *registry.NetworkServiceEndpoint {
	i.RLock()
	defer i.RUnlock()
	for _, e := range i.endpoints {
		if e.EndpointName == name {
			return e
		}
	}
	return nil
}

func (i *impl) AddEndpoint(endpoint *registry.NetworkServiceEndpoint) {
	i.Lock()
	i.endpoints = append(i.endpoints, endpoint)
	i.Unlock()
	logrus.Infof("Endpoint added: %v", endpoint)
}

func (i *impl) DeleteEndpoint(name string) error {
	i.Lock()
	defer i.Unlock()
	for idx, e := range i.endpoints {
		if e.EndpointName == name {
			i.endpoints = append(i.endpoints[:idx], i.endpoints[idx+1:]...)
			return nil
		}
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
	if len(i.dataplanes) == 0 {
		return nil, fmt.Errorf("no dataplanes registered")
	} else {
		return i.dataplanes[0], nil
	}
}

func (i *impl) AddDataplane(dataplane *Dataplane) {
	i.Lock()
	i.dataplanes = append(i.dataplanes, dataplane)
	i.Unlock()
	logrus.Infof("Dataplane added: %v", dataplane)
}

func (i *impl) DeleteDataplane(name string) {}

func NewModel() Model {
	return &impl{}
}
