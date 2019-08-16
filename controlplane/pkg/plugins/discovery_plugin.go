package plugins

import (
	"context"
	"fmt"
	"sync"

	"google.golang.org/grpc"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/plugins"
)

// DiscoveryPluginManager transmits each method call to all registered discovery plugins
type DiscoveryPluginManager interface {
	PluginManager
	FindNetworkService(context.Context, *plugins.FindNetworkServiceRequest) (*plugins.FindNetworkServiceResponse, error)
}

type discoveryPluginManager struct {
	sync.RWMutex
	pluginClients map[string]plugins.DiscoveryPluginClient
}

func createDiscoveryPluginManager() DiscoveryPluginManager {
	return &discoveryPluginManager{
		pluginClients: make(map[string]plugins.DiscoveryPluginClient),
	}
}

func (dpm *discoveryPluginManager) Register(name string, conn *grpc.ClientConn) error {
	client := plugins.NewDiscoveryPluginClient(conn)
	return dpm.addClient(name, client)
}

func (dpm *discoveryPluginManager) addClient(name string, client plugins.DiscoveryPluginClient) error {
	dpm.Lock()
	defer dpm.Unlock()

	if _, ok := dpm.pluginClients[name]; ok {
		return fmt.Errorf("already have a discovery plugin with the same name")
	}

	dpm.pluginClients[name] = client
	return nil
}

func (dpm *discoveryPluginManager) getClients() map[string]plugins.DiscoveryPluginClient {
	dpm.RLock()
	defer dpm.RUnlock()

	return dpm.pluginClients
}

func (dpm *discoveryPluginManager) FindNetworkService(ctx context.Context, request *plugins.FindNetworkServiceRequest) (*plugins.FindNetworkServiceResponse, error) {
	for name, plugin := range dpm.getClients() {
		pluginCtx, cancel := context.WithTimeout(ctx, pluginCallTimeout)

		response, err := plugin.FindNetworkService(pluginCtx, request)
		cancel()

		if err != nil {
			return nil, fmt.Errorf("'%s' discovery plugin returned an error: %v", name, err)
		}

		if response.GetFound() {
			return response, nil
		}
	}
	return &plugins.FindNetworkServiceResponse{Found: false}, nil
}
