package plugins

import (
	"context"
	"sync"

	"google.golang.org/grpc"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/plugins"
	"github.com/pkg/errors"
)

// ConnectionPluginManager transmits each method call to all registered connection plugins
type ConnectionPluginManager interface {
	PluginManager
	plugins.ConnectionPluginServer
}

type connectionPluginManager struct {
	sync.RWMutex
	pluginClients map[string]plugins.ConnectionPluginClient
}

func createConnectionPluginManager() ConnectionPluginManager {
	return &connectionPluginManager{
		pluginClients: make(map[string]plugins.ConnectionPluginClient),
	}
}

func (cpm *connectionPluginManager) Register(name string, conn *grpc.ClientConn) {
	client := plugins.NewConnectionPluginClient(conn)
	cpm.addClient(name, client)
}

func (cpm *connectionPluginManager) addClient(name string, client plugins.ConnectionPluginClient) {
	cpm.Lock()
	defer cpm.Unlock()

	cpm.pluginClients[name] = client
}

func (cpm *connectionPluginManager) getClients() map[string]plugins.ConnectionPluginClient {
	cpm.RLock()
	defer cpm.RUnlock()

	return cpm.pluginClients
}

func (cpm *connectionPluginManager) UpdateConnection(ctx context.Context, wrapper *plugins.ConnectionWrapper) (*plugins.ConnectionWrapper, error) {
	for name, plugin := range cpm.getClients() {
		pluginCtx, cancel := context.WithTimeout(ctx, pluginCallTimeout)

		var err error
		wrapper, err = plugin.UpdateConnection(pluginCtx, wrapper)
		cancel()

		if err != nil {
			return nil, errors.Wrapf(err, "'%s' connection plugin returned an error: %v", name)
		}
	}
	return wrapper.Clone(), nil
}

func (cpm *connectionPluginManager) ValidateConnection(ctx context.Context, wrapper *plugins.ConnectionWrapper) (*plugins.ConnectionValidationResult, error) {
	for name, plugin := range cpm.getClients() {
		pluginCtx, cancel := context.WithTimeout(ctx, pluginCallTimeout)

		result, err := plugin.ValidateConnection(pluginCtx, wrapper)
		cancel()

		if err != nil {
			return nil, errors.Wrapf(err, "'%s' connection plugin returned an error", name)
		}

		if result.GetStatus() != plugins.ConnectionValidationStatus_SUCCESS {
			return result, nil
		}
	}
	return &plugins.ConnectionValidationResult{Status: plugins.ConnectionValidationStatus_SUCCESS}, nil
}
