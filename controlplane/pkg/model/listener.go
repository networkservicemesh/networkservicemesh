package model

import "context"

// Listener represent an interface for listening model changes
type Listener interface {
	EndpointAdded(ctx context.Context, endpoint *Endpoint)
	EndpointUpdated(ctx context.Context, endpoint *Endpoint)
	EndpointDeleted(ctx context.Context, endpoint *Endpoint)

	DataplaneAdded(ctx context.Context, dataplane *Dataplane)
	DataplaneDeleted(ctx context.Context, dataplane *Dataplane)

	ClientConnectionAdded(ctx context.Context, clientConnection *ClientConnection)
	ClientConnectionDeleted(ctx context.Context, clientConnection *ClientConnection)
	ClientConnectionUpdated(ctx context.Context, old, new *ClientConnection)
}

// ListenerImpl is empty implementation of Listener
type ListenerImpl struct{}

// EndpointAdded will be called when Endpoint is added to model, accept pointer to copy
func (ListenerImpl) EndpointAdded(ctx context.Context, endpoint *Endpoint) {}

// EndpointUpdated will be called when Endpoint in model is updated
func (ListenerImpl) EndpointUpdated(ctx context.Context, endpoint *Endpoint) {}

// EndpointDeleted will be called when Endpoint in model is deleted
func (ListenerImpl) EndpointDeleted(ctx context.Context, endpoint *Endpoint) {}

// DataplaneAdded will be called when Dataplane is added to model, accept pointer to copy
func (ListenerImpl) DataplaneAdded(ctx context.Context, dataplane *Dataplane) {}

// DataplaneDeleted will be called when Dataplane in model is deleted
func (ListenerImpl) DataplaneDeleted(ctx context.Context, dataplane *Dataplane) {}

// ClientConnectionAdded will be called when ClientConnection is added to model, accept pointer to copy
func (ListenerImpl) ClientConnectionAdded(ctx context.Context, clientConnection *ClientConnection) {}

// ClientConnectionUpdated will be called when ClientConnection in model is updated
func (ListenerImpl) ClientConnectionUpdated(ctx context.Context, old, new *ClientConnection) {}

// ClientConnectionDeleted will be called when ClientConnection in model is deleted
func (ListenerImpl) ClientConnectionDeleted(ctx context.Context, clientConnection *ClientConnection) {}
