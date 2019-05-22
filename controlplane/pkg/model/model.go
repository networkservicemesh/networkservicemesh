package model

import (
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/registry"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/selector"
	"github.com/sirupsen/logrus"
	"strconv"
	"sync"
)

type ModelListener interface {
	EndpointAdded(endpoint *Endpoint)
	EndpointUpdated(endpoint *Endpoint)
	EndpointDeleted(endpoint *Endpoint)

	DataplaneAdded(dataplane *Dataplane)
	DataplaneDeleted(dataplane *Dataplane)

	ClientConnectionAdded(clientConnection *ClientConnection)
	ClientConnectionDeleted(clientConnection *ClientConnection)
	ClientConnectionUpdated(clientConnection *ClientConnection)
}

type ModelListenerImpl struct{}

func (ModelListenerImpl) EndpointAdded(endpoint *Endpoint)                           {}
func (ModelListenerImpl) EndpointUpdated(endpoint *Endpoint)                         {}
func (ModelListenerImpl) EndpointDeleted(endpoint *Endpoint)                         {}
func (ModelListenerImpl) DataplaneAdded(dataplane *Dataplane)                        {}
func (ModelListenerImpl) DataplaneDeleted(dataplane *Dataplane)                      {}
func (ModelListenerImpl) ClientConnectionAdded(clientConnection *ClientConnection)   {}
func (ModelListenerImpl) ClientConnectionDeleted(clientConnection *ClientConnection) {}
func (ModelListenerImpl) ClientConnectionUpdated(clientConnection *ClientConnection) {}

type Model interface {
	GetEndpointsByNetworkService(nsName string) []*Endpoint

	AddEndpoint(endpoint *Endpoint)
	GetEndpoint(name string) *Endpoint
	UpdateEndpoint(endpoint *Endpoint)
	DeleteEndpoint(name string)

	GetDataplane(name string) *Dataplane
	AddDataplane(dataplane *Dataplane)
	DeleteDataplane(name string)
	SelectDataplane(dataplaneSelector func(dp *Dataplane) bool) (*Dataplane, error)

	AddClientConnection(clientConnection *ClientConnection)
	GetClientConnection(connectionId string) *ClientConnection
	GetAllClientConnections() []*ClientConnection
	UpdateClientConnection(clientConnection *ClientConnection)
	DeleteClientConnection(connectionId string)

	ConnectionId() string
	CorrectIdGenerator(id string)

	AddListener(listener ModelListener)

	SetNsm(nsm *registry.NetworkServiceManager)
	GetNsm() *registry.NetworkServiceManager

	GetSelector() selector.Selector
}

type model struct {
	endpointDomain
	dataplaneDomain
	clientConnectionDomain

	lastConnectionId uint64
	mtx              sync.RWMutex
	selector         selector.Selector
	nsm              *registry.NetworkServiceManager
}

func (m *model) AddListener(listener ModelListener) {
	m.SetEndpointModificationHandler(&ModificationHandler{
		AddFunc: func(new interface{}) {
			listener.EndpointAdded(new.(*Endpoint))
		},
		UpdateFunc: func(old interface{}, new interface{}) {
			listener.EndpointUpdated(new.(*Endpoint))
		},
		DeleteFunc: func(del interface{}) {
			listener.EndpointDeleted(del.(*Endpoint))
		},
	})

	m.SetDataplaneModificationHandler(&ModificationHandler{
		AddFunc: func(new interface{}) {
			listener.DataplaneAdded(new.(*Dataplane))
		},
		DeleteFunc: func(del interface{}) {
			listener.DataplaneDeleted(del.(*Dataplane))
		},
	})

	m.SetClientConnectionModificationHandler(&ModificationHandler{
		AddFunc: func(new interface{}) {
			listener.ClientConnectionAdded(new.(*ClientConnection))
		},
		UpdateFunc: func(old interface{}, new interface{}) {
			listener.ClientConnectionUpdated(new.(*ClientConnection))
		},
		DeleteFunc: func(del interface{}) {
			listener.ClientConnectionDeleted(del.(*ClientConnection))
		},
	})
}

func NewModel() Model {
	return &model{
		selector: selector.NewMatchSelector(),
	}
}

func (m *model) ConnectionId() string {
	m.mtx.Lock()
	defer m.mtx.Unlock()

	m.lastConnectionId++
	return strconv.FormatUint(m.lastConnectionId, 16)
}

func (m *model) CorrectIdGenerator(id string) {
	m.mtx.Lock()
	defer m.mtx.Unlock()

	value, err := strconv.ParseUint(id, 16, 64)
	if err != nil {
		logrus.Errorf("Failed to update id genrator %v", err)
	}
	if m.lastConnectionId < value {
		m.lastConnectionId = value
	}
}

func (m *model) GetNsm() *registry.NetworkServiceManager {
	m.mtx.RLock()
	defer m.mtx.RUnlock()

	return m.nsm
}

func (m *model) SetNsm(nsm *registry.NetworkServiceManager) {
	m.mtx.Lock()
	defer m.mtx.Unlock()

	m.nsm = nsm
}

func (m *model) GetSelector() selector.Selector {
	return m.selector
}
