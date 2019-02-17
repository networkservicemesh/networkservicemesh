package model

import (
	"fmt"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/nsm"
	"strconv"
	"sync"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/crossconnect"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/registry"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/selector"

	local "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/connection"
	remote "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/remote/connection"
	"github.com/sirupsen/logrus"
)

type Dataplane struct {
	RegisteredName   string
	SocketLocation   string
	LocalMechanisms  []*local.Mechanism
	RemoteMechanisms []*remote.Mechanism
}
type ClientConnectionState int8
const (
	ClientConnection_Ready = 0
	ClientConnection_Requesting = 1
	ClientConnection_Healing	= 2
	ClientConnection_Closing	= 3
	ClientConnection_Closed	= 4
)

type ClientConnection struct {
	ConnectionId string
	Xcon         *crossconnect.CrossConnect
	RemoteNsm    *registry.NetworkServiceManager
	Endpoint     *registry.NSERegistration
	Dataplane    *Dataplane
	ConnectionState ClientConnectionState
	Request      nsm.NSMRequest
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

func (cc *ClientConnection) GetSourceConnection() nsm.NSMConnection {
	if cc.Xcon.GetLocalSource() != nil {
		return cc.Xcon.GetLocalSource()
	} else {
		return cc.Xcon.GetRemoteSource()
	}
}

// Model change listener
type ModelListener interface {
	EndpointAdded(endpoint *registry.NetworkServiceEndpoint)
	EndpointDeleted(endpoint *registry.NetworkServiceEndpoint)

	DataplaneAdded(dataplane *Dataplane)
	DataplaneDeleted(dataplane *Dataplane)

	ClientConnectionAdded(clientConnection *ClientConnection)
	ClientConnectionDeleted(clientConnection *ClientConnection)
	ClientConnectionUpdated(clientConnection *ClientConnection)
}

type ModelListenerImpl struct{}

func (ModelListenerImpl) EndpointAdded(endpoint *registry.NetworkServiceEndpoint) {}

func (ModelListenerImpl) EndpointDeleted(endpoint *registry.NetworkServiceEndpoint) {}

func (ModelListenerImpl) DataplaneAdded(dataplane *Dataplane) {}

func (ModelListenerImpl) DataplaneDeleted(dataplane *Dataplane) {}

func (ModelListenerImpl) ClientConnectionAdded(clientConnection *ClientConnection) {}

func (ModelListenerImpl) ClientConnectionDeleted(clientConnection *ClientConnection) {}

func (ModelListenerImpl) ClientConnectionUpdated(clientConnection *ClientConnection) {}

type Model interface {
	GetNetworkServiceEndpoints(name string) []*registry.NSERegistration

	GetEndpoint(name string) *registry.NSERegistration
	AddEndpoint(endpoint *registry.NSERegistration)
	DeleteEndpoint(name string) error

	GetDataplane(name string) *Dataplane
	AddDataplane(dataplane *Dataplane)
	DeleteDataplane(name string)
	SelectDataplane() (*Dataplane, error)

	AddClientConnection(clientConnection *ClientConnection)
	GetClientConnection(connectionId string) *ClientConnection
	GetAllClientConnections() []*ClientConnection
	UpdateClientConnection(clientConnection *ClientConnection)
	DeleteClientConnection(connectionId string)

	ConnectionId() string

	// After listener will be added it will be called for all existing dataplanes/endpoints
	AddListener(listener ModelListener)
	RemoveListener(listener ModelListener)

	SetNsm(nsm *registry.NetworkServiceManager)
	GetNsm() *registry.NetworkServiceManager

	GetSelector() selector.Selector
}

type impl struct {
	sync.RWMutex
	endpoints         map[string]*registry.NSERegistration
	networkServices   map[string][]*registry.NSERegistration
	dataplanes        map[string]*Dataplane
	lastConnnectionId uint64
	nsm               *registry.NetworkServiceManager
	listeners         []ModelListener
	selector          selector.Selector
	clientConnections map[string]*ClientConnection
}

func (i *impl) AddClientConnection(clientConnection *ClientConnection) {
	i.Lock()
	defer i.Unlock()

	i.clientConnections[clientConnection.ConnectionId] = clientConnection
	for _, listener := range i.listeners {
		listener.ClientConnectionAdded(clientConnection)
	}
}

func (i *impl) GetClientConnection(connectionId string) *ClientConnection {
	i.RLock()
	defer i.RUnlock()

	return i.clientConnections[connectionId]
}

func (i *impl) GetAllClientConnections() []*ClientConnection {
	i.RLock()
	defer i.RUnlock()

	var rv []*ClientConnection
	for _, v := range i.clientConnections {
		rv = append(rv, v)
	}

	return rv
}

func (i *impl) UpdateClientConnection(clientConnection *ClientConnection) {
	i.Lock()
	i.clientConnections[clientConnection.ConnectionId] = clientConnection
	i.Unlock()

	for _, listener := range i.listeners {
		listener.ClientConnectionUpdated(clientConnection)
	}
}

func (i *impl) DeleteClientConnection(connectionId string) {
	i.Lock()
	clientConnection := i.clientConnections[connectionId]
	if clientConnection == nil {
		i.Unlock()
		return
	}
	delete(i.clientConnections, connectionId)
	i.Unlock()

	i.RLock()
	defer i.RUnlock()

	for _, listener := range i.listeners {
		listener.ClientConnectionDeleted(clientConnection)
	}
}

func (i *impl) AddListener(listener ModelListener) {
	i.Lock()
	i.listeners = append(i.listeners, listener)
	i.Unlock()

	i.RLock()
	defer i.RUnlock()

	// We need to notify this listener about all already added dataplanes/endpoints
	for _, dp := range i.dataplanes {
		listener.DataplaneAdded(dp)
	}

	for _, ep := range i.endpoints {
		listener.EndpointAdded(ep.NetworkserviceEndpoint)
	}
}

func (i *impl) RemoveListener(listener ModelListener) {
	i.Lock()
	defer i.Unlock()
	for idx, v := range i.listeners {
		if v == listener {
			i.listeners = append(i.listeners[:idx], i.listeners[idx+1:]...)
			return
		}
	}
}

func (i *impl) GetNetworkServiceEndpoints(name string) []*registry.NSERegistration {
	i.RLock()
	defer i.RUnlock()
	var endpoints = i.networkServices[name]
	if endpoints == nil {
		endpoints = []*registry.NSERegistration{}
	}
	return endpoints
}

func (i *impl) GetEndpoint(name string) *registry.NSERegistration {
	i.RLock()
	defer i.RUnlock()
	return i.endpoints[name]
}

func (i *impl) AddEndpoint(endpoint *registry.NSERegistration) {
	i.Lock()
	defer i.Unlock()
	i.endpoints[endpoint.GetNetworkserviceEndpoint().GetEndpointName()] = endpoint
	serviceName := endpoint.GetNetworkService().GetName()
	services := i.networkServices[serviceName]
	if services == nil {
		services = []*registry.NSERegistration{endpoint}
	} else {
		services = append(services, endpoint)
	}
	i.networkServices[serviceName] = services

	logrus.Infof("Endpoint added: %v", endpoint)

	for _, l := range i.listeners {
		l.EndpointAdded(endpoint.GetNetworkserviceEndpoint())
	}
}

func (i *impl) DeleteEndpoint(name string) error {
	i.Lock()
	defer i.Unlock()

	endpoint := i.endpoints[name]
	if endpoint != nil {
		services := i.networkServices[endpoint.GetNetworkService().GetName()]
		if len(services) > 1 {
			for idx, e := range services {
				if e == endpoint {
					services = append(services[:idx], services[idx+1:]...)
					break
				}
			}
			// Update services with removed item.
			i.networkServices[endpoint.GetNetworkService().GetName()] = services
		} else {
			delete(i.networkServices, endpoint.GetNetworkService().GetName())
		}
		delete(i.endpoints, name)

		for _, l := range i.listeners {
			l.EndpointDeleted(endpoint.GetNetworkserviceEndpoint())
		}
		return nil
	}

	return fmt.Errorf("no endpoint with name: %s", name)
}

func (i *impl) GetDataplane(name string) *Dataplane {
	i.RLock()
	defer i.RUnlock()
	for _, dp := range i.dataplanes {
		if dp.RegisteredName == name {
			return dp
		}
	}
	return nil
}

func (i *impl) SelectDataplane() (*Dataplane, error) {
	i.Lock()
	defer i.Unlock()
	for _, v := range i.dataplanes {
		return v, nil // TODO: Return first for now
	}
	return nil, fmt.Errorf("no dataplanes registered")
}

func (i *impl) AddDataplane(dataplane *Dataplane) {
	i.Lock()
	defer i.Unlock()
	i.dataplanes[dataplane.RegisteredName] = dataplane
	logrus.Infof("Dataplane added: %v", dataplane)

	for _, l := range i.listeners {
		l.DataplaneAdded(dataplane)
	}
}

func (i *impl) DeleteDataplane(name string) {
	i.Lock()
	dataplane, ok := i.dataplanes[name]
	if !ok {
		i.Unlock()
		return
	}
	delete(i.dataplanes, name)
	i.Unlock()

	for _, l := range i.listeners {
		l.DataplaneDeleted(dataplane)
	}
}

func (i *impl) GetNsm() *registry.NetworkServiceManager {
	return i.nsm
}

func (i *impl) SetNsm(nsm *registry.NetworkServiceManager) {
	i.nsm = nsm
}

func NewModel() Model {
	return &impl{
		dataplanes:        make(map[string]*Dataplane),
		networkServices:   make(map[string][]*registry.NSERegistration),
		endpoints:         make(map[string]*registry.NSERegistration),
		listeners:         []ModelListener{},
		selector:          selector.NewMatchSelector(),
		clientConnections: make(map[string]*ClientConnection),
	}
}

func (i *impl) ConnectionId() string {
	i.Lock()
	defer i.Unlock()
	i.lastConnnectionId++
	return strconv.FormatUint(i.lastConnnectionId, 16)
}

func (i *impl) GetSelector() selector.Selector {
	return i.selector
}
