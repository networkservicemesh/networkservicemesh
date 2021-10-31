package model

import (
	"context"
	"strconv"
	"sync"
        "os"

	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/registry"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/selector"
)

type Model interface {
	GetEndpointsByNetworkService(nsName string) []*Endpoint

	AddEndpoint(ctx context.Context, endpoint *Endpoint)
	GetEndpoint(name string) *Endpoint
	UpdateEndpoint(ctx context.Context, endpoint *Endpoint)
	DeleteEndpoint(ctx context.Context, name string)

	GetForwarder(name string) *Forwarder
	AddForwarder(ctx context.Context, forwarder *Forwarder)
	UpdateForwarder(ctx context.Context, forwarder *Forwarder)
	DeleteForwarder(ctx context.Context, name string)
	SelectForwarder(forwarderSelector func(dp *Forwarder) bool) (*Forwarder, error)

	AddClientConnection(ctx context.Context, clientConnection *ClientConnection)
	GetClientConnection(connectionID string) *ClientConnection
	GetAllClientConnections() []*ClientConnection
	UpdateClientConnection(ctx context.Context, clientConnection *ClientConnection)
	DeleteClientConnection(ctx context.Context, connectionID string)
	ApplyClientConnectionChanges(ctx context.Context, connectionID string, changeFunc func(*ClientConnection)) *ClientConnection

	ConnectionID() string
	CorrectIDGenerator(id string)

	AddListener(listener Listener)
	RemoveListener(listener Listener)
	ListenerCount() int

	SetNsm(nsm *registry.NetworkServiceManager)
	GetNsm() *registry.NetworkServiceManager

	GetSelector() selector.Selector
}

type model struct {
	endpointDomain
	forwarderDomain
	clientConnectionDomain

	lastConnectionID uint64
	mtx              sync.RWMutex
	selector         selector.Selector
	nsm              *registry.NetworkServiceManager
	listeners        map[Listener]func()
}

func (m *model) AddListener(listener Listener) {
	endpListenerDelete := m.SetEndpointModificationHandler(&ModificationHandler{
		AddFunc: func(ctx context.Context, new interface{}) {
			listener.EndpointAdded(ctx, new.(*Endpoint))
		},
		UpdateFunc: func(ctx context.Context, old interface{}, new interface{}) {
			listener.EndpointUpdated(ctx, new.(*Endpoint))
		},
		DeleteFunc: func(ctx context.Context, del interface{}) {
			listener.EndpointDeleted(ctx, del.(*Endpoint))
		},
	})

	dpListenerDelete := m.SetForwarderModificationHandler(&ModificationHandler{
		AddFunc: func(ctx context.Context, new interface{}) {
			listener.ForwarderAdded(ctx, new.(*Forwarder))
		},
		DeleteFunc: func(ctx context.Context, del interface{}) {
			listener.ForwarderDeleted(ctx, del.(*Forwarder))
		},
	})

	ccListenerDelete := m.SetClientConnectionModificationHandler(&ModificationHandler{
		AddFunc: func(ctx context.Context, new interface{}) {
			listener.ClientConnectionAdded(ctx, new.(*ClientConnection))
		},
		UpdateFunc: func(ctx context.Context, old interface{}, new interface{}) {
			listener.ClientConnectionUpdated(ctx, old.(*ClientConnection), new.(*ClientConnection))
		},
		DeleteFunc: func(ctx context.Context, del interface{}) {
			listener.ClientConnectionDeleted(ctx, del.(*ClientConnection))
		},
	})
	m.mtx.Lock()
	m.listeners[listener] = func() {
		endpListenerDelete()
		dpListenerDelete()
		ccListenerDelete()
	}
	m.mtx.Unlock()
}

func (m *model) RemoveListener(listener Listener) {
	m.mtx.Lock()
	defer m.mtx.Unlock()
	deleter, ok := m.listeners[listener]
	if !ok {
		logrus.Info("No such listener")
	}
	deleter()
	delete(m.listeners, listener)
}

func (m *model) ListenerCount() int {
	m.mtx.Lock()
	l := len(m.listeners)
	m.mtx.Unlock()
	return l
}

// NewModel returns new instance of Model
func NewModel() Model {

	// Check for the selection method : RoundRobin or Maglev
	var selector_method string
        selector_method = os.Getenv("SELECTOR")
	logrus.Infof(" get selector_method %s ",selector_method)
	
	var Selector_method selector.Selector
	if selector_method == "Maglev" {
		Selector_method = selector.NewMatchMaglevSelector()

	} else{ // default selector is RoundRobin
		
		Selector_method = selector.NewMatchSelector()
	
		 
	}
	return &model{
		clientConnectionDomain: newClientConnectionDomain(),
		endpointDomain:         newEndpointDomain(),
		forwarderDomain:        newForwarderDomain(),
		//selector:               selector.NewMatchSelector(),
		selector:               Selector_method,
		listeners:              make(map[Listener]func()),
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
