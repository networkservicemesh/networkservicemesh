package model

import (
	"github.com/golang/protobuf/proto"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/registry"
	"sync"
)

// Endpoint structure in Model that describes NetworkServiceEndpoint
type Endpoint struct {
	Endpoint       *registry.NSERegistration
	SocketLocation string
	Workspace      string
}

// Clone returns pointer to copy of Endpoint
func (ep *Endpoint) Clone() *Endpoint {
	if ep == nil {
		return nil
	}

	var endpoint *registry.NSERegistration
	if ep.Endpoint != nil {
		endpoint = proto.Clone(ep.Endpoint).(*registry.NSERegistration)
	}

	return &Endpoint{
		Endpoint:       endpoint,
		SocketLocation: ep.SocketLocation,
		Workspace:      ep.Workspace,
	}
}

// EndpointName returns name of Endpoint
func (ep *Endpoint) EndpointName() string {
	return ep.Endpoint.GetNetworkserviceEndpoint().GetEndpointName()
}

// NetworkServiceName returns name of NetworkService of that Endpoint
func (ep *Endpoint) NetworkServiceName() string {
	return ep.Endpoint.GetNetworkService().GetName()
}

type endpointDomain struct {
	baseDomain
	inner sync.Map
}

func (d *endpointDomain) AddEndpoint(endpoint *Endpoint) {
	d.inner.Store(endpoint.EndpointName(), endpoint.Clone())
	d.resourceAdded(endpoint.Clone())
}

func (d *endpointDomain) GetEndpoint(name string) *Endpoint {
	v, _ := d.inner.Load(name)
	if v != nil {
		return v.(*Endpoint).Clone()
	}
	return nil

}

func (d *endpointDomain) GetEndpointsByNetworkService(nsName string) []*Endpoint {
	var rv []*Endpoint
	d.inner.Range(func(key, value interface{}) bool {
		endp := value.(*Endpoint)
		if endp.NetworkServiceName() == nsName {
			rv = append(rv, endp.Clone())
		}
		return true
	})
	return rv
}

func (d *endpointDomain) DeleteEndpoint(name string) {
	v := d.GetEndpoint(name)
	if v == nil {
		return
	}
	d.inner.Delete(name)
	d.resourceDeleted(v)
}

func (d *endpointDomain) UpdateEndpoint(endpoint *Endpoint) {
	v := d.GetEndpoint(endpoint.EndpointName())
	if v == nil {
		d.AddEndpoint(endpoint)
		return
	}
	d.inner.Store(endpoint.EndpointName(), endpoint.Clone())
	d.resourceUpdated(v, endpoint.Clone())
}

func (d *endpointDomain) SetEndpointModificationHandler(h *ModificationHandler) func() {
	deleteFunc := d.addHandler(h)
	d.inner.Range(func(key, value interface{}) bool {
		d.resourceAdded(value.(*Endpoint).Clone())
		return true
	})
	return deleteFunc
}
