package plugins

import (
	"context"
	"fmt"
	"sync"

	"google.golang.org/grpc"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/plugins"
)

// RequestPluginManager transmits each method call to all registered request plugins
type RequestPluginManager interface {
	PluginManager
	UpdateRequest(context.Context, *plugins.RequestWrapper) (*plugins.RequestWrapper, error)
}

type requestPluginManager struct {
	sync.RWMutex
	pluginClients map[string]plugins.RequestPluginClient
}

func createRequestPluginManager() RequestPluginManager {
	return &requestPluginManager{
		pluginClients: make(map[string]plugins.RequestPluginClient),
	}
}

func (rpm *requestPluginManager) Register(name string, conn *grpc.ClientConn) error {
	client := plugins.NewRequestPluginClient(conn)
	return rpm.addClient(name, client)
}

func (rpm *requestPluginManager) addClient(name string, client plugins.RequestPluginClient) error {
	rpm.Lock()
	defer rpm.Unlock()

	if _, ok := rpm.pluginClients[name]; ok {
		return fmt.Errorf("already have a request plugin with the same name")
	}

	rpm.pluginClients[name] = client
	return nil
}

func (rpm *requestPluginManager) getClients() map[string]plugins.RequestPluginClient {
	rpm.RLock()
	defer rpm.RUnlock()

	return rpm.pluginClients
}

func (rpm *requestPluginManager) UpdateRequest(ctx context.Context, wrapper *plugins.RequestWrapper) (*plugins.RequestWrapper, error) {
	for name, plugin := range rpm.getClients() {
		pluginCtx, cancel := context.WithTimeout(ctx, pluginCallTimeout)

		var err error
		wrapper, err = plugin.UpdateRequest(pluginCtx, wrapper)
		cancel()

		if err != nil {
			return nil, fmt.Errorf("'%s' request plugin returned an error: %v", name, err)
		}
	}
	return wrapper, nil
}
