package model

import (
	"fmt"
	"strconv"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/registry"
)

func TestAddAndGetEndpoint(t *testing.T) {
	g := NewWithT(t)

	endp := &Endpoint{
		Endpoint: &registry.NSERegistration{
			NetworkService: &registry.NetworkService{
				Name: "ns1",
			},
			NetworkServiceManager: &registry.NetworkServiceManager{
				Name: "worker",
				Url:  "2.2.2.2",
			},
			NetworkServiceEndpoint: &registry.NetworkServiceEndpoint{
				Name:               "endp1",
				NetworkServiceName: "ns1",
			},
		},
		SocketLocation: "/socket",
		Workspace:      "ws",
	}

	ed := newEndpointDomain()
	ed.AddEndpoint(endp)
	getEndp := ed.GetEndpoint("endp1")

	g.Expect(getEndp.SocketLocation).To(Equal(endp.SocketLocation))
	g.Expect(getEndp.Workspace).To(Equal(endp.Workspace))
	g.Expect(getEndp.Endpoint).To(Equal(endp.Endpoint))

	g.Expect(fmt.Sprintf("%p", getEndp.Endpoint)).ToNot(Equal(fmt.Sprintf("%p", endp.Endpoint)))
	g.Expect(fmt.Sprintf("%p", getEndp.Endpoint.NetworkService)).
		ToNot(Equal(fmt.Sprintf("%p", endp.Endpoint.NetworkService)))
	g.Expect(fmt.Sprintf("%p", getEndp.Endpoint.NetworkServiceEndpoint)).
		ToNot(Equal(fmt.Sprintf("%p", endp.Endpoint.NetworkServiceEndpoint)))
	g.Expect(fmt.Sprintf("%p", getEndp.Endpoint.NetworkServiceManager)).
		ToNot(Equal(fmt.Sprintf("%p", endp.Endpoint.NetworkServiceManager)))
}

func TestGetEndpointsByNs(t *testing.T) {
	g := NewWithT(t)

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
				NetworkServiceEndpoint: &registry.NetworkServiceEndpoint{
					Name:               fmt.Sprintf("%d", i),
					NetworkServiceName: "ns1",
				},
			},
			SocketLocation: "/socket",
			Workspace:      "ws",
		})
	}

	ns0 := ed.GetEndpointsByNetworkService("0")
	count0 := (amount + 1) / 2
	g.Expect(len(ns0)).To(Equal(count0))

	ns1 := ed.GetEndpointsByNetworkService("1")
	count1 := amount - count0
	g.Expect(len(ns1)).To(Equal(count1))

	expected := make([]bool, amount)

	for i := 0; i < count0; i++ {
		idxNs, _ := strconv.ParseInt(ns0[i].NetworkServiceName(), 10, 64)
		idxEndp, _ := strconv.ParseInt(ns0[i].EndpointName(), 10, 64)
		g.Expect(idxNs).To(Equal(idxEndp % 2))
		expected[idxEndp] = true
	}

	for i := 0; i < count1; i++ {
		idxNs, _ := strconv.ParseInt(ns1[i].NetworkServiceName(), 10, 64)
		idxEndp, _ := strconv.ParseInt(ns1[i].EndpointName(), 10, 64)
		g.Expect(idxNs).To(Equal(idxEndp % 2))
		expected[idxEndp] = true
	}

	for i := 0; i < amount; i++ {
		g.Expect(expected[i]).To(BeTrue())
	}
}

func TestDeleteEndpoint(t *testing.T) {
	g := NewWithT(t)

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
			NetworkServiceEndpoint: &registry.NetworkServiceEndpoint{
				Name:               "endp1",
				NetworkServiceName: "ns1",
			},
		},
		SocketLocation: "/socket",
		Workspace:      "ws",
	})

	endp := ed.GetEndpoint("endp1")
	g.Expect(endp).ToNot(BeNil())

	ed.DeleteEndpoint("endp1")

	endpDel := ed.GetEndpoint("endp1")
	g.Expect(endpDel).To(BeNil())

	ed.DeleteEndpoint("NotExistingId")
}

func TestUpdateExistingEndpoint(t *testing.T) {
	g := NewWithT(t)

	endp := &Endpoint{
		Endpoint: &registry.NSERegistration{
			NetworkService: &registry.NetworkService{
				Name: "ns1",
			},
			NetworkServiceManager: &registry.NetworkServiceManager{
				Name: "worker",
				Url:  "2.2.2.2",
			},
			NetworkServiceEndpoint: &registry.NetworkServiceEndpoint{
				Name:               "endp1",
				NetworkServiceName: "ns1",
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
	g.Expect(notUpdated.Endpoint.NetworkServiceManager.Url).ToNot(Equal(newUrl))
	g.Expect(notUpdated.Endpoint.NetworkService.Name).ToNot(Equal(newNs))

	ed.UpdateEndpoint(endp)
	updated := ed.GetEndpoint("endp1")
	g.Expect(updated.Endpoint.NetworkServiceManager.Url).To(Equal(newUrl))
	g.Expect(updated.Endpoint.NetworkService.Name).To(Equal(newNs))
}

func TestUpdateNotExisting(t *testing.T) {
	g := NewWithT(t)

	endp := &Endpoint{
		Endpoint: &registry.NSERegistration{
			NetworkService: &registry.NetworkService{
				Name: "ns1",
			},
			NetworkServiceManager: &registry.NetworkServiceManager{
				Name: "worker",
				Url:  "2.2.2.2",
			},
			NetworkServiceEndpoint: &registry.NetworkServiceEndpoint{
				Name:               "endp1",
				NetworkServiceName: "ns1",
			},
		},
		SocketLocation: "/socket",
		Workspace:      "ws",
	}

	ed := newEndpointDomain()

	ed.UpdateEndpoint(endp)
	updated := ed.GetEndpoint("endp1")
	g.Expect(updated.Endpoint.NetworkServiceManager.Url).To(Equal("2.2.2.2"))
	g.Expect(updated.Endpoint.NetworkService.Name).To(Equal("ns1"))
}
