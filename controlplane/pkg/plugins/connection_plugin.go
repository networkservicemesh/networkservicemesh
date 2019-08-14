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
	UpdateConnection(context.Context, *plugins.ConnectionInfo) (*plugins.ConnectionInfo, error)
	ValidateConnection(context.Context, *plugins.ConnectionInfo) (*plugins.ConnectionValidationResult, error)
}

type connectionPluginManager struct {
	sync.RWMutex
	pluginClients []plugins.ConnectionPluginClient
}

func (cpm *connectionPluginManager) Register(conn *grpc.ClientConn) {
	client := plugins.NewConnectionPluginClient(conn)
	cpm.addClient(client)
}

func (cpm *connectionPluginManager) addClient(client plugins.ConnectionPluginClient) {
	cpm.Lock()
	defer cpm.Unlock()

	cpm.pluginClients = append(cpm.pluginClients, client)
}

func (cpm *connectionPluginManager) getClients() []plugins.ConnectionPluginClient {
	cpm.RLock()
	defer cpm.RUnlock()

	return cpm.pluginClients
}

func (cpm *connectionPluginManager) UpdateConnection(ctx context.Context, info *plugins.ConnectionInfo) (*plugins.ConnectionInfo, error) {
	for _, plugin := range cpm.getClients() {
		pluginCtx, cancel := context.WithTimeout(ctx, pluginCallTimeout)

		var err error
		info, err = plugin.UpdateConnection(pluginCtx, info)
		cancel()

		if err != nil {
			return nil, fmt.Errorf("connection plugin returned an error: %v", err)
		}
	}
	return info, nil
}

func (cpm *connectionPluginManager) ValidateConnection(ctx context.Context, info *plugins.ConnectionInfo) (*plugins.ConnectionValidationResult, error) {
	for _, plugin := range cpm.getClients() {
		pluginCtx, cancel := context.WithTimeout(ctx, pluginCallTimeout)

		result, err := plugin.ValidateConnection(pluginCtx, info)
		cancel()

		if err != nil {
			return nil, fmt.Errorf("connection plugin returned an error: %v", err)
		}

		if result.GetStatus() != plugins.ConnectionValidationStatus_SUCCESS {
			return result, nil
		}
	}
	return &plugins.ConnectionValidationResult{Status: plugins.ConnectionValidationStatus_SUCCESS}, nil
}
