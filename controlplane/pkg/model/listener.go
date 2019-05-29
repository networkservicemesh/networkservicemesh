package model

// Listener represent an interface for listening model changes
type Listener interface {
	EndpointAdded(endpoint *Endpoint)
	EndpointUpdated(endpoint *Endpoint)
	EndpointDeleted(endpoint *Endpoint)

	DataplaneAdded(dataplane *Dataplane)
	DataplaneDeleted(dataplane *Dataplane)

	ClientConnectionAdded(clientConnection *ClientConnection)
	ClientConnectionDeleted(clientConnection *ClientConnection)
	ClientConnectionUpdated(old, new *ClientConnection)
}

// ListenerImpl is empty implementation of Listener
type ListenerImpl struct{}

// EndpointAdded will be called when Endpoint is added to model, accept pointer to copy
func (ListenerImpl) EndpointAdded(endpoint *Endpoint) {}

// EndpointUpdated will be called when Endpoint in model is updated
func (ListenerImpl) EndpointUpdated(endpoint *Endpoint) {}

// EndpointDeleted will be called when Endpoint in model is deleted
func (ListenerImpl) EndpointDeleted(endpoint *Endpoint) {}

// DataplaneAdded will be called when Dataplane is added to model, accept pointer to copy
func (ListenerImpl) DataplaneAdded(dataplane *Dataplane) {}

// DataplaneDeleted will be called when Dataplane in model is deleted
func (ListenerImpl) DataplaneDeleted(dataplane *Dataplane) {}

// ClientConnectionAdded will be called when ClientConnection is added to model, accept pointer to copy
func (ListenerImpl) ClientConnectionAdded(clientConnection *ClientConnection) {}

// ClientConnectionUpdated will be called when ClientConnection in model is updated
func (ListenerImpl) ClientConnectionUpdated(old, new *ClientConnection) {}

// ClientConnectionDeleted will be called when ClientConnection in model is deleted
func (ListenerImpl) ClientConnectionDeleted(clientConnection *ClientConnection) {}
