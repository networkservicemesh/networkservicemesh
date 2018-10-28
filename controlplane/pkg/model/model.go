package model

import (
	"fmt"
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
	// GetNetworkService(name string) *netmesh.NetworkService
	GetNetworkServiceEndpoints(name string) []*netmesh.NetworkServiceEndpoint

	GetEndpoint(name string) *netmesh.NetworkServiceEndpoint
	AddEndpoint(endpoint *netmesh.NetworkServiceEndpoint)

	GetDataplane(name string) *Dataplane
	AddDataplane(dataplane *Dataplane)
	DeleteDataplane(name string)
	SelectDataplane() (*Dataplane, error)
}

type impl struct {
	sync.RWMutex
	endpoints  []*netmesh.NetworkServiceEndpoint
	services   []*netmesh.NetworkService
	dataplanes []*Dataplane
}

// func (i *impl) GetNetworkService(name string) *netmesh.NetworkService {
// 	i.RLock()
// 	defer i.RUnlock()
// 	for _, s := range i.services {
// 		if s.NetworkServiceName == name {
// 			return s
// 		}
// 	}
// 	return nil
// }

func (i *impl) GetNetworkServiceEndpoints(name string) []*netmesh.NetworkServiceEndpoint {
	i.RLock()
	defer i.RUnlock()
	var endpoints []*netmesh.NetworkServiceEndpoint
	for _, e := range i.endpoints {
		if e.NetworkServiceName == name {
			endpoints = append(endpoints, e)
		}
	}
	return endpoints
}

func (i *impl) GetEndpoint(name string) *netmesh.NetworkServiceEndpoint {
	i.RLock()
	defer i.RUnlock()
	for _, e := range i.endpoints {
		if e.NseProviderName == name {
			return e
		}
	}
	return nil
}

func (i *impl) AddEndpoint(endpoint *netmesh.NetworkServiceEndpoint) {
	i.Lock()
	i.endpoints = append(i.endpoints, endpoint)
	i.Unlock()
	logrus.Infof("Endpoint added: %v", endpoint)
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

// getLocalEndpoint return a slice of nsmapi.NetworkServiceEndpoint with only
// entries matching NSM Pod ip address.
func FilterEndpointsByHost(endpointList []*netmesh.NetworkServiceEndpoint, host string) []*netmesh.NetworkServiceEndpoint {
	endpoints := []*netmesh.NetworkServiceEndpoint{}
	for _, ep := range endpointList {
		if ep.NetworkServiceHost == host {
			endpoints = append(endpoints, ep)
		}
	}
	return endpoints
}

// getEndpointWithInterface returns a slice of slice of nsmapi.NetworkServiceEndpoint with
// only Endpoints offerring correct Interface type.
func FindEndpointsForMechanism(endpointList []*netmesh.NetworkServiceEndpoint, reqMechanismsSorted []*common.LocalMechanism) []*netmesh.NetworkServiceEndpoint {
	endpoints := []*netmesh.NetworkServiceEndpoint{}
	found := false
	// Loop over a list of required interfaces, since it is sorted, the loop starts with first choice.
	// if no first choice matches found, loop goes to the second choice, etc., otherwise function
	// returns collected slice of endpoints with matching interface type.
	for _, iReq := range reqMechanismsSorted {
		for _, ep := range endpointList {
			for _, intf := range ep.LocalMechanisms {
				if iReq.Type == intf.Type {
					found = true
					endpoints = append(endpoints, ep)
				}
			}
		}
		if found {
			break
		}
	}
	return endpoints
}
