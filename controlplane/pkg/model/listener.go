package model

import "context"

// Listener represent an interface for listening model changes
type Listener interface {
	EndpointAdded(ctx context.Context, endpoint *Endpoint)
	EndpointUpdated(ctx context.Context, endpoint *Endpoint)
	EndpointDeleted(ctx context.Context, endpoint *Endpoint)

	ForwarderAdded(ctx context.Context, forwarder *Forwarder)
	ForwarderDeleted(ctx context.Context, forwarder *Forwarder)

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

// ForwarderAdded will be called when Forwarder is added to model, accept pointer to copy
func (ListenerImpl) ForwarderAdded(ctx context.Context, forwarder *Forwarder) {}

// ForwarderDeleted will be called when Forwarder in model is deleted
func (ListenerImpl) ForwarderDeleted(ctx context.Context, forwarder *Forwarder) {}

// ClientConnectionAdded will be called when ClientConnection is added to model, accept pointer to copy
func (ListenerImpl) ClientConnectionAdded(ctx context.Context, clientConnection *ClientConnection) {}

// ClientConnectionUpdated will be called when ClientConnection in model is updated
func (ListenerImpl) ClientConnectionUpdated(ctx context.Context, old, new *ClientConnection) {}

// ClientConnectionDeleted will be called when ClientConnection in model is deleted
func (ListenerImpl) ClientConnectionDeleted(ctx context.Context, clientConnection *ClientConnection) {
}
