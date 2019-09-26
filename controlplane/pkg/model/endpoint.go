package model

import (
	"context"

	"github.com/golang/protobuf/proto"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/registry"
)

// Endpoint structure in Model that describes NetworkServiceEndpoint
type Endpoint struct {
	Endpoint       *registry.NSERegistration
	SocketLocation string
	Workspace      string
}

// Clone returns pointer to copy of Endpoint
func (ep *Endpoint) clone() cloneable {
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
	return ep.Endpoint.GetNetworkServiceEndpoint().GetName()
}

// NetworkServiceName returns name of NetworkService of that Endpoint
func (ep *Endpoint) NetworkServiceName() string {
	return ep.Endpoint.GetNetworkService().GetName()
}

type endpointDomain struct {
	baseDomain
}

func newEndpointDomain() endpointDomain {
	return endpointDomain{
		baseDomain: newBase(),
	}
}
func (d *endpointDomain) AddEndpoint(ctx context.Context, endpoint *Endpoint) {
	d.store(ctx, endpoint.EndpointName(), endpoint)
}

func (d *endpointDomain) GetEndpoint(name string) *Endpoint {
	v, _ := d.load(name)
	if v != nil {
		return v.(*Endpoint)
	}
	return nil
}

func (d *endpointDomain) GetEndpointsByNetworkService(nsName string) []*Endpoint {
	var rv []*Endpoint
	d.kvRange(func(key string, value interface{}) bool {
		endp := value.(*Endpoint)
		if endp.NetworkServiceName() == nsName {
			rv = append(rv, endp)
		}
		return true
	})
	return rv
}

func (d *endpointDomain) DeleteEndpoint(ctx context.Context, name string) {
	d.delete(ctx, name)
}

func (d *endpointDomain) UpdateEndpoint(ctx context.Context, endpoint *Endpoint) {
	d.store(ctx, endpoint.EndpointName(), endpoint)
}

func (d *endpointDomain) SetEndpointModificationHandler(h *ModificationHandler) func() {
	return d.addHandler(h)
}
