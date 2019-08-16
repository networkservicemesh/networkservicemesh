package plugins

import (
	"context"
	"fmt"
	"sync"

	"google.golang.org/grpc"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/plugins"
)

// ConnectionPluginManager transmits each method call to all registered connection plugins
type ConnectionPluginManager interface {
	PluginManager
	UpdateConnection(context.Context, *plugins.ConnectionWrapper) (*plugins.ConnectionWrapper, error)
	ValidateConnection(context.Context, *plugins.ConnectionWrapper) (*plugins.ConnectionValidationResult, error)
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

func (cpm *connectionPluginManager) Register(name string, conn *grpc.ClientConn) error {
	client := plugins.NewConnectionPluginClient(conn)
	return cpm.addClient(name, client)
}

func (cpm *connectionPluginManager) addClient(name string, client plugins.ConnectionPluginClient) error {
	cpm.Lock()
	defer cpm.Unlock()

	if _, ok := cpm.pluginClients[name]; ok {
		return fmt.Errorf("already have a connection plugin with the same name")
	}

	cpm.pluginClients[name] = client
	return nil
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
			return nil, fmt.Errorf("'%s' connection plugin returned an error: %v", name, err)
		}
	}
	return wrapper, nil
}

func (cpm *connectionPluginManager) ValidateConnection(ctx context.Context, wrapper *plugins.ConnectionWrapper) (*plugins.ConnectionValidationResult, error) {
	for name, plugin := range cpm.getClients() {
		pluginCtx, cancel := context.WithTimeout(ctx, pluginCallTimeout)

		result, err := plugin.ValidateConnection(pluginCtx, wrapper)
		cancel()

		if err != nil {
			return nil, fmt.Errorf("'%s' connection plugin returned an error: %v", name, err)
		}

		if result.GetStatus() != plugins.ConnectionValidationStatus_SUCCESS {
			return result, nil
		}
	}
	return &plugins.ConnectionValidationResult{Status: plugins.ConnectionValidationStatus_SUCCESS}, nil
}
