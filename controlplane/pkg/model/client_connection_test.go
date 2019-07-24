package model

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/crossconnect"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/registry"
	. "github.com/onsi/gomega"
)

func TestAddAndGetСс(t *testing.T) {
	RegisterTestingT(t)

	cc := &ClientConnection{
		ConnectionID: "1",
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
		ConnectionState:         ClientConnectionHealing,
		Request:                 nil,
		DataplaneState:          DataplaneStateReady,
	}

	ccd := newClientConnectionDomain()
	ccd.AddClientConnection(cc)
	getConn := ccd.GetClientConnection("1")

	Expect(getConn.ConnectionID).To(Equal(cc.ConnectionID))
	Expect(getConn.ConnectionState).To(Equal(cc.ConnectionState))
	Expect(getConn.DataplaneState).To(Equal(cc.DataplaneState))
	Expect(getConn.Request).To(BeNil())

	Expect(getConn.GetNetworkService()).To(Equal(cc.GetNetworkService()))
	Expect(getConn.GetID()).To(Equal(cc.GetID()))

	Expect(fmt.Sprintf("%p", getConn.RemoteNsm)).ToNot(Equal(fmt.Sprintf("%p", cc.RemoteNsm)))
	Expect(fmt.Sprintf("%p", getConn.Endpoint)).ToNot(Equal(fmt.Sprintf("%p", cc.Endpoint)))
	Expect(fmt.Sprintf("%p", getConn.Endpoint.NetworkServiceManager)).
		ToNot(Equal(fmt.Sprintf("%p", cc.Endpoint.NetworkServiceManager)))
}

func TestGetAllСс(t *testing.T) {
	RegisterTestingT(t)

	ccd := newClientConnectionDomain()
	amount := 5

	for i := 0; i < amount; i++ {
		ccd.AddClientConnection(&ClientConnection{
			ConnectionID: fmt.Sprintf("%d", i),
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
			ConnectionState:         ClientConnectionHealing,
			Request:                 nil,
			DataplaneState:          DataplaneStateReady,
		})
	}

	all := ccd.GetAllClientConnections()
	Expect(len(all)).To(Equal(amount))

	expected := make([]bool, amount)
	for i := 0; i < amount; i++ {
		index, _ := strconv.ParseInt(all[i].ConnectionID, 10, 64)
		expected[index] = true
	}

	for i := 0; i < amount; i++ {
		Expect(expected[i]).To(BeTrue())
	}
}

func TestDeleteСс(t *testing.T) {
	RegisterTestingT(t)

	ccd := newClientConnectionDomain()
	ccd.AddClientConnection(&ClientConnection{
		ConnectionID: "1",
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
		ConnectionState:         ClientConnectionHealing,
		Request:                 nil,
		DataplaneState:          DataplaneStateReady,
	})

	cc := ccd.GetClientConnection("1")
	Expect(cc).ToNot(BeNil())

	ccd.DeleteClientConnection("1")

	ccDel := ccd.GetClientConnection("1")
	Expect(ccDel).To(BeNil())

	ccd.DeleteClientConnection("NotExistingId")

}

func TestUpdateExistingСс(t *testing.T) {
	RegisterTestingT(t)

	cc := &ClientConnection{
		ConnectionID: "1",
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
		ConnectionState:         ClientConnectionHealing,
		Request:                 nil,
		DataplaneState:          DataplaneStateReady,
	}

	ccd := newClientConnectionDomain()
	ccd.AddClientConnection(cc)

	newUrl := "3.3.3.3"
	newDpName := "updatedName"
	cc.Endpoint.NetworkServiceManager.Url = newUrl
	cc.DataplaneRegisteredName = newDpName

	notUpdated := ccd.GetClientConnection("1")
	Expect(notUpdated.Endpoint.NetworkServiceManager.Url).ToNot(Equal(newUrl))
	Expect(notUpdated.DataplaneRegisteredName).ToNot(Equal(newDpName))

	ccd.UpdateClientConnection(cc)
	updated := ccd.GetClientConnection("1")
	Expect(updated.Endpoint.NetworkServiceManager.Url).To(Equal(newUrl))
	Expect(updated.DataplaneRegisteredName).To(Equal(newDpName))
}

func TestUpdateNotExistingСс(t *testing.T) {
	RegisterTestingT(t)

	cc := &ClientConnection{
		ConnectionID: "1",
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
		ConnectionState:         ClientConnectionHealing,
		Request:                 nil,
		DataplaneState:          DataplaneStateReady,
	}

	ccd := newClientConnectionDomain()

	ccd.UpdateClientConnection(cc)
	updated := ccd.GetClientConnection("1")
	Expect(updated.Endpoint.NetworkServiceManager.Url).To(Equal("2.2.2.2"))
	Expect(updated.DataplaneRegisteredName).To(Equal("dp1"))
}

func TestApplyChanges(t *testing.T) {
	RegisterTestingT(t)

	ccd := newClientConnectionDomain()
	ccd.AddClientConnection(&ClientConnection{
		ConnectionID: "1",
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
		ConnectionState:         ClientConnectionHealing,
		Request:                 nil,
		DataplaneState:          DataplaneStateReady,
	})

	ccd.ApplyClientConnectionChanges("1", func(cc *ClientConnection) {
		cc.RemoteNsm.Name = "updatedMaster"
	})
	upd := ccd.GetClientConnection("1")
	Expect(upd.RemoteNsm.Name).To(Equal("updatedMaster"))
}
