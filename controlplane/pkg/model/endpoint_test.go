package model

import (
	"fmt"
	"strconv"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/registry"
)

func TestAddAndGetEndpoint(t *testing.T) {
	RegisterTestingT(t)

	endp := &Endpoint{
		Endpoint: &registry.NSERegistration{
			NetworkService: &registry.NetworkService{
				Name: "ns1",
			},
			NetworkServiceManager: &registry.NetworkServiceManager{
				Name: "worker",
				Url:  "2.2.2.2",
			},
			NetworkserviceEndpoint: &registry.NetworkServiceEndpoint{
				NetworkServiceName: "ns1",
				EndpointName:       "endp1",
			},
		},
		SocketLocation: "/socket",
		Workspace:      "ws",
	}

	ed := newEndpointDomain()
	ed.AddEndpoint(endp)
	getEndp := ed.GetEndpoint("endp1")

	Expect(getEndp.SocketLocation).To(Equal(endp.SocketLocation))
	Expect(getEndp.Workspace).To(Equal(endp.Workspace))
	Expect(getEndp.Endpoint).To(Equal(endp.Endpoint))

	Expect(fmt.Sprintf("%p", getEndp.Endpoint)).ToNot(Equal(fmt.Sprintf("%p", endp.Endpoint)))
	Expect(fmt.Sprintf("%p", getEndp.Endpoint.NetworkService)).
		ToNot(Equal(fmt.Sprintf("%p", endp.Endpoint.NetworkService)))
	Expect(fmt.Sprintf("%p", getEndp.Endpoint.NetworkserviceEndpoint)).
		ToNot(Equal(fmt.Sprintf("%p", endp.Endpoint.NetworkserviceEndpoint)))
	Expect(fmt.Sprintf("%p", getEndp.Endpoint.NetworkServiceManager)).
		ToNot(Equal(fmt.Sprintf("%p", endp.Endpoint.NetworkServiceManager)))
}

func TestGetEndpointsByNs(t *testing.T) {
	RegisterTestingT(t)

	ed := newEndpointDomain()
	amount := 5

	for i := 0; i < amount; i++ {
		ed.AddEndpoint(&Endpoint{
			Endpoint: &registry.NSERegistration{
				NetworkService: &registry.NetworkService{
					Name: fmt.Sprintf("%d", i%2),
				},
				NetworkServiceManager: &registry.NetworkServiceManager{
					Name: "worker",
					Url:  "2.2.2.2",
				},
				NetworkserviceEndpoint: &registry.NetworkServiceEndpoint{
					NetworkServiceName: "ns1",
					EndpointName:       fmt.Sprintf("%d", i),
				},
			},
			SocketLocation: "/socket",
			Workspace:      "ws",
		})
	}

	ns0 := ed.GetEndpointsByNetworkService("0")
	count0 := (amount + 1) / 2
	Expect(len(ns0)).To(Equal(count0))

	ns1 := ed.GetEndpointsByNetworkService("1")
	count1 := amount - count0
	Expect(len(ns1)).To(Equal(count1))

	expected := make([]bool, amount)

	for i := 0; i < count0; i++ {
		idxNs, _ := strconv.ParseInt(ns0[i].NetworkServiceName(), 10, 64)
		idxEndp, _ := strconv.ParseInt(ns0[i].EndpointName(), 10, 64)
		Expect(idxNs).To(Equal(idxEndp % 2))
		expected[idxEndp] = true
	}

	for i := 0; i < count1; i++ {
		idxNs, _ := strconv.ParseInt(ns1[i].NetworkServiceName(), 10, 64)
		idxEndp, _ := strconv.ParseInt(ns1[i].EndpointName(), 10, 64)
		Expect(idxNs).To(Equal(idxEndp % 2))
		expected[idxEndp] = true
	}

	for i := 0; i < amount; i++ {
		Expect(expected[i]).To(BeTrue())
	}
}

func TestDeleteEndpoint(t *testing.T) {
	RegisterTestingT(t)

	ed := newEndpointDomain()
	ed.AddEndpoint(&Endpoint{
		Endpoint: &registry.NSERegistration{
			NetworkService: &registry.NetworkService{
				Name: "ns1",
			},
			NetworkServiceManager: &registry.NetworkServiceManager{
				Name: "worker",
				Url:  "2.2.2.2",
			},
			NetworkserviceEndpoint: &registry.NetworkServiceEndpoint{
				NetworkServiceName: "ns1",
				EndpointName:       "endp1",
			},
		},
		SocketLocation: "/socket",
		Workspace:      "ws",
	})

	endp := ed.GetEndpoint("endp1")
	Expect(endp).ToNot(BeNil())

	ed.DeleteEndpoint("endp1")

	endpDel := ed.GetEndpoint("endp1")
	Expect(endpDel).To(BeNil())

	ed.DeleteEndpoint("NotExistingId")
}

func TestUpdateExistingEndpoint(t *testing.T) {
	RegisterTestingT(t)

	endp := &Endpoint{
		Endpoint: &registry.NSERegistration{
			NetworkService: &registry.NetworkService{
				Name: "ns1",
			},
			NetworkServiceManager: &registry.NetworkServiceManager{
				Name: "worker",
				Url:  "2.2.2.2",
			},
			NetworkserviceEndpoint: &registry.NetworkServiceEndpoint{
				NetworkServiceName: "ns1",
				EndpointName:       "endp1",
			},
		},
		SocketLocation: "/socket",
		Workspace:      "ws",
	}

	ed := newEndpointDomain()
	ed.AddEndpoint(endp)

	newUrl := "3.3.3.3"
	newNs := "updatedNs"
	endp.Endpoint.NetworkServiceManager.Url = newUrl
	endp.Endpoint.NetworkService.Name = newNs

	notUpdated := ed.GetEndpoint("endp1")
	Expect(notUpdated.Endpoint.NetworkServiceManager.Url).ToNot(Equal(newUrl))
	Expect(notUpdated.Endpoint.NetworkService.Name).ToNot(Equal(newNs))

	ed.UpdateEndpoint(endp)
	updated := ed.GetEndpoint("endp1")
	Expect(updated.Endpoint.NetworkServiceManager.Url).To(Equal(newUrl))
	Expect(updated.Endpoint.NetworkService.Name).To(Equal(newNs))
}

func TestUpdateNotExisting(t *testing.T) {
	RegisterTestingT(t)

	endp := &Endpoint{
		Endpoint: &registry.NSERegistration{
			NetworkService: &registry.NetworkService{
				Name: "ns1",
			},
			NetworkServiceManager: &registry.NetworkServiceManager{
				Name: "worker",
				Url:  "2.2.2.2",
			},
			NetworkserviceEndpoint: &registry.NetworkServiceEndpoint{
				NetworkServiceName: "ns1",
				EndpointName:       "endp1",
			},
		},
		SocketLocation: "/socket",
		Workspace:      "ws",
	}

	ed := newEndpointDomain()

	ed.UpdateEndpoint(endp)
	updated := ed.GetEndpoint("endp1")
	Expect(updated.Endpoint.NetworkServiceManager.Url).To(Equal("2.2.2.2"))
	Expect(updated.Endpoint.NetworkService.Name).To(Equal("ns1"))
}
