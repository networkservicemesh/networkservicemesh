package model

import (
	"fmt"

	"github.com/golang/protobuf/proto"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/crossconnect"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/nsm/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/nsm/networkservice"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/registry"
)

// ClientConnectionState describes state of ClientConnection
type ClientConnectionState int8

const (
	// ClientConnectionReady means connection is in state 'ready'
	ClientConnectionReady ClientConnectionState = 0

	// ClientConnectionRequesting means connection waits answer from NSE or Dp
	ClientConnectionRequesting ClientConnectionState = 1

	// ClientConnectionBroken means connection failed requesting
	ClientConnectionBroken ClientConnectionState = 2

	// ClientConnectionHealing means connection is in 'healing' state
	ClientConnectionHealing ClientConnectionState = 3

	// ClientConnectionWaitingForRequest means connection finished healing on the local side and waits for the request
	ClientConnectionWaitingForRequest ClientConnectionState = 4

	// ClientConnectionClosing means connection is started closing process
	ClientConnectionClosing ClientConnectionState = 5
)

// ClientConnection struct in model that describes cross connect between NetworkServiceClient and NetworkServiceEndpoint
type ClientConnection struct {
	Request                 networkservice.Request
	Xcon                    *crossconnect.CrossConnect
	RemoteNsm               *registry.NetworkServiceManager
	Endpoint                *registry.NSERegistration
	DataplaneRegisteredName string
	DataplaneState          DataplaneState

	id              string
	connectionState ClientConnectionState
}

// GetID returns id of clientConnection
func (cc *ClientConnection) GetID() string {
	if cc == nil {
		return ""
	}
	return cc.id
}

// GetConnectionState returns state of clientConnection
func (cc *ClientConnection) GetConnectionState() ClientConnectionState {
	return cc.connectionState
}

// GetNetworkService returns name of networkService of clientConnection
func (cc *ClientConnection) GetNetworkService() string {
	if cc == nil {
		return ""
	}
	return cc.Endpoint.GetNetworkService().GetName()
}

// GetConnectionSource returns source part of connection
func (cc *ClientConnection) GetConnectionSource() connection.Connection {
	return cc.Xcon.GetSourceConnection()
}

// GetConnectionDestination returns destination part of connection
func (cc *ClientConnection) GetConnectionDestination() connection.Connection {
	return cc.Xcon.GetDestinationConnection()
}

// Clone return pointer to copy of ClientConnection
func (cc *ClientConnection) clone() cloneable {
	if cc == nil {
		return nil
	}

	var xcon *crossconnect.CrossConnect
	if cc.Xcon != nil {
		xcon = proto.Clone(cc.Xcon).(*crossconnect.CrossConnect)
	}

	var remoteNsm *registry.NetworkServiceManager
	if cc.RemoteNsm != nil {
		remoteNsm = proto.Clone(cc.RemoteNsm).(*registry.NetworkServiceManager)
	}

	var endpoint *registry.NSERegistration
	if cc.Endpoint != nil {
		endpoint = proto.Clone(cc.Endpoint).(*registry.NSERegistration)
	}

	var request networkservice.Request
	if cc.Request != nil {
		request = cc.Request.Clone()
	}

	return &ClientConnection{
		id:                      cc.id,
		Xcon:                    xcon,
		RemoteNsm:               remoteNsm,
		Endpoint:                endpoint,
		DataplaneRegisteredName: cc.DataplaneRegisteredName,
		Request:                 request,
		connectionState:         cc.connectionState,
		DataplaneState:          cc.DataplaneState,
	}
}

type clientConnectionDomain struct {
	baseDomain
}

func newClientConnectionDomain() clientConnectionDomain {
	return clientConnectionDomain{
		baseDomain: newBase(),
	}
}

func (d *clientConnectionDomain) AddClientConnection(cc *ClientConnection) error {
	if ok := d.store(cc.id, cc, false); ok {
		return nil
	}
	return fmt.Errorf("trying to add client connection by existing id: %v", cc.id)
}

func (d *clientConnectionDomain) AddOrUpdateClientConnection(cc *ClientConnection) {
	d.store(cc.id, cc, true)
}

func (d *clientConnectionDomain) GetClientConnection(id string) *ClientConnection {
	if v, ok := d.load(id); ok {
		return v.(*ClientConnection)
	}
	return nil
}

func (d *clientConnectionDomain) GetAllClientConnections() []*ClientConnection {
	var rv []*ClientConnection
	d.kvRange(func(_ string, value interface{}) bool {
		rv = append(rv, value.(*ClientConnection))
		return true
	})
	return rv
}

func (d *clientConnectionDomain) DeleteClientConnection(id string) error {
	if ok := d.delete(id); ok {
		return nil
	}
	return fmt.Errorf("trying to delete client connection by not existing id: %v", id)
}

func (d *clientConnectionDomain) ApplyClientConnectionChanges(id string, f func(*ClientConnection)) *ClientConnection {
	if upd := d.applyChanges(id, func(v interface{}) { f(v.(*ClientConnection)) }); upd != nil {
		return upd.(*ClientConnection)
	}
	return nil
}

func (d *clientConnectionDomain) CompareAndSwapClientConnection(cc *ClientConnection, f func(*ClientConnection) bool) bool {
	return d.compareAndSwap(cc.id, cc, func(v interface{}) bool { return f(v.(*ClientConnection)) })
}

func (d *clientConnectionDomain) SetClientConnectionModificationHandler(h *ModificationHandler) func() {
	return d.addHandler(h)
}
