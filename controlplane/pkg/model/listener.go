package model

type ModelListener interface {
	EndpointAdded(endpoint *Endpoint)
	EndpointUpdated(endpoint *Endpoint)
	EndpointDeleted(endpoint *Endpoint)

	DataplaneAdded(dataplane *Dataplane)
	DataplaneDeleted(dataplane *Dataplane)

	ClientConnectionAdded(clientConnection *ClientConnection)
	ClientConnectionDeleted(clientConnection *ClientConnection)
	ClientConnectionUpdated(old, new *ClientConnection)
}

type ModelListenerImpl struct{}

// EndpointAdded will be called when Endpoint is added to model, accept pointer to copy
func (ModelListenerImpl) EndpointAdded(endpoint *Endpoint) {}

// EndpointUpdated will be called when Endpoint in model is updated
func (ModelListenerImpl) EndpointUpdated(endpoint *Endpoint) {}

// EndpointDeleted will be called when Endpoint in model is deleted
func (ModelListenerImpl) EndpointDeleted(endpoint *Endpoint) {}

// DataplaneAdded will be called when Dataplane is added to model, accept pointer to copy
func (ModelListenerImpl) DataplaneAdded(dataplane *Dataplane) {}

// DataplaneDeleted will be called when Dataplane in model is deleted
func (ModelListenerImpl) DataplaneDeleted(dataplane *Dataplane) {}

// ClientConnectionAdded will be called when ClientConnection is added to model, accept pointer to copy
func (ModelListenerImpl) ClientConnectionAdded(clientConnection *ClientConnection) {}

// ClientConnectionUpdated will be called when ClientConnection in model is updated
func (ModelListenerImpl) ClientConnectionUpdated(old, new *ClientConnection) {}

// ClientConnectionDeleted will be called when ClientConnection in model is deleted
func (ModelListenerImpl) ClientConnectionDeleted(clientConnection *ClientConnection) {}
