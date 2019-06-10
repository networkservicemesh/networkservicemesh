package model

import (
	"strconv"
	"sync"

	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/registry"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/selector"
)

type Model interface {
	GetEndpointsByNetworkService(nsName string) []*Endpoint

	AddEndpoint(endpoint *Endpoint) error
	AddOrUpdateEndpoint(endpoint *Endpoint)
	GetEndpoint(name string) *Endpoint
	DeleteEndpoint(name string) error

	AddDataplane(dataplane *Dataplane) error
	AddOrUpdateDataplane(dataplane *Dataplane)
	GetDataplane(name string) *Dataplane
	DeleteDataplane(name string) error
	SelectDataplane(dataplaneSelector func(dp *Dataplane) bool) (*Dataplane, error)
	ApplyDataplaneChanges(name string, changeFunc func(*Dataplane)) *Dataplane

	AddClientConnection(id string, connectionState ClientConnectionState, cc *ClientConnection) (*ClientConnectionEditor, error)
	GetClientConnection(connectionID string) *ClientConnection
	GetAllClientConnections() []*ClientConnection
	DeleteClientConnection(connectionID string) error
	ChangeClientConnectionState(id string, connectionState ClientConnectionState) (*ClientConnectionEditor, error)
	ResetClientConnectionChanges(editor *ClientConnectionEditor)
	CommitClientConnectionChanges(editor *ClientConnectionEditor) error

	ConnectionID() string
	CorrectIDGenerator(id string)

	AddListener(listener Listener)
	RemoveListener(listener Listener)

	SetNsm(nsm *registry.NetworkServiceManager)
	GetNsm() *registry.NetworkServiceManager

	GetSelector() selector.Selector
}

type model struct {
	endpointDomain
	dataplaneDomain
	clientConnectionEditorManager

	lastConnectionID uint64
	mtx              sync.RWMutex
	selector         selector.Selector
	nsm              *registry.NetworkServiceManager
	listeners        map[Listener]func()
}

func (m *model) AddListener(listener Listener) {
	endpListenerDelete := m.SetEndpointModificationHandler(&ModificationHandler{
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

	dpListenerDelete := m.SetDataplaneModificationHandler(&ModificationHandler{
		AddFunc: func(new interface{}) {
			listener.DataplaneAdded(new.(*Dataplane))
		},
		DeleteFunc: func(del interface{}) {
			listener.DataplaneDeleted(del.(*Dataplane))
		},
	})

	ccListenerDelete := m.SetClientConnectionModificationHandler(&ModificationHandler{
		AddFunc: func(new interface{}) {
			listener.ClientConnectionAdded(new.(*ClientConnection))
		},
		UpdateFunc: func(old interface{}, new interface{}) {
			listener.ClientConnectionUpdated(old.(*ClientConnection), new.(*ClientConnection))
		},
		DeleteFunc: func(del interface{}) {
			listener.ClientConnectionDeleted(del.(*ClientConnection))
		},
	})

	m.listeners[listener] = func() {
		endpListenerDelete()
		dpListenerDelete()
		ccListenerDelete()
	}
}

func (m *model) RemoveListener(listener Listener) {
	deleter, ok := m.listeners[listener]
	if !ok {
		logrus.Info("No such listener")
	}
	deleter()
	delete(m.listeners, listener)
}

// NewModel returns new instance of Model
func NewModel() Model {
	return &model{
		endpointDomain:                newEndpointDomain(),
		dataplaneDomain:               newDataplaneDomain(),
		clientConnectionEditorManager: newClientConnectionEditorManager(),
		selector:                      selector.NewMatchSelector(),
		listeners:                     make(map[Listener]func()),
	}
}

func (m *model) ConnectionID() string {
	m.mtx.Lock()
	defer m.mtx.Unlock()

	m.lastConnectionID++
	return strconv.FormatUint(m.lastConnectionID, 16)
}

func (m *model) CorrectIDGenerator(id string) {
	m.mtx.Lock()
	defer m.mtx.Unlock()

	value, err := strconv.ParseUint(id, 16, 64)
	if err != nil {
		logrus.Errorf("Failed to update id genrator %v", err)
	}
	if m.lastConnectionID < value {
		m.lastConnectionID = value
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
