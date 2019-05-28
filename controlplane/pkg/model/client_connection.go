package model

import (
	"github.com/golang/protobuf/proto"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/crossconnect"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/nsm"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/registry"
	"sync"
)

// ClientConnectionState describes state of ClientConnection
type ClientConnectionState int8

const (
	// ClientConnectionReady means connection is in state 'ready'
	ClientConnectionReady ClientConnectionState = 0

	// ClientConnectionRequesting means connection waits answer from NSE or Dp
	ClientConnectionRequesting ClientConnectionState = 1

	// ClientConnectionHealing means connection is in 'healing' state
	ClientConnectionHealing ClientConnectionState = 2

	// ClientConnectionClosing means connection is started closing process
	ClientConnectionClosing ClientConnectionState = 3
)

// ClientConnection struct in model that describes cross connect between NetworkServiceClient and NetworkServiceEndpoint
type ClientConnection struct {
	ConnectionID    string
	Request         nsm.NSMRequest
	Xcon            *crossconnect.CrossConnect
	RemoteNsm       *registry.NetworkServiceManager
	Endpoint        *registry.NSERegistration
	Dataplane       *Dataplane
	ConnectionState ClientConnectionState
	DataplaneState  DataplaneState
}

// GetID returns id of clientConnection
func (cc *ClientConnection) GetID() string {
	if cc == nil {
		return ""
	}
	return cc.ConnectionID
}

// GetNetworkService returns name of networkService of clientConnection
func (cc *ClientConnection) GetNetworkService() string {
	if cc == nil {
		return ""
	}
	return cc.Endpoint.GetNetworkService().GetName()
}

// GetConnectionSource returns source part of connection
func (cc *ClientConnection) GetConnectionSource() nsm.NSMConnection {
	if cc.Xcon.GetLocalSource() != nil {
		return cc.Xcon.GetLocalSource()
	}
	return cc.Xcon.GetRemoteSource()
}

// Clone return pointer to copy of ClientConnection
func (cc *ClientConnection) Clone() *ClientConnection {
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

	var request nsm.NSMRequest
	if cc.Request != nil {
		request = cc.Request.Clone()
	}

	return &ClientConnection{
		ConnectionID:    cc.ConnectionID,
		Xcon:            xcon,
		RemoteNsm:       remoteNsm,
		Endpoint:        endpoint,
		Dataplane:       cc.Dataplane.Clone(),
		Request:         request,
		ConnectionState: cc.ConnectionState,
		DataplaneState:  cc.DataplaneState,
	}
}

type clientConnectionDomain struct {
	baseDomain
	inner sync.Map
}

func (d *clientConnectionDomain) AddClientConnection(cc *ClientConnection) {
	d.inner.Store(cc.ConnectionID, cc.Clone())
	d.resourceAdded(cc.Clone())
}

func (d *clientConnectionDomain) GetClientConnection(id string) *ClientConnection {
	v, _ := d.inner.Load(id)
	if v != nil {
		return v.(*ClientConnection).Clone()
	}
	return nil
}

func (d *clientConnectionDomain) GetAllClientConnections() []*ClientConnection {
	var rv []*ClientConnection
	d.inner.Range(func(_, value interface{}) bool {
		rv = append(rv, value.(*ClientConnection).Clone())
		return true
	})
	return rv
}

func (d *clientConnectionDomain) DeleteClientConnection(id string) {
	v := d.GetClientConnection(id)
	if v == nil {
		return
	}
	d.inner.Delete(id)
	d.resourceDeleted(v)
}

func (d *clientConnectionDomain) UpdateClientConnection(cc *ClientConnection) {
	v := d.GetClientConnection(cc.ConnectionID)
	if v == nil {
		d.AddClientConnection(cc)
		return
	}
	d.inner.Store(cc.ConnectionID, cc.Clone())
	d.resourceUpdated(v, cc.Clone())
}

func (d *clientConnectionDomain) ApplyClientConnectionChanges(id string,
	changeFunc func(*ClientConnection)) *ClientConnection {
	d.mtx.Lock()

	old := d.GetClientConnection(id)
	if old == nil {
		return nil
	}

	new := old.Clone()
	changeFunc(new)

	d.inner.Store(id, new.Clone())

	d.mtx.Unlock()

	d.resourceUpdated(old, new)
	return new
}

func (d *clientConnectionDomain) SetClientConnectionModificationHandler(h *ModificationHandler) func() {
	deleteFunc := d.addHandler(h)
	d.inner.Range(func(key, value interface{}) bool {
		d.resourceAdded(value.(*ClientConnection).Clone())
		return true
	})
	return deleteFunc
}
