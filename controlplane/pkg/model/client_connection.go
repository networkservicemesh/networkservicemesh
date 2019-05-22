package model

import (
	"github.com/golang/protobuf/proto"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/crossconnect"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/nsm"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/registry"
	"sync"
)

type ClientConnectionState int8

const (
	ClientConnection_Ready      ClientConnectionState = 0
	ClientConnection_Requesting ClientConnectionState = 1
	ClientConnection_Healing    ClientConnectionState = 2
	ClientConnection_Closing    ClientConnectionState = 3
	ClientConnection_Closed     ClientConnectionState = 4
)

type ClientConnection struct {
	ConnectionId    string
	Xcon            *crossconnect.CrossConnect
	RemoteNsm       *registry.NetworkServiceManager
	Endpoint        *registry.NSERegistration
	Dataplane       *Dataplane
	ConnectionState ClientConnectionState
	Request         nsm.NSMRequest
	DataplaneState  DataplaneState
}

func (cc *ClientConnection) GetId() string {
	if cc == nil {
		return ""
	}
	return cc.ConnectionId
}

func (cc *ClientConnection) GetNetworkService() string {
	if cc == nil {
		return ""
	}
	return cc.Endpoint.GetNetworkService().GetName()
}

func (cc *ClientConnection) GetConnectionSource() nsm.NSMConnection {
	if cc.Xcon.GetLocalSource() != nil {
		return cc.Xcon.GetLocalSource()
	} else {
		return cc.Xcon.GetRemoteSource()
	}
}

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
		ConnectionId:    cc.ConnectionId,
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
	d.inner.Store(cc.ConnectionId, cc.Clone())
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
	v := d.GetClientConnection(cc.ConnectionId)
	if v == nil {
		d.AddClientConnection(cc)
		return
	}
	d.inner.Store(cc.ConnectionId, cc.Clone())
	d.resourceUpdated(v, cc.Clone())
}

func (d *clientConnectionDomain) SetClientConnectionModificationHandler(h *ModificationHandler) {
	d.addHandler(h)
}
