package model

import (
	"fmt"
	"strconv"
	"sync"

	"github.com/ligato/networkservicemesh/controlplane/pkg/model/registry"

	"github.com/ligato/networkservicemesh/pkg/nsm/apis/common"
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

	GetNsmUrl() string
	ConnectionId() string
}

type impl struct {
	sync.RWMutex
	endpoints         []*registry.NetworkServiceEndpoint
	dataplanes        []*Dataplane
	lastConnnectionId uint64
	nsmUrl            string
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

func (i *impl) DeleteDataplane(name string) {
	i.Lock()
	defer i.Unlock()

	for idx, dp := range i.dataplanes {
		if dp.RegisteredName == name {
			i.dataplanes = append(i.dataplanes[:idx], i.dataplanes[idx+1:]...)
			return
		}
	}
}

func (i *impl) GetNsmUrl() string {
	return i.nsmUrl
}

func NewModel(nsmUrl string) Model {
	return &impl{
		nsmUrl: nsmUrl,
	}
}

func (i *impl) ConnectionId() string {
	i.Lock()
	defer i.Unlock()
	i.lastConnnectionId++
	return strconv.FormatUint(i.lastConnnectionId, 16)
}
