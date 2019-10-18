package model

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/crossconnect"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/registry"
)

func TestAddAndGetСс(t *testing.T) {
	g := NewWithT(t)

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
			NetworkServiceEndpoint: &registry.NetworkServiceEndpoint{
				Name:               "endp1",
				NetworkServiceName: "ns1",
			},
		},
		ForwarderRegisteredName: "dp1",
		ConnectionState:         ClientConnectionHealing,
		Request:                 nil,
		ForwarderState:          ForwarderStateReady,
	}

	ccd := newClientConnectionDomain()
	ccd.AddClientConnection(context.Background(), cc)
	getConn := ccd.GetClientConnection("1")

	g.Expect(getConn.ConnectionID).To(Equal(cc.ConnectionID))
	g.Expect(getConn.ConnectionState).To(Equal(cc.ConnectionState))
	g.Expect(getConn.ForwarderState).To(Equal(cc.ForwarderState))
	g.Expect(getConn.Request).To(BeNil())

	g.Expect(getConn.GetNetworkService()).To(Equal(cc.GetNetworkService()))
	g.Expect(getConn.GetID()).To(Equal(cc.GetID()))

	g.Expect(fmt.Sprintf("%p", getConn.RemoteNsm)).ToNot(Equal(fmt.Sprintf("%p", cc.RemoteNsm)))
	g.Expect(fmt.Sprintf("%p", getConn.Endpoint)).ToNot(Equal(fmt.Sprintf("%p", cc.Endpoint)))
	g.Expect(fmt.Sprintf("%p", getConn.Endpoint.NetworkServiceManager)).
		ToNot(Equal(fmt.Sprintf("%p", cc.Endpoint.NetworkServiceManager)))
}

func TestGetAllСс(t *testing.T) {
	g := NewWithT(t)

	ccd := newClientConnectionDomain()
	amount := 5

	for i := 0; i < amount; i++ {
		ccd.AddClientConnection(context.Background(), &ClientConnection{
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
				NetworkServiceEndpoint: &registry.NetworkServiceEndpoint{
					Name:               "endp1",
					NetworkServiceName: "ns1",
				},
			},
			ForwarderRegisteredName: "dp1",
			ConnectionState:         ClientConnectionHealing,
			Request:                 nil,
			ForwarderState:          ForwarderStateReady,
		})
	}

	all := ccd.GetAllClientConnections()
	g.Expect(len(all)).To(Equal(amount))

	expected := make([]bool, amount)
	for i := 0; i < amount; i++ {
		index, _ := strconv.ParseInt(all[i].ConnectionID, 10, 64)
		expected[index] = true
	}

	for i := 0; i < amount; i++ {
		g.Expect(expected[i]).To(BeTrue())
	}
}

func TestDeleteСс(t *testing.T) {
	g := NewWithT(t)

	ccd := newClientConnectionDomain()
	ccd.AddClientConnection(context.Background(), &ClientConnection{
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
			NetworkServiceEndpoint: &registry.NetworkServiceEndpoint{
				Name:               "endp1",
				NetworkServiceName: "ns1",
			},
		},
		ForwarderRegisteredName: "dp1",
		ConnectionState:         ClientConnectionHealing,
		Request:                 nil,
		ForwarderState:          ForwarderStateReady,
	})

	cc := ccd.GetClientConnection("1")
	g.Expect(cc).ToNot(BeNil())

	ccd.DeleteClientConnection(context.Background(), "1")

	ccDel := ccd.GetClientConnection("1")
	g.Expect(ccDel).To(BeNil())

	ccd.DeleteClientConnection(context.Background(), "NotExistingId")
}

func TestUpdateExistingСс(t *testing.T) {
	g := NewWithT(t)

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
			NetworkServiceEndpoint: &registry.NetworkServiceEndpoint{
				Name:               "endp1",
				NetworkServiceName: "ns1",
			},
		},
		ForwarderRegisteredName: "dp1",
		ConnectionState:         ClientConnectionHealing,
		Request:                 nil,
		ForwarderState:          ForwarderStateReady,
	}

	ccd := newClientConnectionDomain()
	ccd.AddClientConnection(context.Background(), cc)

	newUrl := "3.3.3.3"
	newDpName := "updatedName"
	cc.Endpoint.NetworkServiceManager.Url = newUrl
	cc.ForwarderRegisteredName = newDpName

	notUpdated := ccd.GetClientConnection("1")
	g.Expect(notUpdated.Endpoint.NetworkServiceManager.Url).ToNot(Equal(newUrl))
	g.Expect(notUpdated.ForwarderRegisteredName).ToNot(Equal(newDpName))

	ccd.UpdateClientConnection(context.Background(), cc)
	updated := ccd.GetClientConnection("1")
	g.Expect(updated.Endpoint.NetworkServiceManager.Url).To(Equal(newUrl))
	g.Expect(updated.ForwarderRegisteredName).To(Equal(newDpName))
}

func TestUpdateNotExistingСс(t *testing.T) {
	g := NewWithT(t)

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
			NetworkServiceEndpoint: &registry.NetworkServiceEndpoint{
				Name:               "endp1",
				NetworkServiceName: "ns1",
			},
		},
		ForwarderRegisteredName: "dp1",
		ConnectionState:         ClientConnectionHealing,
		Request:                 nil,
		ForwarderState:          ForwarderStateReady,
	}

	ccd := newClientConnectionDomain()

	ccd.UpdateClientConnection(context.Background(), cc)
	updated := ccd.GetClientConnection("1")
	g.Expect(updated.Endpoint.NetworkServiceManager.Url).To(Equal("2.2.2.2"))
	g.Expect(updated.ForwarderRegisteredName).To(Equal("dp1"))
}

func TestApplyChanges(t *testing.T) {
	g := NewWithT(t)

	ccd := newClientConnectionDomain()
	ccd.AddClientConnection(context.Background(), &ClientConnection{
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
			NetworkServiceEndpoint: &registry.NetworkServiceEndpoint{
				Name:               "endp1",
				NetworkServiceName: "ns1",
			},
		},
		ForwarderRegisteredName: "dp1",
		ConnectionState:         ClientConnectionHealing,
		Request:                 nil,
		ForwarderState:          ForwarderStateReady,
	})

	ccd.ApplyClientConnectionChanges(context.Background(), "1", func(cc *ClientConnection) {
		cc.RemoteNsm.Name = "updatedMaster"
	})
	upd := ccd.GetClientConnection("1")
	g.Expect(upd.RemoteNsm.Name).To(Equal("updatedMaster"))
}
