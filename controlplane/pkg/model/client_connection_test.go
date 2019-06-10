package model

import (
	"fmt"
	"strconv"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/crossconnect"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/registry"
)

func TestAddAndGetCc(t *testing.T) {
	RegisterTestingT(t)

	cc := &ClientConnection{
		id: "1",
		Xcon: &crossconnect.CrossConnect{
			Id: "1",
		},
		RemoteNsm: &registry.NetworkServiceManager{
			Name: "master",
			Url:  "1.1.1.1",
		},
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
		DataplaneRegisteredName: "dp1",
		connectionState:         ClientConnectionHealing,
		Request:                 nil,
		DataplaneState:          DataplaneStateReady,
	}

	ccd := newClientConnectionDomain()

	err := ccd.AddClientConnection(cc)
	Expect(err).To(BeNil())

	getConn := ccd.GetClientConnection("1")

	Expect(getConn.id).To(Equal(cc.id))
	Expect(getConn.connectionState).To(Equal(cc.connectionState))
	Expect(getConn.DataplaneState).To(Equal(cc.DataplaneState))
	Expect(getConn.Request).To(BeNil())

	Expect(getConn.GetNetworkService()).To(Equal(cc.GetNetworkService()))
	Expect(getConn.GetID()).To(Equal(cc.GetID()))

	Expect(fmt.Sprintf("%p", getConn.RemoteNsm)).ToNot(Equal(fmt.Sprintf("%p", cc.RemoteNsm)))
	Expect(fmt.Sprintf("%p", getConn.Endpoint)).ToNot(Equal(fmt.Sprintf("%p", cc.Endpoint)))
	Expect(fmt.Sprintf("%p", getConn.Endpoint.NetworkServiceManager)).
		ToNot(Equal(fmt.Sprintf("%p", cc.Endpoint.NetworkServiceManager)))

	err = ccd.AddClientConnection(cc)
	Expect(err).NotTo(BeNil())
}

func TestGetAllCc(t *testing.T) {
	RegisterTestingT(t)

	ccd := newClientConnectionDomain()
	amount := 5

	for i := 0; i < amount; i++ {
		ccd.AddOrUpdateClientConnection(&ClientConnection{
			id: fmt.Sprintf("%d", i),
			Xcon: &crossconnect.CrossConnect{
				Id: "1",
			},
			RemoteNsm: &registry.NetworkServiceManager{
				Name: "master",
				Url:  "1.1.1.1",
			},
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
			DataplaneRegisteredName: "dp1",
			connectionState:         ClientConnectionHealing,
			Request:                 nil,
			DataplaneState:          DataplaneStateReady,
		})
	}

	all := ccd.GetAllClientConnections()
	Expect(len(all)).To(Equal(amount))

	expected := make([]bool, amount)
	for i := 0; i < amount; i++ {
		index, _ := strconv.ParseInt(all[i].id, 10, 64)
		expected[index] = true
	}

	for i := 0; i < amount; i++ {
		Expect(expected[i]).To(BeTrue())
	}
}

func TestDeleteCc(t *testing.T) {
	RegisterTestingT(t)

	ccd := newClientConnectionDomain()
	ccd.AddOrUpdateClientConnection(&ClientConnection{
		id: "1",
		Xcon: &crossconnect.CrossConnect{
			Id: "1",
		},
		RemoteNsm: &registry.NetworkServiceManager{
			Name: "master",
			Url:  "1.1.1.1",
		},
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
		DataplaneRegisteredName: "dp1",
		connectionState:         ClientConnectionHealing,
		Request:                 nil,
		DataplaneState:          DataplaneStateReady,
	})

	cc := ccd.GetClientConnection("1")
	Expect(cc).ToNot(BeNil())

	err := ccd.DeleteClientConnection("1")
	Expect(err).To(BeNil())

	ccDel := ccd.GetClientConnection("1")
	Expect(ccDel).To(BeNil())

	err = ccd.DeleteClientConnection("1")
	Expect(err).NotTo(BeNil())
}

func TestAddOrUpdateExistingCc(t *testing.T) {
	RegisterTestingT(t)

	cc := &ClientConnection{
		id: "1",
		Xcon: &crossconnect.CrossConnect{
			Id: "1",
		},
		RemoteNsm: &registry.NetworkServiceManager{
			Name: "master",
			Url:  "1.1.1.1",
		},
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
		DataplaneRegisteredName: "dp1",
		connectionState:         ClientConnectionHealing,
		Request:                 nil,
		DataplaneState:          DataplaneStateReady,
	}

	ccd := newClientConnectionDomain()
	ccd.AddOrUpdateClientConnection(cc)

	newUrl := "3.3.3.3"
	newDpName := "updatedName"
	cc.Endpoint.NetworkServiceManager.Url = newUrl
	cc.DataplaneRegisteredName = newDpName

	notUpdated := ccd.GetClientConnection("1")
	Expect(notUpdated.Endpoint.NetworkServiceManager.Url).ToNot(Equal(newUrl))
	Expect(notUpdated.DataplaneRegisteredName).ToNot(Equal(newDpName))

	ccd.AddOrUpdateClientConnection(cc)
	updated := ccd.GetClientConnection("1")
	Expect(updated.Endpoint.NetworkServiceManager.Url).To(Equal(newUrl))
	Expect(updated.DataplaneRegisteredName).To(Equal(newDpName))
}

func TestAddOrUpdateNotExistingCc(t *testing.T) {
	RegisterTestingT(t)

	cc := &ClientConnection{
		id: "1",
		Xcon: &crossconnect.CrossConnect{
			Id: "1",
		},
		RemoteNsm: &registry.NetworkServiceManager{
			Name: "master",
			Url:  "1.1.1.1",
		},
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
		DataplaneRegisteredName: "dp1",
		connectionState:         ClientConnectionHealing,
		Request:                 nil,
		DataplaneState:          DataplaneStateReady,
	}

	ccd := newClientConnectionDomain()

	ccd.AddOrUpdateClientConnection(cc)
	updated := ccd.GetClientConnection("1")
	Expect(updated.Endpoint.NetworkServiceManager.Url).To(Equal("2.2.2.2"))
	Expect(updated.DataplaneRegisteredName).To(Equal("dp1"))
}

func TestApplyChanges(t *testing.T) {
	RegisterTestingT(t)

	ccd := newClientConnectionDomain()
	ccd.AddOrUpdateClientConnection(&ClientConnection{
		id: "1",
		Xcon: &crossconnect.CrossConnect{
			Id: "1",
		},
		RemoteNsm: &registry.NetworkServiceManager{
			Name: "master",
			Url:  "1.1.1.1",
		},
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
		DataplaneRegisteredName: "dp1",
		connectionState:         ClientConnectionHealing,
		Request:                 nil,
		DataplaneState:          DataplaneStateReady,
	})

	ccd.ApplyClientConnectionChanges("1", func(cc *ClientConnection) {
		cc.RemoteNsm.Name = "updatedMaster"
	})
	upd := ccd.GetClientConnection("1")
	Expect(upd.RemoteNsm.Name).To(Equal("updatedMaster"))
}
