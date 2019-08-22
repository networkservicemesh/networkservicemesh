package plugins

import (
	"context"
	"fmt"
	"sync"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/plugins"
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
	logrus.Infof("PLUGINS: calling UpdateConnection")
	for name, plugin := range cpm.getClients() {
		logrus.Infof("PLUGINS: calling UpdateConnection on %s", name)
		pluginCtx, cancel := context.WithTimeout(ctx, pluginCallTimeout)

		var err error
		wrapper, err = plugin.UpdateConnection(pluginCtx, wrapper)
		cancel()
		logrus.Infof("PLUGINS: calling UpdateConnection res: %v %v", wrapper, err)

		if err != nil {
			return nil, fmt.Errorf("'%s' connection plugin returned an error: %v", name, err)
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
			return nil, fmt.Errorf("'%s' connection plugin returned an error: %v", name, err)
		}

		if result.GetStatus() != plugins.ConnectionValidationStatus_SUCCESS {
			return result, nil
		}
	}
	return &plugins.ConnectionValidationResult{Status: plugins.ConnectionValidationStatus_SUCCESS}, nil
}
